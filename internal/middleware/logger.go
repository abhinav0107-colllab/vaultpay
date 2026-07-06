package middleware

import (
	"net/http"
	"time"

	"github.com/abhinav0107-collab/vaultpay/internal/logger" // ◄ Import your new global logger package
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

func StructuredLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		startTime := time.Now()

		next.ServeHTTP(ww, r)

		// ◄ Blazing fast production structured logging output!
		logger.Log.Info("HTTP Request Processed",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", ww.Status()),
			zap.Duration("duration", time.Since(startTime)),
			zap.String("req_id", middleware.GetReqID(r.Context())),
		)
	})
}
