-- Remove key_prefix column from api_keys table

ALTER TABLE api_keys DROP COLUMN IF EXISTS key_prefix;
