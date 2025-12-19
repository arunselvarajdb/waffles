-- Migration: Rename server_groups to namespaces
-- This migration renames tables, columns, and indexes for the namespace refactoring

-- Rename tables
ALTER TABLE server_groups RENAME TO namespaces;
ALTER TABLE server_group_members RENAME TO namespace_members;
ALTER TABLE role_server_group_access RENAME TO role_namespace_access;

-- Rename columns (group_id -> namespace_id)
ALTER TABLE namespace_members RENAME COLUMN group_id TO namespace_id;
ALTER TABLE role_namespace_access RENAME COLUMN group_id TO namespace_id;

-- Rename indexes
ALTER INDEX idx_server_group_members_server_id RENAME TO idx_namespace_members_server_id;
ALTER INDEX idx_server_group_members_group_id RENAME TO idx_namespace_members_namespace_id;
ALTER INDEX idx_role_server_group_access_role_id RENAME TO idx_role_namespace_access_role_id;
ALTER INDEX idx_role_server_group_access_group_id RENAME TO idx_role_namespace_access_namespace_id;
