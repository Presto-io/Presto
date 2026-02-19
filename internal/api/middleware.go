package api

import (
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// allowedOrigins for CORS whitelist (SEC-08)
var allowedOrigins = map[string]bool{
	"http://localhost:8080":   true,
	"http://localhost:5173":   true,
	"http://127.0.0.1:8080":  true,
	"http://127.0.0.1:5173":  true,
	"wails://wails":          true,
	"https://wails.localhost": true,
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Work-Dir")
			w.Header().Set("Vary", "Origin")
		}
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// authMiddleware enforces Bearer token auth for API routes (SEC-09)
func authMiddleware(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// No auth if no API key configured (desktop mode)
			if apiKey == "" {
				next.ServeHTTP(w, r)
				return
			}
			// Health endpoint always accessible
			if r.URL.Path == "/api/health" {
				next.ServeHTTP(w, r)
				return
			}
			// Skip auth for non-API paths (static files)
			if !strings.HasPrefix(r.URL.Path, "/api/") {
				next.ServeHTTP(w, r)
				return
			}
			// OPTIONS handled by CORS middleware before auth
			if r.Method == "OPTIONS" {
				next.ServeHTTP(w, r)
				return
			}
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") || auth[7:] != apiKey {
				writeJSONError(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// rateLimiter implements a token bucket rate limiter (SEC-19)
type rateLimiter struct {
	mu       sync.Mutex
	tokens   float64
	maxBurst float64
	rate     float64 // tokens per second
	lastTime time.Time
}

func newRateLimiter(rate float64, burst int) *rateLimiter {
	return &rateLimiter{
		tokens:   float64(burst),
		maxBurst: float64(burst),
		rate:     rate,
		lastTime: time.Now(),
	}
}

func (rl *rateLimiter) allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	rl.tokens += now.Sub(rl.lastTime).Seconds() * rl.rate
	rl.lastTime = now
	if rl.tokens > rl.maxBurst {
		rl.tokens = rl.maxBurst
	}
	if rl.tokens < 1 {
		return false
	}
	rl.tokens--
	return true
}

func rateLimitMiddleware(rl *rateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !rl.allow() {
				writeJSONError(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// dotfileFilterHandler blocks requests for hidden files/directories (SEC-27)
func dotfileFilterHandler(fs http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, part := range strings.Split(r.URL.Path, "/") {
			if strings.HasPrefix(part, ".") && part != "." && part != ".." {
				http.NotFound(w, r)
				return
			}
		}
		fs.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lw := &loggingResponseWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(lw, r)
		log.Printf("[%s] %s %s → %d (%s)", r.Method, r.URL.Path, r.RemoteAddr, lw.status, time.Since(start))
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (lw *loggingResponseWriter) WriteHeader(code int) {
	lw.status = code
	lw.ResponseWriter.WriteHeader(code)
}
