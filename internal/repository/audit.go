package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/waffles/waffles/internal/domain"
)

// AuditRepository handles audit log data access
type AuditRepository struct {
	pool *pgxpool.Pool
}

// NewAuditRepository creates a new audit repository
func NewAuditRepository(pool *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{pool: pool}
}

// Create inserts a new audit log entry
func (r *AuditRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	query := `
		INSERT INTO audit_logs (
			user_id, server_id, request_id, method, path,
			query_params, request_body, response_status, response_body,
			latency_ms, ip_address, user_agent, error_message
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12, $13
		)
		RETURNING id, created_at
	`

	err := r.pool.QueryRow(ctx, query,
		log.UserID,
		log.ServerID,
		log.RequestID,
		log.Method,
		log.Path,
		log.QueryParams,
		log.RequestBody,
		log.ResponseStatus,
		log.ResponseBody,
		log.LatencyMS,
		log.IPAddress,
		log.UserAgent,
		log.ErrorMessage,
	).Scan(&log.ID, &log.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

// Get retrieves a single audit log by ID
func (r *AuditRepository) Get(ctx context.Context, id string) (*domain.AuditLog, error) {
	query := `
		SELECT
			id, user_id, server_id, request_id, method, path,
			query_params, request_body, response_status, response_body,
			latency_ms, ip_address::TEXT, user_agent, error_message, created_at
		FROM audit_logs
		WHERE id = $1
	`

	log := &domain.AuditLog{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&log.ID,
		&log.UserID,
		&log.ServerID,
		&log.RequestID,
		&log.Method,
		&log.Path,
		&log.QueryParams,
		&log.RequestBody,
		&log.ResponseStatus,
		&log.ResponseBody,
		&log.LatencyMS,
		&log.IPAddress,
		&log.UserAgent,
		&log.ErrorMessage,
		&log.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get audit log: %w", err)
	}

	return log, nil
}

// List retrieves audit logs with filters
func (r *AuditRepository) List(ctx context.Context, filter domain.AuditLogFilter) ([]*domain.AuditLog, error) {
	query := `
		SELECT
			id, user_id, server_id, request_id, method, path,
			query_params, request_body, response_status, response_body,
			latency_ms, ip_address::TEXT, user_agent, error_message, created_at
		FROM audit_logs
		WHERE 1=1
	`

	args := []interface{}{}
	argIndex := 1

	// Add filters
	if filter.UserID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argIndex)
		args = append(args, *filter.UserID)
		argIndex++
	}

	if filter.ServerID != nil {
		query += fmt.Sprintf(" AND server_id = $%d", argIndex)
		args = append(args, *filter.ServerID)
		argIndex++
	}

	if filter.RequestID != nil {
		query += fmt.Sprintf(" AND request_id = $%d", argIndex)
		args = append(args, *filter.RequestID)
		argIndex++
	}

	if filter.Method != nil {
		query += fmt.Sprintf(" AND method = $%d", argIndex)
		args = append(args, *filter.Method)
		argIndex++
	}

	if filter.ResponseStatus != nil {
		query += fmt.Sprintf(" AND response_status = $%d", argIndex)
		args = append(args, *filter.ResponseStatus)
		argIndex++
	}

	if filter.FromDate != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, *filter.FromDate)
		argIndex++
	}

	if filter.ToDate != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, *filter.ToDate)
		argIndex++
	}

	// Order by created_at DESC
	query += " ORDER BY created_at DESC"

	// Add limit and offset
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filter.Offset)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list audit logs: %w", err)
	}
	defer rows.Close()

	logs := []*domain.AuditLog{}
	for rows.Next() {
		log := &domain.AuditLog{}
		err := rows.Scan(
			&log.ID,
			&log.UserID,
			&log.ServerID,
			&log.RequestID,
			&log.Method,
			&log.Path,
			&log.QueryParams,
			&log.RequestBody,
			&log.ResponseStatus,
			&log.ResponseBody,
			&log.LatencyMS,
			&log.IPAddress,
			&log.UserAgent,
			&log.ErrorMessage,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		logs = append(logs, log)
	}

	return logs, nil
}
