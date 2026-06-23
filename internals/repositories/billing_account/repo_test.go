//go:build integration

package billing_account_repository_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/suite"

	"github.com/yca-software/2chi-go-api/internals/constants"
	"github.com/yca-software/2chi-go-api/internals/models"
	billing_account_repository "github.com/yca-software/2chi-go-api/internals/repositories/billing_account"
	"github.com/yca-software/2chi-go-api/internals/packages/testutil"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

const (
	seedBillingOrgID          = "22222222-2222-2222-2222-222222222401"
	seedBillingUpdateOrgID    = "22222222-2222-2222-2222-222222222402"
	seedBillingScheduledOrgID = "22222222-2222-2222-2222-222222222403"
	seedBillingCreateOrgID    = "22222222-2222-2222-2222-222222222404"
	seedBillingTxOrgID        = "22222222-2222-2222-2222-222222222405"
	seedBillingCustomerID     = "paddle-cust-seed-401"
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
	repo billing_account_repository.Repository
	ctx  context.Context
}

func (s *RepositorySuite) SetupSuite() {
	testDB, err := testutil.GetIntegrationDB()
	s.Require().NoError(err)

	s.db, err = testDB.SQLx()
	s.Require().NoError(err)

	s.repo = billing_account_repository.NewRepository(s.db, nil)
	s.ctx = context.Background()
}

func (s *RepositorySuite) SetupTest() {
	_, err := s.db.ExecContext(s.ctx, `
INSERT INTO organizations (id, created_at, deleted_at, name, address, city, zip, country, place_id, geo, timezone) VALUES
	('22222222-2222-2222-2222-222222222401', '2024-01-01T00:00:00Z', NULL, 'Billing Org', '1 Main St', 'Oslo', '0001', 'NO', 'place_seed_bill_1', ST_SetSRID(ST_MakePoint(10.7, 59.9), 4326), 'Europe/Oslo'),
	('22222222-2222-2222-2222-222222222402', '2024-01-01T00:00:00Z', NULL, 'Billing Update Org', '1 Main St', 'Oslo', '0001', 'NO', 'place_seed_bill_2', ST_SetSRID(ST_MakePoint(10.7, 59.9), 4326), 'Europe/Oslo'),
	('22222222-2222-2222-2222-222222222403', '2024-01-01T00:00:00Z', NULL, 'Billing Scheduled Org', '1 Main St', 'Oslo', '0001', 'NO', 'place_seed_bill_3', ST_SetSRID(ST_MakePoint(10.7, 59.9), 4326), 'Europe/Oslo'),
	('22222222-2222-2222-2222-222222222404', '2024-01-01T00:00:00Z', NULL, 'Billing Create Org', '1 Main St', 'Oslo', '0001', 'NO', 'place_seed_bill_4', ST_SetSRID(ST_MakePoint(10.7, 59.9), 4326), 'Europe/Oslo'),
	('22222222-2222-2222-2222-222222222405', '2024-01-01T00:00:00Z', NULL, 'Billing Tx Org', '1 Main St', 'Oslo', '0001', 'NO', 'place_seed_bill_5', ST_SetSRID(ST_MakePoint(10.7, 59.9), 4326), 'Europe/Oslo');
INSERT INTO organization_billing_accounts (
	created_at, organization_id, billing_email, provider, provider_customer_id,
	subscription_tier, subscription_payment_interval, subscription_seats,
	provider_subscription_id, subscription_expires_at, subscription_scheduled_plan_price_id
) VALUES
	('2024-01-01T00:00:00Z', '22222222-2222-2222-2222-222222222401',
		'billing@example.com', 'paddle', 'paddle-cust-seed-401', 'free', 'monthly', 1, 'paddle-sub-placeholder-401', NULL, NULL),
	('2024-01-01T00:00:00Z', '22222222-2222-2222-2222-222222222402',
		'update@example.com', 'paddle', 'paddle-cust-update', 'free', 'monthly', 1, 'paddle-sub-placeholder-402', NULL, NULL),
	('2024-01-01T00:00:00Z', '22222222-2222-2222-2222-222222222403',
		'scheduled@example.com', 'paddle', 'paddle-cust-scheduled', 'pro', 'annual', 25,
		'paddle-sub-scheduled', '2024-01-01T00:00:00Z', 'pri_scheduled_plan')`)
	s.Require().NoError(err)
}

func (s *RepositorySuite) TearDownTest() {
	_, err := s.db.ExecContext(s.ctx, `TRUNCATE TABLE organization_billing_accounts, organizations CASCADE`)
	s.Require().NoError(err)
}

func (s *RepositorySuite) TestCreate() {
	account := s.newAccount(seedBillingCreateOrgID, "new@example.com", "paddle-new-cust")
	s.Require().NoError(s.repo.Create(s.ctx, account))

	got, err := s.repo.GetByOrganizationID(s.ctx, seedBillingCreateOrgID)
	s.Require().NoError(err)
	s.Equal("new@example.com", got.BillingEmail)
}

func (s *RepositorySuite) TestUpdate() {
	account, err := s.repo.GetByOrganizationID(s.ctx, seedBillingUpdateOrgID)
	s.Require().NoError(err)
	account.BillingEmail = "updated@example.com"
	s.Require().NoError(s.repo.Update(s.ctx, account))

	got, err := s.repo.GetByOrganizationID(s.ctx, seedBillingUpdateOrgID)
	s.Require().NoError(err)
	s.Equal("updated@example.com", got.BillingEmail)
}

func (s *RepositorySuite) TestGetByOrganizationID() {
	got, err := s.repo.GetByOrganizationID(s.ctx, seedBillingOrgID)
	s.Require().NoError(err)
	s.Equal(seedBillingOrgID, got.OrganizationID.String())
}

func (s *RepositorySuite) TestGetByProviderAndProviderCustomerID() {
	got, err := s.repo.GetByProviderAndProviderCustomerID(s.ctx, "paddle", seedBillingCustomerID)
	s.Require().NoError(err)
	s.Equal(seedBillingOrgID, got.OrganizationID.String())
}

func (s *RepositorySuite) TestUpdateSubscriptionFields() {
	account, err := s.repo.GetByOrganizationID(s.ctx, seedBillingUpdateOrgID)
	s.Require().NoError(err)
	account.SubscriptionTier = constants.TIER_PRO
	account.SubscriptionSeats = 25
	account.ProviderSubscriptionID = "paddle-sub-updated"
	s.Require().NoError(s.repo.Update(s.ctx, account))

	got, err := s.repo.GetByOrganizationID(s.ctx, seedBillingUpdateOrgID)
	s.Require().NoError(err)
	s.Equal(constants.TIER_PRO, got.SubscriptionTier)
	s.Equal(25, got.SubscriptionSeats)
	s.Equal("paddle-sub-updated", got.ProviderSubscriptionID)
}

func (s *RepositorySuite) TestGetByProviderAndProviderSubscriptionID() {
	got, err := s.repo.GetByProviderAndProviderSubscriptionID(s.ctx, "paddle", "paddle-sub-scheduled")
	s.Require().NoError(err)
	s.Equal(seedBillingScheduledOrgID, got.OrganizationID.String())
}

func (s *RepositorySuite) TestListWithScheduledPlanChangeDue() {
	rows, err := s.repo.ListWithScheduledPlanChangeDue(s.ctx)
	s.Require().NoError(err)
	s.Require().NotNil(rows)
	s.Len(*rows, 1)
	s.Equal(seedBillingScheduledOrgID, (*rows)[0].OrganizationID.String())
}

func (s *RepositorySuite) TestWithTx() {
	account := s.newAccount(seedBillingTxOrgID, "tx@example.com", "paddle-tx-cust")
	err := chi_repository.RunInTx(s.ctx, s.db, nil, func(tx chi_repository.Tx) error {
		return s.repo.WithTx(tx).Create(s.ctx, account)
	})
	s.Require().NoError(err)

	got, err := s.repo.GetByOrganizationID(s.ctx, seedBillingTxOrgID)
	s.Require().NoError(err)
	s.Equal("tx@example.com", got.BillingEmail)
}

func (s *RepositorySuite) newAccount(organizationID, email, customerID string) *models.OrganizationBillingAccount {
	return &models.OrganizationBillingAccount{
		OrganizationID:     uuid.MustParse(organizationID),
		CreatedAt:          seedCreatedAtTime,
		BillingEmail:       email,
		Provider:           "paddle",
		ProviderCustomerID: customerID,
		SubscriptionTier:   constants.TIER_FREE,
	}
}
