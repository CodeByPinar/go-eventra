ALTER TABLE refresh_tokens
    ADD COLUMN IF NOT EXISTS access_jti TEXT,
    ADD COLUMN IF NOT EXISTS access_expires_at TIMESTAMPTZ;

UPDATE refresh_tokens
SET access_jti = COALESCE(access_jti, id::text),
    access_expires_at = COALESCE(access_expires_at, expires_at)
WHERE access_jti IS NULL OR access_expires_at IS NULL;

ALTER TABLE refresh_tokens
    ALTER COLUMN access_jti SET NOT NULL,
    ALTER COLUMN access_expires_at SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_access_jti ON refresh_tokens(access_jti);

CREATE TABLE IF NOT EXISTS revoked_jtis (
    jti TEXT PRIMARY KEY,
    expires_at TIMESTAMPTZ NOT NULL,
    reason TEXT NOT NULL,
    revoked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_revoked_jtis_expires_at ON revoked_jtis(expires_at);

CREATE TABLE IF NOT EXISTS account_security_state (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    failed_attempts INT NOT NULL DEFAULT 0,
    last_failed_at TIMESTAMPTZ,
    last_login_at TIMESTAMPTZ,
    last_login_ip TEXT,
    locked_until TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_account_security_state_locked_until ON account_security_state(locked_until);

CREATE TABLE IF NOT EXISTS security_audit_logs (
    id UUID PRIMARY KEY,
    event_type TEXT NOT NULL,
    severity TEXT NOT NULL,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    ip TEXT,
    user_agent TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_security_audit_logs_event_type ON security_audit_logs(event_type);
CREATE INDEX IF NOT EXISTS idx_security_audit_logs_created_at ON security_audit_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_security_audit_logs_severity ON security_audit_logs(severity);
