package logger

import "context"

// NopLogger is a logger that does nothing (useful for testing)
type NopLogger struct{}

// NewNopLogger creates a new no-op logger for testing
func NewNopLogger() Logger {
	return &NopLogger{}
}

// NewNop is an alias for NewNopLogger for convenience in tests
func NewNop() Logger {
	return &NopLogger{}
}

func (n *NopLogger) Debug() Event                           { return &NopEvent{} }
func (n *NopLogger) Info() Event                            { return &NopEvent{} }
func (n *NopLogger) Warn() Event                            { return &NopEvent{} }
func (n *NopLogger) Error() Event                           { return &NopEvent{} }
func (n *NopLogger) Fatal() Event                           { return &NopEvent{} }
func (n *NopLogger) With() Context                          { return &NopContext{} }
func (n *NopLogger) WithContext(ctx context.Context) Logger { return n }

// NopEvent is an event that does nothing
type NopEvent struct{}

func (e *NopEvent) Str(_, _ string) Event             { return e }
func (e *NopEvent) Int(_ string, _ int) Event         { return e }
func (e *NopEvent) Uint(_ string, _ uint) Event       { return e }
func (e *NopEvent) Bool(_ string, _ bool) Event       { return e }
func (e *NopEvent) Err(_ error) Event                 { return e }
func (e *NopEvent) Dur(_ string, _ interface{}) Event { return e }
func (e *NopEvent) Any(_ string, _ interface{}) Event { return e }
func (e *NopEvent) Msg(_ string)                      {}
func (e *NopEvent) Msgf(_ string, _ ...interface{})   {}
func (e *NopEvent) Send()                             {}

// NopContext is a context that does nothing
type NopContext struct{}

func (c *NopContext) Str(_, _ string) Context       { return c }
func (c *NopContext) Int(_ string, _ int) Context   { return c }
func (c *NopContext) Bool(_ string, _ bool) Context { return c }
func (c *NopContext) Err(_ error) Context           { return c }
func (c *NopContext) Logger() Logger                { return &NopLogger{} }
