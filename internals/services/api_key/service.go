package api_key_service

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/yca-software/2chi-go-api/internals/constants"
	"github.com/yca-software/2chi-go-api/internals/models"
	"github.com/yca-software/2chi-go-api/internals/packages/audit"
	"github.com/yca-software/2chi-go-api/internals/packages/authz"
	"github.com/yca-software/2chi-go-api/internals/repositories"
	api_key_repository "github.com/yca-software/2chi-go-api/internals/repositories/api_key"
	billing_account_repository "github.com/yca-software/2chi-go-api/internals/repositories/billing_account"
	organization_repository "github.com/yca-software/2chi-go-api/internals/repositories/organization"
	audit_service "github.com/yca-software/2chi-go-api/internals/services/audit"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_types "github.com/yca-software/2chi-go-types"
	chi_validator "github.com/yca-software/2chi-go-validator"
)

type Dependencies struct {
	GenerateID    func() (uuid.UUID, error)
	Now           func() time.Time
	Validator     chi_validator.Validator
	Logger        chi_logger.Logger
	GenerateToken func() (string, error)
	HashToken     func(token string) string
	Authorizer    *authz.Authorizer
	Repositories  *repositories.Repositories
	AuditService  audit_service.Service
}

type Service interface {
	Create(ctx context.Context, req *CreateRequest, access *chi_types.AccessInfo) (*CreateResponse, error)
	Update(ctx context.Context, req *UpdateRequest, access *chi_types.AccessInfo) (*models.APIKey, error)
	Delete(ctx context.Context, req *DeleteRequest, access *chi_types.AccessInfo) error
	List(ctx context.Context, req *ListRequest, access *chi_types.AccessInfo) (*[]models.APIKey, error)
}

type service struct {
	generateID          func() (uuid.UUID, error)
	now                 func() time.Time
	validator           chi_validator.Validator
	logger              chi_logger.Logger
	generateToken       func() (string, error)
	hashToken           func(token string) string
	authorizer          *authz.Authorizer
	billingAccountsRepo billing_account_repository.Repository
	apiKeysRepo         api_key_repository.Repository
	organizationsRepo   organization_repository.Repository
	auditService        audit_service.Service
}

func New(deps Dependencies) Service {
	return &service{
		generateID:          deps.GenerateID,
		now:                 deps.Now,
		validator:           deps.Validator,
		logger:              deps.Logger,
		generateToken:       deps.GenerateToken,
		hashToken:           deps.HashToken,
		authorizer:          deps.Authorizer,
		billingAccountsRepo: deps.Repositories.OrganizationBillingAccounts,
		apiKeysRepo:         deps.Repositories.APIKeys,
		organizationsRepo:   deps.Repositories.Organizations,
		auditService:        deps.AuditService,
	}
}

func (s *service) Create(ctx context.Context, req *CreateRequest, access *chi_types.AccessInfo) (*CreateResponse, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	for _, permission := range req.Permissions {
		if !slices.Contains(constants.ASSIGNABLE_API_KEY_PERMISSIONS, permission) {
			return nil, chi_error.NewUnprocessableEntityError(errors.New("invalid api key permission"), "InvalidAPIKeyPermission", nil)
		}
	}

	billingAccount, err := s.billingAccountsRepo.GetByOrganizationID(ctx, req.OrganizationID)
	if err != nil {
		return nil, err
	}

	if err := s.authorizer.CheckOrganizationPermissionWithSubscription(access, billingAccount, constants.PERMISSION_API_KEY_WRITE); err != nil {
		return nil, err
	}
	if err := s.authorizer.CheckOrganizationFeature(access, billingAccount, constants.FEATURE_API_ACCESS); err != nil {
		return nil, err
	}

	rawKey, err := s.generateToken()
	if err != nil {
		return nil, err
	}

	keyPrefix := constants.API_KEY_PREFIX + rawKey[:constants.API_KEY_PREFIX_LEN]

	apiKeyID, err := s.generateID()
	if err != nil {
		return nil, err
	}

	apiKey := &models.APIKey{
		ModelBase: chi_types.ModelBase{
			ID: apiKeyID,
		},
		OrganizationID: uuid.MustParse(req.OrganizationID),
		Name:           req.Name,
		KeyPrefix:      keyPrefix,
		KeyHash:        s.hashToken(rawKey),
		Permissions:    req.Permissions,
		ExpiresAt:      req.ExpiresAt,
	}

	if err := s.apiKeysRepo.Create(ctx, apiKey); err != nil {
		return nil, err
	}

	s.logAPIKeyAudit(ctx, access, req.OrganizationID, constants.AUDIT_ACTION_TYPE_CREATE, apiKey, audit.CreatePayload(map[string]any{
		"name":        apiKey.Name,
		"permissions": apiKey.Permissions,
		"expiresAt":   apiKey.ExpiresAt,
	}))

	return &CreateResponse{
		APIKey: apiKey,
		Secret: constants.API_KEY_PREFIX + rawKey,
	}, nil
}

func (s *service) Update(ctx context.Context, req *UpdateRequest, access *chi_types.AccessInfo) (*models.APIKey, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	for _, permission := range req.Permissions {
		if !slices.Contains(constants.ASSIGNABLE_API_KEY_PERMISSIONS, permission) {
			return nil, chi_error.NewUnprocessableEntityError(errors.New("invalid api key permission"), "InvalidAPIKeyPermission", nil)
		}
	}

	billingAccount, err := s.billingAccountsRepo.GetByOrganizationID(ctx, req.OrganizationID)
	if err != nil {
		return nil, err
	}

	if err := s.authorizer.CheckOrganizationPermissionWithSubscription(access, billingAccount, constants.PERMISSION_API_KEY_WRITE); err != nil {
		return nil, err
	}
	if err := s.authorizer.CheckOrganizationFeature(access, billingAccount, constants.FEATURE_API_ACCESS); err != nil {
		return nil, err
	}

	apiKey, err := s.apiKeysRepo.GetByID(ctx, req.OrganizationID, req.APIKeyID)
	if err != nil {
		return nil, err
	}

	previous := map[string]any{
		"name":        apiKey.Name,
		"permissions": apiKey.Permissions,
	}

	apiKey.Name = req.Name
	apiKey.Permissions = req.Permissions

	if err := s.apiKeysRepo.Update(ctx, apiKey); err != nil {
		return nil, err
	}

	s.logAPIKeyAudit(ctx, access, req.OrganizationID, constants.AUDIT_ACTION_TYPE_UPDATE, apiKey, audit.UpdatePayload(previous, map[string]any{
		"name":        apiKey.Name,
		"permissions": apiKey.Permissions,
	}))

	return apiKey, nil
}

func (s *service) Delete(ctx context.Context, req *DeleteRequest, access *chi_types.AccessInfo) error {
	if err := s.validator.ValidateStruct(req); err != nil {
		return chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	billingAccount, err := s.billingAccountsRepo.GetByOrganizationID(ctx, req.OrganizationID)
	if err != nil {
		return err
	}

	if err := s.authorizer.CheckOrganizationPermissionWithSubscription(access, billingAccount, constants.PERMISSION_API_KEY_DELETE); err != nil {
		return err
	}
	if err := s.authorizer.CheckOrganizationFeature(access, billingAccount, constants.FEATURE_API_ACCESS); err != nil {
		return err
	}

	apiKey, err := s.apiKeysRepo.GetByID(ctx, req.OrganizationID, req.APIKeyID)
	if err != nil {
		return err
	}

	if err := s.apiKeysRepo.Delete(ctx, req.OrganizationID, req.APIKeyID); err != nil {
		return err
	}

	s.logAPIKeyAudit(ctx, access, req.OrganizationID, constants.AUDIT_ACTION_TYPE_DELETE, apiKey, audit.DeletePayload(map[string]any{
		"name":        apiKey.Name,
		"permissions": apiKey.Permissions,
		"expiresAt":   apiKey.ExpiresAt,
	}))

	return nil
}

func (s *service) List(ctx context.Context, req *ListRequest, access *chi_types.AccessInfo) (*[]models.APIKey, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	billingAccount, err := s.billingAccountsRepo.GetByOrganizationID(ctx, req.OrganizationID)
	if err != nil {
		return nil, err
	}

	if err := s.authorizer.CheckOrganizationPermissionWithSubscription(access, billingAccount, constants.PERMISSION_API_KEY_READ); err != nil {
		return nil, err
	}
	if err := s.authorizer.CheckOrganizationFeature(access, billingAccount, constants.FEATURE_API_ACCESS); err != nil {
		return nil, err
	}

	return s.apiKeysRepo.ListByOrganizationID(ctx, req.OrganizationID)
}

func (s *service) logAPIKeyAudit(ctx context.Context, access *chi_types.AccessInfo, orgID, action string, apiKey *models.APIKey, payload map[string]any) {
	auditPayload, err := json.Marshal(payload)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to marshal api key audit payload", "error", err, "organizationId", orgID)
		return
	}
	auditRaw := json.RawMessage(auditPayload)
	if _, err := s.auditService.Create(ctx, &audit_service.CreateRequest{
		OrganizationID: orgID,
		Action:         action,
		ResourceType:   constants.RESOURCE_TYPE_API_KEY,
		ResourceID:     apiKey.ID.String(),
		ResourceName:   apiKey.Name,
		Data:           &auditRaw,
	}, access); err != nil {
		s.logger.WithContext(ctx).Error("failed to create api key audit log", "error", err, "organizationId", orgID)
	}
}
