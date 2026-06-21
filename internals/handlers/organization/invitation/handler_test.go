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
	invitation_handlers "github.com/yca-software/2chi-go-api/internals/handlers/organization/invitation"
	"github.com/yca-software/2chi-go-api/internals/models"
	invitation_service "github.com/yca-software/2chi-go-api/internals/services/invitation"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_types "github.com/yca-software/2chi-go-types"
)

const (
	testOrgID        = "33333333-3333-4333-8333-333333333303"
	testRoleID       = "55555555-5555-4555-8555-555555555501"
	testInvitationID = "77777777-7777-4777-8777-777777777701"
)

type InvitationsHandlerSuite struct {
	suite.Suite
	echo               *echo.Echo
	invitationsService *invitation_service.MockService
	handler            *invitation_handlers.InvitationsHandler
	userAccess         *chi_types.AccessInfo
}

func TestInvitationsHandlerSuite(t *testing.T) {
	suite.Run(t, new(InvitationsHandlerSuite))
}

func (s *InvitationsHandlerSuite) SetupTest() {
	s.echo = echo.New()
	s.echo.HTTPErrorHandler = func(err error, c echo.Context) {
		if apiErr, ok := chi_error.AsError(err); ok {
			_ = c.JSON(apiErr.StatusCode, apiErr)
			return
		}
		_ = c.JSON(http.StatusInternalServerError, map[string]any{"message": err.Error()})
	}

	s.invitationsService = new(invitation_service.MockService)
	s.handler = invitation_handlers.NewInvitationsHandler(s.invitationsService, &chi_logger.MockLogger{})
	s.userAccess = &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeUser,
		SubjectID: uuid.MustParse("11111111-1111-4111-8111-111111111101"),
		Email:     "user@example.com",
	}
}

func (s *InvitationsHandlerSuite) withAccess(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Set("accessInfo", s.userAccess)
		return next(c)
	}
}

func (s *InvitationsHandlerSuite) TestListInvitations_Success() {
	invitations := []models.Invitation{{Email: "invitee@example.com"}}
	s.invitationsService.On("ListInvitations", mock.Anything, mock.MatchedBy(func(req *invitation_service.ListInvitationsRequest) bool {
		return req.OrganizationID == testOrgID
	}), s.userAccess).Return(&invitations, nil).Once()

	s.echo.GET("/api/v1/organization/:orgId/invitation", s.handler.ListInvitations, s.withAccess)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/organization/"+testOrgID+"/invitation", nil)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)
	s.invitationsService.AssertExpectations(s.T())
}

func (s *InvitationsHandlerSuite) TestCreateInvitation_Success() {
	s.invitationsService.On("CreateInvitation", mock.Anything, mock.MatchedBy(func(req *invitation_service.CreateInvitationRequest) bool {
		return req.OrganizationID == testOrgID && req.Email == "invitee@example.com"
	}), s.userAccess).Return(&invitation_service.CreateInvitationResponse{}, nil).Once()

	s.echo.POST("/api/v1/organization/:orgId/invitation", s.handler.CreateInvitation, s.withAccess)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/organization/"+testOrgID+"/invitation", strings.NewReader(`{"email":"invitee@example.com","roleId":"`+testRoleID+`"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusCreated, rec.Code)
	s.invitationsService.AssertExpectations(s.T())
}

func (s *InvitationsHandlerSuite) TestRevokeInvitation_Success() {
	s.invitationsService.On("RevokeInvitation", mock.Anything, mock.MatchedBy(func(req *invitation_service.RevokeInvitationRequest) bool {
		return req.OrganizationID == testOrgID && req.InvitationID == testInvitationID
	}), s.userAccess).Return(nil).Once()

	s.echo.DELETE("/api/v1/organization/:orgId/invitation/:invitationId", s.handler.RevokeInvitation, s.withAccess)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/organization/"+testOrgID+"/invitation/"+testInvitationID, nil)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusNoContent, rec.Code)
	s.invitationsService.AssertExpectations(s.T())
}
