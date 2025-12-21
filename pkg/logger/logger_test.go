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

func TestLogger_AllEventMethods(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Level:  DebugLevel,
		Format: "json",
		Output: &buf,
	}
	log := NewZerolog(cfg)

	// Test Uint method
	log.Info().Uint("count", 100).Msg("uint test")

	// Test Err method
	log.Error().Err(assert.AnError).Msg("error test")

	// Test Dur method with time.Duration
	log.Info().Dur("duration", 5*1e9).Msg("duration test") // 5 seconds in nanoseconds

	// Test Any method
	log.Info().Any("data", map[string]int{"key": 42}).Msg("any test")

	// Test Msgf method
	log.Info().Msgf("formatted: %s %d", "test", 123)

	// Test Send method
	log.Info().Str("key", "value").Send()

	output := buf.String()
	assert.Contains(t, output, "uint test")
	assert.Contains(t, output, "error test")
	assert.Contains(t, output, "duration test")
	assert.Contains(t, output, "any test")
	assert.Contains(t, output, "formatted: test 123")
	assert.Contains(t, output, "key")
}

func TestLogger_WithContextMethod(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Level:  InfoLevel,
		Format: "json",
		Output: &buf,
	}
	log := NewZerolog(cfg)

	// Test With() to add persistent fields
	logWithFields := log.With().Str("service", "test-service").Int("version", 1).Logger()

	logWithFields.Info().Msg("message with persistent fields")

	output := buf.String()
	assert.Contains(t, output, "test-service")
	assert.Contains(t, output, "message with persistent fields")
}

func TestZeroContext_AllMethods(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Level:  InfoLevel,
		Format: "json",
		Output: &buf,
	}
	log := NewZerolog(cfg)

	// Test all Context methods
	ctxLogger := log.With().
		Str("string_ctx", "value").
		Int("int_ctx", 42).
		Bool("bool_ctx", true).
		Err(assert.AnError).
		Logger()

	ctxLogger.Info().Msg("context logger test")

	output := buf.String()
	assert.Contains(t, output, "string_ctx")
	assert.Contains(t, output, "context logger test")
}

func TestParseLevel(t *testing.T) {
	// This tests the parseLevel function indirectly through NewZerolog
	tests := []struct {
		name   string
		level  Level
		format string
	}{
		{"debug level", DebugLevel, "json"},
		{"info level", InfoLevel, "json"},
		{"warn level", WarnLevel, "json"},
		{"error level", ErrorLevel, "json"},
		{"fatal level", FatalLevel, "json"},
		{"unknown level defaults to info", Level("unknown"), "json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			cfg := Config{
				Level:  tt.level,
				Format: tt.format,
				Output: &buf,
			}
			log := NewZerolog(cfg)
			assert.NotNil(t, log)
		})
	}
}

func TestLogger_ConsoleFormat(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Level:  InfoLevel,
		Format: "console",
		Output: &buf,
	}
	log := NewZerolog(cfg)

	log.Info().Str("key", "value").Msg("console format test")

	output := buf.String()
	assert.Contains(t, output, "console format test")
}

func TestLogger_DefaultOutput(t *testing.T) {
	// Test that nil output defaults to stdout
	cfg := Config{
		Level:  InfoLevel,
		Format: "json",
		Output: nil, // Should default to os.Stdout
	}
	log := NewZerolog(cfg)
	assert.NotNil(t, log)
}

func TestLogger_WithContext_EmptyContext(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Level:  InfoLevel,
		Format: "json",
		Output: &buf,
	}
	log := NewZerolog(cfg)

	// Test with empty context (no request_id or user_id)
	ctx := context.Background()
	ctxLogger := log.WithContext(ctx)

	ctxLogger.Info().Msg("empty context test")

	// Parse JSON to verify no request_id or user_id
	var logEntry map[string]interface{}
	lines := bytes.Split(buf.Bytes(), []byte("\n"))
	err := json.Unmarshal(lines[0], &logEntry)
	require.NoError(t, err)

	_, hasRequestID := logEntry["request_id"]
	_, hasUserID := logEntry["user_id"]
	assert.False(t, hasRequestID)
	assert.False(t, hasUserID)
}

func TestLogger_WithContext_OnlyRequestID(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Level:  InfoLevel,
		Format: "json",
		Output: &buf,
	}
	log := NewZerolog(cfg)

	ctx := WithRequestID(context.Background(), "req-only")
	ctxLogger := log.WithContext(ctx)

	ctxLogger.Info().Msg("request id only test")

	var logEntry map[string]interface{}
	lines := bytes.Split(buf.Bytes(), []byte("\n"))
	err := json.Unmarshal(lines[0], &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "req-only", logEntry["request_id"])
	_, hasUserID := logEntry["user_id"]
	assert.False(t, hasUserID)
}

func TestLogger_WithContext_OnlyUserID(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Level:  InfoLevel,
		Format: "json",
		Output: &buf,
	}
	log := NewZerolog(cfg)

	ctx := WithUserID(context.Background(), "user-only")
	ctxLogger := log.WithContext(ctx)

	ctxLogger.Info().Msg("user id only test")

	var logEntry map[string]interface{}
	lines := bytes.Split(buf.Bytes(), []byte("\n"))
	err := json.Unmarshal(lines[0], &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "user-only", logEntry["user_id"])
	_, hasRequestID := logEntry["request_id"]
	assert.False(t, hasRequestID)
}
