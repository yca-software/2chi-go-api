package handlers_test

import (
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	core_handlers "github.com/yca-software/2chi-go-api/internals/domains/core/handlers"
	audit_service "github.com/yca-software/2chi-go-api/internals/domains/core/services/audit"
	auth_service "github.com/yca-software/2chi-go-api/internals/domains/core/services/auth"
	organization_service "github.com/yca-software/2chi-go-api/internals/domains/core/services/organization"
	user_service "github.com/yca-software/2chi-go-api/internals/domains/core/services/user"
)

type AdminHandlerSuite struct {
	suite.Suite
	echo         *echo.Echo
	authService  *auth_service.MockService
	usersService *user_service.MockService
	orgService   *organization_service.MockService
	auditService *audit_service.MockService
	handler      *core_handlers.AdminHandler
}

func TestAdminHandlerSuite(t *testing.T) {
	suite.Run(t, new(AdminHandlerSuite))
}

func (s *AdminHandlerSuite) SetupTest() {
	s.echo = newEchoWithAPIErrors()
	s.authService = new(auth_service.MockService)
	s.usersService = new(user_service.MockService)
	s.orgService = new(organization_service.MockService)
	s.auditService = new(audit_service.MockService)
	s.handler = core_handlers.NewAdminHandler(
		s.authService,
		s.usersService,
		s.orgService,
		s.auditService,
		testConfig(),
		testLogger(),
	)
}

func (s *AdminHandlerSuite) TestListUsers_Success() {
	s.usersService.On("ListUsers", mock.Anything, mock.Anything, mock.Anything).
		Return(&user_service.ListUsersResponse{}, nil).Once()

	s.echo.GET("/api/v1/admin/user", s.handler.ListUsers, injectAccess(testAdminAccess()))
	rec := serve(s.echo, jsonRequest(http.MethodGet, "/api/v1/admin/user", nil))

	s.Equal(http.StatusOK, rec.Code)
	s.usersService.AssertExpectations(s.T())
}

func (s *AdminHandlerSuite) TestGetUser_Success() {
	s.usersService.On("GetUser", mock.Anything, mock.MatchedBy(func(req *user_service.GetUserRequest) bool {
		return req.UserID == testUserID.String()
	}), mock.Anything).Return(&user_service.GetUserResponse{}, nil).Once()

	s.echo.GET("/api/v1/admin/user/:userId", s.handler.GetUser, injectAccess(testAdminAccess()))
	rec := serve(s.echo, jsonRequest(http.MethodGet, "/api/v1/admin/user/"+testUserID.String(), nil))

	s.Equal(http.StatusOK, rec.Code)
	s.usersService.AssertExpectations(s.T())
}

func (s *AdminHandlerSuite) TestImpersonateUser_Success() {
	s.authService.On("Impersonate", mock.Anything, mock.MatchedBy(func(req *auth_service.ImpersonateRequest) bool {
		return req.UserID == testUserID.String()
	}), mock.Anything).Return(&auth_service.AuthenticateResponse{AccessToken: "at", RefreshToken: "rt"}, nil).Once()

	s.echo.POST("/api/v1/admin/user/:userId/impersonate", s.handler.ImpersonateUser, injectAccess(testAdminAccess()))
	rec := serve(s.echo, jsonRequest(http.MethodPost, "/api/v1/admin/user/"+testUserID.String()+"/impersonate", nil))

	s.Equal(http.StatusOK, rec.Code)
	s.authService.AssertExpectations(s.T())
}

func (s *AdminHandlerSuite) TestListOrganizations_Success() {
	s.orgService.On("ListOrganizations", mock.Anything, mock.Anything, mock.Anything).
		Return(&organization_service.ListOrganizationsResponse{}, nil).Once()

	s.echo.GET("/api/v1/admin/organization", s.handler.ListOrganizations, injectAccess(testAdminAccess()))
	rec := serve(s.echo, jsonRequest(http.MethodGet, "/api/v1/admin/organization", nil))

	s.Equal(http.StatusOK, rec.Code)
	s.orgService.AssertExpectations(s.T())
}

func (s *AdminHandlerSuite) TestListOrganizationAuditLogs_Success() {
	s.auditService.On("ListAuditLogsForOrganization", mock.Anything, mock.Anything, mock.Anything).
		Return(&audit_service.ListAuditLogsForOrganizationResponse{}, nil).Once()

	s.echo.GET("/api/v1/admin/organization/:orgId/audit-log", s.handler.ListOrganizationAuditLogs, injectAccess(testAdminAccess()))
	rec := serve(s.echo, jsonRequest(http.MethodGet, "/api/v1/admin/organization/"+testOrgID.String()+"/audit-log", nil))

	s.Equal(http.StatusOK, rec.Code)
	s.auditService.AssertExpectations(s.T())
}
