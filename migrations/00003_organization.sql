-- +goose Up

/*
 * Organizations
 */
CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    name VARCHAR(255) NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_organizations_deleted_at ON organizations(deleted_at);

/*
 * Organization Locations
 */
CREATE TABLE IF NOT EXISTS organization_locations (
  id UUID PRIMARY KEY,
  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),

  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

  address TEXT NOT NULL,
  city VARCHAR(100) NOT NULL,
  zip VARCHAR(20) NOT NULL,
  country VARCHAR(70) NOT NULL,
  place_id VARCHAR(255) NOT NULL,
  geo GEOMETRY(Point,4326) NOT NULL,
  timezone VARCHAR(100) NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_organization_locations_organization_id 
  ON organization_locations(organization_id);
CREATE INDEX IF NOT EXISTS idx_organization_locations_geo 
  ON organization_locations USING GIST (geo);


-- +goose Down

DROP TABLE IF EXISTS organization_locations;
DROP TABLE IF EXISTS organizations;