package logger

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// ZeroLogger wraps zerolog.Logger to implement Logger interface
type ZeroLogger struct {
	zl zerolog.Logger
}

// NewZerolog creates a new Zerolog-based logger
func NewZerolog(cfg Config) Logger {
	// Set global level
	level := parseLevel(cfg.Level)
	zerolog.SetGlobalLevel(level)

	// Configure time format
	zerolog.TimeFieldFormat = time.RFC3339

	// Set output
	output := cfg.Output
	if output == nil {
		output = os.Stdout
	}

	// Create logger based on format
	var zl zerolog.Logger
	if cfg.Format == "console" {
		zl = zerolog.New(zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: time.RFC3339,
		}).With().Timestamp().Logger()
	} else {
		zl = zerolog.New(output).With().Timestamp().Logger()
	}

	return &ZeroLogger{zl: zl}
}

// parseLevel converts Level to zerolog.Level
func parseLevel(level Level) zerolog.Level {
	switch level {
	case DebugLevel:
		return zerolog.DebugLevel
	case InfoLevel:
		return zerolog.InfoLevel
	case WarnLevel:
		return zerolog.WarnLevel
	case ErrorLevel:
		return zerolog.ErrorLevel
	case FatalLevel:
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}

// Debug logs a debug message
func (z *ZeroLogger) Debug() Event {
	return &ZeroEvent{ze: z.zl.Debug()}
}

// Info logs an info message
func (z *ZeroLogger) Info() Event {
	return &ZeroEvent{ze: z.zl.Info()}
}

// Warn logs a warning message
func (z *ZeroLogger) Warn() Event {
	return &ZeroEvent{ze: z.zl.Warn()}
}

// Error logs an error message
func (z *ZeroLogger) Error() Event {
	return &ZeroEvent{ze: z.zl.Error()}
}

// Fatal logs a fatal message
func (z *ZeroLogger) Fatal() Event {
	return &ZeroEvent{ze: z.zl.Fatal()}
}

// With returns a context for adding fields
func (z *ZeroLogger) With() Context {
	return &ZeroContext{zc: z.zl.With()}
}

// WithContext returns a logger with request context fields
func (z *ZeroLogger) WithContext(ctx context.Context) Logger {
	zl := z.zl

	// Extract request ID from context if present
	if reqID := GetRequestID(ctx); reqID != "" {
		zl = zl.With().Str("request_id", reqID).Logger()
	}

	// Extract user ID from context if present
	if userID := GetUserID(ctx); userID != "" {
		zl = zl.With().Str("user_id", userID).Logger()
	}

	return &ZeroLogger{zl: zl}
}

// ZeroEvent wraps zerolog.Event to implement Event interface
type ZeroEvent struct {
	ze *zerolog.Event
}

// Str adds a string field
func (e *ZeroEvent) Str(key, val string) Event {
	e.ze = e.ze.Str(key, val)
	return e
}

// Int adds an integer field
func (e *ZeroEvent) Int(key string, val int) Event {
	e.ze = e.ze.Int(key, val)
	return e
}

// Uint adds an unsigned integer field
func (e *ZeroEvent) Uint(key string, val uint) Event {
	e.ze = e.ze.Uint(key, val)
	return e
}

// Bool adds a boolean field
func (e *ZeroEvent) Bool(key string, val bool) Event {
	e.ze = e.ze.Bool(key, val)
	return e
}

// Err adds an error field
func (e *ZeroEvent) Err(err error) Event {
	e.ze = e.ze.Err(err)
	return e
}

// Dur adds a duration field
func (e *ZeroEvent) Dur(key string, val interface{}) Event {
	if d, ok := val.(time.Duration); ok {
		e.ze = e.ze.Dur(key, d)
	}
	return e
}

// Any adds any value field
func (e *ZeroEvent) Any(key string, val interface{}) Event {
	e.ze = e.ze.Interface(key, val)
	return e
}

// Msg sends the event with a message
func (e *ZeroEvent) Msg(msg string) {
	e.ze.Msg(msg)
}

// Msgf sends the event with a formatted message
func (e *ZeroEvent) Msgf(format string, v ...interface{}) {
	e.ze.Msgf(format, v...)
}

// Send sends the event
func (e *ZeroEvent) Send() {
	e.ze.Send()
}

// ZeroContext wraps zerolog.Context to implement Context interface
type ZeroContext struct {
	zc zerolog.Context
}

// Str adds a string field
func (c *ZeroContext) Str(key, val string) Context {
	c.zc = c.zc.Str(key, val)
	return c
}

// Int adds an integer field
func (c *ZeroContext) Int(key string, val int) Context {
	c.zc = c.zc.Int(key, val)
	return c
}

// Bool adds a boolean field
func (c *ZeroContext) Bool(key string, val bool) Context {
	c.zc = c.zc.Bool(key, val)
	return c
}

// Err adds an error field
func (c *ZeroContext) Err(err error) Context {
	c.zc = c.zc.Err(err)
	return c
}

// Logger returns the logger with the context fields
func (c *ZeroContext) Logger() Logger {
	return &ZeroLogger{zl: c.zc.Logger()}
}
