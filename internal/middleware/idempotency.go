package middleware

import (
	"bytes"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

type IdempotencyMiddleware struct {
	rdb *redis.Client
}

func NewIdempotencyMiddleware(rdb *redis.Client) *IdempotencyMiddleware {
	return &IdempotencyMiddleware{rdb: rdb}
}

func (m *IdempotencyMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only enforce idempotency protections on state-changing POST actions
		if r.Method != http.MethodPost {
			next.ServeHTTP(w, r)
			return
		}

		idemKey := r.Header.Get("Idempotency-Key")
		if idemKey == "" {
			next.ServeHTTP(w, r)
			return
		}

		ctx := r.Context()
		// Check if we've seen this exact key in Redis cache memory
		cachedResponse, err := m.rdb.Get(ctx, "idem:"+idemKey).Result()
		if err == nil {
			// Key match found! Return cached data instantly to stop double-deduction
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Cache-Idempotent", "true")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(cachedResponse))
			return
		}

		// Intercept the handler's output data so we can mirror it into Redis cache memory
		rec := &responseRecorder{ResponseWriter: w, body: bytes.NewBuffer(nil)}
		next.ServeHTTP(rec, r)

		if rec.statusCode == http.StatusOK || rec.statusCode == http.StatusCreated {
			m.rdb.Set(ctx, "idem:"+idemKey, rec.body.String(), 24*time.Hour)
		}
	})
}

type responseRecorder struct {
	http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
}

func (rec *responseRecorder) WriteHeader(statusCode int) {
	rec.statusCode = statusCode
	rec.ResponseWriter.WriteHeader(statusCode)
}

func (rec *responseRecorder) Write(b []byte) (int, error) {
	rec.body.Write(b)
	return rec.ResponseWriter.Write(b)
}
