-- Server Groups table
CREATE TABLE server_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_server_groups_name ON server_groups(name);

-- Server to Group mapping (many-to-many)
CREATE TABLE server_group_members (
    server_id UUID NOT NULL REFERENCES mcp_servers(id) ON DELETE CASCADE,
    group_id UUID NOT NULL REFERENCES server_groups(id) ON DELETE CASCADE,
    added_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (server_id, group_id)
);

CREATE INDEX idx_server_group_members_server ON server_group_members(server_id);
CREATE INDEX idx_server_group_members_group ON server_group_members(group_id);

-- Role to Server Group access mapping
CREATE TABLE role_server_group_access (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    group_id UUID NOT NULL REFERENCES server_groups(id) ON DELETE CASCADE,
    access_level VARCHAR(50) NOT NULL CHECK (access_level IN ('view', 'execute')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (role_id, group_id)
);

CREATE INDEX idx_role_group_access_role ON role_server_group_access(role_id);
CREATE INDEX idx_role_group_access_group ON role_server_group_access(group_id);

-- Seed default server groups
INSERT INTO server_groups (name, description) VALUES
    ('default', 'Default server group - new servers are added here automatically'),
    ('engineering', 'Engineering team MCP servers'),
    ('data-science', 'Data science and ML MCP servers');

-- Give admin role execute access to all default groups
INSERT INTO role_server_group_access (role_id, group_id, access_level)
SELECT r.id, g.id, 'execute'
FROM roles r, server_groups g
WHERE r.name = 'admin';

-- Give operator role execute access to default and engineering groups
INSERT INTO role_server_group_access (role_id, group_id, access_level)
SELECT r.id, g.id, 'execute'
FROM roles r, server_groups g
WHERE r.name = 'operator' AND g.name IN ('default', 'engineering');

-- Give viewer role view access to default group
INSERT INTO role_server_group_access (role_id, group_id, access_level)
SELECT r.id, g.id, 'view'
FROM roles r, server_groups g
WHERE r.name = 'viewer' AND g.name = 'default';

-- Give user role view access to default group
INSERT INTO role_server_group_access (role_id, group_id, access_level)
SELECT r.id, g.id, 'view'
FROM roles r, server_groups g
WHERE r.name = 'user' AND g.name = 'default';

-- Add all existing servers to the default group
INSERT INTO server_group_members (server_id, group_id)
SELECT s.id, g.id
FROM mcp_servers s, server_groups g
WHERE g.name = 'default';
