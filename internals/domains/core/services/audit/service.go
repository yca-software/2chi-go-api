package audit_service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/yca-software/2chi-go-api/internals/constants"
	"github.com/yca-software/2chi-go-api/internals/domains/core/models"
	"github.com/yca-software/2chi-go-api/internals/domains/core/repositories"
	audit_log_repository "github.com/yca-software/2chi-go-api/internals/domains/core/repositories/audit/audit_log"
	billing_account_repository "github.com/yca-software/2chi-go-api/internals/domains/core/repositories/organization/billing_account"
	organization_repository "github.com/yca-software/2chi-go-api/internals/domains/core/repositories/organization/organization"
	"github.com/yca-software/2chi-go-api/internals/platform/audit"
	"github.com/yca-software/2chi-go-api/internals/platform/authz"
	platform_subscription "github.com/yca-software/2chi-go-api/internals/platform/subscription"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_types "github.com/yca-software/2chi-go-types"
	chi_validator "github.com/yca-software/2chi-go-validator"
)

type Dependencies struct {
	GenerateID   func() (uuid.UUID, error)
	Now          func() time.Time
	Validator    chi_validator.Validator
	Logger       chi_logger.Logger
	Authorizer   *authz.Authorizer
	Repositories *repositories.Repositories
}

type Service interface {
	CreateAuditLog(ctx context.Context, req *CreateAuditLogRequest, access *chi_types.AccessInfo) (*models.AuditLog, error)
	ListAuditLogsForOrganization(ctx context.Context, req *ListAuditLogsForOrganizationRequest, access *chi_types.AccessInfo) (*ListAuditLogsForOrganizationResponse, error)
}

type service struct {
	generateID                      func() (uuid.UUID, error)
	now                             func() time.Time
	validator                       chi_validator.Validator
	logger                          chi_logger.Logger
	authorizer                      *authz.Authorizer
	auditLogsRepo                   audit_log_repository.AuditLogsRepository
	organizationsRepo               organization_repository.OrganizationsRepository
	organizationBillingAccountsRepo billing_account_repository.OrganizationBillingAccountsRepository
}

func New(deps Dependencies) Service {
	return &service{
		generateID:                      deps.GenerateID,
		now:                             deps.Now,
		validator:                       deps.Validator,
		logger:                          deps.Logger,
		authorizer:                      deps.Authorizer,
		auditLogsRepo:                   deps.Repositories.AuditLogs,
		organizationsRepo:               deps.Repositories.Organizations,
		organizationBillingAccountsRepo: deps.Repositories.OrganizationBillingAccounts,
	}
}

// Create audit log. Only used by other services
func (s *service) CreateAuditLog(ctx context.Context, req *CreateAuditLogRequest, access *chi_types.AccessInfo) (*models.AuditLog, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	auditLogID, err := s.generateID()
	if err != nil {
		return nil, err
	}

	now := s.now()
	var actorID uuid.UUID
	var actorInfo string
	if access != nil && access.Type == chi_types.AccessTypeUser {
		actorID = access.SubjectID
		actorInfo = access.Email
	} else if access != nil && access.Type == chi_types.AccessTypeAPIKey {
		actorID = access.SubjectID
		actorInfo = access.Email
	}

	var data *json.RawMessage
	if req.Data != nil {
		sanitized := audit.SanitizeAuditDataJSON(*req.Data)
		data = &sanitized
	}

	auditLog := models.AuditLog{
		ID:             auditLogID,
		CreatedAt:      now,
		OrganizationID: uuid.MustParse(req.OrganizationID),
		ActorID:        actorID,
		ActorInfo:      actorInfo,
		Action:         req.Action,
		ResourceType:   req.ResourceType,
		ResourceID:     uuid.MustParse(req.ResourceID),
		ResourceName:   req.ResourceName,
		Data:           data,
	}

	if access != nil && access.ImpersonatedBy.Valid {
		auditLog.ImpersonatedByID = access.ImpersonatedBy
		auditLog.ImpersonatedByEmail = access.ImpersonatedByEmail
	}

	if err := s.auditLogsRepo.CreateAuditLog(ctx, &auditLog); err != nil {
		return nil, err
	}

	return &auditLog, nil
}

func (s *service) ListAuditLogsForOrganization(ctx context.Context, req *ListAuditLogsForOrganizationRequest, access *chi_types.AccessInfo) (*ListAuditLogsForOrganizationResponse, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	if _, err := s.organizationsRepo.GetOrganizationByID(ctx, req.OrganizationID); err != nil {
		return nil, err
	}

	orgBillingAccount, err := s.organizationBillingAccountsRepo.GetOrganizationBillingAccountByOrganizationID(ctx, req.OrganizationID)
	if err != nil {
		return nil, err
	}

	if err := s.authorizer.CheckOrganizationPermissionWithSubscription(access, orgBillingAccount, constants.PERMISSION_AUDIT_READ); err != nil {
		return nil, err
	}
	if err := s.authorizer.CheckOrganizationFeature(access, orgBillingAccount, constants.FEATURE_AUDIT_LOG); err != nil {
		return nil, err
	}

	filters := req.Filters
	minStart := platform_subscription.AuditLogMinStartDate(orgBillingAccount.SubscriptionTier, s.now())
	if filters == nil {
		filters = &audit_log_repository.AuditLogFilters{StartDate: &minStart}
	} else if filters.StartDate == nil || filters.StartDate.Before(minStart) {
		f := *filters
		f.StartDate = &minStart
		filters = &f
	}

	auditLogs, err := s.auditLogsRepo.ListAuditLogsByOrganizationID(ctx, req.OrganizationID, filters, req.Limit+1, req.Offset)
	if err != nil {
		return nil, err
	}

	hasNext := len(*auditLogs) > req.Limit
	if hasNext {
		items := (*auditLogs)[:req.Limit]
		auditLogs = &items
	}

	publicItems := make([]models.AuditLogPublic, 0, len(*auditLogs))
	for i := range *auditLogs {
		publicItems = append(publicItems, audit.ToPublicAuditLog(&(*auditLogs)[i]))
	}

	return &ListAuditLogsForOrganizationResponse{
		Items:   publicItems,
		HasNext: hasNext,
	}, nil
}
