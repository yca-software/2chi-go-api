package handlers_test

import (
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	core_handlers "github.com/yca-software/2chi-go-api/internals/domains/core/handlers"
	"github.com/yca-software/2chi-go-api/internals/domains/core/models"
	invitation_service "github.com/yca-software/2chi-go-api/internals/domains/core/services/invitation"
)

type InvitationsHandlerSuite struct {
	suite.Suite
	echo               *echo.Echo
	invitationsService *invitation_service.MockService
	handler            *core_handlers.InvitationsHandler
}

func TestInvitationsHandlerSuite(t *testing.T) {
	suite.Run(t, new(InvitationsHandlerSuite))
}

func (s *InvitationsHandlerSuite) SetupTest() {
	s.echo = newEchoWithAPIErrors()
	s.invitationsService = new(invitation_service.MockService)
	s.handler = core_handlers.NewInvitationsHandler(s.invitationsService, testLogger())
}

func (s *InvitationsHandlerSuite) TestListInvitations_Success() {
	invitations := []models.Invitation{{Email: "invitee@example.com"}}
	s.invitationsService.On("ListInvitations", mock.Anything, mock.MatchedBy(func(req *invitation_service.ListInvitationsRequest) bool {
		return req.OrganizationID == testOrgID.String()
	}), mock.Anything).Return(&invitations, nil).Once()

	s.echo.GET("/api/v1/organization/:orgId/invitation", s.handler.ListInvitations, injectAccess(testUserAccess()))
	rec := serve(s.echo, jsonRequest(http.MethodGet, "/api/v1/organization/"+testOrgID.String()+"/invitation", nil))

	s.Equal(http.StatusOK, rec.Code)
	s.invitationsService.AssertExpectations(s.T())
}

func (s *InvitationsHandlerSuite) TestCreateInvitation_Success() {
	s.invitationsService.On("CreateInvitation", mock.Anything, mock.MatchedBy(func(req *invitation_service.CreateInvitationRequest) bool {
		return req.OrganizationID == testOrgID.String() && req.Email == "invitee@example.com"
	}), mock.Anything).Return(&invitation_service.CreateInvitationResponse{}, nil).Once()

	s.echo.POST("/api/v1/organization/:orgId/invitation", s.handler.CreateInvitation, injectAccess(testUserAccess()))
	rec := serve(s.echo, jsonRequest(http.MethodPost, "/api/v1/organization/"+testOrgID.String()+"/invitation", map[string]string{
		"email":  "invitee@example.com",
		"roleId": testRoleID.String(),
	}))

	s.Equal(http.StatusCreated, rec.Code)
	s.invitationsService.AssertExpectations(s.T())
}

func (s *InvitationsHandlerSuite) TestRevokeInvitation_Success() {
	s.invitationsService.On("RevokeInvitation", mock.Anything, mock.MatchedBy(func(req *invitation_service.RevokeInvitationRequest) bool {
		return req.OrganizationID == testOrgID.String() && req.InvitationID == testInvitationID.String()
	}), mock.Anything).Return(nil).Once()

	s.echo.DELETE("/api/v1/organization/:orgId/invitation/:invitationId", s.handler.RevokeInvitation, injectAccess(testUserAccess()))
	rec := serve(s.echo, jsonRequest(http.MethodDelete, "/api/v1/organization/"+testOrgID.String()+"/invitation/"+testInvitationID.String(), nil))

	s.Equal(http.StatusNoContent, rec.Code)
	s.invitationsService.AssertExpectations(s.T())
}
