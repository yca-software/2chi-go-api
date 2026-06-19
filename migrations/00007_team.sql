-- +goose Up
CREATE TABLE IF NOT EXISTS teams (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),

    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    name VARCHAR(255) NOT NULL,
    description TEXT,

    CONSTRAINT uniq_team_name_org UNIQUE (organization_id, name)
);
CREATE INDEX IF NOT EXISTS idx_teams_org ON teams(organization_id);

CREATE TABLE IF NOT EXISTS team_members (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),

    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    CONSTRAINT team_members_unique UNIQUE (team_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_team_members_org ON team_members(organization_id, team_id);
CREATE INDEX IF NOT EXISTS idx_team_members_user ON team_members(user_id);

-- +goose Down
DROP TABLE IF EXISTS team_members;
DROP TABLE IF EXISTS teams;