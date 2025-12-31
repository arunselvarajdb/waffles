-- Add key_prefix column to api_keys table for displaying obfuscated key
-- Format: mcpgw_<first8>...<last4> for user identification

ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS key_prefix VARCHAR(50);

-- Add comment for documentation
COMMENT ON COLUMN api_keys.key_prefix IS 'Obfuscated key prefix for display (e.g., mcpgw_abc12345...6789)';
