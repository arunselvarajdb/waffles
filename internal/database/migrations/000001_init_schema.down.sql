-- Drop tables in reverse order (respecting foreign key constraints)
DROP TABLE IF EXISTS usage_stats;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS server_health;
DROP TABLE IF EXISTS server_permissions;
DROP TABLE IF EXISTS mcp_servers;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS users;

-- Drop extension
DROP EXTENSION IF EXISTS "pgcrypto";
