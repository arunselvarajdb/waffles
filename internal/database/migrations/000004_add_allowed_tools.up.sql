-- Add allowed_tools column to mcp_servers table
-- This stores a list of tool names that users are allowed to access
-- If NULL or empty, all tools are allowed (default behavior)
ALTER TABLE mcp_servers ADD COLUMN allowed_tools TEXT[];

-- Add index for efficient searching
CREATE INDEX idx_servers_allowed_tools ON mcp_servers USING GIN(allowed_tools);
