package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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

	// Update headers to fit the destination service network protocol
	r.URL.Host = target.Host
	r.URL.Scheme = target.Scheme
	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
	r.Host = target.Host

	proxy.ServeHTTP(w, r)
}
