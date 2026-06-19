package handlers_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	core_handlers "github.com/yca-software/2chi-go-api/internals/domains/core/handlers"
	"github.com/yca-software/2chi-go-api/internals/domains/core/models"
	audit_service "github.com/yca-software/2chi-go-api/internals/domains/core/services/audit"
	billing_service "github.com/yca-software/2chi-go-api/internals/domains/core/services/billing"
	invitation_service "github.com/yca-software/2chi-go-api/internals/domains/core/services/invitation"
	organization_service "github.com/yca-software/2chi-go-api/internals/domains/core/services/organization"
)

type OrganizationsHandlerSuite struct {
	suite.Suite
	echo         *echo.Echo
	orgService   *organization_service.MockService
	billingSvc   *billing_service.MockService
	auditSvc     *audit_service.MockService
	handler      *core_handlers.OrganizationsHandler
}

func TestOrganizationsHandlerSuite(t *testing.T) {
	suite.Run(t, new(OrganizationsHandlerSuite))
}

func (s *OrganizationsHandlerSuite) SetupTest() {
	s.echo = newEchoWithAPIErrors()
	s.orgService = new(organization_service.MockService)
	s.billingSvc = new(billing_service.MockService)
	s.auditSvc = new(audit_service.MockService)
	s.handler = core_handlers.NewOrganizationsHandler(
		s.orgService,
		s.billingSvc,
		s.auditSvc,
		nil, nil, nil, nil,
		testLogger(),
	)
}

func (s *OrganizationsHandlerSuite) TestCreateOrganization_Success() {
	s.orgService.On("CreateOrganization", mock.Anything, mock.Anything, mock.Anything).
		Return(&organization_service.CreateOrganizationResponse{}, nil).Once()

	s.echo.POST("/api/v1/organization", s.handler.CreateOrganization, injectAccess(testUserAccess()))
	rec := serve(s.echo, jsonRequest(http.MethodPost, "/api/v1/organization", map[string]string{
		"name": "Acme",
	}))

	s.Equal(http.StatusCreated, rec.Code)
	s.orgService.AssertExpectations(s.T())
}

func (s *OrganizationsHandlerSuite) TestGetOrganization_Success() {
	s.orgService.On("GetOrganization", mock.Anything, mock.MatchedBy(func(req *organization_service.GetOrganizationRequest) bool {
		return req.OrganizationID == testOrgID.String()
	}), mock.Anything).Return(&models.Organization{}, nil).Once()

	s.echo.GET("/api/v1/organization/:orgId", s.handler.GetOrganization, injectAccess(testUserAccess()))
	rec := serve(s.echo, jsonRequest(http.MethodGet, "/api/v1/organization/"+testOrgID.String(), nil))

	s.Equal(http.StatusOK, rec.Code)
	s.orgService.AssertExpectations(s.T())
}

func (s *OrganizationsHandlerSuite) TestListAuditLogs_InvalidStartDate() {
	s.echo.GET("/api/v1/organization/:orgId/audit-log", s.handler.ListAuditLogs, injectAccess(testUserAccess()))
	rec := serve(s.echo, jsonRequest(http.MethodGet, "/api/v1/organization/"+testOrgID.String()+"/audit-log?startDate=bad", nil))

	s.Equal(http.StatusUnprocessableEntity, rec.Code)
}

func (s *OrganizationsHandlerSuite) TestListAuditLogs_Success() {
	start := time.Now().UTC().Format(time.RFC3339)
	s.auditSvc.On("ListAuditLogsForOrganization", mock.Anything, mock.Anything, mock.Anything).
		Return(&audit_service.ListAuditLogsForOrganizationResponse{}, nil).Once()

	s.echo.GET("/api/v1/organization/:orgId/audit-log", s.handler.ListAuditLogs, injectAccess(testUserAccess()))
	rec := serve(s.echo, jsonRequest(http.MethodGet, "/api/v1/organization/"+testOrgID.String()+"/audit-log?startDate="+start, nil))

	s.Equal(http.StatusOK, rec.Code)
	s.auditSvc.AssertExpectations(s.T())
}

func (s *OrganizationsHandlerSuite) TestCreateCheckoutSession_Success() {
	s.billingSvc.On("CreateCheckoutSession", mock.Anything, mock.Anything, mock.Anything).
		Return(&billing_service.CheckoutSessionResponse{TransactionID: "txn_123"}, nil).Once()

	s.echo.POST("/api/v1/organization/:orgId/subscription/checkout", s.handler.CreateCheckoutSession, injectAccess(testUserAccess()))
	rec := serve(s.echo, jsonRequest(http.MethodPost, "/api/v1/organization/"+testOrgID.String()+"/subscription/checkout", map[string]string{
		"planId": "basic_monthly",
	}))

	s.Equal(http.StatusOK, rec.Code)
	s.billingSvc.AssertExpectations(s.T())
}

func (s *OrganizationsHandlerSuite) TestListOrganizationMembers_Success() {
	s.orgService.On("ListOrganizationMembers", mock.Anything, mock.Anything, mock.Anything).
		Return(&[]models.OrganizationMemberWithUser{}, nil).Once()

	s.echo.GET("/api/v1/organization/:orgId/member", s.handler.ListOrganizationMembers, injectAccess(testUserAccess()))
	rec := serve(s.echo, jsonRequest(http.MethodGet, "/api/v1/organization/"+testOrgID.String()+"/member", nil))

	s.Equal(http.StatusOK, rec.Code)
	s.orgService.AssertExpectations(s.T())
}

// Ensure invitation service wiring compiles through constructor.
func (s *OrganizationsHandlerSuite) TestHandlerWithInvitationService() {
	h := core_handlers.NewOrganizationsHandler(
		s.orgService, s.billingSvc, s.auditSvc,
		nil, nil, new(invitation_service.MockService), nil,
		testLogger(),
	)
	s.NotNil(h)
}
