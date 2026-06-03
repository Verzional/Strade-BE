package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"github.com/golang-jwt/jwt/v5"
)

// Route defines the configuration schema for an endpoint
type Route struct {
	Path        string `json:"path"`         // e.g., "/api/v1/reviews"
	TargetURL   string `json:"target_url"`   // e.g., "http://review:3001"
	RequireAuth bool   `json:"require_auth"` // If true, validates JWT
}

// RouteConfig holds an array of routes from a configuration file
type RouteConfig struct {
	Routes []Route `json:"routes"`
}

var routes []Route

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 1. Load all independent route configurations
	err := loadRoutes("./routes")
	if err != nil {
		log.Fatalf("Failed to load route configurations: %v", err)
	}

	// 2. Set up the catch-all multiplexer routing handler
	http.HandleFunc("/", gatewayHandler)

	log.Printf("API Gateway successfully running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// loadRoutes reads all JSON files inside the target directory dynamically
func loadRoutes(dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			filePath := filepath.Join(dir, file.Name())
			jsonData, err := ioutil.ReadFile(filePath)
			if err != nil {
				log.Printf("Warning: Could not read route file %s: %v", file.Name(), err)
				continue
			}

			var config RouteConfig
			if err := json.Unmarshal(jsonData, &config); err != nil {
				log.Printf("Warning: Malformed JSON in file %s: %v", file.Name(), err)
				continue
			}

			for _, route := range config.Routes {
				log.Printf("Registered Route: [Auth: %t] %s -> %s", route.RequireAuth, route.Path, route.TargetURL)
				routes = append(routes, route)
			}
		}
	}
	return nil
}

// gatewayHandler inspects the request paths and reverse proxies them
func gatewayHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization")
	w.Header().Set("Access-Control-Allow-Credentials", "true")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	// Find matching registered route configuration
	var matchedRoute *Route
	for _, route := range routes {
		if strings.HasPrefix(r.URL.Path, route.Path) {
			matchedRoute = &route
			break
		}
	}

	if matchedRoute == nil {
		http.Error(w, "Gateway Error: Route Not Found", http.StatusNotFound)
		return
	}

	// Handle Authentication Middleware Barrier
	if matchedRoute.RequireAuth {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Gateway Error: Unauthorized (Missing or Invalid Token)", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		isValid, err := validateToken(tokenString)
		
		if err != nil || !isValid {
			log.Printf("Blocked invalid token attempt: %v", err)
			http.Error(w, "Gateway Error: Unauthorized (Invalid or Expired Token)", http.StatusUnauthorized)
			return
		}

		// Optional: Parse/Validate JWT token here or forward to User Service token validator
		// tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		// userId, err := validateToken(tokenString)
		// r.Header.Set("X-User-Id", userId)
	}

	// Execute Reverse Proxy forwarding
	target, err := url.Parse(matchedRoute.TargetURL)
	if err != nil {
		http.Error(w, "Gateway Error: Bad Target Configuration", http.StatusInternalServerError)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, err error) {
		log.Printf("Microservice is down/unreachable (%s): %v", target.Host, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable) // HTTP 503
		w.Write([]byte(`{"error": "Service Unavailable: The requested service is currently offline."}`))
	}

	// Update headers to fit the destination service network protocol
	r.URL.Host = target.Host
	r.URL.Scheme = target.Scheme
	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
	r.Host = target.Host

	proxy.ServeHTTP(w, r)
}

func validateToken(tokenString string) (bool, error) {
	// Get the secret key from environment variables
	secretKey := os.Getenv("JWT_SECRET") 
	if secretKey == "" {
		return false, fmt.Errorf("JWT_SECRET environment variable is missing in Gateway")
	}

	// Parse and validate the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Ensure the signing method is exactly what we expect (HMAC)
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		return false, err
	}

	return token.Valid, nil
}
