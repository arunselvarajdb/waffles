package logger

import (
	"context"
	"io"
)

// Logger defines the interface for structured logging
type Logger interface {
	// Debug logs a debug message
	Debug() Event
	// Info logs an info message
	Info() Event
	// Warn logs a warning message
	Warn() Event
	// Error logs an error message
	Error() Event
	// Fatal logs a fatal message and exits
	Fatal() Event

	// With returns a logger with the specified fields
	With() Context

	// WithContext returns a logger with request context fields
	WithContext(ctx context.Context) Logger
}

// Event represents a logging event
type Event interface {
	// Str adds a string field
	Str(key, val string) Event
	// Int adds an integer field
	Int(key string, val int) Event
	// Uint adds an unsigned integer field
	Uint(key string, val uint) Event
	// Bool adds a boolean field
	Bool(key string, val bool) Event
	// Err adds an error field
	Err(err error) Event
	// Dur adds a duration field
	Dur(key string, val interface{}) Event
	// Any adds any value field
	Any(key string, val interface{}) Event
	// Msg sends the event with a message
	Msg(msg string)
	// Msgf sends the event with a formatted message
	Msgf(format string, v ...interface{})
	// Send sends the event
	Send()
}

// Context represents a logging context for adding fields
type Context interface {
	// Str adds a string field
	Str(key, val string) Context
	// Int adds an integer field
	Int(key string, val int) Context
	// Bool adds a boolean field
	Bool(key string, val bool) Context
	// Err adds an error field
	Err(err error) Context
	// Logger returns the logger with the context fields
	Logger() Logger
}

// Level represents the logging level
type Level string

const (
	DebugLevel Level = "debug"
	InfoLevel  Level = "info"
	WarnLevel  Level = "warn"
	ErrorLevel Level = "error"
	FatalLevel Level = "fatal"
)

// Config holds logger configuration
type Config struct {
	Level  Level
	Format string // "json" or "console"
	Output io.Writer
}
