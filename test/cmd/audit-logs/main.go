package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Connect to database
	connStr := "postgres://postgres:postgres@localhost:5432/mcp_gateway?sslmode=disable"
	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Query audit logs
	query := `
		SELECT id, method, path, response_status, latency_ms, ip_address, created_at
		FROM audit_logs
		ORDER BY created_at DESC
		LIMIT 10
	`

	rows, err := pool.Query(context.Background(), query)
	if err != nil {
		log.Fatalf("Failed to query audit logs: %v", err)
	}
	defer rows.Close()

	fmt.Println("=== Recent Audit Logs ===\n")
	count := 0
	for rows.Next() {
		var id, method, path, ipAddress string
		var responseStatus, latencyMS *int
		var createdAt string

		err := rows.Scan(&id, &method, &path, &responseStatus, &latencyMS, &ipAddress, &createdAt)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		count++
		fmt.Printf("%d. %s %s\n", count, method, path)
		if responseStatus != nil {
			fmt.Printf("   Status: %d", *responseStatus)
		}
		if latencyMS != nil {
			fmt.Printf(", Latency: %dms", *latencyMS)
		}
		fmt.Printf(", IP: %s\n", ipAddress)
		fmt.Printf("   Created: %s\n\n", createdAt)
	}

	if count == 0 {
		fmt.Println("No audit logs found")
	} else {
		fmt.Printf("Total: %d audit logs\n", count)
	}
}
