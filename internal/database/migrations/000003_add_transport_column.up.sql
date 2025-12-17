-- Add transport column to mcp_servers table
-- Values: 'http', 'sse', 'streamable_http', 'stdio'
ALTER TABLE mcp_servers ADD COLUMN IF NOT EXISTS transport VARCHAR(50) DEFAULT 'http';

-- Add comment for documentation
COMMENT ON COLUMN mcp_servers.transport IS 'Transport type: http (REST), sse (Server-Sent Events, deprecated), streamable_http (MCP 2025-11-25), stdio';
