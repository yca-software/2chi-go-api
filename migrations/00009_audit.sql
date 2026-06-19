-- +goose Up
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    
    actor_id UUID NOT NULL, -- User ID or API Key ID
    actor_info TEXT NOT NULL, -- User email or API Key name
    impersonated_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    impersonated_by_email TEXT NOT NULL,
    
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id UUID NOT NULL,
    resource_name TEXT,

    data JSONB,

    CONSTRAINT audit_logs_action_check CHECK (action <> '')
);
CREATE INDEX IF NOT EXISTS idx_audit_logs_org ON audit_logs(organization_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_actor ON audit_logs(actor_id, created_at DESC) WHERE actor_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_audit_logs_impersonated_by ON audit_logs(impersonated_by_id, created_at DESC) WHERE impersonated_by_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS impersonation_sessions (
    id UUID PRIMARY KEY,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ended_at TIMESTAMPTZ,
    end_reason TEXT,

    admin_id UUID NOT NULL,
    admin_email TEXT NOT NULL,

    target_user_id UUID NOT NULL,
    target_user_email TEXT NOT NULL,

    refresh_token_id UUID NOT NULL,

    ip INET NOT NULL,
    user_agent TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_impersonation_sessions_admin
    ON impersonation_sessions(admin_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_impersonation_sessions_target
    ON impersonation_sessions(target_user_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_impersonation_sessions_refresh_token
    ON impersonation_sessions(refresh_token_id);
CREATE INDEX IF NOT EXISTS idx_impersonation_sessions_active
    ON impersonation_sessions(started_at DESC) WHERE ended_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS impersonation_sessions;
DROP TABLE IF EXISTS audit_logs;