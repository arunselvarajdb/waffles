package domain

import (
	"encoding/json"
	"time"
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID             string
	UserID         *string // Nullable for Phase 2 (no auth)
	ServerID       *string // Nullable
	RequestID      string
	Method         string
	Path           string
	QueryParams    json.RawMessage // JSONB
	RequestBody    json.RawMessage // JSONB
	ResponseStatus *int            // Nullable
	ResponseBody   json.RawMessage // JSONB
	LatencyMS      *int            // Nullable
	IPAddress      string
	UserAgent      string
	ErrorMessage   *string // Nullable
	CreatedAt      time.Time
}

// AuditLogFilter represents filter criteria for querying audit logs
type AuditLogFilter struct {
	UserID         *string
	ServerID       *string
	RequestID      *string
	Method         *string
	ResponseStatus *int
	FromDate       *time.Time
	ToDate         *time.Time
	Limit          int
	Offset         int
}
