-- Remove allowed_tools column and index
DROP INDEX IF EXISTS idx_servers_allowed_tools;
ALTER TABLE mcp_servers DROP COLUMN IF EXISTS allowed_tools;
