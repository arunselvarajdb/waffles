-- Rollback: Rename namespaces back to server_groups
-- This migration reverts the namespace refactoring

-- Rename indexes back
ALTER INDEX idx_namespace_members_server_id RENAME TO idx_server_group_members_server_id;
ALTER INDEX idx_namespace_members_namespace_id RENAME TO idx_server_group_members_group_id;
ALTER INDEX idx_role_namespace_access_role_id RENAME TO idx_role_server_group_access_role_id;
ALTER INDEX idx_role_namespace_access_namespace_id RENAME TO idx_role_server_group_access_group_id;

-- Rename columns back (namespace_id -> group_id)
ALTER TABLE namespace_members RENAME COLUMN namespace_id TO group_id;
ALTER TABLE role_namespace_access RENAME COLUMN namespace_id TO group_id;

-- Rename tables back
ALTER TABLE namespaces RENAME TO server_groups;
ALTER TABLE namespace_members RENAME TO server_group_members;
ALTER TABLE role_namespace_access RENAME TO role_server_group_access;
