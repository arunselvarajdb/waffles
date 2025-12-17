-- Remove transport column from mcp_servers table
ALTER TABLE mcp_servers DROP COLUMN IF EXISTS transport;
