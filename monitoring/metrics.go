package monitoring

import (
	"runtime"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

// MetricsCollector holds all Prometheus metrics
type MetricsCollector struct {
	// HTTP request metrics
	HTTPRequestDuration *prometheus.HistogramVec
	HTTPRequestTotal    *prometheus.CounterVec
	HTTPResponseSize    *prometheus.HistogramVec

	// Database metrics
	DBQueryDuration      *prometheus.HistogramVec
	DBConnectionPoolSize *prometheus.GaugeVec

	// Authentication metrics
	AuthRequestsTotal *prometheus.CounterVec
	AuthFailuresTotal *prometheus.CounterVec

	// Tenant metrics
	TenantOperationsTotal *prometheus.CounterVec
	ActiveTenants         prometheus.Gauge

	// System metrics
	MemoryUsage prometheus.Gauge
	Goroutines  prometheus.Gauge
}

var (
	metrics *MetricsCollector
	logger  *logrus.Logger
)

// InitMetrics initializes the Prometheus metrics collector
func InitMetrics() {
	metrics = &MetricsCollector{
		// HTTP request metrics
		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "Duration of HTTP requests in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "endpoint", "status_code", "tenant_id"},
		),
		HTTPRequestTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status_code", "tenant_id"},
		),
		HTTPResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_response_size_bytes",
				Help:    "Size of HTTP responses in bytes",
				Buckets: []float64{100, 1000, 10000, 100000, 1000000},
			},
			[]string{"method", "endpoint", "tenant_id"},
		),

		// Database metrics
		DBQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "db_query_duration_seconds",
				Help:    "Duration of database queries in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"operation", "table", "tenant_id"},
		),
		DBConnectionPoolSize: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "db_connection_pool_size",
				Help: "Size of database connection pools",
			},
			[]string{"database_type", "tenant_id"},
		),

		// Authentication metrics
		AuthRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "auth_requests_total",
				Help: "Total number of authentication requests",
			},
			[]string{"auth_type", "result", "tenant_id"},
		),
		AuthFailuresTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "auth_failures_total",
				Help: "Total number of authentication failures",
			},
			[]string{"auth_type", "reason", "tenant_id"},
		),

		// Tenant metrics
		TenantOperationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "tenant_operations_total",
				Help: "Total number of tenant operations",
			},
			[]string{"operation", "result"},
		),
		ActiveTenants: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "active_tenants",
				Help: "Number of active tenants",
			},
		),

		// System metrics
		MemoryUsage: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "memory_usage_bytes",
				Help: "Current memory usage in bytes",
			},
		),
		Goroutines: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "goroutines_count",
				Help: "Number of active goroutines",
			},
		),
	}

	// Initialize structured logger
	logger = logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})
	logger.SetLevel(logrus.ErrorLevel)
}

// GetMetrics returns the metrics collector instance
func GetMetrics() *MetricsCollector {
	return metrics
}

// GetLogger returns the structured logger instance
func GetLogger() *logrus.Logger {
	return logger
}

// Middleware returns a Gin middleware for collecting HTTP metrics
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		tenantID := c.GetHeader("X-Tenant-ID")
		if tenantID == "" {
			tenantID = "unknown"
		}

		// Extract request ID from context or generate one
		requestID := c.GetString("request_id")
		if requestID == "" {
			requestID = "unknown"
		}

		c.Next()

		duration := time.Since(start).Seconds()
		statusCode := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method
		endpoint := c.FullPath()
		if endpoint == "" {
			endpoint = "unknown"
		}

		// Record metrics
		if metrics != nil {
			metrics.HTTPRequestDuration.WithLabelValues(method, endpoint, statusCode, tenantID).Observe(duration)
			metrics.HTTPRequestTotal.WithLabelValues(method, endpoint, statusCode, tenantID).Inc()
			metrics.HTTPResponseSize.WithLabelValues(method, endpoint, tenantID).Observe(float64(c.Writer.Size()))
		}

		// Suppress noisy structured logging; metrics capture visibility for failures
	}
}

// RecordDBQuery records database query metrics
func RecordDBQuery(operation, table, tenantID string, duration time.Duration) {
	if metrics != nil {
		metrics.DBQueryDuration.WithLabelValues(operation, table, tenantID).Observe(duration.Seconds())
	}
}

// RecordAuthRequest records authentication request metrics
func RecordAuthRequest(authType, result, tenantID string) {
	if metrics != nil {
		metrics.AuthRequestsTotal.WithLabelValues(authType, result, tenantID).Inc()
	}
}

// RecordAuthFailure records authentication failure metrics
func RecordAuthFailure(authType, reason, tenantID string) {
	if metrics != nil {
		metrics.AuthFailuresTotal.WithLabelValues(authType, reason, tenantID).Inc()
	}
}

// RecordTenantOperation records tenant operation metrics
func RecordTenantOperation(operation, result string) {
	if metrics != nil {
		metrics.TenantOperationsTotal.WithLabelValues(operation, result).Inc()
	}
}

// UpdateSystemMetrics updates system-level metrics
func UpdateSystemMetrics() {
	if metrics != nil {
		metrics.Goroutines.Set(float64(runtime.NumGoroutine()))

		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		metrics.MemoryUsage.Set(float64(m.Alloc))
	}
}
