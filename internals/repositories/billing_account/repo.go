package billing_account_repository

import (
	"context"
	"maps"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"

	"github.com/yca-software/2chi-go-api/internals/constants"
	"github.com/yca-software/2chi-go-api/internals/models"
	chi_observer "github.com/yca-software/2chi-go-observer"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

const (
	TableName = "organization_billing_accounts"
)

var (
	Columns = []string{
		"organization_id", "created_at", "updated_at",
		"billing_email", "provider", "provider_customer_id",
		"provider_subscription_id",
		"subscription_expires_at", "subscription_payment_interval",
		"subscription_tier", "subscription_seats", "subscription_in_trial",
		"subscription_scheduled_plan_price_id",
	}
)

type Repository interface {
	WithTx(tx chi_repository.Tx) Repository

	Create(ctx context.Context, account *models.OrganizationBillingAccount) error
	Update(ctx context.Context, account *models.OrganizationBillingAccount) error

	GetByOrganizationID(ctx context.Context, organizationID string) (*models.OrganizationBillingAccount, error)
	GetByProviderAndProviderCustomerID(ctx context.Context, provider, providerCustomerID string) (*models.OrganizationBillingAccount, error)
	GetByProviderAndProviderSubscriptionID(ctx context.Context, provider, providerSubscriptionID string) (*models.OrganizationBillingAccount, error)

	ListWithScheduledPlanChangeDue(ctx context.Context) (*[]models.OrganizationBillingAccount, error)
}

type repository struct {
	repo chi_repository.Repository[models.OrganizationBillingAccount]
}

func NewRepository(db *sqlx.DB, metricsHook chi_observer.QueryMetricsHook) Repository {
	return &repository{
		repo: chi_repository.NewRepository[models.OrganizationBillingAccount](db, TableName, Columns, metricsHook),
	}
}

func (r *repository) WithTx(tx chi_repository.Tx) Repository {
	return &repository{
		repo: r.repo.WithTx(tx),
	}
}

func (r *repository) Create(ctx context.Context, account *models.OrganizationBillingAccount) error {
	now := time.Now()
	data := map[string]any{
		"organization_id":      account.OrganizationID,
		"created_at":           now,
		"updated_at":           now,
		"billing_email":        account.BillingEmail,
		"provider":             account.Provider,
		"provider_customer_id": account.ProviderCustomerID,
	}
	maps.Copy(data, organizationBillingAccountSubscriptionFields(account))
	return r.repo.Create(ctx, data)
}

func (r *repository) Update(ctx context.Context, account *models.OrganizationBillingAccount) error {
	data := organizationBillingAccountSubscriptionFields(account)
	data["billing_email"] = account.BillingEmail
	data["provider"] = account.Provider
	data["provider_customer_id"] = account.ProviderCustomerID
	data["updated_at"] = time.Now()
	return r.repo.Update(ctx, squirrel.Eq{"organization_id": account.OrganizationID}, data)
}

func (r *repository) GetByOrganizationID(ctx context.Context, organizationID string) (*models.OrganizationBillingAccount, error) {
	return r.repo.Get(ctx, squirrel.Eq{"organization_id": organizationID}, nil)
}

func (r *repository) GetByProviderAndProviderCustomerID(ctx context.Context, provider, providerCustomerID string) (*models.OrganizationBillingAccount, error) {
	return r.repo.Get(ctx, squirrel.Eq{
		"provider":             provider,
		"provider_customer_id": providerCustomerID,
	}, nil)
}

func (r *repository) GetByProviderAndProviderSubscriptionID(ctx context.Context, provider, providerSubscriptionID string) (*models.OrganizationBillingAccount, error) {
	return r.repo.Get(ctx, squirrel.Eq{
		"provider":                 provider,
		"provider_subscription_id": providerSubscriptionID,
	}, nil)
}

func (r *repository) ListWithScheduledPlanChangeDue(ctx context.Context) (*[]models.OrganizationBillingAccount, error) {
	now := time.Now()
	return r.repo.Select(ctx, squirrel.And{
		squirrel.NotEq{"subscription_scheduled_plan_price_id": ""},
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
		"subscription_payment_interval":        interval,
		"subscription_tier":                    tier,
		"subscription_seats":                   seats,
		"subscription_in_trial":                account.SubscriptionInTrial,
		"subscription_scheduled_plan_price_id": account.SubscriptionScheduledPlanPriceID,
	}
}
