-- 000005_add_token_revocation.down.sql

DROP TABLE IF EXISTS revoked_tokens;
ALTER TABLE users DROP COLUMN IF EXISTS token_version;
