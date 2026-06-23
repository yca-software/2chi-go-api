//go:build integration

package organization_repository_test

import (
	"context"
	"errors"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/suite"

	"github.com/yca-software/2chi-go-api/internals/models"
	organization_repository "github.com/yca-software/2chi-go-api/internals/repositories/organization"
	"github.com/yca-software/2chi-go-api/internals/packages/testutil"
	chi_archive "github.com/yca-software/2chi-go-archive"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_repository "github.com/yca-software/2chi-go-repository"
	chi_types "github.com/yca-software/2chi-go-types"
)

const (
	seedActiveOrgID        = "22222222-2222-2222-2222-222222222001"
	seedUpdateOrgID        = "22222222-2222-2222-2222-222222222002"
	seedArchiveTargetOrgID = "22222222-2222-2222-2222-222222222003"
	seedRestoreOrgID       = "22222222-2222-2222-2222-222222222004"
	seedArchivedOrgID      = "22222222-2222-2222-2222-222222222005"
	seedStaleArchivedOrgID = "22222222-2222-2222-2222-222222222006"
	seedSearchActiveOrgID  = "22222222-2222-2222-2222-222222222007"
	seedNewOrgID           = "22222222-2222-2222-2222-22222222200b"
)

var seedCreatedAtTime = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

func TestMain(m *testing.M) {
	os.Exit(testutil.IntegrationTestMain(m))
}

func TestRepositorySuite(t *testing.T) {
	suite.Run(t, new(RepositorySuite))
}

type RepositorySuite struct {
	suite.Suite

	db   *sqlx.DB
	repo organization_repository.Repository
	ctx  context.Context
}

func (s *RepositorySuite) SetupSuite() {
	testDB, err := testutil.GetIntegrationDB()
	s.Require().NoError(err)

	s.db, err = testDB.SQLx()
	s.Require().NoError(err)

	s.repo = organization_repository.NewRepository(s.db, nil)
	s.ctx = context.Background()
}

func (s *RepositorySuite) SetupTest() {
	_, err := s.db.ExecContext(s.ctx, `
INSERT INTO organizations (id, created_at, deleted_at, name, address, city, zip, country, place_id, geo, timezone) VALUES
	('22222222-2222-2222-2222-222222222001', '2024-01-01T00:00:00Z', NULL, 'Active Org', '1 Main St', 'Oslo', '0001', 'NO', 'place_seed_001', ST_SetSRID(ST_MakePoint(10.7, 59.9), 4326), 'Europe/Oslo'),
	('22222222-2222-2222-2222-222222222002', '2024-01-01T00:00:00Z', NULL, 'Update Org', '1 Main St', 'Oslo', '0001', 'NO', 'place_seed_002', ST_SetSRID(ST_MakePoint(10.7, 59.9), 4326), 'Europe/Oslo'),
	('22222222-2222-2222-2222-222222222003', '2024-01-01T00:00:00Z', NULL, 'Archive Org', '1 Main St', 'Oslo', '0001', 'NO', 'place_seed_003', ST_SetSRID(ST_MakePoint(10.7, 59.9), 4326), 'Europe/Oslo'),
	('22222222-2222-2222-2222-222222222004', '2024-01-01T00:00:00Z', '2026-06-06T00:00:00Z', 'Restore Org', '1 Main St', 'Oslo', '0001', 'NO', 'place_seed_004', ST_SetSRID(ST_MakePoint(10.7, 59.9), 4326), 'Europe/Oslo'),
	('22222222-2222-2222-2222-222222222005', '2024-01-01T00:00:00Z', '2026-06-06T00:00:00Z', 'Archived Org', '1 Main St', 'Oslo', '0001', 'NO', 'place_seed_005', ST_SetSRID(ST_MakePoint(10.7, 59.9), 4326), 'Europe/Oslo'),
	('22222222-2222-2222-2222-222222222006', '2024-01-01T00:00:00Z', '2020-01-01T00:00:00Z', 'Stale Org', '1 Main St', 'Oslo', '0001', 'NO', 'place_seed_006', ST_SetSRID(ST_MakePoint(10.7, 59.9), 4326), 'Europe/Oslo'),
	('22222222-2222-2222-2222-222222222007', '2024-01-01T00:00:00Z', NULL, 'FindMeActive Org', '1 Main St', 'Oslo', '0001', 'NO', 'place_seed_007', ST_SetSRID(ST_MakePoint(10.7, 59.9), 4326), 'Europe/Oslo')`)
	s.Require().NoError(err)
}

func (s *RepositorySuite) TearDownTest() {
	_, err := s.db.ExecContext(s.ctx, `TRUNCATE TABLE organizations CASCADE`)
	s.Require().NoError(err)
}

func (s *RepositorySuite) TestCreate() {
	org := &models.Organization{
		ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
			ModelBase: chi_types.ModelBase{
				ID:        uuid.MustParse(seedNewOrgID),
				CreatedAt: seedCreatedAtTime,
			},
		},
		Name:     "New Org",
		Address:  "2 Side St",
		City:     "Oslo",
		Zip:      "0002",
		Country:  "NO",
		PlaceID:  "place_new",
		Geo:      chi_types.Point{Lat: 59.9, Lng: 10.7},
		Timezone: "Europe/Oslo",
	}
	s.Require().NoError(s.repo.Create(s.ctx, org))

	got, err := s.repo.GetByID(s.ctx, seedNewOrgID)
	s.Require().NoError(err)
	s.Equal("New Org", got.Name)
}

func (s *RepositorySuite) TestUpdate() {
	org, err := s.repo.GetByID(s.ctx, seedUpdateOrgID)
	s.Require().NoError(err)
	originalUpdatedAt := org.UpdatedAt

	org.Name = "Updated Org"
	s.Require().NoError(s.repo.Update(s.ctx, org))

	got, err := s.repo.GetByID(s.ctx, seedUpdateOrgID)
	s.Require().NoError(err)
	s.Equal("Updated Org", got.Name)
	s.True(got.UpdatedAt.After(originalUpdatedAt))
}

func (s *RepositorySuite) TestArchiveAndRestore() {
	org, err := s.repo.GetByID(s.ctx, seedArchiveTargetOrgID)
	s.Require().NoError(err)
	s.Require().NoError(s.repo.Archive(s.ctx, org))

	_, err = s.repo.GetByID(s.ctx, seedArchiveTargetOrgID)
	s.requireNotFound(err)

	s.Require().NoError(s.repo.Restore(s.ctx, seedRestoreOrgID))
	got, err := s.repo.GetByID(s.ctx, seedRestoreOrgID)
	s.Require().NoError(err)
	s.Nil(got.DeletedAt)
}

func (s *RepositorySuite) TestSearch() {
	activeRows, err := s.repo.Search(s.ctx, "FindMe", chi_archive.ArchiveFilterActive, 10, 0)
	s.Require().NoError(err)
	s.Require().Len(*activeRows, 1)
	s.Equal(seedSearchActiveOrgID, (*activeRows)[0].ID.String())
}

func (s *RepositorySuite) TestWithTx() {
	org := &models.Organization{
		ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
			ModelBase: chi_types.ModelBase{
				ID:        uuid.MustParse("22222222-2222-2222-2222-22222222200c"),
				CreatedAt: seedCreatedAtTime,
			},
		},
		Name:     "Tx Org",
		Address:  "3 Tx St",
		City:     "Oslo",
		Zip:      "0003",
		Country:  "NO",
		PlaceID:  "place_tx",
		Geo:      chi_types.Point{Lat: 59.9, Lng: 10.7},
		Timezone: "Europe/Oslo",
	}
	err := chi_repository.RunInTx(s.ctx, s.db, nil, func(tx chi_repository.Tx) error {
		return s.repo.WithTx(tx).Create(s.ctx, org)
	})
	s.Require().NoError(err)

	got, err := s.repo.GetByID(s.ctx, "22222222-2222-2222-2222-22222222200c")
	s.Require().NoError(err)
	s.Equal("Tx Org", got.Name)
}

func (s *RepositorySuite) requireNotFound(err error) {
	s.T().Helper()
	s.Require().Error(err)
	var apiErr *chi_error.Error
	s.Require().True(errors.As(err, &apiErr))
	s.Equal(http.StatusNotFound, apiErr.StatusCode)
}
