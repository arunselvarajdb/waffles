package metrics

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waffles/waffles/pkg/logger"
)

func TestNewServer(t *testing.T) {
	t.Run("creates server with valid parameters", func(t *testing.T) {
		reg := NewRegistry()
		log := logger.NewNopLogger()

		server := NewServer(reg, 9090, log)

		require.NotNil(t, server)
		assert.Equal(t, 9090, server.port)
		assert.NotNil(t, server.registry)
		assert.NotNil(t, server.logger)
	})

	t.Run("creates server with different port", func(t *testing.T) {
		reg := NewRegistry()
		log := logger.NewNopLogger()

		server := NewServer(reg, 8080, log)

		require.NotNil(t, server)
		assert.Equal(t, 8080, server.port)
	})
}

func TestServer_StartAndShutdown(t *testing.T) {
	t.Run("starts and shuts down cleanly", func(t *testing.T) {
		reg := NewRegistry()
		log := logger.NewNopLogger()
		server := NewServer(reg, 9091, log)

		ctx := context.Background()

		// Start server
		err := server.Start(ctx)
		require.NoError(t, err)

		// Give server time to start
		time.Sleep(100 * time.Millisecond)

		// Verify server is running
		resp, err := http.Get("http://localhost:9091/health")
		if err == nil {
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			resp.Body.Close()
		}

		// Verify metrics endpoint
		resp, err = http.Get("http://localhost:9091/metrics")
		if err == nil {
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			resp.Body.Close()
		}

		// Shutdown server
		err = server.Shutdown(ctx)
		require.NoError(t, err)
	})

	t.Run("shutdown with nil httpServer returns nil", func(t *testing.T) {
		reg := NewRegistry()
		log := logger.NewNopLogger()
		server := NewServer(reg, 9092, log)

		// Don't start the server, just try to shutdown
		err := server.Shutdown(context.Background())

		assert.NoError(t, err)
	})
}

func TestServer_Endpoints(t *testing.T) {
	t.Run("health endpoint returns OK", func(t *testing.T) {
		reg := NewRegistry()
		log := logger.NewNopLogger()
		server := NewServer(reg, 9093, log)

		ctx := context.Background()
		err := server.Start(ctx)
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		resp, err := http.Get("http://localhost:9093/health")
		if err == nil {
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			resp.Body.Close()
		}

		_ = server.Shutdown(ctx)
	})

	t.Run("metrics endpoint returns prometheus data", func(t *testing.T) {
		reg := NewRegistry()
		log := logger.NewNopLogger()
		server := NewServer(reg, 9094, log)

		ctx := context.Background()
		err := server.Start(ctx)
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		resp, err := http.Get("http://localhost:9094/metrics")
		if err == nil {
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			resp.Body.Close()
		}

		_ = server.Shutdown(ctx)
	})
}
