package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vaultpay_http_requests_total",
			Help: "Total volume of incoming HTTP requests to VaultPay",
		},
		[]string{"path", "method", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "vaultpay_http_request_duration_seconds",
			Help:    "Latency tracking matrix measuring API request processing speeds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"path", "method"},
	)
)

type responseWriterInterceptor struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriterInterceptor) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func MetricsTracker(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		interceptor := &responseWriterInterceptor{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(interceptor, r)

		duration := time.Since(startTime).Seconds()
		routePath := r.URL.Path
		method := r.Method
		statusStr := strconv.Itoa(interceptor.statusCode)

		httpRequestsTotal.WithLabelValues(routePath, method, statusStr).Inc()
		httpRequestDuration.WithLabelValues(routePath, method).Observe(duration)
	})
}
