CREATE TABLE IF NOT EXISTS resources (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        UUID NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    resource_key  VARCHAR(255) NOT NULL,
    name          VARCHAR(200) NOT NULL,
    attributes    JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    updated_at    TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(org_id, resource_type, resource_key)
);

CREATE TABLE IF NOT EXISTS policies (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        UUID NOT NULL,
    name          VARCHAR(200) NOT NULL,
    effect        VARCHAR(20) NOT NULL DEFAULT 'allow',
    resource_type VARCHAR(100) NOT NULL,
    action        VARCHAR(50) NOT NULL,
    condition     JSONB NOT NULL DEFAULT '{}',
    is_enabled    BOOLEAN DEFAULT true,
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    updated_at    TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS policy_bindings (
    policy_id    UUID NOT NULL REFERENCES policies(id) ON DELETE CASCADE,
    subject_type VARCHAR(50) NOT NULL,
    subject_id   VARCHAR(255) NOT NULL,
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (policy_id, subject_type, subject_id)
);

CREATE INDEX IF NOT EXISTS idx_resources_org_type_key ON resources(org_id, resource_type, resource_key);
CREATE INDEX IF NOT EXISTS idx_policies_org_enabled ON policies(org_id, is_enabled);
CREATE INDEX IF NOT EXISTS idx_policy_bindings_subject ON policy_bindings(subject_type, subject_id);
