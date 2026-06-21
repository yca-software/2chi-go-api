package audit_service_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/yca-software/2chi-go-api/internals/constants"
	"github.com/yca-software/2chi-go-api/internals/models"
	"github.com/yca-software/2chi-go-api/internals/packages/authz"
	"github.com/yca-software/2chi-go-api/internals/repositories"
	audit_log_repository "github.com/yca-software/2chi-go-api/internals/repositories/audit_log"
	billing_account_repository "github.com/yca-software/2chi-go-api/internals/repositories/billing_account"
	organization_repository "github.com/yca-software/2chi-go-api/internals/repositories/organization"
	audit_service "github.com/yca-software/2chi-go-api/internals/services/audit"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_types "github.com/yca-software/2chi-go-types"
	chi_validator "github.com/yca-software/2chi-go-validator"
)

type AuditServiceSuite struct {
	suite.Suite
	ctx             context.Context
	now             time.Time
	orgID           uuid.UUID
	auditRepo       *audit_log_repository.MockAuditLogsRepository
	orgsRepo        *organization_repository.MockOrganizationsRepository
	billingAccounts *billing_account_repository.MockOrganizationBillingAccountsRepository
	svc             audit_service.Service
}

func TestAuditServiceSuite(t *testing.T) {
	suite.Run(t, new(AuditServiceSuite))
}

func (s *AuditServiceSuite) SetupTest() {
	s.ctx = context.Background()
	s.now = fixedNow()
	s.orgID = uuid.New()
	s.auditRepo = &audit_log_repository.MockAuditLogsRepository{}
	s.orgsRepo = &organization_repository.MockOrganizationsRepository{}
	s.billingAccounts = &billing_account_repository.MockOrganizationBillingAccountsRepository{}

	s.svc = audit_service.New(audit_service.Dependencies{
		GenerateID: uuid.NewV7,
		Now:        func() time.Time { return s.now },
		Validator:  chi_validator.New(),
		Logger:     mockLogger(),
		Authorizer: authz.NewAuthorizer(func() time.Time { return s.now }),
		Repositories: &repositories.Repositories{
			AuditLogs:                   s.auditRepo,
			Organizations:               s.orgsRepo,
			OrganizationBillingAccounts: s.billingAccounts,
		},
	})
}

func (s *AuditServiceSuite) readAccess() *chi_types.AccessInfo {
	return &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeUser,
		SubjectID: uuid.New(),
		Email:     "admin@example.com",
		Roles: []chi_types.JWTAccessTokenPermissionData{{
			OrganizationID: s.orgID,
			Permissions:    []string{constants.PERMISSION_AUDIT_READ},
		}},
	}
}

func (s *AuditServiceSuite) TestCreateAuditLog_Success() {
	resourceID := uuid.New()
	s.auditRepo.On("CreateAuditLog", s.ctx, mock.AnythingOfType("*models.AuditLog")).Return(nil).Once()

	log, err := s.svc.CreateAuditLog(s.ctx, &audit_service.CreateAuditLogRequest{
		OrganizationID: s.orgID.String(),
		Action:         constants.AUDIT_ACTION_TYPE_CREATE,
		ResourceType:   constants.RESOURCE_TYPE_ORGANIZATION,
		ResourceID:     resourceID.String(),
		ResourceName:   "Acme",
	}, s.readAccess())
	s.Require().NoError(err)
	s.Equal(resourceID, log.ResourceID)
}

func (s *AuditServiceSuite) TestCreateAuditLog_ImpersonationAttribution() {
	adminID := uuid.New()
	resourceID := uuid.New()
	access := &chi_types.AccessInfo{
		Type:                chi_types.AccessTypeUser,
		SubjectID:           uuid.New(),
		Email:               "target@example.com",
		ImpersonatedBy:      uuid.NullUUID{UUID: adminID, Valid: true},
		ImpersonatedByEmail: "admin@example.com",
	}
	s.auditRepo.On("CreateAuditLog", s.ctx, mock.MatchedBy(func(log *models.AuditLog) bool {
		return log.ImpersonatedByID.Valid &&
			log.ImpersonatedByID.UUID == adminID &&
			log.ImpersonatedByEmail == "admin@example.com"
	})).Return(nil).Once()

	log, err := s.svc.CreateAuditLog(s.ctx, &audit_service.CreateAuditLogRequest{
		OrganizationID: s.orgID.String(),
		Action:         constants.AUDIT_ACTION_TYPE_UPDATE,
		ResourceType:   constants.RESOURCE_TYPE_ORGANIZATION,
		ResourceID:     resourceID.String(),
		ResourceName:   "Acme",
	}, access)
	s.Require().NoError(err)
	s.True(log.ImpersonatedByID.Valid)
	s.Equal(adminID, log.ImpersonatedByID.UUID)
	s.Equal("admin@example.com", log.ImpersonatedByEmail)
}

func (s *AuditServiceSuite) TestCreateAuditLog_Validation_MissingAction() {
	log, err := s.svc.CreateAuditLog(s.ctx, &audit_service.CreateAuditLogRequest{
		OrganizationID: s.orgID.String(),
		ResourceType:   constants.RESOURCE_TYPE_ORGANIZATION,
		ResourceID:     uuid.New().String(),
		ResourceName:   "Acme",
	}, s.readAccess())
	s.Error(err)
	s.Nil(log)
}

func (s *AuditServiceSuite) TestListAuditLogs_FreePlanFeatureDenied() {
	s.orgsRepo.On("GetOrganizationByID", s.ctx, s.orgID.String()).
		Return(organization(s.orgID, "Acme"), nil).Once()
	s.billingAccounts.On("GetOrganizationBillingAccountByOrganizationID", s.ctx, s.orgID.String()).
		Return(billingAccount(s.orgID, constants.TIER_FREE, constants.BILLING_PROVIDER_PADDLE, nil), nil).Once()

	_, err := s.svc.ListAuditLogsForOrganization(s.ctx, &audit_service.ListAuditLogsForOrganizationRequest{
		OrganizationID: s.orgID.String(),
		Limit:          20,
	}, s.readAccess())
	s.Error(err)
}

func (s *AuditServiceSuite) TestListAuditLogs_Success() {
	logs := []models.AuditLog{{
		ID:             uuid.New(),
		OrganizationID: s.orgID,
		Action:         constants.AUDIT_ACTION_TYPE_CREATE,
		ResourceType:   constants.RESOURCE_TYPE_ORGANIZATION,
		ResourceID:     uuid.New(),
	}}
	expiresAt := s.now.Add(24 * time.Hour)
	s.orgsRepo.On("GetOrganizationByID", s.ctx, s.orgID.String()).
		Return(organization(s.orgID, "Acme"), nil).Once()
	s.billingAccounts.On("GetOrganizationBillingAccountByOrganizationID", s.ctx, s.orgID.String()).
		Return(billingAccount(s.orgID, constants.TIER_BASIC, constants.BILLING_PROVIDER_PADDLE, &expiresAt), nil).Once()
	s.auditRepo.On("ListAuditLogsByOrganizationID", s.ctx, s.orgID.String(), mock.Anything, 21, 0).
		Return(&logs, nil).Once()

	resp, err := s.svc.ListAuditLogsForOrganization(s.ctx, &audit_service.ListAuditLogsForOrganizationRequest{
		OrganizationID: s.orgID.String(),
		Limit:          20,
	}, s.readAccess())
	s.Require().NoError(err)
	s.Len(resp.Items, 1)
}

func (s *AuditServiceSuite) TestListAuditLogs_HasNext() {
	logs := make([]models.AuditLog, 21)
	for i := range logs {
		logs[i] = models.AuditLog{
			ID:             uuid.New(),
			OrganizationID: s.orgID,
			Action:         constants.AUDIT_ACTION_TYPE_CREATE,
			ResourceType:   constants.RESOURCE_TYPE_ORGANIZATION,
			ResourceID:     uuid.New(),
		}
	}
	expiresAt := s.now.Add(24 * time.Hour)
	s.orgsRepo.On("GetOrganizationByID", s.ctx, s.orgID.String()).
		Return(organization(s.orgID, "Acme"), nil).Once()
	s.billingAccounts.On("GetOrganizationBillingAccountByOrganizationID", s.ctx, s.orgID.String()).
		Return(billingAccount(s.orgID, constants.TIER_BASIC, constants.BILLING_PROVIDER_PADDLE, &expiresAt), nil).Once()
	s.auditRepo.On("ListAuditLogsByOrganizationID", s.ctx, s.orgID.String(), mock.Anything, 21, 0).
		Return(&logs, nil).Once()

	resp, err := s.svc.ListAuditLogsForOrganization(s.ctx, &audit_service.ListAuditLogsForOrganizationRequest{
		OrganizationID: s.orgID.String(),
		Limit:          20,
	}, s.readAccess())
	s.Require().NoError(err)
	s.True(resp.HasNext)
	s.Len(resp.Items, 20)
}

func (s *AuditServiceSuite) TestCreateAuditLog_SanitizesData() {
	raw := json.RawMessage(`{"password":"secret","name":"Acme"}`)
	s.auditRepo.On("CreateAuditLog", s.ctx, mock.MatchedBy(func(log *models.AuditLog) bool {
		if log.Data == nil {
			return false
		}
		var payload map[string]any
		if err := json.Unmarshal(*log.Data, &payload); err != nil {
			return false
		}
		_, hasPassword := payload["password"]
		return hasPassword && payload["password"] == "[redacted]" && payload["name"] == "Acme"
	})).Return(nil).Once()

	_, err := s.svc.CreateAuditLog(s.ctx, &audit_service.CreateAuditLogRequest{
		OrganizationID: s.orgID.String(),
		Action:         constants.AUDIT_ACTION_TYPE_UPDATE,
		ResourceType:   constants.RESOURCE_TYPE_ORGANIZATION,
		ResourceID:     uuid.New().String(),
		ResourceName:   "Acme",
		Data:           &raw,
	}, s.readAccess())
	s.NoError(err)
}

func (s *AuditServiceSuite) TestListAuditLogs_Validation() {
	_, err := s.svc.ListAuditLogsForOrganization(s.ctx, &audit_service.ListAuditLogsForOrganizationRequest{
		OrganizationID: s.orgID.String(),
		Limit:          0,
	}, s.readAccess())
	s.Error(err)
}

func fixedNow() time.Time {
	return time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
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

func organization(orgID uuid.UUID, name string) *models.Organization {
	return &models.Organization{
		ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
			ModelBase: chi_types.ModelBase{ID: orgID},
		},
		Name: name,
	}
}

func billingAccount(orgID uuid.UUID, tier, provider string, expiresAt *time.Time) *models.OrganizationBillingAccount {
	return &models.OrganizationBillingAccount{
		ModelBase:             chi_types.ModelBase{ID: orgID},
		OrganizationID:        orgID,
		Provider:              provider,
		SubscriptionTier:      tier,
		SubscriptionExpiresAt: expiresAt,
	}
}
