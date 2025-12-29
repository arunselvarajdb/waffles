-- Remove scope and restriction fields from api_keys table

DROP INDEX IF EXISTS idx_api_keys_namespaces;
DROP INDEX IF EXISTS idx_api_keys_scopes;

ALTER TABLE api_keys DROP COLUMN IF EXISTS description;
ALTER TABLE api_keys DROP COLUMN IF EXISTS read_only;
ALTER TABLE api_keys DROP COLUMN IF EXISTS ip_whitelist;
ALTER TABLE api_keys DROP COLUMN IF EXISTS namespaces;
ALTER TABLE api_keys DROP COLUMN IF EXISTS allowed_tools;
ALTER TABLE api_keys DROP COLUMN IF EXISTS allowed_servers;
ALTER TABLE api_keys DROP COLUMN IF EXISTS scopes;
