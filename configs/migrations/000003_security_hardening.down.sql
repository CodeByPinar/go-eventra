DROP TABLE IF EXISTS security_audit_logs;
DROP TABLE IF EXISTS account_security_state;
DROP TABLE IF EXISTS revoked_jtis;

DROP INDEX IF EXISTS idx_refresh_tokens_access_jti;

ALTER TABLE refresh_tokens
    DROP COLUMN IF EXISTS access_jti,
    DROP COLUMN IF EXISTS access_expires_at;
