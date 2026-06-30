package handler

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	clients = make(map[string]*client)
	mu      sync.Mutex
)

// BackgroundCleaner evicts stale clients from memory to prevent memory leaks
func init() {
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			mu.Lock()
			for ip, c := range clients {
				if time.Since(c.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()
}

// RateLimitMiddleware enforces a strict quota of 5 requests per second with a burst allowance of 10
func RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { // Extract user IP address safely
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			WriteProblemResponse(w, r.URL.Path, http.StatusBadRequest, "Invalid Remote Address", "Could not evaluate client origin IP pointer mappings.")
			return
		}

		mu.Lock()
		if _, exists := clients[ip]; !exists {
			// rate.Limit(5) means 5 tokens refilled per second, with a burst capacity of 10
			clients[ip] = &client{limiter: rate.NewLimiter(rate.Limit(5), 10)}
		}
		clients[ip].lastSeen = time.Now()

		if !clients[ip].limiter.Allow() {
			mu.Unlock()
			w.Header().Set("Retry-After", "1")
			WriteProblemResponse(w, r.URL.Path, http.StatusTooManyRequests, "Rate Limit Exceeded", "You have generated too many automated concurrent queries. Please wait before retrying.")
			return
		}
		mu.Unlock()

		next.ServeHTTP(w, r)
	})
}

// CORSSecurityMiddleware ensures secure, authorized cross-origin requests from frontend apps
func CORSSecurityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*") // Adjust to specific frontend domains in production
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight browser check requests cleanly
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
