-- +goose Up
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),

    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,

    name VARCHAR(255) NOT NULL,
    key_prefix VARCHAR(20) NOT NULL,
    key_hash TEXT NOT NULL,
    permissions JSONB NOT NULL DEFAULT '[]'::jsonb
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_org ON api_keys(organization_id);

-- +goose Down
DROP TABLE IF EXISTS api_keys;