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
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// --- CIRCUIT BREAKER ---

type CircuitBreaker struct {
	mu               sync.Mutex
	state            string // "CLOSED", "OPEN", "HALF_OPEN"
	failureCount     int
	failureThreshold int
	resetTimeout     time.Duration
	nextAttempt      time.Time
}

func NewCircuitBreaker(threshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:            "CLOSED",
		failureThreshold: threshold,
		resetTimeout:     timeout,
	}
}

func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == "OPEN" {
		if time.Now().After(cb.nextAttempt) {
			cb.state = "HALF_OPEN"
			return true
		}
		return false
	}
	return true
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state != "CLOSED" {
		log.Println("[Circuit Breaker] Service restored! Circuit state dropped to CLOSED.")
	}
	cb.failureCount = 0
	cb.state = "CLOSED"
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	if cb.failureCount >= cb.failureThreshold {
		cb.state = "OPEN"
		cb.nextAttempt = time.Now().Add(cb.resetTimeout)
		log.Printf("[Circuit Breaker] High failure rate! Circuit TRIPPED to OPEN. Cooldown: %v", cb.resetTimeout)
	}
}

// --- LOAD BALANCER ---

type Backend struct {
	URL *url.URL
	CB  *CircuitBreaker
}

type RoundRobinBalancer struct {
	backends []*Backend
	current  uint32
}

func (r *RoundRobinBalancer) NextBackend() *Backend {
	n := len(r.backends)
	if n == 0 {
		return nil
	}

	// Cycle through targets to find a healthy node
	for i := 0; i < n; i++ {
		idx := atomic.AddUint32(&r.current, 1) % uint32(n)
		backend := r.backends[idx]

		if backend.CB.AllowRequest() {
			return backend
		}
	}
	return nil // All backends are tripped open
}

// --- ROUTING SETUP ---

type Route struct {
	Path        string              `json:"path"`
	TargetURL   string              `json:"target_url,omitempty"`  // Fallback for single target
	TargetURLs  []string            `json:"target_urls,omitempty"` // Multiple targets for Load Balancing
	RequireAuth bool                `json:"require_auth"`
	Balancer    *RoundRobinBalancer `json:"-"`
}

type RouteConfig struct {
	Routes []Route `json:"routes"`
}

var routes []Route

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := loadRoutes("./routes"); err != nil {
		log.Fatalf("Critical error setting up gateway routes: %v", err)
	}

	http.HandleFunc("/", gatewayHandler)

	log.Printf("API Gateway successfully running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

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
				log.Printf("Skipping unreadable route configuration %s: %v", file.Name(), err)
				continue
			}

			var config RouteConfig
			if err := json.Unmarshal(jsonData, &config); err != nil {
				log.Printf("Skipping malformed route file %s: %v", file.Name(), err)
				continue
			}

			for _, route := range config.Routes {
				var targets []string
				if len(route.TargetURLs) > 0 {
					targets = route.TargetURLs
				} else if route.TargetURL != "" {
					targets = append(targets, route.TargetURL)
				}

				balancer := &RoundRobinBalancer{}
				for _, targetStr := range targets {
					parsedURL, err := url.Parse(targetStr)
					if err != nil {
						log.Printf("Invalid target URL syntax: %s", targetStr)
						continue
					}
					balancer.backends = append(balancer.backends, &Backend{
						URL: parsedURL,
						CB:  NewCircuitBreaker(3, 10*time.Second), // Trip after 3 fails, 10s cool down
					})
				}

				route.Balancer = balancer
				routes = append(routes, route)
				log.Printf("Configured Path Proxy: %s mapped to -> %v", route.Path, targets)
			}
		}
	}
	return nil
}

func gatewayHandler(w http.ResponseWriter, r *http.Request) {
	// Global CORS Rules
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization")
	w.Header().Set("Access-Control-Allow-Credentials", "true")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	var matchedRoute *Route
	for _, route := range routes {
		if strings.HasPrefix(r.URL.Path, route.Path) {
			matchedRoute = &route
			break
		}
	}

	if matchedRoute == nil {
		http.Error(w, `{"error":"Route Not Found"}`, http.StatusNotFound)
		return
	}

	if matchedRoute.RequireAuth {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, `{"error":"Unauthorized: Missing or invalid authentication token"}`, http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		isValid, err := validateToken(tokenString)
		if err != nil || !isValid {
			http.Error(w, `{"error":"Unauthorized: Access token has expired or is invalid"}`, http.StatusUnauthorized)
			return
		}
	}

	backend := matchedRoute.Balancer.NextBackend()
	if backend == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error":"Service Unavailable","message":"All backend instances are currently unresponsive. Circuit Breaker is OPEN."}`))
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(backend.URL)

	// Intercept bad target responses
	proxy.ModifyResponse = func(resp *http.Response) error {
		if resp.StatusCode >= 500 {
			backend.CB.RecordFailure()
		} else {
			backend.CB.RecordSuccess()
		}
		return nil
	}

	// Intercept dropped connections / crashed instances
	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
		backend.CB.RecordFailure()
		log.Printf("Proxy Connection Failure to %s: %v", backend.URL.Host, err)
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusServiceUnavailable)
		rw.Write([]byte(`{"error":"Service Unreachable","message":"The application service instance failed to respond."}`))
	}

	r.URL.Host = backend.URL.Host
	r.URL.Scheme = backend.URL.Scheme
	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
	r.Host = backend.URL.Host

	proxy.ServeHTTP(w, r)
}

func validateToken(tokenString string) (bool, error) {
	secretKey := os.Getenv("JWT_SECRET")
	if secretKey == "" {
		return false, fmt.Errorf("JWT_SECRET environment variable not initialized")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected parsing algorithm: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		return false, err
	}
	return token.Valid, nil
}