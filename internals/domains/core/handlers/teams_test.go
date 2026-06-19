package handlers_test

import (
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	core_handlers "github.com/yca-software/2chi-go-api/internals/domains/core/handlers"
	"github.com/yca-software/2chi-go-api/internals/domains/core/models"
	team_service "github.com/yca-software/2chi-go-api/internals/domains/core/services/team"
)

type TeamsHandlerSuite struct {
	suite.Suite
	echo        *echo.Echo
	teamService *team_service.MockService
	handler     *core_handlers.TeamsHandler
}

func TestTeamsHandlerSuite(t *testing.T) {
	suite.Run(t, new(TeamsHandlerSuite))
}

func (s *TeamsHandlerSuite) SetupTest() {
	s.echo = newEchoWithAPIErrors()
	s.teamService = new(team_service.MockService)
	s.handler = core_handlers.NewTeamsHandler(s.teamService, testLogger())
}

func (s *TeamsHandlerSuite) TestListTeams_Success() {
	teams := []models.Team{{Name: "Engineering"}}
	s.teamService.On("ListTeams", mock.Anything, mock.MatchedBy(func(req *team_service.ListTeamsRequest) bool {
		return req.OrganizationID == testOrgID.String()
	}), mock.Anything).Return(&teams, nil).Once()

	s.echo.GET("/api/v1/organization/:orgId/team", s.handler.ListTeams, injectAccess(testUserAccess()))
	rec := serve(s.echo, jsonRequest(http.MethodGet, "/api/v1/organization/"+testOrgID.String()+"/team", nil))

	s.Equal(http.StatusOK, rec.Code)
	s.teamService.AssertExpectations(s.T())
}

func (s *TeamsHandlerSuite) TestCreateTeam_Success() {
	s.teamService.On("CreateTeam", mock.Anything, mock.MatchedBy(func(req *team_service.CreateTeamRequest) bool {
		return req.OrganizationID == testOrgID.String() && req.Name == "Engineering"
	}), mock.Anything).Return(&models.Team{Name: "Engineering"}, nil).Once()

	s.echo.POST("/api/v1/organization/:orgId/team", s.handler.CreateTeam, injectAccess(testUserAccess()))
	rec := serve(s.echo, jsonRequest(http.MethodPost, "/api/v1/organization/"+testOrgID.String()+"/team", map[string]string{
		"name": "Engineering",
	}))

	s.Equal(http.StatusCreated, rec.Code)
	s.teamService.AssertExpectations(s.T())
}

func (s *TeamsHandlerSuite) TestAddTeamMember_Success() {
	s.teamService.On("AddTeamMember", mock.Anything, mock.MatchedBy(func(req *team_service.AddTeamMemberRequest) bool {
		return req.OrganizationID == testOrgID.String() && req.TeamID == testTeamID.String()
	}), mock.Anything).Return(&models.TeamMemberWithUser{}, nil).Once()

	s.echo.POST("/api/v1/organization/:orgId/team/:teamId/member", s.handler.AddTeamMember, injectAccess(testUserAccess()))
	rec := serve(s.echo, jsonRequest(http.MethodPost, "/api/v1/organization/"+testOrgID.String()+"/team/"+testTeamID.String()+"/member", map[string]string{
		"userId": testMemberID.String(),
	}))

	s.Equal(http.StatusCreated, rec.Code)
	s.teamService.AssertExpectations(s.T())
}
