-- Seed admin user with password: admin123
-- Password hash: bcrypt cost 12
INSERT INTO users (id, email, password_hash, name, auth_provider, is_active)
VALUES (
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'admin@example.com',
    '$2a$12$52pXBWj3xdpuPXizEmrRremy0QjnThrV/FHrE.Nb38euJxImkeSQ6',
    'System Administrator',
    'local',
    true
) ON CONFLICT (email) DO NOTHING;

-- Assign admin role to the admin user
INSERT INTO user_roles (user_id, role_id)
SELECT
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    r.id
FROM roles r
WHERE r.name = 'admin'
ON CONFLICT (user_id, role_id) DO NOTHING;

-- Add 'viewer' role to the roles table if it doesn't exist (for Casbin compatibility)
INSERT INTO roles (name, description)
VALUES ('viewer', 'Read-only access to view servers and resources')
ON CONFLICT (name) DO NOTHING;

-- Seed a viewer user with password: viewer123
INSERT INTO users (id, email, password_hash, name, auth_provider, is_active)
VALUES (
    'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22',
    'viewer@example.com',
    '$2a$12$M0rjvZpu0nIYU2zh1mpZreIHfQtEqwt2TEaEoYT7XxLzo6Fas2dFa',
    'Demo Viewer',
    'local',
    true
) ON CONFLICT (email) DO NOTHING;

-- Assign viewer role to the viewer user
INSERT INTO user_roles (user_id, role_id)
SELECT
    'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22',
    r.id
FROM roles r
WHERE r.name = 'viewer'
ON CONFLICT (user_id, role_id) DO NOTHING;
