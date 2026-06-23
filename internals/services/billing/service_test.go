package billing_service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/yca-software/2chi-go-api/internals/constants"
	"github.com/yca-software/2chi-go-api/internals/models"
	"github.com/yca-software/2chi-go-api/internals/packages/authz"
	"github.com/yca-software/2chi-go-api/internals/repositories"
	billing_account_repository "github.com/yca-software/2chi-go-api/internals/repositories/billing_account"
	organization_repository "github.com/yca-software/2chi-go-api/internals/repositories/organization"
	audit_service "github.com/yca-software/2chi-go-api/internals/services/audit"
	billing_service "github.com/yca-software/2chi-go-api/internals/services/billing"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_paddle_customer "github.com/yca-software/2chi-go-paddle/customer"
	chi_paddle_subscription "github.com/yca-software/2chi-go-paddle/subscription"
	chi_paddle_transaction "github.com/yca-software/2chi-go-paddle/transaction"
	chi_types "github.com/yca-software/2chi-go-types"
	chi_validator "github.com/yca-software/2chi-go-validator"
)

type BillingServiceSuite struct {
	suite.Suite
	ctx                 context.Context
	now                 time.Time
	orgID               uuid.UUID
	orgsRepo            *organization_repository.MockRepository
	billingAccountsRepo *billing_account_repository.MockRepository
	paddleCustomer      *chi_paddle_customer.MockCustomerService
	paddleSubscription  *chi_paddle_subscription.MockSubscriptionService
	paddleTransaction   *chi_paddle_transaction.MockTransactionService
	auditSvc            *audit_service.MockService
	svc                 billing_service.Service
	priceID             string
}

func TestBillingServiceSuite(t *testing.T) {
	suite.Run(t, new(BillingServiceSuite))
}

func (s *BillingServiceSuite) SetupTest() {
	s.ctx = context.Background()
	s.now = time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	s.orgID = uuid.New()
	s.priceID = "pri_basic_monthly"
	s.orgsRepo = &organization_repository.MockRepository{}
	s.billingAccountsRepo = &billing_account_repository.MockRepository{}
	s.paddleCustomer = &chi_paddle_customer.MockCustomerService{}
	s.paddleSubscription = &chi_paddle_subscription.MockSubscriptionService{}
	s.paddleTransaction = &chi_paddle_transaction.MockTransactionService{}
	s.auditSvc = &audit_service.MockService{}

	s.svc = billing_service.New(billing_service.Dependencies{
		Validator:  chi_validator.New(),
		Logger:     mockLogger(),
		Authorizer: authz.NewAuthorizer(func() time.Time { return s.now }),
		Repositories: &repositories.Repositories{
			Organizations:               s.orgsRepo,
			OrganizationBillingAccounts: s.billingAccountsRepo,
		},
		PaddleCustomer:     s.paddleCustomer,
		PaddleSubscription: s.paddleSubscription,
		PaddleTransaction:  s.paddleTransaction,
		PriceCatalog: billing_service.PriceCatalog{
			PriceIDs: billing_service.PriceIDs{BasicMonthly: s.priceID},
		},
		AuditService: s.auditSvc,
	})
}

func (s *BillingServiceSuite) writeAccess() *chi_types.AccessInfo {
	return &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeUser,
		SubjectID: uuid.New(),
		Roles: []chi_types.JWTAccessTokenPermissionData{{
			OrganizationID: s.orgID,
			Permissions:    []string{constants.PERMISSION_SUBSCRIPTION_WRITE},
		}},
	}
}

func (s *BillingServiceSuite) organization() *models.Organization {
	return &models.Organization{
		ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
			ModelBase: chi_types.ModelBase{ID: s.orgID},
		},
		Name: "Acme",
	}
}

func (s *BillingServiceSuite) billingAccount(customerID string) *models.OrganizationBillingAccount {
	return &models.OrganizationBillingAccount{
		OrganizationID:              s.orgID,
		Provider:                    constants.BILLING_PROVIDER_PADDLE,
		ProviderCustomerID:          customerID,
		SubscriptionTier:            constants.TIER_BASIC,
		SubscriptionSeats:           constants.SUBSCRIPTION_TYPE_SEATS_INCLUDED_BASIC,
		SubscriptionPaymentInterval: constants.PAYMENT_INTERVAL_MONTHLY,
	}
}

func (s *BillingServiceSuite) TestCreateCheckoutSession_Validation() {
	resp, err := s.svc.CreateCheckoutSession(s.ctx, &billing_service.CreateCheckoutSessionRequest{
		OrganizationID: "not-a-uuid",
		PlanID:         s.priceID,
	}, s.writeAccess())
	s.Error(err)
	s.Nil(resp)
}

func (s *BillingServiceSuite) TestCreateCheckoutSession_MissingPaddleCustomer() {
	s.orgsRepo.On("GetByID", s.ctx, s.orgID.String()).
		Return(s.organization(), nil).Once()
	s.billingAccountsRepo.On("GetByOrganizationID", s.ctx, s.orgID.String()).
		Return(s.billingAccount(""), nil).Once()

	resp, err := s.svc.CreateCheckoutSession(s.ctx, &billing_service.CreateCheckoutSessionRequest{
		OrganizationID: s.orgID.String(),
		PlanID:         s.priceID,
	}, s.writeAccess())
	s.Error(err)
	s.Nil(resp)
	if apiErr, ok := chi_error.AsError(err); ok {
		s.Equal("InternalServerError", apiErr.ErrorCode)
	}
}

func (s *BillingServiceSuite) TestCreateCheckoutSession_Success() {
	s.orgsRepo.On("GetByID", s.ctx, s.orgID.String()).
		Return(s.organization(), nil).Once()
	s.billingAccountsRepo.On("GetByOrganizationID", s.ctx, s.orgID.String()).
		Return(s.billingAccount("ctm_123"), nil).Once()
	s.paddleTransaction.On("CreateCheckoutSession", s.ctx, "ctm_123", s.priceID).
		Return(&chi_paddle_transaction.CheckoutSessionResult{TransactionID: "txn_123"}, nil).Once()

	resp, err := s.svc.CreateCheckoutSession(s.ctx, &billing_service.CreateCheckoutSessionRequest{
		OrganizationID: s.orgID.String(),
		PlanID:         s.priceID,
	}, s.writeAccess())
	s.Require().NoError(err)
	s.Equal("txn_123", resp.TransactionID)
}

func (s *BillingServiceSuite) TestCancelSubscription_NoOpWhenEmptyID() {
	s.NoError(s.svc.CancelSubscription(s.ctx, ""))
}

func (s *BillingServiceSuite) TestCreateCustomerPortalSession_Success() {
	s.orgsRepo.On("GetByID", s.ctx, s.orgID.String()).
		Return(s.organization(), nil).Once()
	s.billingAccountsRepo.On("GetByOrganizationID", s.ctx, s.orgID.String()).
		Return(s.billingAccount("ctm_123"), nil).Once()
	s.paddleCustomer.On("CreateCustomerPortalSession", s.ctx, "ctm_123").
		Return(&chi_paddle_customer.CustomerPortalSessionResult{URL: "https://portal.example.com"}, nil).Once()

	resp, err := s.svc.CreateCustomerPortalSession(s.ctx, &billing_service.CreateCustomerPortalSessionRequest{
		OrganizationID: s.orgID.String(),
	}, s.writeAccess())
	s.Require().NoError(err)
	s.Equal("https://portal.example.com", resp.PortalURL)
}

func (s *BillingServiceSuite) TestApplyScheduledPlanChanges_NoOrganizations() {
	s.billingAccountsRepo.On("ListWithScheduledPlanChangeDue", s.ctx).
		Return(&[]models.OrganizationBillingAccount{}, nil).Once()
	s.NoError(s.svc.ApplyScheduledPlanChanges(s.ctx))
}

func (s *BillingServiceSuite) TestCreateCustomer_PaddleNotConfigured() {
	svc := billing_service.New(billing_service.Dependencies{
		Logger:       mockLogger(),
		Repositories: &repositories.Repositories{},
	})
	_, err := svc.CreateCustomer(s.ctx, &billing_service.CreateCustomerInput{
		OrganizationID: s.orgID.String(),
		BillingEmail:   "b@example.com",
	})
	s.Error(err)
}

func (s *BillingServiceSuite) TestUpdateCustomer_NoOpWithoutClient() {
	svc := billing_service.New(billing_service.Dependencies{
		Logger:       mockLogger(),
		Repositories: &repositories.Repositories{},
	})
	err := svc.UpdateCustomer(s.ctx, &billing_service.UpdateCustomerInput{
		OrganizationID: s.orgID.String(),
		BillingAccount: s.billingAccount("ctm_1"),
	})
	s.NoError(err)
}

func (s *BillingServiceSuite) TestReleaseProvisionedCustomer_NoOpWhenEmptyCustomer() {
	s.NoError(s.svc.ReleaseProvisionedCustomer(s.ctx, s.orgID.String(), s.billingAccount("")))
}

func (s *BillingServiceSuite) TestProcessTransaction_Validation() {
	account, err := s.svc.ProcessTransaction(s.ctx, &billing_service.ProcessTransactionRequest{
		OrganizationID: "bad",
		TransactionID:  "txn",
		PriceID:        s.priceID,
	}, s.writeAccess())
	s.Error(err)
	s.Nil(account)
}

func (s *BillingServiceSuite) TestChangePlan_NoBillingAccount() {
	s.orgsRepo.On("GetByID", s.ctx, s.orgID.String()).
		Return(s.organization(), nil).Once()
	s.billingAccountsRepo.On("GetByOrganizationID", s.ctx, s.orgID.String()).
		Return(nil, chi_error.NewNotFoundError(nil, "NotFound", nil)).Once()

	resp, err := s.svc.ChangePlan(s.ctx, &billing_service.ChangePlanRequest{
		OrganizationID: s.orgID.String(),
		PlanID:         s.priceID,
	}, s.writeAccess())
	s.Error(err)
	s.Nil(resp)
}

func (s *BillingServiceSuite) TestChangePlan_SamePlanNoOp() {
	account := s.billingAccount("ctm_123")
	account.ProviderSubscriptionID = "sub_123"
	s.orgsRepo.On("GetByID", s.ctx, s.orgID.String()).
		Return(s.organization(), nil).Once()
	s.billingAccountsRepo.On("GetByOrganizationID", s.ctx, s.orgID.String()).
		Return(account, nil).Once()

	resp, err := s.svc.ChangePlan(s.ctx, &billing_service.ChangePlanRequest{
		OrganizationID: s.orgID.String(),
		PlanID:         s.priceID,
	}, s.writeAccess())
	s.Require().NoError(err)
	s.Equal(constants.PLAN_EFFECTIVE_IMMEDIATELY, resp.EffectiveAt)
}

func (s *BillingServiceSuite) TestHandleWebhook_InvalidSignature() {
	err := s.svc.HandleWebhook(s.ctx, []byte(`{}`), "bad-signature")
	s.Error(err)
	if apiErr, ok := chi_error.AsError(err); ok {
		s.Equal("Unauthorized", apiErr.ErrorCode)
	}
}

func mockLogger() chi_logger.Logger {
	m := new(chi_logger.MockLogger)
	for n := 0; n <= 8; n++ {
		args := make([]any, n)
		for i := range args {
			args[i] = mock.Anything
		}
		if n == 0 {
			m.On("With").Return(m).Maybe()
			continue
		}
		m.On("With", args...).Return(m).Maybe()
	}
	m.On("WithContext", mock.Anything).Return(m).Maybe()
	for _, method := range []string{"Debug", "Info", "Warn", "Error"} {
		for n := 0; n <= 8; n++ {
			args := make([]any, n+1)
			for i := range args {
				args[i] = mock.Anything
			}
			m.On(method, args...).Return().Maybe()
		}
	}
	return m
}
