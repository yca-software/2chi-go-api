package billing_account_repository

import (
	"context"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"

	"github.com/yca-software/2chi-go-api/internals/constants"
	"github.com/yca-software/2chi-go-api/internals/domains/core/models"
	chi_observer "github.com/yca-software/2chi-go-observer"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

const (
	OrganizationBillingAccountsTableName = "organization_billing_accounts"
)

var (
	OrganizationBillingAccountsColumns = []string{
		"id", "created_at", "updated_at", "organization_id",
		"billing_email", "provider", "provider_customer_id",
		"provider_subscription_id",
		"subscription_expires_at", "subscription_payment_interval",
		"subscription_tier", "subscription_seats", "subscription_in_trial",
		"subscription_scheduled_plan_price_id",
	}
)

type OrganizationBillingAccountsRepository interface {
	WithTx(tx chi_repository.Tx) OrganizationBillingAccountsRepository

	CreateOrganizationBillingAccount(ctx context.Context, account *models.OrganizationBillingAccount) error
	UpdateOrganizationBillingAccount(ctx context.Context, account *models.OrganizationBillingAccount) error

	GetOrganizationBillingAccountByID(ctx context.Context, organizationID, id string) (*models.OrganizationBillingAccount, error)
	GetOrganizationBillingAccountByOrganizationID(ctx context.Context, organizationID string) (*models.OrganizationBillingAccount, error)
	GetOrganizationBillingAccountByProviderAndProviderCustomerID(ctx context.Context, provider, providerCustomerID string) (*models.OrganizationBillingAccount, error)
	GetOrganizationBillingAccountByProviderAndProviderSubscriptionID(ctx context.Context, provider, providerSubscriptionID string) (*models.OrganizationBillingAccount, error)

	ListOrganizationBillingAccountsWithScheduledPlanChangeDue(ctx context.Context) (*[]models.OrganizationBillingAccount, error)
}

type organizationBillingAccountsRepository struct {
	billingAccountsRepo chi_repository.Repository[models.OrganizationBillingAccount]
}

func NewOrganizationBillingAccountsRepository(db *sqlx.DB, metricsHook chi_observer.QueryMetricsHook) OrganizationBillingAccountsRepository {
	return &organizationBillingAccountsRepository{
		billingAccountsRepo: chi_repository.NewRepository[models.OrganizationBillingAccount](db, OrganizationBillingAccountsTableName, OrganizationBillingAccountsColumns, metricsHook),
	}
}

func (r *organizationBillingAccountsRepository) WithTx(tx chi_repository.Tx) OrganizationBillingAccountsRepository {
	return &organizationBillingAccountsRepository{
		billingAccountsRepo: r.billingAccountsRepo.WithTx(tx),
	}
}

func (r *organizationBillingAccountsRepository) CreateOrganizationBillingAccount(ctx context.Context, account *models.OrganizationBillingAccount) error {
	now := time.Now()
	data := map[string]any{
		"id":                   account.ID,
		"created_at":           now,
		"updated_at":           now,
		"organization_id":      account.OrganizationID,
		"billing_email":        account.BillingEmail,
		"provider":             account.Provider,
		"provider_customer_id": account.ProviderCustomerID,
	}
	for k, v := range organizationBillingAccountSubscriptionFields(account) {
		data[k] = v
	}
	return r.billingAccountsRepo.Create(ctx, data)
}

func (r *organizationBillingAccountsRepository) UpdateOrganizationBillingAccount(ctx context.Context, account *models.OrganizationBillingAccount) error {
	return r.billingAccountsRepo.Update(ctx, squirrel.And{
		squirrel.Eq{"id": account.ID},
		squirrel.Eq{"organization_id": account.OrganizationID},
	}, organizationBillingAccountUpdateFields(account))
}

func (r *organizationBillingAccountsRepository) GetOrganizationBillingAccountByID(ctx context.Context, organizationID, id string) (*models.OrganizationBillingAccount, error) {
	return r.billingAccountsRepo.Get(ctx, squirrel.And{
		squirrel.Eq{"id": id},
		squirrel.Eq{"organization_id": organizationID},
	}, nil)
}

func (r *organizationBillingAccountsRepository) GetOrganizationBillingAccountByOrganizationID(ctx context.Context, organizationID string) (*models.OrganizationBillingAccount, error) {
	return r.billingAccountsRepo.Get(ctx, squirrel.Eq{"organization_id": organizationID}, nil)
}

func (r *organizationBillingAccountsRepository) GetOrganizationBillingAccountByProviderAndProviderCustomerID(ctx context.Context, provider, providerCustomerID string) (*models.OrganizationBillingAccount, error) {
	return r.billingAccountsRepo.Get(ctx, squirrel.Eq{
		"provider":             provider,
		"provider_customer_id": providerCustomerID,
	}, nil)
}

func (r *organizationBillingAccountsRepository) GetOrganizationBillingAccountByProviderAndProviderSubscriptionID(ctx context.Context, provider, providerSubscriptionID string) (*models.OrganizationBillingAccount, error) {
	return r.billingAccountsRepo.Get(ctx, squirrel.Eq{
		"provider":                 provider,
		"provider_subscription_id": providerSubscriptionID,
	}, nil)
}

func (r *organizationBillingAccountsRepository) ListOrganizationBillingAccountsWithScheduledPlanChangeDue(ctx context.Context) (*[]models.OrganizationBillingAccount, error) {
	now := time.Now()
	return r.billingAccountsRepo.Select(ctx, squirrel.And{
		squirrel.NotEq{"subscription_scheduled_plan_price_id": nil},
		squirrel.NotEq{"subscription_scheduled_plan_price_id": ""},
		squirrel.NotEq{"provider_subscription_id": nil},
		squirrel.NotEq{"provider_subscription_id": ""},
		squirrel.LtOrEq{"subscription_expires_at": now},
	}, nil, "")
}

func organizationBillingAccountSubscriptionFields(account *models.OrganizationBillingAccount) map[string]any {
	tier := account.SubscriptionTier
	if tier == "" {
		tier = constants.TIER_FREE
	}
	interval := account.SubscriptionPaymentInterval
	if interval == "" {
		interval = constants.PAYMENT_INTERVAL_MONTHLY
	}
	seats := account.SubscriptionSeats
	if seats == 0 {
		seats = constants.SUBSCRIPTION_TYPE_SEATS_INCLUDED_FREE
	}

	return map[string]any{
		"provider_subscription_id":             account.ProviderSubscriptionID,
		"subscription_expires_at":              account.SubscriptionExpiresAt,
		"subscription_payment_interval":      interval,
		"subscription_tier":                  tier,
		"subscription_seats":                   seats,
		"subscription_in_trial":                account.SubscriptionInTrial,
		"subscription_scheduled_plan_price_id": account.SubscriptionScheduledPlanPriceID,
	}
}

func organizationBillingAccountUpdateFields(account *models.OrganizationBillingAccount) map[string]any {
	fields := organizationBillingAccountSubscriptionFields(account)
	fields["billing_email"] = account.BillingEmail
	fields["provider"] = account.Provider
	fields["provider_customer_id"] = account.ProviderCustomerID
	fields["updated_at"] = time.Now()
	return fields
}
