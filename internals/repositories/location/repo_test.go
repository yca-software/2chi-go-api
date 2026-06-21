package organization_location_repository_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/suite"

	"github.com/yca-software/2chi-go-api/internals/models"
	"github.com/yca-software/2chi-go-api/internals/packages/location"
	organization_location_repository "github.com/yca-software/2chi-go-api/internals/repositories/location"
	chi_repository "github.com/yca-software/2chi-go-repository"
	chi_test "github.com/yca-software/2chi-go-test"
	chi_types "github.com/yca-software/2chi-go-types"
)

const (
	seedLocOrgID       = "22222222-2222-2222-2222-222222222301"
	seedLocUpdateOrgID = "22222222-2222-2222-2222-222222222302"
	seedLocActiveID    = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa301"
	seedLocUpdateID    = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa302"
	seedLocNewID       = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa303"
	seedLocTxID        = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa304"
)

var seedCreatedAtTime = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

func moduleMigrationsDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "..", "..", "migrations"))
}

func TestMain(m *testing.M) {
	code := m.Run()
	chi_test.Cleanup()
	os.Exit(code)
}

func TestOrganizationLocationsRepositorySuite(t *testing.T) {
	suite.Run(t, new(OrganizationLocationsRepositorySuite))
}

type OrganizationLocationsRepositorySuite struct {
	suite.Suite

	db   *sqlx.DB
	repo organization_location_repository.OrganizationLocationsRepository
	ctx  context.Context
}

func (s *OrganizationLocationsRepositorySuite) SetupSuite() {
	testDB, err := chi_test.Get(moduleMigrationsDir())
	s.Require().NoError(err)

	s.db, err = testDB.SQLx()
	s.Require().NoError(err)

	s.repo = organization_location_repository.NewOrganizationLocationsRepository(s.db, nil)
	s.ctx = context.Background()
}

func (s *OrganizationLocationsRepositorySuite) SetupTest() {
	_, err := s.db.ExecContext(s.ctx, `
INSERT INTO organizations (id, created_at, deleted_at, name) VALUES
	('22222222-2222-2222-2222-222222222301', '2024-01-01T00:00:00Z', NULL, 'Location Org'),
	('22222222-2222-2222-2222-222222222302', '2024-01-01T00:00:00Z', NULL, 'Location Update Org');
INSERT INTO organization_locations (
	id, created_at, organization_id, address, city, zip, country, place_id, geo, timezone
) VALUES
	('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa301', '2024-01-01T00:00:00Z', '22222222-2222-2222-2222-222222222301',
		'1 Main St', 'Oslo', '0001', 'NO', 'place-1', ST_SetSRID(ST_MakePoint(10.7, 59.9), 4326), 'Europe/Oslo'),
	('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa302', '2024-01-01T00:00:00Z', '22222222-2222-2222-2222-222222222302',
		'2 Main St', 'Oslo', '0002', 'NO', 'place-2', ST_SetSRID(ST_MakePoint(10.8, 59.8), 4326), 'Europe/Oslo')`)
	s.Require().NoError(err)
}

func (s *OrganizationLocationsRepositorySuite) TearDownTest() {
	_, err := s.db.ExecContext(s.ctx, `TRUNCATE TABLE organization_locations, organizations CASCADE`)
	s.Require().NoError(err)
}

func (s *OrganizationLocationsRepositorySuite) TestCreateOrganizationLocation() {
	loc := s.newLocation(seedLocNewID, "3 Main St", "Bergen")
	s.Require().NoError(s.repo.CreateOrganizationLocation(s.ctx, loc))

	got, err := s.repo.GetOrganizationLocationByID(s.ctx, seedLocOrgID, seedLocNewID)
	s.Require().NoError(err)
	s.Equal("Bergen", got.City)
}

func (s *OrganizationLocationsRepositorySuite) TestUpdateOrganizationLocation() {
	loc, err := s.repo.GetOrganizationLocationByID(s.ctx, seedLocUpdateOrgID, seedLocUpdateID)
	s.Require().NoError(err)
	loc.City = "Trondheim"
	s.Require().NoError(s.repo.UpdateOrganizationLocation(s.ctx, loc))

	got, err := s.repo.GetOrganizationLocationByID(s.ctx, seedLocUpdateOrgID, seedLocUpdateID)
	s.Require().NoError(err)
	s.Equal("Trondheim", got.City)
}

func (s *OrganizationLocationsRepositorySuite) TestGetOrganizationLocationByOrganizationID() {
	got, err := s.repo.GetOrganizationLocationByOrganizationID(s.ctx, seedLocOrgID)
	s.Require().NoError(err)
	s.Equal(seedLocActiveID, got.ID.String())
}

func (s *OrganizationLocationsRepositorySuite) TestWithTx() {
	loc := s.newLocation(seedLocTxID, "99 Tx St", "Oslo")
	err := chi_repository.RunInTx(s.ctx, s.db, nil, func(tx chi_repository.Tx) error {
		return s.repo.WithTx(tx).CreateOrganizationLocation(s.ctx, loc)
	})
	s.Require().NoError(err)

	got, err := s.repo.GetOrganizationLocationByID(s.ctx, seedLocOrgID, seedLocTxID)
	s.Require().NoError(err)
	s.Equal("99 Tx St", got.Address)
}

func (s *OrganizationLocationsRepositorySuite) newLocation(id, address, city string) *models.OrganizationLocation {
	return &models.OrganizationLocation{
		LocationModel: location.LocationModel{
			ModelBase: chi_types.ModelBase{
				ID:        uuid.MustParse(id),
				CreatedAt: seedCreatedAtTime,
			},
			Address:  address,
			City:     city,
			Zip:      "0001",
			Country:  "NO",
			PlaceID:  "place-new",
			Geo:      chi_types.Point{Lng: 10.7, Lat: 59.9},
			Timezone: "Europe/Oslo",
		},
		OrganizationID: uuid.MustParse(seedLocOrgID),
	}
}
