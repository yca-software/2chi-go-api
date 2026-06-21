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
	chi_types "github.com/yca-software/2chi-go-types"
)

const (
	seedBillingOrgID          = "22222222-2222-2222-2222-222222222401"
	seedBillingUpdateOrgID    = "22222222-2222-2222-2222-222222222402"
	seedBillingScheduledOrgID = "22222222-2222-2222-2222-222222222403"
	seedBillingActiveID       = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb401"
	seedBillingUpdateID       = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb402"
	seedBillingScheduledID    = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb405"
	seedBillingCustomerID     = "paddle-cust-seed-401"
	seedBillingNewID          = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb403"
	seedBillingTxID           = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb404"
)

var seedCreatedAtTime = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

func TestMain(m *testing.M) {
	os.Exit(testutil.IntegrationTestMain(m))
}

func TestOrganizationBillingAccountsRepositorySuite(t *testing.T) {
	suite.Run(t, new(OrganizationBillingAccountsRepositorySuite))
}

type OrganizationBillingAccountsRepositorySuite struct {
	suite.Suite

	db   *sqlx.DB
	repo billing_account_repository.OrganizationBillingAccountsRepository
	ctx  context.Context
}

func (s *OrganizationBillingAccountsRepositorySuite) SetupSuite() {
	testDB, err := testutil.GetIntegrationDB()
	s.Require().NoError(err)

	s.db, err = testDB.SQLx()
	s.Require().NoError(err)

	s.repo = billing_account_repository.NewOrganizationBillingAccountsRepository(s.db, nil)
	s.ctx = context.Background()
}

func (s *OrganizationBillingAccountsRepositorySuite) SetupTest() {
	_, err := s.db.ExecContext(s.ctx, `
INSERT INTO organizations (id, created_at, deleted_at, name) VALUES
	('22222222-2222-2222-2222-222222222401', '2024-01-01T00:00:00Z', NULL, 'Billing Org'),
	('22222222-2222-2222-2222-222222222402', '2024-01-01T00:00:00Z', NULL, 'Billing Update Org'),
	('22222222-2222-2222-2222-222222222403', '2024-01-01T00:00:00Z', NULL, 'Billing Scheduled Org');
INSERT INTO organization_billing_accounts (
	id, created_at, organization_id, billing_email, provider, provider_customer_id,
	subscription_tier, subscription_payment_interval, subscription_seats,
	provider_subscription_id, subscription_expires_at, subscription_scheduled_plan_price_id
) VALUES
	('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb401', '2024-01-01T00:00:00Z', '22222222-2222-2222-2222-222222222401',
		'billing@example.com', 'paddle', 'paddle-cust-seed-401', 'free', 'monthly', 1, 'paddle-sub-placeholder-401', NULL, NULL),
	('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb402', '2024-01-01T00:00:00Z', '22222222-2222-2222-2222-222222222402',
		'update@example.com', 'paddle', 'paddle-cust-update', 'free', 'monthly', 1, 'paddle-sub-placeholder-402', NULL, NULL),
	('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb405', '2024-01-01T00:00:00Z', '22222222-2222-2222-2222-222222222403',
		'scheduled@example.com', 'paddle', 'paddle-cust-scheduled', 'pro', 'annual', 25,
		'paddle-sub-scheduled', '2024-01-01T00:00:00Z', 'pri_scheduled_plan')`)
	s.Require().NoError(err)
}

func (s *OrganizationBillingAccountsRepositorySuite) TearDownTest() {
	_, err := s.db.ExecContext(s.ctx, `TRUNCATE TABLE organization_billing_accounts, organizations CASCADE`)
	s.Require().NoError(err)
}

func (s *OrganizationBillingAccountsRepositorySuite) TestCreateOrganizationBillingAccount() {
	account := s.newAccount(seedBillingNewID, "new@example.com", "paddle-new-cust")
	s.Require().NoError(s.repo.CreateOrganizationBillingAccount(s.ctx, account))

	got, err := s.repo.GetOrganizationBillingAccountByID(s.ctx, seedBillingOrgID, seedBillingNewID)
	s.Require().NoError(err)
	s.Equal("new@example.com", got.BillingEmail)
}

func (s *OrganizationBillingAccountsRepositorySuite) TestUpdateOrganizationBillingAccount() {
	account, err := s.repo.GetOrganizationBillingAccountByID(s.ctx, seedBillingUpdateOrgID, seedBillingUpdateID)
	s.Require().NoError(err)
	account.BillingEmail = "updated@example.com"
	s.Require().NoError(s.repo.UpdateOrganizationBillingAccount(s.ctx, account))

	got, err := s.repo.GetOrganizationBillingAccountByID(s.ctx, seedBillingUpdateOrgID, seedBillingUpdateID)
	s.Require().NoError(err)
	s.Equal("updated@example.com", got.BillingEmail)
}

func (s *OrganizationBillingAccountsRepositorySuite) TestGetOrganizationBillingAccountByOrganizationID() {
	got, err := s.repo.GetOrganizationBillingAccountByOrganizationID(s.ctx, seedBillingOrgID)
	s.Require().NoError(err)
	s.Equal(seedBillingActiveID, got.ID.String())
}

func (s *OrganizationBillingAccountsRepositorySuite) TestGetOrganizationBillingAccountByProviderAndProviderCustomerID() {
	got, err := s.repo.GetOrganizationBillingAccountByProviderAndProviderCustomerID(s.ctx, "paddle", seedBillingCustomerID)
	s.Require().NoError(err)
	s.Equal(seedBillingOrgID, got.OrganizationID.String())
}

func (s *OrganizationBillingAccountsRepositorySuite) TestUpdateOrganizationBillingAccountSubscriptionFields() {
	account, err := s.repo.GetOrganizationBillingAccountByID(s.ctx, seedBillingUpdateOrgID, seedBillingUpdateID)
	s.Require().NoError(err)
	account.SubscriptionTier = constants.TIER_PRO
	account.SubscriptionSeats = 25
	account.ProviderSubscriptionID = "paddle-sub-updated"
	s.Require().NoError(s.repo.UpdateOrganizationBillingAccount(s.ctx, account))

	got, err := s.repo.GetOrganizationBillingAccountByID(s.ctx, seedBillingUpdateOrgID, seedBillingUpdateID)
	s.Require().NoError(err)
	s.Equal(constants.TIER_PRO, got.SubscriptionTier)
	s.Equal(25, got.SubscriptionSeats)
	s.Equal("paddle-sub-updated", got.ProviderSubscriptionID)
}

func (s *OrganizationBillingAccountsRepositorySuite) TestGetOrganizationBillingAccountByProviderAndProviderSubscriptionID() {
	got, err := s.repo.GetOrganizationBillingAccountByProviderAndProviderSubscriptionID(s.ctx, "paddle", "paddle-sub-scheduled")
	s.Require().NoError(err)
	s.Equal(seedBillingScheduledID, got.ID.String())
}

func (s *OrganizationBillingAccountsRepositorySuite) TestListOrganizationBillingAccountsWithScheduledPlanChangeDue() {
	rows, err := s.repo.ListOrganizationBillingAccountsWithScheduledPlanChangeDue(s.ctx)
	s.Require().NoError(err)
	s.Require().NotNil(rows)
	s.Len(*rows, 1)
	s.Equal(seedBillingScheduledOrgID, (*rows)[0].OrganizationID.String())
}

func (s *OrganizationBillingAccountsRepositorySuite) TestWithTx() {
	account := s.newAccount(seedBillingTxID, "tx@example.com", "paddle-tx-cust")
	err := chi_repository.RunInTx(s.ctx, s.db, nil, func(tx chi_repository.Tx) error {
		return s.repo.WithTx(tx).CreateOrganizationBillingAccount(s.ctx, account)
	})
	s.Require().NoError(err)

	got, err := s.repo.GetOrganizationBillingAccountByID(s.ctx, seedBillingOrgID, seedBillingTxID)
	s.Require().NoError(err)
	s.Equal("tx@example.com", got.BillingEmail)
}

func (s *OrganizationBillingAccountsRepositorySuite) newAccount(id, email, customerID string) *models.OrganizationBillingAccount {
	return &models.OrganizationBillingAccount{
		ModelBase: chi_types.ModelBase{
			ID:        uuid.MustParse(id),
			CreatedAt: seedCreatedAtTime,
		},
		OrganizationID:     uuid.MustParse(seedBillingOrgID),
		BillingEmail:       email,
		Provider:           "paddle",
		ProviderCustomerID: customerID,
		SubscriptionTier:   constants.TIER_FREE,
	}
}
