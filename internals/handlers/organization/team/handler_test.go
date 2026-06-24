package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	team_handlers "github.com/yca-software/2chi-go-api/internals/handlers/organization/team"
	"github.com/yca-software/2chi-go-api/internals/models"
	team_service "github.com/yca-software/2chi-go-api/internals/services/team"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_types "github.com/yca-software/2chi-go-types"
)

const (
	testOrgID    = "33333333-3333-4333-8333-333333333303"
	testTeamID   = "44444444-4444-4444-8444-444444444401"
	testMemberID = "88888888-8888-4888-8888-888888888801"
)

type TeamsHandlerSuite struct {
	suite.Suite
	echo        *echo.Echo
	teamService *team_service.MockService
	handler     *team_handlers.TeamsHandler
	userAccess  *chi_types.AccessInfo
}

func TestTeamsHandlerSuite(t *testing.T) {
	suite.Run(t, new(TeamsHandlerSuite))
}

func (s *TeamsHandlerSuite) SetupTest() {
	s.echo = echo.New()
	s.echo.HTTPErrorHandler = func(err error, c echo.Context) {
		if apiErr, ok := chi_error.AsError(err); ok {
			_ = c.JSON(apiErr.StatusCode, apiErr)
			return
		}
		_ = c.JSON(http.StatusInternalServerError, map[string]any{"message": err.Error()})
	}

	s.teamService = new(team_service.MockService)
	s.handler = team_handlers.NewTeamsHandler(s.teamService, &chi_logger.MockLogger{})
	s.userAccess = &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeUser,
		SubjectID: uuid.MustParse("11111111-1111-4111-8111-111111111101"),
		Email:     "user@example.com",
	}
}

func (s *TeamsHandlerSuite) withAccess(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Set("accessInfo", s.userAccess)
		return next(c)
	}
}

func (s *TeamsHandlerSuite) TestListTeams_Success() {
	teams := []models.Team{{Name: "Engineering"}}
	s.teamService.On("ListTeams", mock.Anything, mock.MatchedBy(func(req *team_service.ListTeamsRequest) bool {
		return req.OrganizationID == testOrgID
	}), s.userAccess).Return(&teams, nil).Once()

	s.echo.GET("/api/v1/organization/:orgId/team", s.handler.ListTeams, s.withAccess)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/organization/"+testOrgID+"/team", nil)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)
	s.teamService.AssertExpectations(s.T())
}

func (s *TeamsHandlerSuite) TestCreateTeam_Success() {
	s.teamService.On("CreateTeam", mock.Anything, mock.MatchedBy(func(req *team_service.CreateTeamRequest) bool {
		return req.OrganizationID == testOrgID && req.Name == "Engineering"
	}), s.userAccess).Return(&models.Team{Name: "Engineering"}, nil).Once()

	s.echo.POST("/api/v1/organization/:orgId/team", s.handler.CreateTeam, s.withAccess)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/organization/"+testOrgID+"/team", strings.NewReader(`{"name":"Engineering"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusCreated, rec.Code)
	s.teamService.AssertExpectations(s.T())
}

func (s *TeamsHandlerSuite) TestAddTeamMember_Success() {
	s.teamService.On("AddTeamMember", mock.Anything, mock.MatchedBy(func(req *team_service.AddTeamMemberRequest) bool {
		return req.OrganizationID == testOrgID && req.TeamID == testTeamID
	}), s.userAccess).Return(&models.TeamMemberWithUser{}, nil).Once()

	s.echo.POST("/api/v1/organization/:orgId/team/:teamId/member", s.handler.AddTeamMember, s.withAccess)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/organization/"+testOrgID+"/team/"+testTeamID+"/member", strings.NewReader(`{"userId":"`+testMemberID+`"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusCreated, rec.Code)
	s.teamService.AssertExpectations(s.T())
}

func (s *TeamsHandlerSuite) TestUpdateTeam_Success() {
	s.teamService.On("UpdateTeam", mock.Anything, mock.MatchedBy(func(req *team_service.UpdateTeamRequest) bool {
		return req.OrganizationID == testOrgID && req.TeamID == testTeamID && req.Name == "Platform"
	}), s.userAccess).Return(&models.Team{Name: "Platform"}, nil).Once()

	s.echo.PATCH("/api/v1/organization/:orgId/team/:teamId", s.handler.UpdateTeam, s.withAccess)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/organization/"+testOrgID+"/team/"+testTeamID, strings.NewReader(`{"name":"Platform"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)
	s.teamService.AssertExpectations(s.T())
}

func (s *TeamsHandlerSuite) TestDeleteTeam_Success() {
	s.teamService.On("DeleteTeam", mock.Anything, mock.MatchedBy(func(req *team_service.DeleteTeamRequest) bool {
		return req.OrganizationID == testOrgID && req.TeamID == testTeamID
	}), s.userAccess).Return(nil).Once()

	s.echo.DELETE("/api/v1/organization/:orgId/team/:teamId", s.handler.DeleteTeam, s.withAccess)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/organization/"+testOrgID+"/team/"+testTeamID, nil)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusNoContent, rec.Code)
	s.teamService.AssertExpectations(s.T())
}

func (s *TeamsHandlerSuite) TestListTeamMembers_Success() {
	members := []models.TeamMemberWithUser{{TeamMember: models.TeamMember{}}}
	s.teamService.On("ListTeamMembers", mock.Anything, mock.MatchedBy(func(req *team_service.ListTeamMembersRequest) bool {
		return req.OrganizationID == testOrgID && req.TeamID == testTeamID
	}), s.userAccess).Return(&members, nil).Once()

	s.echo.GET("/api/v1/organization/:orgId/team/:teamId/member", s.handler.ListTeamMembers, s.withAccess)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/organization/"+testOrgID+"/team/"+testTeamID+"/member", nil)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)
	s.teamService.AssertExpectations(s.T())
}

func (s *TeamsHandlerSuite) TestRemoveTeamMember_Success() {
	s.teamService.On("RemoveTeamMember", mock.Anything, mock.MatchedBy(func(req *team_service.RemoveTeamMemberRequest) bool {
		return req.OrganizationID == testOrgID && req.TeamID == testTeamID && req.MemberID == testMemberID
	}), s.userAccess).Return(nil).Once()

	s.echo.DELETE("/api/v1/organization/:orgId/team/:teamId/member/:memberId", s.handler.RemoveTeamMember, s.withAccess)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/organization/"+testOrgID+"/team/"+testTeamID+"/member/"+testMemberID, nil)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusNoContent, rec.Code)
	s.teamService.AssertExpectations(s.T())
}
