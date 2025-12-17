-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Users and Authentication
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255),
    name VARCHAR(255),
    auth_provider VARCHAR(50) DEFAULT 'local', -- local, oauth, ldap
    external_id VARCHAR(255),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_external_id ON users(external_id);
CREATE INDEX idx_users_is_active ON users(is_active);

-- API Keys
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP,
    last_used_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX idx_api_keys_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_expires_at ON api_keys(expires_at);

-- Roles and Permissions
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE user_roles (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,
    resource VARCHAR(100) NOT NULL,      -- e.g., 'servers', 'users', 'audit'
    action VARCHAR(50) NOT NULL,         -- e.g., 'read', 'write', 'delete'
    description TEXT
);

CREATE INDEX idx_permissions_resource ON permissions(resource);

CREATE TABLE role_permissions (
    role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- MCP Server Registry
CREATE TABLE mcp_servers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    url VARCHAR(500) NOT NULL,
    protocol_version VARCHAR(50) DEFAULT '1.0.0',
    auth_type VARCHAR(50),               -- none, basic, bearer, oauth
    auth_config JSONB,                   -- Encrypted credentials
    health_check_url VARCHAR(500),
    health_check_interval INT DEFAULT 60, -- seconds
    timeout_seconds INT DEFAULT 30,
    max_connections INT DEFAULT 100,
    is_active BOOLEAN DEFAULT true,
    tags TEXT[],
    metadata JSONB,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_servers_name ON mcp_servers(name);
CREATE INDEX idx_servers_tags ON mcp_servers USING GIN(tags);
CREATE INDEX idx_servers_is_active ON mcp_servers(is_active);
CREATE INDEX idx_servers_created_by ON mcp_servers(created_by);

-- Server Access Control
CREATE TABLE server_permissions (
    server_id UUID REFERENCES mcp_servers(id) ON DELETE CASCADE,
    role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
    access_level VARCHAR(50) NOT NULL,   -- read, execute, admin
    PRIMARY KEY (server_id, role_id)
);

-- Server Health Status
CREATE TABLE server_health (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID REFERENCES mcp_servers(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL,         -- healthy, degraded, unhealthy
    response_time_ms INT,
    error_message TEXT,
    checked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_health_server_id ON server_health(server_id);
CREATE INDEX idx_health_checked_at ON server_health(checked_at);
CREATE INDEX idx_health_status ON server_health(status);

-- Audit Logs
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    server_id UUID REFERENCES mcp_servers(id) ON DELETE SET NULL,
    request_id VARCHAR(255) NOT NULL,    -- Correlation ID
    method VARCHAR(10) NOT NULL,
    path TEXT NOT NULL,
    query_params JSONB,
    request_body JSONB,
    response_status INT,
    response_body JSONB,
    latency_ms INT,
    ip_address INET,
    user_agent TEXT,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_audit_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_server_id ON audit_logs(server_id);
CREATE INDEX idx_audit_created_at ON audit_logs(created_at DESC);
CREATE INDEX idx_audit_request_id ON audit_logs(request_id);
CREATE INDEX idx_audit_response_status ON audit_logs(response_status);

-- Usage Analytics (Aggregated)
CREATE TABLE usage_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID REFERENCES mcp_servers(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    request_count INT DEFAULT 0,
    error_count INT DEFAULT 0,
    avg_latency_ms INT,
    total_bytes BIGINT DEFAULT 0,
    UNIQUE(server_id, user_id, date)
);

CREATE INDEX idx_usage_date ON usage_stats(date DESC);
CREATE INDEX idx_usage_server_id ON usage_stats(server_id);
CREATE INDEX idx_usage_user_id ON usage_stats(user_id);

-- Insert default roles
INSERT INTO roles (name, description) VALUES
    ('admin', 'Administrator with full system access'),
    ('operator', 'Operator who can manage servers and view logs'),
    ('user', 'Regular user who can use the gateway'),
    ('readonly', 'Read-only access to the system');

-- Insert default permissions
INSERT INTO permissions (name, resource, action, description) VALUES
    -- Server permissions
    ('servers.list', 'servers', 'list', 'List all servers'),
    ('servers.read', 'servers', 'read', 'Read server details'),
    ('servers.create', 'servers', 'create', 'Create new servers'),
    ('servers.update', 'servers', 'update', 'Update server configuration'),
    ('servers.delete', 'servers', 'delete', 'Delete servers'),
    ('servers.execute', 'servers', 'execute', 'Execute tools on servers'),

    -- User permissions
    ('users.list', 'users', 'list', 'List all users'),
    ('users.read', 'users', 'read', 'Read user details'),
    ('users.create', 'users', 'create', 'Create new users'),
    ('users.update', 'users', 'update', 'Update user details'),
    ('users.delete', 'users', 'delete', 'Delete users'),

    -- Audit permissions
    ('audit.read', 'audit', 'read', 'Read audit logs'),

    -- Analytics permissions
    ('analytics.read', 'analytics', 'read', 'Read usage analytics');

-- Assign permissions to roles
-- Admin gets all permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'admin';

-- Operator gets server and audit permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'operator'
AND p.name IN ('servers.list', 'servers.read', 'servers.create', 'servers.update', 'servers.execute', 'audit.read', 'analytics.read');

-- User gets basic server execute permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'user'
AND p.name IN ('servers.list', 'servers.read', 'servers.execute');

-- Readonly gets only read permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'readonly'
AND p.name IN ('servers.list', 'servers.read', 'audit.read', 'analytics.read');
