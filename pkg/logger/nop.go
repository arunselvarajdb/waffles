package logger

import "context"

// NopLogger is a logger that does nothing (useful for testing)
type NopLogger struct{}

// NewNopLogger creates a new no-op logger for testing
func NewNopLogger() Logger {
	return &NopLogger{}
}

func (n *NopLogger) Debug() Event       { return &NopEvent{} }
func (n *NopLogger) Info() Event        { return &NopEvent{} }
func (n *NopLogger) Warn() Event        { return &NopEvent{} }
func (n *NopLogger) Error() Event       { return &NopEvent{} }
func (n *NopLogger) Fatal() Event       { return &NopEvent{} }
func (n *NopLogger) With() Context      { return &NopContext{} }
func (n *NopLogger) WithContext(ctx context.Context) Logger { return n }

// NopEvent is an event that does nothing
type NopEvent struct{}

func (e *NopEvent) Str(key, val string) Event            { return e }
func (e *NopEvent) Int(key string, val int) Event        { return e }
func (e *NopEvent) Uint(key string, val uint) Event      { return e }
func (e *NopEvent) Bool(key string, val bool) Event      { return e }
func (e *NopEvent) Err(err error) Event                  { return e }
func (e *NopEvent) Dur(key string, val interface{}) Event { return e }
func (e *NopEvent) Any(key string, val interface{}) Event { return e }
func (e *NopEvent) Msg(msg string)                       {}
func (e *NopEvent) Msgf(format string, v ...interface{}) {}
func (e *NopEvent) Send()                                {}

// NopContext is a context that does nothing
type NopContext struct{}

func (c *NopContext) Str(key, val string) Context  { return c }
func (c *NopContext) Int(key string, val int) Context { return c }
func (c *NopContext) Bool(key string, val bool) Context { return c }
func (c *NopContext) Err(err error) Context          { return c }
func (c *NopContext) Logger() Logger                 { return &NopLogger{} }
