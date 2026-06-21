package role_service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yca-software/2chi-go-api/internals/constants"
	"github.com/yca-software/2chi-go-api/internals/models"
	"github.com/yca-software/2chi-go-api/internals/repositories"
	billing_account_repository "github.com/yca-software/2chi-go-api/internals/repositories/billing_account"
	organization_member_repository "github.com/yca-software/2chi-go-api/internals/repositories/org_member"
	organization_repository "github.com/yca-software/2chi-go-api/internals/repositories/organization"
	role_repository "github.com/yca-software/2chi-go-api/internals/repositories/role"
	audit_service "github.com/yca-software/2chi-go-api/internals/services/audit"
	"github.com/yca-software/2chi-go-api/internals/packages/audit"
	"github.com/yca-software/2chi-go-api/internals/packages/authz"
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
	AuditService audit_service.Service
	SessionCache *authz.SessionCache
}

type Service interface {
	CreateRole(ctx context.Context, req *CreateRoleRequest, access *chi_types.AccessInfo) (*models.Role, error)
	UpdateRole(ctx context.Context, req *UpdateRoleRequest, access *chi_types.AccessInfo) (*models.Role, error)
	DeleteRole(ctx context.Context, req *DeleteRoleRequest, access *chi_types.AccessInfo) error
	ListRoles(ctx context.Context, req *ListRolesRequest, access *chi_types.AccessInfo) (*[]models.Role, error)
}

type service struct {
	generateID              func() (uuid.UUID, error)
	now                     func() time.Time
	validator               chi_validator.Validator
	logger                  chi_logger.Logger
	authorizer              *authz.Authorizer
	billingAccountsRepo     billing_account_repository.OrganizationBillingAccountsRepository
	rolesRepo               role_repository.RolesRepository
	organizationsRepo       organization_repository.OrganizationsRepository
	organizationMembersRepo organization_member_repository.OrganizationMembersRepository
	auditService            audit_service.Service
	sessionCache            *authz.SessionCache
}

func New(deps Dependencies) Service {
	return &service{
		generateID:              deps.GenerateID,
		now:                     deps.Now,
		validator:               deps.Validator,
		logger:                  deps.Logger,
		authorizer:              deps.Authorizer,
		billingAccountsRepo:     deps.Repositories.OrganizationBillingAccounts,
		rolesRepo:               deps.Repositories.Roles,
		organizationsRepo:       deps.Repositories.Organizations,
		organizationMembersRepo: deps.Repositories.OrganizationMembers,
		auditService:            deps.AuditService,
		sessionCache:            deps.SessionCache,
	}
}

func (s *service) CreateRole(ctx context.Context, req *CreateRoleRequest, access *chi_types.AccessInfo) (*models.Role, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	if _, err := s.organizationsRepo.GetOrganizationByID(ctx, req.OrganizationID); err != nil {
		return nil, err
	}

	billing, err := s.billingAccountsRepo.GetOrganizationBillingAccountByOrganizationID(ctx, req.OrganizationID)
	if err != nil {
		return nil, err
	}

	if err := s.authorizer.CheckOrganizationPermissionWithSubscription(access, billing, constants.PERMISSION_ROLE_WRITE); err != nil {
		return nil, err
	}
	if err := s.authorizer.CheckOrganizationFeature(access, billing, constants.FEATURE_ROLES); err != nil {
		return nil, err
	}

	roleID, err := s.generateID()
	if err != nil {
		return nil, err
	}

	role := &models.Role{
		ModelBase: chi_types.ModelBase{
			ID:        roleID,
			CreatedAt: s.now(),
		},
		OrganizationID: uuid.MustParse(req.OrganizationID),
		Name:           strings.TrimSpace(req.Name),
		Description:    strings.TrimSpace(req.Description),
		Permissions:    req.Permissions,
	}

	if err := s.rolesRepo.CreateRole(ctx, role); err != nil {
		return nil, err
	}

	s.logRoleAudit(ctx, access, req.OrganizationID, constants.AUDIT_ACTION_TYPE_CREATE, role, audit.CreatePayload(map[string]any{
		"name":        role.Name,
		"description": role.Description,
		"permissions": role.Permissions,
	}))

	return role, nil
}

func (s *service) UpdateRole(ctx context.Context, req *UpdateRoleRequest, access *chi_types.AccessInfo) (*models.Role, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	if _, err := s.organizationsRepo.GetOrganizationByID(ctx, req.OrganizationID); err != nil {
		return nil, err
	}

	billing, err := s.billingAccountsRepo.GetOrganizationBillingAccountByOrganizationID(ctx, req.OrganizationID)
	if err != nil {
		return nil, err
	}

	if err := s.authorizer.CheckOrganizationPermissionWithSubscription(access, billing, constants.PERMISSION_ROLE_WRITE); err != nil {
		return nil, err
	}
	if err := s.authorizer.CheckOrganizationFeature(access, billing, constants.FEATURE_ROLES); err != nil {
		return nil, err
	}

	role, err := s.rolesRepo.GetRoleByID(ctx, req.OrganizationID, req.RoleID)
	if err != nil {
		return nil, err
	}

	if role.Locked {
		return nil, chi_error.NewForbiddenError(errors.New("role is locked"), "RoleLocked", nil)
	}

	updatedRole := *role
	updatedRole.Name = strings.TrimSpace(req.Name)
	updatedRole.Description = strings.TrimSpace(req.Description)
	updatedRole.Permissions = req.Permissions

	if err := s.rolesRepo.UpdateRole(ctx, &updatedRole); err != nil {
		return nil, err
	}

	s.logRoleAudit(ctx, access, req.OrganizationID, constants.AUDIT_ACTION_TYPE_UPDATE, &updatedRole, audit.UpdatePayload(
		map[string]any{
			"name":        role.Name,
			"description": role.Description,
			"permissions": role.Permissions,
		},
		map[string]any{
			"name":        updatedRole.Name,
			"description": updatedRole.Description,
			"permissions": updatedRole.Permissions,
		},
	))

	s.invalidateSessionsForRole(ctx, req.OrganizationID, role.ID.String())

	return &updatedRole, nil
}

func (s *service) DeleteRole(ctx context.Context, req *DeleteRoleRequest, access *chi_types.AccessInfo) error {
	if err := s.validator.ValidateStruct(req); err != nil {
		return chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	if _, err := s.organizationsRepo.GetOrganizationByID(ctx, req.OrganizationID); err != nil {
		return err
	}

	billing, err := s.billingAccountsRepo.GetOrganizationBillingAccountByOrganizationID(ctx, req.OrganizationID)
	if err != nil {
		return err
	}

	if err := s.authorizer.CheckOrganizationPermissionWithSubscription(access, billing, constants.PERMISSION_ROLE_DELETE); err != nil {
		return err
	}
	if err := s.authorizer.CheckOrganizationFeature(access, billing, constants.FEATURE_ROLES); err != nil {
		return err
	}

	role, err := s.rolesRepo.GetRoleByID(ctx, req.OrganizationID, req.RoleID)
	if err != nil {
		return err
	}

	if role.Locked {
		return chi_error.NewForbiddenError(errors.New("role is locked"), "RoleLocked", nil)
	}

	emails, err := s.organizationMembersRepo.ListUserEmailsForRole(ctx, req.OrganizationID, req.RoleID)
	if err != nil {
		return err
	}
	if len(emails) > 0 {
		return chi_error.NewConflictError(errors.New("role has members"), "RoleHasMembers", map[string]any{
			"memberEmails":     emails,
			"memberEmailsText": strings.Join(emails, ", "),
		})
	}

	if err := s.rolesRepo.DeleteRole(ctx, req.OrganizationID, req.RoleID); err != nil {
		return err
	}

	s.logRoleAudit(ctx, access, req.OrganizationID, constants.AUDIT_ACTION_TYPE_DELETE, role, audit.DeletePayload(map[string]any{
		"name":        role.Name,
		"description": role.Description,
		"permissions": role.Permissions,
	}))

	return nil
}

func (s *service) ListRoles(ctx context.Context, req *ListRolesRequest, access *chi_types.AccessInfo) (*[]models.Role, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	if _, err := s.organizationsRepo.GetOrganizationByID(ctx, req.OrganizationID); err != nil {
		return nil, err
	}

	if err := s.authorizer.CheckOrganizationPermission(access, req.OrganizationID, constants.PERMISSION_ROLE_READ); err != nil {
		return nil, err
	}

	return s.rolesRepo.ListRolesByOrganizationID(ctx, req.OrganizationID)
}

func (s *service) logRoleAudit(ctx context.Context, access *chi_types.AccessInfo, orgID, action string, role *models.Role, payload map[string]any) {
	changes, err := json.Marshal(payload)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to marshal role audit payload", "error", err, "organizationId", orgID)
		return
	}
	changesRaw := json.RawMessage(changes)
	if _, err := s.auditService.CreateAuditLog(ctx, &audit_service.CreateAuditLogRequest{
		OrganizationID: orgID,
		Action:         action,
		ResourceType:   constants.RESOURCE_TYPE_ROLE,
		ResourceID:     role.ID.String(),
		ResourceName:   role.Name,
		Data:           &changesRaw,
	}, access); err != nil {
		s.logger.WithContext(ctx).Error("failed to create role audit log", "error", err, "organizationId", orgID)
	}
}

func (s *service) invalidateSessionsForRole(ctx context.Context, organizationID, roleID string) {
	if s.sessionCache == nil {
		return
	}
	members, err := s.organizationMembersRepo.ListByOrganizationID(ctx, organizationID)
	if err != nil {
		return
	}
	for _, member := range *members {
		if member.RoleID.String() == roleID {
			_ = s.sessionCache.InvalidateSession(ctx, member.UserID.String())
		}
	}
}
