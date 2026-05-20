package middleware

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// StructuredLogger intercepts incoming HTTP traffic and logs core execution metrics
func StructuredLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		startTime := time.Now()

		// Let the request proceed downstream to our handlers
		next.ServeHTTP(ww, r)

		// Print clean execution metrics after the request completes
		log.Printf("[HTTP] METHOD=%s PATH=%s STATUS=%d DURATION=%s REQ_ID=%s",
			r.Method,
			r.URL.Path,
			ww.Status(),
			time.Since(startTime).String(),
			middleware.GetReqID(r.Context()),
		)
	})
}
