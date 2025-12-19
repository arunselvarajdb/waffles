#!/bin/bash
# Seed script for local testing - creates test users and RBAC configuration
# Usage: ./scripts/seed-users.sh
#
# Test Users:
#   admin@example.com / admin123 - Full admin access
#   viewer@example.com / viewer123 - Viewer role (limited access)
#
# This script will:
# 1. Create test users with hashed passwords
# 2. Assign roles to users
# 3. Create sample namespaces
# 4. Configure role access to namespaces

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

echo "=== Seeding test users and RBAC configuration ==="

# Check if docker-compose is running
if ! docker-compose ps postgres | grep -q "Up"; then
    echo "Error: PostgreSQL container is not running. Start it with: docker-compose up -d postgres"
    exit 1
fi

# Wait for postgres to be ready
echo "Waiting for PostgreSQL to be ready..."
until docker-compose exec -T postgres pg_isready -U postgres > /dev/null 2>&1; do
    sleep 1
done

echo "Running seed SQL..."

docker-compose exec -T postgres psql -U postgres -d mcp_gateway << 'EOSQL'
-- ============================================
-- SEED TEST USERS
-- ============================================

-- Ensure viewer role exists
INSERT INTO roles (name, description)
VALUES ('viewer', 'Read-only access to view servers and resources')
ON CONFLICT (name) DO NOTHING;

-- Admin user (password: admin123)
-- Hash generated with: bcrypt.hashpw("admin123".encode(), bcrypt.gensalt(12))
INSERT INTO users (id, email, password_hash, name, auth_provider, is_active)
VALUES (
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'admin@example.com',
    '$2a$12$52pXBWj3xdpuPXizEmrRremy0QjnThrV/FHrE.Nb38euJxImkeSQ6',
    'System Administrator',
    'local',
    true
) ON CONFLICT (email) DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    name = EXCLUDED.name,
    is_active = EXCLUDED.is_active;

-- Assign admin role
INSERT INTO user_roles (user_id, role_id)
SELECT
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    r.id
FROM roles r
WHERE r.name = 'admin'
ON CONFLICT (user_id, role_id) DO NOTHING;

-- Viewer user (password: viewer123)
-- Hash generated with: bcrypt.hashpw("viewer123".encode(), bcrypt.gensalt(12))
INSERT INTO users (id, email, password_hash, name, auth_provider, is_active)
VALUES (
    'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22',
    'viewer@example.com',
    '$2a$12$M0rjvZpu0nIYU2zh1mpZreIHfQtEqwt2TEaEoYT7XxLzo6Fas2dFa',
    'Demo Viewer',
    'local',
    true
) ON CONFLICT (email) DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    name = EXCLUDED.name,
    is_active = EXCLUDED.is_active;

-- Assign viewer role
INSERT INTO user_roles (user_id, role_id)
SELECT
    'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22',
    r.id
FROM roles r
WHERE r.name = 'viewer'
ON CONFLICT (user_id, role_id) DO NOTHING;

-- Operator user (password: operator123)
-- Hash generated with: go run with bcrypt.GenerateFromPassword([]byte("operator123"), 12)
INSERT INTO users (id, email, password_hash, name, auth_provider, is_active)
VALUES (
    'c2eebc99-9c0b-4ef8-bb6d-6bb9bd380a33',
    'operator@example.com',
    '$2a$12$HcS17bNdanTNUchIRTsF1u1gCAO4j42EhnyLJ0jD72FiPl0X61p0O',
    'Demo Operator',
    'local',
    true
) ON CONFLICT (email) DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    name = EXCLUDED.name,
    is_active = EXCLUDED.is_active;

-- Assign operator role
INSERT INTO user_roles (user_id, role_id)
SELECT
    'c2eebc99-9c0b-4ef8-bb6d-6bb9bd380a33',
    r.id
FROM roles r
WHERE r.name = 'operator'
ON CONFLICT (user_id, role_id) DO NOTHING;

-- ============================================
-- SEED NAMESPACES
-- ============================================

INSERT INTO namespaces (name, description)
VALUES
    ('default', 'Default namespace for general servers'),
    ('engineering', 'Engineering team MCP servers'),
    ('data-science', 'Data science and analytics servers'),
    ('production', 'Production environment servers')
ON CONFLICT (name) DO NOTHING;

-- ============================================
-- SEED ROLE ACCESS TO NAMESPACES
-- ============================================

-- Viewer role: view access to engineering namespace
INSERT INTO role_namespace_access (role_id, namespace_id, access_level)
SELECT r.id, n.id, 'view'
FROM roles r, namespaces n
WHERE r.name = 'viewer' AND n.name = 'engineering'
ON CONFLICT (role_id, namespace_id) DO UPDATE SET access_level = 'view';

-- Operator role: execute access to engineering and data-science namespaces
INSERT INTO role_namespace_access (role_id, namespace_id, access_level)
SELECT r.id, n.id, 'execute'
FROM roles r, namespaces n
WHERE r.name = 'operator' AND n.name IN ('engineering', 'data-science')
ON CONFLICT (role_id, namespace_id) DO UPDATE SET access_level = 'execute';

-- ============================================
-- VERIFY SEED DATA
-- ============================================

SELECT '=== USERS ===' as section;
SELECT u.email, u.name, string_agg(r.name, ', ') as roles
FROM users u
LEFT JOIN user_roles ur ON u.id = ur.user_id
LEFT JOIN roles r ON ur.role_id = r.id
GROUP BY u.id, u.email, u.name
ORDER BY u.email;

SELECT '=== NAMESPACES ===' as section;
SELECT name, description FROM namespaces ORDER BY name;

SELECT '=== ROLE NAMESPACE ACCESS ===' as section;
SELECT r.name as role, n.name as namespace, rna.access_level
FROM role_namespace_access rna
JOIN roles r ON rna.role_id = r.id
JOIN namespaces n ON rna.namespace_id = n.id
ORDER BY r.name, n.name;

EOSQL

echo ""
echo "=== Seed completed successfully! ==="
echo ""
echo "Test Users:"
echo "  admin@example.com    / admin123     (admin role - full access)"
echo "  operator@example.com / operator123  (operator role - execute on engineering, data-science)"
echo "  viewer@example.com   / viewer123    (viewer role - view on engineering only)"
echo ""
echo "To test Resource RBAC, make sure these settings are in configs/config.yaml:"
echo "  auth.enabled: true"
echo "  auth.resource_rbac_enabled: true"
echo ""
