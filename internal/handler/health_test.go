package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waffles/waffles/pkg/logger"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ======================== Mock implementations ========================

type mockDBHealthChecker struct {
	message          string
	totalConnections int32
	idleConnections  int32
	maxConnections   int32
	healthy          bool
}

func (m *mockDBHealthChecker) Health(ctx context.Context) DatabaseHealthStatus {
	return DatabaseHealthStatus{
		Healthy:          m.healthy,
		TotalConnections: m.totalConnections,
		IdleConnections:  m.idleConnections,
		MaxConnections:   m.maxConnections,
		Message:          m.message,
	}
}

// ======================== Tests ========================

func TestNewHealthHandler(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("creates handler with nil db", func(t *testing.T) {
		handler := NewHealthHandler(nil, log)

		require.NotNil(t, handler)
		assert.Nil(t, handler.db)
		assert.NotNil(t, handler.logger)
	})

	t.Run("creates handler with interface", func(t *testing.T) {
		mockDB := &mockDBHealthChecker{healthy: true}
		handler := NewHealthHandlerWithInterface(mockDB, log)

		require.NotNil(t, handler)
		assert.NotNil(t, handler.db)
		assert.NotNil(t, handler.logger)
	})
}

func TestHealthHandler_Health(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("returns healthy status", func(t *testing.T) {
		handler := NewHealthHandler(nil, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/health", nil)

		handler.Health(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "healthy", response["status"])
		assert.Equal(t, "waffles", response["service"])
	})
}

func TestHealthHandler_Ready(t *testing.T) {
	log := logger.NewNopLogger()

	t.Run("returns not ready when database is nil", func(t *testing.T) {
		handler := NewHealthHandler(nil, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/ready", nil)

		handler.Ready(c)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "not_ready", response["status"])

		checks := response["checks"].(map[string]interface{})
		dbCheck := checks["database"].(map[string]interface{})
		assert.False(t, dbCheck["healthy"].(bool))
		assert.Equal(t, "database not configured", dbCheck["message"])
	})

	t.Run("returns ready when database is healthy", func(t *testing.T) {
		mockDB := &mockDBHealthChecker{
			healthy:          true,
			totalConnections: 10,
			idleConnections:  5,
			maxConnections:   100,
		}
		handler := NewHealthHandlerWithInterface(mockDB, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/ready", nil)

		handler.Ready(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "ready", response["status"])

		checks := response["checks"].(map[string]interface{})
		dbCheck := checks["database"].(map[string]interface{})
		assert.True(t, dbCheck["healthy"].(bool))
		assert.Equal(t, float64(10), dbCheck["total_connections"])
		assert.Equal(t, float64(5), dbCheck["idle_connections"])
		assert.Equal(t, float64(100), dbCheck["max_connections"])
	})

	t.Run("returns not ready when database is unhealthy", func(t *testing.T) {
		mockDB := &mockDBHealthChecker{
			healthy:          false,
			totalConnections: 0,
			idleConnections:  0,
			maxConnections:   100,
			message:          "connection failed",
		}
		handler := NewHealthHandlerWithInterface(mockDB, log)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/ready", nil)

		handler.Ready(c)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "not_ready", response["status"])

		checks := response["checks"].(map[string]interface{})
		dbCheck := checks["database"].(map[string]interface{})
		assert.False(t, dbCheck["healthy"].(bool))
		assert.Equal(t, "connection failed", dbCheck["message"])
	})
}

func TestDatabaseHealthStatus(t *testing.T) {
	t.Run("struct fields", func(t *testing.T) {
		status := DatabaseHealthStatus{
			Healthy:          true,
			TotalConnections: 10,
			IdleConnections:  5,
			MaxConnections:   100,
			Message:          "all good",
		}

		assert.True(t, status.Healthy)
		assert.Equal(t, int32(10), status.TotalConnections)
		assert.Equal(t, int32(5), status.IdleConnections)
		assert.Equal(t, int32(100), status.MaxConnections)
		assert.Equal(t, "all good", status.Message)
	})
}
