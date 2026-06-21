package team_service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/yca-software/2chi-go-api/internals/constants"
	"github.com/yca-software/2chi-go-api/internals/models"
	"github.com/yca-software/2chi-go-api/internals/repositories"
	billing_account_repository "github.com/yca-software/2chi-go-api/internals/repositories/billing_account"
	organization_member_repository "github.com/yca-software/2chi-go-api/internals/repositories/org_member"
	organization_repository "github.com/yca-software/2chi-go-api/internals/repositories/organization"
	team_member_repository "github.com/yca-software/2chi-go-api/internals/repositories/team_member"
	team_repository "github.com/yca-software/2chi-go-api/internals/repositories/team"
	audit_service "github.com/yca-software/2chi-go-api/internals/services/audit"
	team_service "github.com/yca-software/2chi-go-api/internals/services/team"
	"github.com/yca-software/2chi-go-api/internals/packages/authz"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_types "github.com/yca-software/2chi-go-types"
	chi_validator "github.com/yca-software/2chi-go-validator"
)

type TeamServiceSuite struct {
	suite.Suite
	ctx             context.Context
	now             time.Time
	orgID           uuid.UUID
	teamsRepo       *team_repository.MockTeamsRepository
	teamMembers     *team_member_repository.MockTeamMembersRepository
	orgsRepo        *organization_repository.MockOrganizationsRepository
	membersRepo     *organization_member_repository.MockOrganizationMembersRepository
	billingAccounts *billing_account_repository.MockOrganizationBillingAccountsRepository
	auditSvc        *audit_service.MockService
	svc             team_service.Service
}

func TestTeamServiceSuite(t *testing.T) {
	suite.Run(t, new(TeamServiceSuite))
}

func (s *TeamServiceSuite) SetupTest() {
	s.ctx = context.Background()
	s.now = time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	s.orgID = uuid.New()
	s.teamsRepo = &team_repository.MockTeamsRepository{}
	s.teamMembers = &team_member_repository.MockTeamMembersRepository{}
	s.orgsRepo = &organization_repository.MockOrganizationsRepository{}
	s.membersRepo = &organization_member_repository.MockOrganizationMembersRepository{}
	s.billingAccounts = &billing_account_repository.MockOrganizationBillingAccountsRepository{}
	s.auditSvc = &audit_service.MockService{}

	s.svc = team_service.New(team_service.Dependencies{
		GenerateID: uuid.NewV7,
		Now:        func() time.Time { return s.now },
		Validator:  chi_validator.New(),
		Logger:     mockLogger(),
		Authorizer: authz.NewAuthorizer(func() time.Time { return s.now }),
		Repositories: &repositories.Repositories{
			Teams:                       s.teamsRepo,
			TeamMembers:                 s.teamMembers,
			Organizations:               s.orgsRepo,
			OrganizationMembers:         s.membersRepo,
			OrganizationBillingAccounts: s.billingAccounts,
		},
		AuditService: s.auditSvc,
	})
}

func (s *TeamServiceSuite) expectProOrg() {
	expiresAt := s.now.Add(24 * time.Hour)
	s.orgsRepo.On("GetOrganizationByID", s.ctx, s.orgID.String()).
		Return(&models.Organization{
			ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
				ModelBase: chi_types.ModelBase{ID: s.orgID},
			},
			Name: "Acme",
		}, nil).Once()
	s.billingAccounts.On("GetOrganizationBillingAccountByOrganizationID", s.ctx, s.orgID.String()).
		Return(&models.OrganizationBillingAccount{
			ModelBase:             chi_types.ModelBase{ID: s.orgID},
			OrganizationID:        s.orgID,
			Provider:              constants.BILLING_PROVIDER_PADDLE,
			SubscriptionTier:      constants.TIER_PRO,
			SubscriptionExpiresAt: &expiresAt,
		}, nil).Once()
}

func (s *TeamServiceSuite) writeAccess() *chi_types.AccessInfo {
	return &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeUser,
		SubjectID: uuid.New(),
		Roles: []chi_types.JWTAccessTokenPermissionData{{
			OrganizationID: s.orgID,
			Permissions:    []string{constants.PERMISSION_TEAM_WRITE},
		}},
	}
}

func (s *TeamServiceSuite) TestCreateTeam_Validation_MissingName() {
	team, err := s.svc.CreateTeam(s.ctx, &team_service.CreateTeamRequest{
		OrganizationID: s.orgID.String(),
		Name:           "",
	}, s.writeAccess())
	s.Error(err)
	s.Nil(team)
}

func (s *TeamServiceSuite) TestCreateTeam_Success() {
	s.expectProOrg()
	s.teamsRepo.On("CreateTeam", s.ctx, mock.AnythingOfType("*models.Team")).Return(nil).Once()
	s.auditSvc.On("CreateAuditLog", s.ctx, mock.Anything, mock.Anything).Return(&models.AuditLog{}, nil).Maybe()

	team, err := s.svc.CreateTeam(s.ctx, &team_service.CreateTeamRequest{
		OrganizationID: s.orgID.String(),
		Name:           "  Platform  ",
		Description:    "Core team",
	}, s.writeAccess())
	s.Require().NoError(err)
	s.Equal("Platform", team.Name)
}

func (s *TeamServiceSuite) TestCreateTeam_FreePlanFeatureDenied() {
	s.orgsRepo.On("GetOrganizationByID", s.ctx, s.orgID.String()).
		Return(&models.Organization{
			ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
				ModelBase: chi_types.ModelBase{ID: s.orgID},
			},
			Name: "Acme",
		}, nil).Once()
	s.billingAccounts.On("GetOrganizationBillingAccountByOrganizationID", s.ctx, s.orgID.String()).
		Return(&models.OrganizationBillingAccount{
			ModelBase:        chi_types.ModelBase{ID: s.orgID},
			OrganizationID:   s.orgID,
			Provider:         constants.BILLING_PROVIDER_PADDLE,
			SubscriptionTier: constants.TIER_FREE,
		}, nil).Once()

	team, err := s.svc.CreateTeam(s.ctx, &team_service.CreateTeamRequest{
		OrganizationID: s.orgID.String(),
		Name:           "Platform",
	}, s.writeAccess())
	s.Error(err)
	s.Nil(team)
}

func (s *TeamServiceSuite) TestListTeams_Success() {
	teamID := uuid.New()
	teams := []models.Team{{
		ModelBase:      chi_types.ModelBase{ID: teamID},
		OrganizationID: s.orgID,
		Name:           "Platform",
	}}
	s.orgsRepo.On("GetOrganizationByID", s.ctx, s.orgID.String()).
		Return(&models.Organization{
			ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
				ModelBase: chi_types.ModelBase{ID: s.orgID},
			},
			Name: "Acme",
		}, nil).Once()
	s.teamsRepo.On("ListTeamsByOrganizationID", s.ctx, s.orgID.String()).Return(&teams, nil).Once()

	access := s.writeAccess()
	access.Roles[0].Permissions = []string{constants.PERMISSION_TEAM_READ}

	result, err := s.svc.ListTeams(s.ctx, &team_service.ListTeamsRequest{
		OrganizationID: s.orgID.String(),
	}, access)
	s.Require().NoError(err)
	s.Len(*result, 1)
}

func (s *TeamServiceSuite) TestDeleteTeam_Success() {
	teamID := uuid.New()
	s.expectProOrg()
	s.teamsRepo.On("GetTeamByID", s.ctx, s.orgID.String(), teamID.String()).
		Return(&models.Team{
			ModelBase:      chi_types.ModelBase{ID: teamID},
			OrganizationID: s.orgID,
			Name:           "Platform",
		}, nil).Once()
	s.teamsRepo.On("DeleteTeam", s.ctx, s.orgID.String(), teamID.String()).Return(nil).Once()
	s.auditSvc.On("CreateAuditLog", s.ctx, mock.Anything, mock.Anything).Return(&models.AuditLog{}, nil).Maybe()

	access := s.writeAccess()
	access.Roles[0].Permissions = []string{constants.PERMISSION_TEAM_DELETE}

	err := s.svc.DeleteTeam(s.ctx, &team_service.DeleteTeamRequest{
		OrganizationID: s.orgID.String(),
		TeamID:         teamID.String(),
	}, access)
	s.NoError(err)
}

func (s *TeamServiceSuite) TestUpdateTeam_Success() {
	teamID := uuid.New()
	s.expectProOrg()
	s.teamsRepo.On("GetTeamByID", s.ctx, s.orgID.String(), teamID.String()).
		Return(&models.Team{
			ModelBase:      chi_types.ModelBase{ID: teamID},
			OrganizationID: s.orgID,
			Name:           "Old",
			Description:    "d",
		}, nil).Once()
	s.teamsRepo.On("UpdateTeam", s.ctx, mock.AnythingOfType("*models.Team")).Return(nil).Once()
	s.auditSvc.On("CreateAuditLog", s.ctx, mock.Anything, mock.Anything).Return(&models.AuditLog{}, nil).Maybe()

	team, err := s.svc.UpdateTeam(s.ctx, &team_service.UpdateTeamRequest{
		OrganizationID: s.orgID.String(),
		TeamID:         teamID.String(),
		Name:           "  New  ",
		Description:    "updated",
	}, s.writeAccess())
	s.Require().NoError(err)
	s.Equal("New", team.Name)
}

func (s *TeamServiceSuite) TestAddTeamMember_Success() {
	teamID := uuid.New()
	userID := uuid.New()
	s.expectProOrg()
	s.teamsRepo.On("GetTeamByID", s.ctx, s.orgID.String(), teamID.String()).
		Return(&models.Team{
			ModelBase:      chi_types.ModelBase{ID: teamID},
			OrganizationID: s.orgID,
			Name:           "Platform",
		}, nil).Once()
	s.membersRepo.On("GetOrganizationMemberByUserIDAndOrganizationID", s.ctx, userID.String(), s.orgID.String()).
		Return(&models.OrganizationMember{}, nil).Once()
	s.teamMembers.On("CreateTeamMember", s.ctx, mock.AnythingOfType("*models.TeamMember")).Return(nil).Once()
	s.teamMembers.On("GetTeamMemberByIDWithUser", s.ctx, s.orgID.String(), mock.Anything).
		Return(&models.TeamMemberWithUser{
			TeamMember: models.TeamMember{
				ModelBase:      chi_types.ModelBase{ID: uuid.New()},
				OrganizationID: s.orgID,
				TeamID:         teamID,
				UserID:         userID,
			},
			UserEmail: "member@example.com",
		}, nil).Once()
	s.auditSvc.On("CreateAuditLog", s.ctx, mock.Anything, mock.Anything).Return(&models.AuditLog{}, nil).Once()

	access := s.writeAccess()
	access.Roles[0].Permissions = []string{constants.PERMISSION_TEAM_MEMBER_WRITE}

	member, err := s.svc.AddTeamMember(s.ctx, &team_service.AddTeamMemberRequest{
		OrganizationID: s.orgID.String(),
		TeamID:         teamID.String(),
		UserID:         userID.String(),
	}, access)
	s.Require().NoError(err)
	s.NotNil(member)
}

func (s *TeamServiceSuite) TestRemoveTeamMember_Success() {
	teamID := uuid.New()
	memberID := uuid.New()
	userID := uuid.New()
	s.expectProOrg()
	s.teamMembers.On("GetTeamMemberByIDWithUser", s.ctx, s.orgID.String(), memberID.String()).
		Return(&models.TeamMemberWithUser{
			TeamMember: models.TeamMember{
				ModelBase:      chi_types.ModelBase{ID: memberID},
				OrganizationID: s.orgID,
				TeamID:         teamID,
				UserID:         userID,
			},
			UserEmail: "member@example.com",
		}, nil).Once()
	s.teamsRepo.On("GetTeamByID", s.ctx, s.orgID.String(), teamID.String()).
		Return(&models.Team{
			ModelBase:      chi_types.ModelBase{ID: teamID},
			OrganizationID: s.orgID,
			Name:           "Platform",
		}, nil).Once()
	s.teamMembers.On("DeleteTeamMember", s.ctx, s.orgID.String(), memberID.String()).Return(nil).Once()
	s.auditSvc.On("CreateAuditLog", s.ctx, mock.Anything, mock.Anything).Return(&models.AuditLog{}, nil).Once()

	access := s.writeAccess()
	access.Roles[0].Permissions = []string{constants.PERMISSION_TEAM_MEMBER_DELETE}

	err := s.svc.RemoveTeamMember(s.ctx, &team_service.RemoveTeamMemberRequest{
		OrganizationID: s.orgID.String(),
		TeamID:         teamID.String(),
		MemberID:       memberID.String(),
	}, access)
	s.NoError(err)
}

func (s *TeamServiceSuite) TestListTeamMembers_Success() {
	teamID := uuid.New()
	memberID := uuid.New()
	members := []models.TeamMemberWithUser{{
		TeamMember: models.TeamMember{
			ModelBase:      chi_types.ModelBase{ID: memberID},
			OrganizationID: s.orgID,
			TeamID:         teamID,
		},
	}}
	s.expectProOrg()
	s.teamsRepo.On("GetTeamByID", s.ctx, s.orgID.String(), teamID.String()).
		Return(&models.Team{
			ModelBase:      chi_types.ModelBase{ID: teamID},
			OrganizationID: s.orgID,
		}, nil).Once()
	s.teamMembers.On("ListTeamMembersByTeamID", s.ctx, s.orgID.String(), teamID.String()).Return(&members, nil).Once()

	access := s.writeAccess()
	access.Roles[0].Permissions = []string{constants.PERMISSION_TEAM_MEMBER_READ}

	result, err := s.svc.ListTeamMembers(s.ctx, &team_service.ListTeamMembersRequest{
		OrganizationID: s.orgID.String(),
		TeamID:         teamID.String(),
	}, access)
	s.Require().NoError(err)
	s.Len(*result, 1)
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
