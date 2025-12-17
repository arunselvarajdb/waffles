package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewZerolog(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "debug level json format",
			config: Config{
				Level:  DebugLevel,
				Format: "json",
			},
		},
		{
			name: "info level console format",
			config: Config{
				Level:  InfoLevel,
				Format: "console",
			},
		},
		{
			name: "error level",
			config: Config{
				Level:  ErrorLevel,
				Format: "json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := NewZerolog(tt.config)
			assert.NotNil(t, log)
		})
	}
}

func TestLogger_Levels(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Level:  DebugLevel,
		Format: "json",
		Output: &buf,
	}
	log := NewZerolog(cfg)

	// Test each log level
	log.Debug().Str("test", "debug").Msg("debug message")
	log.Info().Str("test", "info").Msg("info message")
	log.Warn().Str("test", "warn").Msg("warn message")
	log.Error().Str("test", "error").Msg("error message")

	output := buf.String()
	assert.Contains(t, output, "debug message")
	assert.Contains(t, output, "info message")
	assert.Contains(t, output, "warn message")
	assert.Contains(t, output, "error message")
}

func TestLogger_WithFields(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Level:  InfoLevel,
		Format: "json",
		Output: &buf,
	}
	log := NewZerolog(cfg)

	log.Info().
		Str("string_field", "value").
		Int("int_field", 42).
		Bool("bool_field", true).
		Msg("test message")

	// Parse JSON to verify fields
	var logEntry map[string]interface{}
	lines := bytes.Split(buf.Bytes(), []byte("\n"))
	err := json.Unmarshal(lines[0], &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "value", logEntry["string_field"])
	assert.Equal(t, float64(42), logEntry["int_field"]) // JSON numbers are float64
	assert.Equal(t, true, logEntry["bool_field"])
	assert.Equal(t, "test message", logEntry["message"])
}

func TestLogger_WithContext(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Level:  InfoLevel,
		Format: "json",
		Output: &buf,
	}
	log := NewZerolog(cfg)

	// Create context with request_id and user_id
	ctx := context.Background()
	ctx = WithRequestID(ctx, "req-123")
	ctx = WithUserID(ctx, "user-456")

	// Log with context
	log.WithContext(ctx).Info().Msg("context test")

	// Verify context values are in log
	var logEntry map[string]interface{}
	lines := bytes.Split(buf.Bytes(), []byte("\n"))
	err := json.Unmarshal(lines[0], &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "req-123", logEntry["request_id"])
	assert.Equal(t, "user-456", logEntry["user_id"])
}

func TestLogger_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Level:  WarnLevel, // Only warn and above
		Format: "json",
		Output: &buf,
	}
	log := NewZerolog(cfg)

	log.Debug().Msg("debug message") // Should be filtered
	log.Info().Msg("info message")   // Should be filtered
	log.Warn().Msg("warn message")   // Should appear
	log.Error().Msg("error message") // Should appear

	output := buf.String()
	assert.NotContains(t, output, "debug message")
	assert.NotContains(t, output, "info message")
	assert.Contains(t, output, "warn message")
	assert.Contains(t, output, "error message")
}

func TestContext_RequestID(t *testing.T) {
	ctx := context.Background()

	// Test setting and getting request ID
	requestID := "test-request-123"
	ctx = WithRequestID(ctx, requestID)

	retrieved := GetRequestID(ctx)
	assert.Equal(t, requestID, retrieved)

	// Test empty context
	emptyCtx := context.Background()
	emptyID := GetRequestID(emptyCtx)
	assert.Empty(t, emptyID)
}

func TestContext_UserID(t *testing.T) {
	ctx := context.Background()

	// Test setting and getting user ID
	userID := "test-user-456"
	ctx = WithUserID(ctx, userID)

	retrieved := GetUserID(ctx)
	assert.Equal(t, userID, retrieved)

	// Test empty context
	emptyCtx := context.Background()
	emptyID := GetUserID(emptyCtx)
	assert.Empty(t, emptyID)
}

func TestContext_BothIDs(t *testing.T) {
	ctx := context.Background()

	// Test setting both IDs
	ctx = WithRequestID(ctx, "req-123")
	ctx = WithUserID(ctx, "user-456")

	assert.Equal(t, "req-123", GetRequestID(ctx))
	assert.Equal(t, "user-456", GetUserID(ctx))
}
