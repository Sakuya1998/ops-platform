-- Create notification tables
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS notification_channels (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL,
    name        VARCHAR(200),
    channel_type VARCHAR(20) NOT NULL,
    config      TEXT,
    is_enabled  BOOLEAN DEFAULT true,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS notification_templates (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_type  VARCHAR(20) NOT NULL,
    name          VARCHAR(200),
    title_template TEXT,
    body_template  TEXT,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS notification_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id  UUID,
    event_type  VARCHAR(100),
    recipient   VARCHAR(500),
    title       VARCHAR(500),
    status      VARCHAR(20),
    error_msg   TEXT,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
