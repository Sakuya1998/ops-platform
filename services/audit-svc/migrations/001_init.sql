-- Create audit_logs table
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS audit_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID,
    user_id         UUID,
    username        VARCHAR(100),
    event_type      VARCHAR(100) NOT NULL,
    action          VARCHAR(50),
    resource_type   VARCHAR(50),
    resource_id     VARCHAR(100),
    detail          TEXT,
    ip              VARCHAR(50),
    user_agent      VARCHAR(500),
    session_id      VARCHAR(100),
    reason_code     VARCHAR(100),
    request_id      VARCHAR(100),
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_org_created ON audit_logs(org_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_event_type ON audit_logs(event_type);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_audit_logs_session_id ON audit_logs(session_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_reason_code ON audit_logs(reason_code);
CREATE INDEX IF NOT EXISTS idx_audit_logs_request_id ON audit_logs(request_id);
