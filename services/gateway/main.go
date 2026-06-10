package main

import (
	"bytes"
	"encoding/json"
	"io"
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

// --- 1. CIRCUIT BREAKER ---
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

// --- 2. LOAD BALANCER ---
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

	for i := 0; i < n; i++ {
		idx := atomic.AddUint32(&r.current, 1) % uint32(n)
		backend := r.backends[idx]

		if backend.CB.AllowRequest() {
			return backend
		}
	}
	return nil
}

// --- 3. RETRY TRANSPORT (INVISIBLE FAILOVER) ---
type RetryTransport struct {
	Transport http.RoundTripper
	Balancer  *RoundRobinBalancer
}


func (t *RetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var lastErr error
	var lastResp *http.Response

	maxAttempts := len(t.Balancer.backends)
	if maxAttempts == 0 {
		return nil, fmt.Errorf("no backend targets configured")
	}

	// 1. SAFELY handle request bodies (Crucial fix for GET requests)
	var bodyBytes []byte
	var hasBody bool
	
	// Only read the body if it actually exists and is not an empty GET request
	if req.Body != nil && req.Body != http.NoBody {
		bodyBytes, _ = ioutil.ReadAll(req.Body)
		req.Body.Close()
		hasBody = true
	}

	for i := 0; i < maxAttempts; i++ {
		backend := t.Balancer.NextBackend()
		if backend == nil {
			break // All nodes are down or circuit breakers are OPEN
		}

		// Clone the request for this specific target
		attemptReq := req.Clone(req.Context())
		attemptReq.URL.Scheme = backend.URL.Scheme
		attemptReq.URL.Host = backend.URL.Host
		// attemptReq.Host = backend.URL.Host

		// 2. Only re-attach a body stream if the original request actually had one (POST/PUT)
		if hasBody {
			attemptReq.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
			attemptReq.GetBody = func() (io.ReadCloser, error) {
				return ioutil.NopCloser(bytes.NewBuffer(bodyBytes)), nil
			}
		} else {
			// Explicitly enforce that GET requests remain entirely body-less
			attemptReq.Body = http.NoBody
			attemptReq.GetBody = func() (io.ReadCloser, error) { return http.NoBody, nil }
		}

		// Execute the network request
		resp, err := t.Transport.RoundTrip(attemptReq)

		// Target is totally dead/unreachable
		if err != nil {
			log.Printf("[Gateway Retry] Connection to %s failed: %v. Switching...", backend.URL.Host, err)
			backend.CB.RecordFailure()
			lastErr = err
			continue 
		}

		// Target is online but crashing internally (HTTP 500+)
		if resp.StatusCode >= 500 {
			log.Printf("[Gateway Retry] Target %s returned %d. Switching...", backend.URL.Host, resp.StatusCode)
			backend.CB.RecordFailure()
			lastResp = resp
			continue 
		}

		// Success!
		backend.CB.RecordSuccess()
		return resp, nil
	}

	// If we exhausted all nodes, return the last error generated
	if lastResp != nil {
		return lastResp, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("all circuit breakers tripped open")
}

// --- 4. ROUTING SETUP ---
type Route struct {
	Path        string              `json:"path"`
	TargetURLs  []string            `json:"target_urls,omitempty"` 
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
				continue
			}

			var config RouteConfig
			if err := json.Unmarshal(jsonData, &config); err != nil {
				continue
			}

			for _, route := range config.Routes {
				balancer := &RoundRobinBalancer{}
				for _, targetStr := range route.TargetURLs {
					parsedURL, _ := url.Parse(targetStr)
					balancer.backends = append(balancer.backends, &Backend{
						URL: parsedURL,
						CB:  NewCircuitBreaker(3, 10*time.Second),
					})
				}

				route.Balancer = balancer
				routes = append(routes, route)
				log.Printf("Configured Proxy: %s -> %v", route.Path, route.TargetURLs)
			}
		}
	}
	return nil
}

func gatewayHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization")
	w.Header().Set("Access-Control-Allow-Credentials", "true")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	var matchedRoute *Route
	for i := range routes {
		if strings.HasPrefix(r.URL.Path, routes[i].Path) {
			matchedRoute = &routes[i]
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
			http.Error(w, `{"error":"Unauthorized: Access token has expired"}`, http.StatusUnauthorized)
			return
		}
	}

	// We no longer pull the backend here. We hand it over to the intelligent Reverse Proxy!
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			// Forward correct headers to the backend
			req.Header.Set("X-Forwarded-Host", req.Host)
		},
		Transport: &RetryTransport{
			Transport: http.DefaultTransport,
			Balancer:  matchedRoute.Balancer,
		},
		ErrorHandler: func(rw http.ResponseWriter, req *http.Request, err error) {
			log.Printf("Gateway Proxy Error: All retries failed - %v", err)
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusServiceUnavailable)
			rw.Write([]byte(`{"error":"Service Unreachable","message":"All backend instances are currently offline."}`))
		},
	}

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