package billing_service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/PaddleHQ/paddle-go-sdk/v4"
	"github.com/yca-software/2chi-go-api/internals/constants"
	"github.com/yca-software/2chi-go-api/internals/models"
	"github.com/yca-software/2chi-go-api/internals/packages/audit"
	"github.com/yca-software/2chi-go-api/internals/packages/authz"
	"github.com/yca-software/2chi-go-api/internals/repositories"
	billing_account_repository "github.com/yca-software/2chi-go-api/internals/repositories/billing_account"
	organization_repository "github.com/yca-software/2chi-go-api/internals/repositories/organization"
	audit_service "github.com/yca-software/2chi-go-api/internals/services/audit"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_paddle "github.com/yca-software/2chi-go-paddle"
	chi_paddle_customer "github.com/yca-software/2chi-go-paddle/customer"
	chi_paddle_subscription "github.com/yca-software/2chi-go-paddle/subscription"
	chi_paddle_transaction "github.com/yca-software/2chi-go-paddle/transaction"
	chi_types "github.com/yca-software/2chi-go-types"
	chi_validator "github.com/yca-software/2chi-go-validator"
)

type Dependencies struct {
	Validator           chi_validator.Validator
	Logger              chi_logger.Logger
	Authorizer          *authz.Authorizer
	Repositories        *repositories.Repositories
	PaddleCustomer      chi_paddle_customer.CustomerService
	PaddleSubscription  chi_paddle_subscription.SubscriptionService
	PaddleTransaction   chi_paddle_transaction.TransactionService
	PaddleWebhookSecret string
	PriceCatalog        PriceCatalog
	AuditService        audit_service.Service
}

type Service interface {
	CreateCustomer(ctx context.Context, input *CreateCustomerInput) (string, error)
	UpdateCustomer(ctx context.Context, input *UpdateCustomerInput) error
	ReleaseProvisionedCustomer(ctx context.Context, organizationID string, billingAccount *models.OrganizationBillingAccount) error
	CancelSubscription(ctx context.Context, providerSubscriptionID string) error

	CreateCheckoutSession(ctx context.Context, req *CreateCheckoutSessionRequest, access *chi_types.AccessInfo) (*CheckoutSessionResponse, error)
	CreateCustomerPortalSession(ctx context.Context, req *CreateCustomerPortalSessionRequest, access *chi_types.AccessInfo) (*CustomerPortalSessionResponse, error)
	ProcessTransaction(ctx context.Context, req *ProcessTransactionRequest, access *chi_types.AccessInfo) (*models.OrganizationBillingAccount, error)
	ChangePlan(ctx context.Context, req *ChangePlanRequest, access *chi_types.AccessInfo) (*ChangePlanResponse, error)
	HandleWebhook(ctx context.Context, payload []byte, signature string) error
	ApplyScheduledPlanChanges(ctx context.Context) error
}

type service struct {
	validator           chi_validator.Validator
	logger              chi_logger.Logger
	authorizer          *authz.Authorizer
	organizationsRepo   organization_repository.OrganizationsRepository
	billingAccountsRepo billing_account_repository.OrganizationBillingAccountsRepository
	paddleCustomer      chi_paddle_customer.CustomerService
	paddleSubscription  chi_paddle_subscription.SubscriptionService
	paddleTransaction   chi_paddle_transaction.TransactionService
	getPaddleCustomer   func(ctx context.Context, customerID string) (*paddle.Customer, error)
	paddleWebhookSecret string
	priceCatalog        PriceCatalog
	auditService        audit_service.Service
}

func New(deps Dependencies) Service {
	return &service{
		validator:           deps.Validator,
		logger:              deps.Logger,
		authorizer:          deps.Authorizer,
		organizationsRepo:   deps.Repositories.Organizations,
		billingAccountsRepo: deps.Repositories.OrganizationBillingAccounts,
		paddleCustomer:      deps.PaddleCustomer,
		paddleSubscription:  deps.PaddleSubscription,
		paddleTransaction:   deps.PaddleTransaction,
		paddleWebhookSecret: deps.PaddleWebhookSecret,
		priceCatalog:        deps.PriceCatalog,
		auditService:        deps.AuditService,
	}
}

func (s *service) customerInput(input *CreateCustomerInput) chi_paddle_customer.CustomerInput {
	req := chi_paddle_customer.CustomerInput{
		OrganizationID: input.OrganizationID,
		Name:           input.OrganizationName,
		BillingEmail:   input.BillingEmail,
		Address:        input.Address,
		City:           input.City,
		Zip:            input.Zip,
		Country:        input.Country,
		Timezone:       input.Timezone,
	}
	return req
}

func (s *service) syncSubscriptionExpiry(account *models.OrganizationBillingAccount, paddleSub *paddle.Subscription) {
	if paddleSub == nil {
		return
	}
	if paddleSub.CurrentBillingPeriod != nil && paddleSub.CurrentBillingPeriod.EndsAt != "" {
		if t, err := time.Parse(time.RFC3339, paddleSub.CurrentBillingPeriod.EndsAt); err == nil {
			account.SubscriptionExpiresAt = &t
		}
	}
	if account.SubscriptionExpiresAt == nil && paddleSub.NextBilledAt != nil && *paddleSub.NextBilledAt != "" {
		if t, err := time.Parse(time.RFC3339, *paddleSub.NextBilledAt); err == nil {
			account.SubscriptionExpiresAt = &t
		}
	}
}

func (s *service) CreateCustomer(ctx context.Context, input *CreateCustomerInput) (string, error) {
	if s.paddleCustomer == nil {
		return "", chi_error.NewInternalServerError(errors.New("paddle not configured"), "InternalServerError", nil)
	}
	customer, err := s.paddleCustomer.CreateCustomer(ctx, s.customerInput(input))
	if err != nil {
		return "", err
	}
	if customer == nil || customer.ID == "" {
		return "", chi_error.NewInternalServerError(errors.New("paddle customer missing id"), "InternalServerError", nil)
	}
	return customer.ID, nil
}

func (s *service) UpdateCustomer(ctx context.Context, input *UpdateCustomerInput) error {
	if s.paddleCustomer == nil || input.BillingAccount == nil || input.BillingAccount.ProviderCustomerID == "" {
		return nil
	}
	createInput := &CreateCustomerInput{
		OrganizationID:   input.OrganizationID,
		OrganizationName: input.OrganizationName,
		BillingEmail:     input.BillingAccount.BillingEmail,
		Address:          input.Address,
		City:             input.City,
		Zip:              input.Zip,
		Country:          input.Country,
		Timezone:         input.Timezone,
	}
	_, err := s.paddleCustomer.UpdateCustomer(ctx, input.BillingAccount.ProviderCustomerID, s.customerInput(createInput))
	return err
}

func (s *service) ReleaseProvisionedCustomer(ctx context.Context, organizationID string, billingAccount *models.OrganizationBillingAccount) error {
	if s.paddleCustomer == nil || billingAccount == nil || billingAccount.ProviderCustomerID == "" {
		return nil
	}
	if s.getPaddleCustomer != nil {
		customer, err := s.getPaddleCustomer(ctx, billingAccount.ProviderCustomerID)
		if err != nil {
			s.logger.WithContext(ctx).Error("failed to load paddle customer for release", "error", err, "organizationId", organizationID, "paddleCustomerId", billingAccount.ProviderCustomerID)
			return err
		}
		if customer.CustomData == nil || customer.CustomData["organization_id"] != organizationID {
			return nil
		}
	}
	if err := s.paddleCustomer.ArchiveCustomer(ctx, billingAccount.ProviderCustomerID); err != nil {
		s.logger.WithContext(ctx).Error("failed to archive paddle customer after org provision rollback", "error", err, "organizationId", organizationID, "paddleCustomerId", billingAccount.ProviderCustomerID)
		return err
	}
	return nil
}

func (s *service) CancelSubscription(ctx context.Context, providerSubscriptionID string) error {
	if s.paddleSubscription == nil || providerSubscriptionID == "" {
		return nil
	}
	_, err := s.paddleSubscription.CancelSubscription(ctx, providerSubscriptionID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to cancel paddle subscription", "error", err, "paddleSubscriptionId", providerSubscriptionID)
	}
	return err
}

func (s *service) loadBillingAccount(ctx context.Context, organizationID string) (*models.OrganizationBillingAccount, error) {
	account, err := s.billingAccountsRepo.GetOrganizationBillingAccountByOrganizationID(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	if account.ProviderCustomerID == "" {
		return nil, chi_error.NewInternalServerError(errors.New("organization without paddle customer"), "InternalServerError", nil)
	}
	return account, nil
}

func (s *service) CreateCheckoutSession(ctx context.Context, req *CreateCheckoutSessionRequest, access *chi_types.AccessInfo) (*CheckoutSessionResponse, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}
	if err := s.authorizer.CheckOrganizationPermission(access, req.OrganizationID, constants.PERMISSION_SUBSCRIPTION_WRITE); err != nil {
		return nil, err
	}
	if _, err := s.organizationsRepo.GetOrganizationByID(ctx, req.OrganizationID); err != nil {
		return nil, err
	}

	billingAccount, err := s.loadBillingAccount(ctx, req.OrganizationID)
	if err != nil {
		return nil, err
	}

	result, err := s.paddleTransaction.CreateCheckoutSession(ctx, billingAccount.ProviderCustomerID, req.PlanID)
	if err != nil {
		return nil, err
	}
	return &CheckoutSessionResponse{TransactionID: result.TransactionID}, nil
}

func (s *service) CreateCustomerPortalSession(ctx context.Context, req *CreateCustomerPortalSessionRequest, access *chi_types.AccessInfo) (*CustomerPortalSessionResponse, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}
	if err := s.authorizer.CheckOrganizationPermission(access, req.OrganizationID, constants.PERMISSION_SUBSCRIPTION_WRITE); err != nil {
		return nil, err
	}
	if _, err := s.organizationsRepo.GetOrganizationByID(ctx, req.OrganizationID); err != nil {
		return nil, err
	}

	billingAccount, err := s.loadBillingAccount(ctx, req.OrganizationID)
	if err != nil {
		return nil, err
	}

	result, err := s.paddleCustomer.CreateCustomerPortalSession(ctx, billingAccount.ProviderCustomerID)
	if err != nil {
		return nil, err
	}
	return &CustomerPortalSessionResponse{PortalURL: result.URL}, nil
}

func (s *service) ProcessTransaction(ctx context.Context, req *ProcessTransactionRequest, access *chi_types.AccessInfo) (*models.OrganizationBillingAccount, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}
	if err := s.authorizer.CheckOrganizationPermission(access, req.OrganizationID, constants.PERMISSION_SUBSCRIPTION_WRITE); err != nil {
		return nil, err
	}
	if _, err := s.organizationsRepo.GetOrganizationByID(ctx, req.OrganizationID); err != nil {
		return nil, err
	}

	billingAccount, err := s.loadBillingAccount(ctx, req.OrganizationID)
	if err != nil {
		return nil, err
	}

	tx, err := s.paddleTransaction.GetTransaction(ctx, req.TransactionID)
	if err != nil {
		return nil, err
	}
	if tx.CustomerID == nil || *tx.CustomerID != billingAccount.ProviderCustomerID {
		return nil, chi_error.NewForbiddenError(errors.New("transaction customer mismatch"), "Forbidden", nil)
	}

	switch tx.Status {
	case paddle.TransactionStatusCompleted,
		paddle.TransactionStatusBilled,
		paddle.TransactionStatus("paid"):
	default:
		return nil, chi_error.NewUnprocessableEntityError(errors.New("transaction is not completed"), "UnprocessableEntity", nil)
	}

	var ourPriceID string
	for _, item := range tx.Items {
		if s.priceCatalog.IsOurPriceID(item.Price.ID) {
			ourPriceID = item.Price.ID
			break
		}
	}
	if ourPriceID == "" {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("transaction is not for this product"), "UnprocessableEntity", nil)
	}
	if req.PriceID != "" && req.PriceID != ourPriceID {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("transaction price mismatch"), "UnprocessableEntity", nil)
	}

	if tx.SubscriptionID != nil && *tx.SubscriptionID != "" {
		billingAccount.ProviderSubscriptionID = *tx.SubscriptionID
	}
	billingAccount.SubscriptionInTrial = false
	if tx.BillingPeriod != nil && tx.BillingPeriod.EndsAt != "" {
		if t, parseErr := time.Parse(time.RFC3339, tx.BillingPeriod.EndsAt); parseErr == nil {
			billingAccount.SubscriptionExpiresAt = &t
		}
	}
	applySubscriptionFromPrice(billingAccount, s.priceCatalog, ourPriceID)

	if err := s.billingAccountsRepo.UpdateOrganizationBillingAccount(ctx, billingAccount); err != nil {
		return nil, err
	}
	return billingAccount, nil
}

func (s *service) ChangePlan(ctx context.Context, req *ChangePlanRequest, access *chi_types.AccessInfo) (*ChangePlanResponse, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}
	if err := s.authorizer.CheckOrganizationPermission(access, req.OrganizationID, constants.PERMISSION_SUBSCRIPTION_WRITE); err != nil {
		return nil, err
	}
	if _, err := s.organizationsRepo.GetOrganizationByID(ctx, req.OrganizationID); err != nil {
		return nil, err
	}

	account, err := s.billingAccountsRepo.GetOrganizationBillingAccountByOrganizationID(ctx, req.OrganizationID)
	if err != nil {
		return nil, err
	}
	if account.ProviderSubscriptionID == "" {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("organization has no paddle subscription"), "UnprocessableEntity", nil)
	}

	currentTier := normalizeTier(account.SubscriptionTier)
	currentInterval := account.SubscriptionPaymentInterval
	targetTier := s.priceCatalog.TierFromPriceID(req.PlanID)
	targetInterval := s.priceCatalog.IntervalFromPriceID(req.PlanID)

	if targetTier == currentTier && targetInterval == currentInterval {
		return &ChangePlanResponse{BillingAccount: account, EffectiveAt: constants.PLAN_EFFECTIVE_IMMEDIATELY}, nil
	}

	tierDowngrade := tierRank(targetTier) < tierRank(currentTier)
	sameTierAnnualToMonthly := targetTier == currentTier &&
		targetInterval == constants.PAYMENT_INTERVAL_MONTHLY && currentInterval == constants.PAYMENT_INTERVAL_ANNUAL
	if tierDowngrade || sameTierAnnualToMonthly {
		account.SubscriptionScheduledPlanPriceID = &req.PlanID
		if err := s.billingAccountsRepo.UpdateOrganizationBillingAccount(ctx, account); err != nil {
			return nil, err
		}
		s.auditSubscriptionChange(ctx, req.OrganizationID, access, currentTier, currentInterval, targetTier, targetInterval, constants.PLAN_EFFECTIVE_NEXT_BILLING_PERIOD)
		return &ChangePlanResponse{BillingAccount: account, EffectiveAt: constants.PLAN_EFFECTIVE_NEXT_BILLING_PERIOD}, nil
	}

	account.SubscriptionScheduledPlanPriceID = nil
	paddleSub, err := s.paddleSubscription.UpdateSubscriptionItems(ctx, account.ProviderSubscriptionID, req.PlanID)
	if err != nil {
		return nil, chi_error.NewUnprocessableEntityError(err, "SubscriptionChangeFailed", nil)
	}

	applySubscriptionFromPrice(account, s.priceCatalog, req.PlanID)
	s.syncSubscriptionExpiry(account, paddleSub)
	if err := s.billingAccountsRepo.UpdateOrganizationBillingAccount(ctx, account); err != nil {
		return nil, err
	}
	s.auditSubscriptionChange(ctx, req.OrganizationID, access, currentTier, currentInterval, targetTier, targetInterval, constants.PLAN_EFFECTIVE_IMMEDIATELY)
	return &ChangePlanResponse{BillingAccount: account, EffectiveAt: constants.PLAN_EFFECTIVE_IMMEDIATELY}, nil
}

func (s *service) HandleWebhook(ctx context.Context, payload []byte, signature string) error {
	if !chi_paddle.VerifyWebhook(s.paddleWebhookSecret, payload, signature) {
		return chi_error.NewUnauthorizedError(errors.New("invalid paddle signature"), "Unauthorized", nil)
	}

	var event chi_paddle.WebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return chi_error.NewUnprocessableEntityError(err, "UnprocessableEntity", nil)
	}

	switch event.EventType {
	case "subscription.created", "subscription.trialing":
		return s.handleSubscriptionUpdate(ctx, event.Data, true)
	case "subscription.updated", "subscription.activated":
		return s.handleSubscriptionUpdate(ctx, event.Data, false)
	case "subscription.canceled":
		return s.handleSubscriptionCanceled(ctx, event.Data)
	case "transaction.completed":
		return s.handleTransactionCompleted(ctx, event.Data)
	default:
		return nil
	}
}

func (s *service) handleSubscriptionUpdate(ctx context.Context, data map[string]any, applyPlan bool) error {
	subscriptionData, ok := data["subscription"].(map[string]any)
	if !ok {
		subscriptionData = data
	}

	subscriptionID, _ := subscriptionData["id"].(string)
	if subscriptionID == "" {
		return chi_error.NewUnprocessableEntityError(errors.New("missing subscription id"), "MissingPaddleSubscriptionID", nil)
	}

	customerID, _ := subscriptionData["customer_id"].(string)
	if customerID == "" {
		return chi_error.NewUnprocessableEntityError(errors.New("missing customer id"), "MissingPaddleCustomerID", nil)
	}

	var items []any
	if rawItems, ok := subscriptionData["items"].([]any); ok {
		items = rawItems
	}
	ourPriceID := s.priceCatalog.OurPriceIDFromWebhookItems(items)
	if ourPriceID == "" {
		return nil
	}

	billingAccount, err := s.billingAccountsRepo.GetOrganizationBillingAccountByProviderAndProviderCustomerID(ctx, constants.BILLING_PROVIDER_PADDLE, customerID)
	if err != nil {
		if webhookNotRelevant(err) {
			return nil
		}
		return err
	}

	if !applyPlan && billingAccount.ProviderSubscriptionID != "" && billingAccount.ProviderSubscriptionID != subscriptionID {
		return nil
	}

	billingAccount.ProviderSubscriptionID = subscriptionID
	if status, _ := subscriptionData["status"].(string); status == "trialing" {
		billingAccount.SubscriptionInTrial = true
	} else {
		billingAccount.SubscriptionInTrial = false
	}
	if applyPlan {
		applySubscriptionFromPrice(billingAccount, s.priceCatalog, ourPriceID)
	}

	if currentBillingPeriod, ok := subscriptionData["current_billing_period"].(map[string]any); ok {
		if endsAt, ok := currentBillingPeriod["ends_at"].(string); ok {
			if t, parseErr := time.Parse(time.RFC3339, endsAt); parseErr == nil {
				billingAccount.SubscriptionExpiresAt = &t
			}
		}
	} else if nextBilledAt, ok := subscriptionData["next_billed_at"].(string); ok {
		if t, parseErr := time.Parse(time.RFC3339, nextBilledAt); parseErr == nil {
			billingAccount.SubscriptionExpiresAt = &t
		}
	}

	return s.billingAccountsRepo.UpdateOrganizationBillingAccount(ctx, billingAccount)
}

func (s *service) handleSubscriptionCanceled(ctx context.Context, data map[string]any) error {
	subscriptionData, ok := data["subscription"].(map[string]any)
	if !ok {
		subscriptionData = data
	}

	customerID, _ := subscriptionData["customer_id"].(string)
	if customerID == "" {
		return chi_error.NewUnprocessableEntityError(errors.New("missing customer id"), "MissingPaddleCustomerID", nil)
	}

	billingAccount, err := s.billingAccountsRepo.GetOrganizationBillingAccountByProviderAndProviderCustomerID(ctx, constants.BILLING_PROVIDER_PADDLE, customerID)
	if err != nil {
		if webhookNotRelevant(err) {
			return nil
		}
		return err
	}

	webhookSubID, _ := subscriptionData["id"].(string)
	if billingAccount.ProviderSubscriptionID != webhookSubID {
		return nil
	}

	billingAccount.SubscriptionTier = constants.TIER_FREE
	billingAccount.SubscriptionSeats = constants.SUBSCRIPTION_TYPE_SEATS_INCLUDED_FREE
	billingAccount.SubscriptionPaymentInterval = constants.PAYMENT_INTERVAL_MONTHLY
	billingAccount.SubscriptionInTrial = false
	billingAccount.ProviderSubscriptionID = ""

	if canceledAt, ok := subscriptionData["canceled_at"].(string); ok {
		if t, parseErr := time.Parse(time.RFC3339, canceledAt); parseErr == nil {
			billingAccount.SubscriptionExpiresAt = &t
		}
	} else if currentBillingPeriod, ok := subscriptionData["current_billing_period"].(map[string]any); ok {
		if endsAt, ok := currentBillingPeriod["ends_at"].(string); ok {
			if t, parseErr := time.Parse(time.RFC3339, endsAt); parseErr == nil {
				billingAccount.SubscriptionExpiresAt = &t
			}
		}
	}

	return s.billingAccountsRepo.UpdateOrganizationBillingAccount(ctx, billingAccount)
}

func (s *service) handleTransactionCompleted(ctx context.Context, data map[string]any) error {
	transactionData, ok := data["transaction"].(map[string]any)
	if !ok {
		transactionData = data
	}

	customerID, _ := transactionData["customer_id"].(string)
	if customerID == "" {
		return chi_error.NewUnprocessableEntityError(errors.New("missing customer id"), "MissingPaddleCustomerID", nil)
	}

	var items []any
	if rawItems, ok := transactionData["items"].([]any); ok {
		items = rawItems
	}
	ourPriceID := s.priceCatalog.OurPriceIDFromWebhookItems(items)
	if ourPriceID == "" {
		return nil
	}

	billingAccount, err := s.billingAccountsRepo.GetOrganizationBillingAccountByProviderAndProviderCustomerID(ctx, constants.BILLING_PROVIDER_PADDLE, customerID)
	if err != nil {
		if webhookNotRelevant(err) {
			return nil
		}
		return err
	}

	if subscriptionID, ok := transactionData["subscription_id"].(string); ok && subscriptionID != "" {
		billingAccount.ProviderSubscriptionID = subscriptionID
	}
	billingAccount.SubscriptionInTrial = false
	applySubscriptionFromPrice(billingAccount, s.priceCatalog, ourPriceID)

	if billingPeriod, ok := transactionData["billing_period"].(map[string]any); ok {
		if endsAt, ok := billingPeriod["ends_at"].(string); ok {
			if t, parseErr := time.Parse(time.RFC3339, endsAt); parseErr == nil {
				billingAccount.SubscriptionExpiresAt = &t
			}
		}
	}

	return s.billingAccountsRepo.UpdateOrganizationBillingAccount(ctx, billingAccount)
}

func (s *service) ApplyScheduledPlanChanges(ctx context.Context) error {
	accounts, err := s.billingAccountsRepo.ListOrganizationBillingAccountsWithScheduledPlanChangeDue(ctx)
	if err != nil {
		return err
	}
	if accounts == nil || len(*accounts) == 0 {
		return nil
	}

	for _, account := range *accounts {
		if account.SubscriptionScheduledPlanPriceID == nil || *account.SubscriptionScheduledPlanPriceID == "" || account.ProviderSubscriptionID == "" {
			continue
		}
		planID := *account.SubscriptionScheduledPlanPriceID
		paddleSub, err := s.paddleSubscription.UpdateSubscriptionItems(ctx, account.ProviderSubscriptionID, planID)
		if err != nil {
			s.logger.WithContext(ctx).Error("apply scheduled plan: paddle update failed", "error", err, "organizationId", account.OrganizationID.String())
			continue
		}

		account.SubscriptionScheduledPlanPriceID = nil
		applySubscriptionFromPrice(&account, s.priceCatalog, planID)
		s.syncSubscriptionExpiry(&account, paddleSub)
		if err := s.billingAccountsRepo.UpdateOrganizationBillingAccount(ctx, &account); err != nil {
			s.logger.WithContext(ctx).Error("apply scheduled plan: billing account update failed", "error", err, "organizationId", account.OrganizationID.String())
		}
	}
	return nil
}

func (s *service) auditSubscriptionChange(ctx context.Context, organizationID string, access *chi_types.AccessInfo, fromTier string, fromInterval string, toTier string, toInterval string, effectiveAt string) {
	data, _ := json.Marshal(audit.UpdatePayload(
		map[string]any{
			"tier":     fromTier,
			"interval": fromInterval,
		},
		map[string]any{
			"tier":        toTier,
			"interval":    toInterval,
			"effectiveAt": effectiveAt,
		},
	))
	raw := json.RawMessage(data)
	if _, err := s.auditService.CreateAuditLog(ctx, &audit_service.CreateAuditLogRequest{
		OrganizationID: organizationID,
		Action:         constants.AUDIT_ACTION_TYPE_UPDATE,
		ResourceType:   constants.RESOURCE_TYPE_SUBSCRIPTION,
		ResourceID:     organizationID,
		ResourceName:   "Subscription",
		Data:           &raw,
	}, access); err != nil {
		s.logger.WithContext(ctx).Error("failed to create subscription audit log", "error", err, "organizationId", organizationID)
	}
}
