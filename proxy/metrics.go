package proxy

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	totalRequest = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "goproxy",
		Subsystem: "router",
		Name:      "request_total",
		Help:      "total request in HTTP",
	}, []string{"mode", "status"})
)

func init() {
	prometheus.MustRegister(totalRequest)
}

type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (mw metricsResponseWriter) status() string {
	return fmt.Sprintf("%d", mw.statusCode)
}

// NewMetricsResponseWriter creates custom metrics response writer.
func NewMetricsResponseWriter(w http.ResponseWriter) *metricsResponseWriter {
	// WriteHeader(int) is not called if our response implicitly returns 0, so
	// we default to that status code.
	return &metricsResponseWriter{w, 0}
}

// WriteHeader implements http.ResponseWriter.
func (mw *metricsResponseWriter) WriteHeader(code int) {
	mw.statusCode = code
	mw.ResponseWriter.WriteHeader(code)
}
