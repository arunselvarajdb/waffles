package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistry(t *testing.T) {
	reg := NewRegistry()

	require.NotNil(t, reg)

	// Verify HTTP metrics are initialized
	assert.NotNil(t, reg.HTTPRequestsTotal)
	assert.NotNil(t, reg.HTTPRequestDuration)
	assert.NotNil(t, reg.HTTPRequestsInFlight)

	// Verify Gateway metrics are initialized
	assert.NotNil(t, reg.GatewayRequestsTotal)
	assert.NotNil(t, reg.GatewayRequestDuration)
	assert.NotNil(t, reg.GatewayRequestsInFlight)
	assert.NotNil(t, reg.GatewayServerHealthStatus)

	// Verify Database metrics are initialized
	assert.NotNil(t, reg.DBConnectionsOpen)
	assert.NotNil(t, reg.DBConnectionsInUse)
	assert.NotNil(t, reg.DBConnectionsIdle)
	assert.NotNil(t, reg.DBConnectionWaitCount)
	assert.NotNil(t, reg.DBConnectionWaitDuration)

	// Verify Audit metrics are initialized
	assert.NotNil(t, reg.AuditLogsWrittenTotal)
	assert.NotNil(t, reg.AuditLogsWriteDuration)

	// Verify Registry metrics are initialized
	assert.NotNil(t, reg.RegistryServersTotal)
	assert.NotNil(t, reg.RegistryHealthChecksTotal)
}

func TestGetRegistry(t *testing.T) {
	reg := NewRegistry()

	promReg := reg.GetRegistry()

	require.NotNil(t, promReg)
	assert.IsType(t, &prometheus.Registry{}, promReg)
}

func TestHTTPMetrics_RecordRequest(t *testing.T) {
	reg := NewRegistry()

	// Record some HTTP requests
	reg.HTTPRequestsTotal.WithLabelValues("GET", "/api/v1/servers", "200").Inc()
	reg.HTTPRequestsTotal.WithLabelValues("POST", "/api/v1/servers", "201").Inc()
	reg.HTTPRequestsTotal.WithLabelValues("GET", "/api/v1/servers/123", "404").Inc()

	// Verify metrics can be gathered
	families, err := reg.GetRegistry().Gather()
	require.NoError(t, err)

	var found bool
	for _, mf := range families {
		if mf.GetName() == "http_requests_total" {
			found = true
			assert.GreaterOrEqual(t, len(mf.GetMetric()), 1)
		}
	}
	assert.True(t, found, "http_requests_total metric should be found")
}

func TestHTTPMetrics_RecordDuration(t *testing.T) {
	reg := NewRegistry()

	// Record some request durations
	reg.HTTPRequestDuration.WithLabelValues("GET", "/api/v1/servers").Observe(0.05)
	reg.HTTPRequestDuration.WithLabelValues("GET", "/api/v1/servers").Observe(0.1)
	reg.HTTPRequestDuration.WithLabelValues("POST", "/api/v1/servers").Observe(0.2)

	// Verify metrics can be gathered
	families, err := reg.GetRegistry().Gather()
	require.NoError(t, err)

	var found bool
	for _, mf := range families {
		if mf.GetName() == "http_request_duration_seconds" {
			found = true
		}
	}
	assert.True(t, found, "http_request_duration_seconds metric should be found")
}

func TestHTTPMetrics_InFlight(t *testing.T) {
	reg := NewRegistry()

	// Simulate in-flight requests
	reg.HTTPRequestsInFlight.Inc()
	reg.HTTPRequestsInFlight.Inc()
	reg.HTTPRequestsInFlight.Dec()

	// Verify metrics can be gathered
	families, err := reg.GetRegistry().Gather()
	require.NoError(t, err)

	var found bool
	for _, mf := range families {
		if mf.GetName() == "http_requests_in_flight" {
			found = true
			assert.Equal(t, 1, len(mf.GetMetric()))
			// Value should be 1 (2 increments - 1 decrement)
			assert.Equal(t, float64(1), mf.GetMetric()[0].GetGauge().GetValue())
		}
	}
	assert.True(t, found, "http_requests_in_flight metric should be found")
}

func TestGatewayMetrics_RecordRequest(t *testing.T) {
	reg := NewRegistry()

	// Record some gateway requests
	reg.GatewayRequestsTotal.WithLabelValues("server-1", "MCP Server 1", "success").Inc()
	reg.GatewayRequestsTotal.WithLabelValues("server-2", "MCP Server 2", "error").Inc()

	// Verify metrics can be gathered
	families, err := reg.GetRegistry().Gather()
	require.NoError(t, err)

	var found bool
	for _, mf := range families {
		if mf.GetName() == "gateway_requests_total" {
			found = true
		}
	}
	assert.True(t, found, "gateway_requests_total metric should be found")
}

func TestGatewayMetrics_RecordDuration(t *testing.T) {
	reg := NewRegistry()

	// Record gateway request durations
	reg.GatewayRequestDuration.WithLabelValues("server-1", "MCP Server 1").Observe(0.5)
	reg.GatewayRequestDuration.WithLabelValues("server-2", "MCP Server 2").Observe(1.0)

	families, err := reg.GetRegistry().Gather()
	require.NoError(t, err)

	var found bool
	for _, mf := range families {
		if mf.GetName() == "gateway_request_duration_seconds" {
			found = true
		}
	}
	assert.True(t, found, "gateway_request_duration_seconds metric should be found")
}

func TestGatewayMetrics_InFlight(t *testing.T) {
	reg := NewRegistry()

	// Track in-flight gateway requests
	reg.GatewayRequestsInFlight.WithLabelValues("server-1", "MCP Server 1").Inc()
	reg.GatewayRequestsInFlight.WithLabelValues("server-1", "MCP Server 1").Dec()

	families, err := reg.GetRegistry().Gather()
	require.NoError(t, err)

	var found bool
	for _, mf := range families {
		if mf.GetName() == "gateway_requests_in_flight" {
			found = true
		}
	}
	assert.True(t, found, "gateway_requests_in_flight metric should be found")
}

func TestGatewayMetrics_ServerHealthStatus(t *testing.T) {
	reg := NewRegistry()

	// Record server health status
	reg.GatewayServerHealthStatus.WithLabelValues("server-1", "MCP Server 1", "healthy").Set(1)
	reg.GatewayServerHealthStatus.WithLabelValues("server-2", "MCP Server 2", "unhealthy").Set(0)

	families, err := reg.GetRegistry().Gather()
	require.NoError(t, err)

	var found bool
	for _, mf := range families {
		if mf.GetName() == "gateway_server_health_status" {
			found = true
		}
	}
	assert.True(t, found, "gateway_server_health_status metric should be found")
}

func TestDatabaseMetrics(t *testing.T) {
	reg := NewRegistry()

	// Record database connection stats
	reg.DBConnectionsOpen.Set(10)
	reg.DBConnectionsInUse.Set(5)
	reg.DBConnectionsIdle.Set(5)
	reg.DBConnectionWaitCount.Add(3)
	reg.DBConnectionWaitDuration.Observe(0.01)

	families, err := reg.GetRegistry().Gather()
	require.NoError(t, err)

	expectedMetrics := []string{
		"db_connections_open",
		"db_connections_in_use",
		"db_connections_idle",
		"db_connections_wait_count",
		"db_connections_wait_duration_seconds",
	}

	foundMetrics := make(map[string]bool)
	for _, mf := range families {
		for _, expected := range expectedMetrics {
			if mf.GetName() == expected {
				foundMetrics[expected] = true
			}
		}
	}

	for _, expected := range expectedMetrics {
		assert.True(t, foundMetrics[expected], "metric %s should be found", expected)
	}
}

func TestAuditMetrics(t *testing.T) {
	reg := NewRegistry()

	// Record audit log writes
	reg.AuditLogsWrittenTotal.WithLabelValues("success").Inc()
	reg.AuditLogsWrittenTotal.WithLabelValues("success").Inc()
	reg.AuditLogsWrittenTotal.WithLabelValues("error").Inc()
	reg.AuditLogsWriteDuration.Observe(0.005)

	families, err := reg.GetRegistry().Gather()
	require.NoError(t, err)

	var foundTotal, foundDuration bool
	for _, mf := range families {
		if mf.GetName() == "audit_logs_written_total" {
			foundTotal = true
		}
		if mf.GetName() == "audit_logs_write_duration_seconds" {
			foundDuration = true
		}
	}

	assert.True(t, foundTotal, "audit_logs_written_total metric should be found")
	assert.True(t, foundDuration, "audit_logs_write_duration_seconds metric should be found")
}

func TestRegistryMetrics(t *testing.T) {
	reg := NewRegistry()

	// Record registry stats
	reg.RegistryServersTotal.WithLabelValues("active").Set(10)
	reg.RegistryServersTotal.WithLabelValues("inactive").Set(2)
	reg.RegistryHealthChecksTotal.WithLabelValues("server-1", "success").Inc()
	reg.RegistryHealthChecksTotal.WithLabelValues("server-1", "failure").Inc()

	families, err := reg.GetRegistry().Gather()
	require.NoError(t, err)

	var foundServers, foundHealthChecks bool
	for _, mf := range families {
		if mf.GetName() == "registry_servers_total" {
			foundServers = true
		}
		if mf.GetName() == "registry_health_checks_total" {
			foundHealthChecks = true
		}
	}

	assert.True(t, foundServers, "registry_servers_total metric should be found")
	assert.True(t, foundHealthChecks, "registry_health_checks_total metric should be found")
}

func TestNewRegistry_MultipleInstances(t *testing.T) {
	// Test that multiple registry instances can coexist
	reg1 := NewRegistry()
	reg2 := NewRegistry()

	require.NotNil(t, reg1)
	require.NotNil(t, reg2)

	// They should have separate prometheus registries
	assert.NotSame(t, reg1.GetRegistry(), reg2.GetRegistry())
}
