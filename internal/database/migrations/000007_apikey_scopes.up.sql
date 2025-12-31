-- Add scope and restriction fields to api_keys table
-- Supports least-privilege principle for API key access control

-- Add scope columns to api_keys
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS scopes TEXT[] DEFAULT '{}';
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS allowed_servers UUID[] DEFAULT '{}';
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS allowed_tools TEXT[] DEFAULT '{}';
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS namespaces UUID[] DEFAULT '{}';
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS ip_whitelist TEXT[] DEFAULT '{}';
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS read_only BOOLEAN DEFAULT false;

-- Add description field for better key management
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS description TEXT DEFAULT '';

-- Create index for scope lookups
CREATE INDEX IF NOT EXISTS idx_api_keys_scopes ON api_keys USING GIN (scopes);
CREATE INDEX IF NOT EXISTS idx_api_keys_namespaces ON api_keys USING GIN (namespaces);

-- Add comments for documentation
COMMENT ON COLUMN api_keys.scopes IS 'Permission scopes: servers:read, servers:write, gateway:execute, audit:read, users:read, users:write';
COMMENT ON COLUMN api_keys.allowed_servers IS 'UUIDs of servers this key can access (empty = all allowed by role)';
COMMENT ON COLUMN api_keys.allowed_tools IS 'Tool names this key can execute (empty = all)';
COMMENT ON COLUMN api_keys.namespaces IS 'UUIDs of namespaces this key can access (empty = all allowed by role)';
COMMENT ON COLUMN api_keys.ip_whitelist IS 'CIDR ranges allowed to use this key (empty = any IP)';
COMMENT ON COLUMN api_keys.read_only IS 'If true, only GET/HEAD/OPTIONS requests allowed';
