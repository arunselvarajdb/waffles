package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Registry holds all Prometheus metrics
type Registry struct {
	// HTTP Metrics
	HTTPRequestsTotal          *prometheus.CounterVec
	HTTPRequestDuration        *prometheus.HistogramVec
	HTTPRequestsInFlight       prometheus.Gauge

	// Gateway Metrics
	GatewayRequestsTotal       *prometheus.CounterVec
	GatewayRequestDuration     *prometheus.HistogramVec
	GatewayRequestsInFlight    *prometheus.GaugeVec
	GatewayServerHealthStatus  *prometheus.GaugeVec

	// Database Metrics (custom collectors will populate these)
	DBConnectionsOpen          prometheus.Gauge
	DBConnectionsInUse         prometheus.Gauge
	DBConnectionsIdle          prometheus.Gauge
	DBConnectionWaitCount      prometheus.Counter
	DBConnectionWaitDuration   prometheus.Histogram

	// Audit Metrics
	AuditLogsWrittenTotal      *prometheus.CounterVec
	AuditLogsWriteDuration     prometheus.Histogram

	// Registry Metrics
	RegistryServersTotal       *prometheus.GaugeVec
	RegistryHealthChecksTotal  *prometheus.CounterVec

	// Prometheus registry
	registry *prometheus.Registry
}

// NewRegistry creates a new metrics registry with all metrics defined
func NewRegistry() *Registry {
	reg := prometheus.NewRegistry()

	r := &Registry{
		registry: reg,
	}

	// HTTP Metrics
	r.HTTPRequestsTotal = promauto.With(reg).NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	r.HTTPRequestDuration = promauto.With(reg).NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets, // [.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10]
		},
		[]string{"method", "path"},
	)

	r.HTTPRequestsInFlight = promauto.With(reg).NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Current number of HTTP requests being served",
		},
	)

	// Gateway Metrics
	r.GatewayRequestsTotal = promauto.With(reg).NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_requests_total",
			Help: "Total number of gateway proxy requests",
		},
		[]string{"server_id", "server_name", "status"},
	)

	r.GatewayRequestDuration = promauto.With(reg).NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gateway_request_duration_seconds",
			Help:    "Gateway proxy request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"server_id", "server_name"},
	)

	r.GatewayRequestsInFlight = promauto.With(reg).NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gateway_requests_in_flight",
			Help: "Current number of gateway proxy requests being served",
		},
		[]string{"server_id", "server_name"},
	)

	r.GatewayServerHealthStatus = promauto.With(reg).NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gateway_server_health_status",
			Help: "Health status of MCP servers (1=healthy, 0=unhealthy)",
		},
		[]string{"server_id", "server_name", "status"},
	)

	// Database Metrics
	r.DBConnectionsOpen = promauto.With(reg).NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_open",
			Help: "Number of established connections both in use and idle",
		},
	)

	r.DBConnectionsInUse = promauto.With(reg).NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_in_use",
			Help: "Number of connections currently in use",
		},
	)

	r.DBConnectionsIdle = promauto.With(reg).NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_idle",
			Help: "Number of idle connections",
		},
	)

	r.DBConnectionWaitCount = promauto.With(reg).NewCounter(
		prometheus.CounterOpts{
			Name: "db_connections_wait_count",
			Help: "Total number of connections waited for",
		},
	)

	r.DBConnectionWaitDuration = promauto.With(reg).NewHistogram(
		prometheus.HistogramOpts{
			Name:    "db_connections_wait_duration_seconds",
			Help:    "Time spent waiting for connections",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		},
	)

	// Audit Metrics
	r.AuditLogsWrittenTotal = promauto.With(reg).NewCounterVec(
		prometheus.CounterOpts{
			Name: "audit_logs_written_total",
			Help: "Total number of audit logs written",
		},
		[]string{"status"}, // success, error
	)

	r.AuditLogsWriteDuration = promauto.With(reg).NewHistogram(
		prometheus.HistogramOpts{
			Name:    "audit_logs_write_duration_seconds",
			Help:    "Time spent writing audit logs",
			Buckets: []float64{.001, .005, .01, .025, .05, .1},
		},
	)

	// Registry Metrics
	r.RegistryServersTotal = promauto.With(reg).NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "registry_servers_total",
			Help: "Total number of registered MCP servers",
		},
		[]string{"status"}, // active, inactive
	)

	r.RegistryHealthChecksTotal = promauto.With(reg).NewCounterVec(
		prometheus.CounterOpts{
			Name: "registry_health_checks_total",
			Help: "Total number of health checks performed",
		},
		[]string{"server_id", "result"}, // success, failure
	)

	return r
}

// GetRegistry returns the underlying Prometheus registry
// This is needed for the HTTP handler to expose metrics
func (r *Registry) GetRegistry() *prometheus.Registry {
	return r.registry
}
