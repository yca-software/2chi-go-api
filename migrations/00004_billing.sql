-- +goose Up

/*
 * Organization Billing Accounts
 */
CREATE TABLE IF NOT EXISTS organization_billing_accounts (
  id UUID PRIMARY KEY,
  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
  
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

  billing_email CITEXT NOT NULL,

  provider VARCHAR(10) NOT NULL,
  provider_customer_id VARCHAR(255) NOT NULL,

  provider_subscription_id VARCHAR(255),

  subscription_expires_at TIMESTAMP WITH TIME ZONE,
  subscription_payment_interval VARCHAR(10) NOT NULL DEFAULT 'monthly', -- monthly, annual
  subscription_tier VARCHAR(10) NOT NULL,
  subscription_seats INT NOT NULL DEFAULT 1,
  subscription_in_trial BOOLEAN NOT NULL DEFAULT FALSE,
  
  subscription_scheduled_plan_price_id VARCHAR(255)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_organization_billing_accounts_provider_customer_id 
  ON organization_billing_accounts(provider, provider_customer_id);
CREATE INDEX IF NOT EXISTS idx_organization_billing_accounts_organization_id 
  ON organization_billing_accounts(organization_id);
CREATE INDEX IF NOT EXISTS idx_organization_billing_accounts_billing_email 
  ON organization_billing_accounts(billing_email);
CREATE UNIQUE INDEX IF NOT EXISTS idx_organization_billing_accounts_provider_subscription_id 
  ON organization_billing_accounts(provider, provider_subscription_id)
  WHERE provider_subscription_id IS NOT NULL;


-- +goose Down

DROP TABLE IF EXISTS organization_billing_accounts;