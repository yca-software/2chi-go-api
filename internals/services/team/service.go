package team_service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yca-software/2chi-go-api/internals/constants"
	"github.com/yca-software/2chi-go-api/internals/models"
	"github.com/yca-software/2chi-go-api/internals/packages/audit"
	"github.com/yca-software/2chi-go-api/internals/packages/authz"
	"github.com/yca-software/2chi-go-api/internals/repositories"
	billing_account_repository "github.com/yca-software/2chi-go-api/internals/repositories/billing_account"
	organization_member_repository "github.com/yca-software/2chi-go-api/internals/repositories/org_member"
	organization_repository "github.com/yca-software/2chi-go-api/internals/repositories/organization"
	team_repository "github.com/yca-software/2chi-go-api/internals/repositories/team"
	team_member_repository "github.com/yca-software/2chi-go-api/internals/repositories/team_member"
	audit_service "github.com/yca-software/2chi-go-api/internals/services/audit"
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
}

type Service interface {
	CreateTeam(ctx context.Context, req *CreateTeamRequest, access *chi_types.AccessInfo) (*models.Team, error)
	UpdateTeam(ctx context.Context, req *UpdateTeamRequest, access *chi_types.AccessInfo) (*models.Team, error)
	DeleteTeam(ctx context.Context, req *DeleteTeamRequest, access *chi_types.AccessInfo) error
	ListTeams(ctx context.Context, req *ListTeamsRequest, access *chi_types.AccessInfo) (*[]models.Team, error)

	AddTeamMember(ctx context.Context, req *AddTeamMemberRequest, access *chi_types.AccessInfo) (*models.TeamMemberWithUser, error)
	RemoveTeamMember(ctx context.Context, req *RemoveTeamMemberRequest, access *chi_types.AccessInfo) error
	ListTeamMembers(ctx context.Context, req *ListTeamMembersRequest, access *chi_types.AccessInfo) (*[]models.TeamMemberWithUser, error)
}

type service struct {
	generateID              func() (uuid.UUID, error)
	now                     func() time.Time
	validator               chi_validator.Validator
	logger                  chi_logger.Logger
	authorizer              *authz.Authorizer
	teamsRepo               team_repository.Repository
	teamMembersRepo         team_member_repository.Repository
	billingAccountsRepo     billing_account_repository.Repository
	organizationsRepo       organization_repository.Repository
	organizationMembersRepo organization_member_repository.Repository
	auditService            audit_service.Service
}

func New(deps Dependencies) Service {
	return &service{
		generateID:              deps.GenerateID,
		now:                     deps.Now,
		validator:               deps.Validator,
		logger:                  deps.Logger,
		authorizer:              deps.Authorizer,
		billingAccountsRepo:     deps.Repositories.OrganizationBillingAccounts,
		teamsRepo:               deps.Repositories.Teams,
		teamMembersRepo:         deps.Repositories.TeamMembers,
		organizationsRepo:       deps.Repositories.Organizations,
		organizationMembersRepo: deps.Repositories.OrganizationMembers,
		auditService:            deps.AuditService,
	}
}

func (s *service) CreateTeam(ctx context.Context, req *CreateTeamRequest, access *chi_types.AccessInfo) (*models.Team, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	org, err := s.organizationsRepo.GetByID(ctx, req.OrganizationID)
	if err != nil {
		return nil, err
	}

	billing, err := s.billingAccountsRepo.GetByOrganizationID(ctx, req.OrganizationID)
	if err != nil {
		return nil, err
	}

	if err := s.authorizer.CheckOrganizationPermissionWithSubscription(access, billing, constants.PERMISSION_TEAM_WRITE); err != nil {
		return nil, err
	}
	if err := s.authorizer.CheckOrganizationFeature(access, billing, constants.FEATURE_TEAMS); err != nil {
		return nil, err
	}

	teamID, err := s.generateID()
	if err != nil {
		return nil, err
	}

	team := &models.Team{
		ModelBase: chi_types.ModelBase{
			ID:        teamID,
			CreatedAt: s.now(),
		},
		OrganizationID: org.ID,
		Name:           strings.TrimSpace(req.Name),
		Description:    req.Description,
	}

	if err := s.teamsRepo.Create(ctx, team); err != nil {
		return nil, err
	}

	s.logTeamAudit(ctx, access, org.ID.String(), constants.AUDIT_ACTION_TYPE_CREATE, team, audit.CreatePayload(map[string]any{
		"name":        team.Name,
		"description": team.Description,
	}))

	return team, nil
}

func (s *service) UpdateTeam(ctx context.Context, req *UpdateTeamRequest, access *chi_types.AccessInfo) (*models.Team, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	if _, err := s.organizationsRepo.GetByID(ctx, req.OrganizationID); err != nil {
		return nil, err
	}

	billing, err := s.billingAccountsRepo.GetByOrganizationID(ctx, req.OrganizationID)
	if err != nil {
		return nil, err
	}

	if err := s.authorizer.CheckOrganizationPermissionWithSubscription(access, billing, constants.PERMISSION_TEAM_WRITE); err != nil {
		return nil, err
	}
	if err := s.authorizer.CheckOrganizationFeature(access, billing, constants.FEATURE_TEAMS); err != nil {
		return nil, err
	}

	team, err := s.teamsRepo.GetByID(ctx, req.OrganizationID, req.TeamID)
	if err != nil {
		return nil, err
	}

	updatedTeam := *team
	updatedTeam.Name = strings.TrimSpace(req.Name)
	updatedTeam.Description = req.Description

	if err := s.teamsRepo.Update(ctx, &updatedTeam); err != nil {
		return nil, err
	}

	s.logTeamAudit(ctx, access, req.OrganizationID, constants.AUDIT_ACTION_TYPE_UPDATE, &updatedTeam, audit.UpdatePayload(
		map[string]any{
			"name":        team.Name,
			"description": team.Description,
		},
		map[string]any{
			"name":        updatedTeam.Name,
			"description": updatedTeam.Description,
		},
	))

	return &updatedTeam, nil
}

func (s *service) DeleteTeam(ctx context.Context, req *DeleteTeamRequest, access *chi_types.AccessInfo) error {
	if err := s.validator.ValidateStruct(req); err != nil {
		return chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	if _, err := s.organizationsRepo.GetByID(ctx, req.OrganizationID); err != nil {
		return err
	}

	billing, err := s.billingAccountsRepo.GetByOrganizationID(ctx, req.OrganizationID)
	if err != nil {
		return err
	}

	if err := s.authorizer.CheckOrganizationPermissionWithSubscription(access, billing, constants.PERMISSION_TEAM_DELETE); err != nil {
		return err
	}
	if err := s.authorizer.CheckOrganizationFeature(access, billing, constants.FEATURE_TEAMS); err != nil {
		return err
	}

	team, err := s.teamsRepo.GetByID(ctx, req.OrganizationID, req.TeamID)
	if err != nil {
		return err
	}

	if err := s.teamsRepo.Delete(ctx, req.OrganizationID, req.TeamID); err != nil {
		return err
	}

	s.logTeamAudit(ctx, access, req.OrganizationID, constants.AUDIT_ACTION_TYPE_DELETE, team, audit.DeletePayload(map[string]any{
		"name":        team.Name,
		"description": team.Description,
	}))

	return nil
}

func (s *service) ListTeams(ctx context.Context, req *ListTeamsRequest, access *chi_types.AccessInfo) (*[]models.Team, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	if _, err := s.organizationsRepo.GetByID(ctx, req.OrganizationID); err != nil {
		return nil, err
	}

	if err := s.authorizer.CheckOrganizationPermission(access, req.OrganizationID, constants.PERMISSION_TEAM_READ); err != nil {
		return nil, err
	}

	return s.teamsRepo.ListByOrganizationID(ctx, req.OrganizationID)
}

func (s *service) AddTeamMember(ctx context.Context, req *AddTeamMemberRequest, access *chi_types.AccessInfo) (*models.TeamMemberWithUser, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	org, err := s.organizationsRepo.GetByID(ctx, req.OrganizationID)
	if err != nil {
		return nil, err
	}

	billing, err := s.billingAccountsRepo.GetByOrganizationID(ctx, req.OrganizationID)
	if err != nil {
		return nil, err
	}

	if err := s.authorizer.CheckOrganizationPermissionWithSubscription(access, billing, constants.PERMISSION_TEAM_MEMBER_WRITE); err != nil {
		return nil, err
	}
	if err := s.authorizer.CheckOrganizationFeature(access, billing, constants.FEATURE_TEAMS); err != nil {
		return nil, err
	}

	team, err := s.teamsRepo.GetByID(ctx, req.OrganizationID, req.TeamID)
	if err != nil {
		return nil, err
	}

	if _, err := s.organizationMembersRepo.GetByUserID(ctx, req.OrganizationID, req.UserID); err != nil {
		return nil, err
	}

	memberID, err := s.generateID()
	if err != nil {
		return nil, err
	}

	member := &models.TeamMember{
		ModelBase: chi_types.ModelBase{
			ID: memberID,
		},
		OrganizationID: org.ID,
		TeamID:         uuid.MustParse(req.TeamID),
		UserID:         uuid.MustParse(req.UserID),
	}

	if err := s.teamMembersRepo.Create(ctx, member); err != nil {
		return nil, err
	}

	createdMember, err := s.teamMembersRepo.GetByIDWithUser(ctx, req.OrganizationID, memberID.String())
	if err != nil {
		return nil, err
	}

	s.logTeamMemberAudit(ctx, access, req.OrganizationID, constants.AUDIT_ACTION_TYPE_CREATE, createdMember, team, audit.CreatePayload(map[string]any{
		"userId":    createdMember.UserID,
		"userEmail": createdMember.UserEmail,
		"teamId":    team.ID,
		"teamName":  team.Name,
	}))

	return createdMember, nil
}

func (s *service) RemoveTeamMember(ctx context.Context, req *RemoveTeamMemberRequest, access *chi_types.AccessInfo) error {
	if err := s.validator.ValidateStruct(req); err != nil {
		return chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	billing, err := s.billingAccountsRepo.GetByOrganizationID(ctx, req.OrganizationID)
	if err != nil {
		return err
	}

	if err := s.authorizer.CheckOrganizationPermissionWithSubscription(access, billing, constants.PERMISSION_TEAM_MEMBER_DELETE); err != nil {
		return err
	}
	if err := s.authorizer.CheckOrganizationFeature(access, billing, constants.FEATURE_TEAMS); err != nil {
		return err
	}

	member, err := s.teamMembersRepo.GetByIDWithUser(ctx, req.OrganizationID, req.MemberID)
	if err != nil {
		return err
	}

	team, err := s.teamsRepo.GetByID(ctx, req.OrganizationID, req.TeamID)
	if err != nil {
		return err
	}

	if err := s.teamMembersRepo.Delete(ctx, req.OrganizationID, req.MemberID); err != nil {
		return err
	}

	s.logTeamMemberAudit(ctx, access, req.OrganizationID, constants.AUDIT_ACTION_TYPE_DELETE, member, team, audit.DeletePayload(map[string]any{
		"userId":    member.UserID,
		"userEmail": member.UserEmail,
		"teamId":    team.ID,
		"teamName":  team.Name,
	}))

	return nil
}

func (s *service) ListTeamMembers(ctx context.Context, req *ListTeamMembersRequest, access *chi_types.AccessInfo) (*[]models.TeamMemberWithUser, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	billing, err := s.billingAccountsRepo.GetByOrganizationID(ctx, req.OrganizationID)
	if err != nil {
		return nil, err
	}

	if err := s.authorizer.CheckOrganizationPermissionWithSubscription(access, billing, constants.PERMISSION_TEAM_MEMBER_READ); err != nil {
		return nil, err
	}
	if err := s.authorizer.CheckOrganizationFeature(access, billing, constants.FEATURE_TEAMS); err != nil {
		return nil, err
	}

	return s.teamMembersRepo.ListByTeamID(ctx, req.OrganizationID, req.TeamID)
}

func (s *service) logTeamMemberAudit(ctx context.Context, access *chi_types.AccessInfo, orgID, action string, member *models.TeamMemberWithUser, team *models.Team, payload map[string]any) {
	changes, err := json.Marshal(payload)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to marshal team member audit payload", "error", err, "organizationId", orgID)
		return
	}
	changesRaw := json.RawMessage(changes)
	resourceName := member.UserEmail
	if resourceName == "" {
		resourceName = "Team member"
	}
	if _, err := s.auditService.Create(ctx, &audit_service.CreateRequest{
		OrganizationID: orgID,
		Action:         action,
		ResourceType:   constants.RESOURCE_TYPE_TEAM_MEMBER,
		ResourceID:     member.ID.String(),
		ResourceName:   resourceName,
		Data:           &changesRaw,
	}, access); err != nil {
		s.logger.WithContext(ctx).Error("failed to create team member audit log", "error", err, "organizationId", orgID)
	}
}

func (s *service) logTeamAudit(ctx context.Context, access *chi_types.AccessInfo, orgID, action string, team *models.Team, payload map[string]any) {
	changes, err := json.Marshal(payload)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to marshal team audit payload", "error", err, "organizationId", orgID)
		return
	}
	changesRaw := json.RawMessage(changes)
	if _, err := s.auditService.Create(ctx, &audit_service.CreateRequest{
		OrganizationID: orgID,
		Action:         action,
		ResourceType:   constants.RESOURCE_TYPE_TEAM,
		ResourceID:     team.ID.String(),
		ResourceName:   team.Name,
		Data:           &changesRaw,
	}, access); err != nil {
		s.logger.WithContext(ctx).Error("failed to create team audit log", "error", err, "organizationId", orgID)
	}
}
