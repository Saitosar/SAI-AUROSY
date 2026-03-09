package observability

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)
	requestsErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_errors_total",
			Help: "Total number of HTTP error responses (4xx, 5xx)",
		},
		[]string{"method", "path", "status"},
	)
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

func init() {
	prometheus.MustRegister(requestsTotal, requestsErrorsTotal, requestDuration)
}

// MetricsMiddleware records Prometheus metrics for each request.
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		duration := time.Since(start).Seconds()
		path := routePath(r)
		method := r.Method
		status := strconv.Itoa(rw.status)
		requestsTotal.WithLabelValues(method, path, status).Inc()
		if rw.status >= 400 {
			requestsErrorsTotal.WithLabelValues(method, path, status).Inc()
		}
		requestDuration.WithLabelValues(method, path).Observe(duration)
	})
}

func routePath(r *http.Request) string {
	if route := mux.CurrentRoute(r); route != nil {
		if path, err := route.GetPathTemplate(); err == nil {
			return path
		}
	}
	return r.URL.Path
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// Handler returns the Prometheus metrics handler.
func Handler() http.Handler {
	return promhttp.Handler()
}
