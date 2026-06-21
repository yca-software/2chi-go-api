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
	role_handlers "github.com/yca-software/2chi-go-api/internals/handlers/organization/role"
	"github.com/yca-software/2chi-go-api/internals/models"
	role_service "github.com/yca-software/2chi-go-api/internals/services/role"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_types "github.com/yca-software/2chi-go-types"
)

const (
	testOrgID  = "33333333-3333-4333-8333-333333333303"
	testRoleID = "55555555-5555-4555-8555-555555555501"
)

type RolesHandlerSuite struct {
	suite.Suite
	echo        *echo.Echo
	roleService *role_service.MockService
	handler     *role_handlers.RolesHandler
	userAccess  *chi_types.AccessInfo
}

func TestRolesHandlerSuite(t *testing.T) {
	suite.Run(t, new(RolesHandlerSuite))
}

func (s *RolesHandlerSuite) SetupTest() {
	s.echo = echo.New()
	s.echo.HTTPErrorHandler = func(err error, c echo.Context) {
		if apiErr, ok := chi_error.AsError(err); ok {
			_ = c.JSON(apiErr.StatusCode, apiErr)
			return
		}
		_ = c.JSON(http.StatusInternalServerError, map[string]any{"message": err.Error()})
	}

	s.roleService = new(role_service.MockService)
	s.handler = role_handlers.NewRolesHandler(s.roleService, &chi_logger.MockLogger{})
	s.userAccess = &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeUser,
		SubjectID: uuid.MustParse("11111111-1111-4111-8111-111111111101"),
		Email:     "user@example.com",
	}
}

func (s *RolesHandlerSuite) withAccess(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Set("accessInfo", s.userAccess)
		return next(c)
	}
}

func (s *RolesHandlerSuite) TestListRoles_Success() {
	roles := []models.Role{{Name: "Admin"}}
	s.roleService.On("ListRoles", mock.Anything, mock.MatchedBy(func(req *role_service.ListRolesRequest) bool {
		return req.OrganizationID == testOrgID
	}), s.userAccess).Return(&roles, nil).Once()

	s.echo.GET("/api/v1/organization/:orgId/role", s.handler.ListRoles, s.withAccess)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/organization/"+testOrgID+"/role", nil)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)
	s.roleService.AssertExpectations(s.T())
}

func (s *RolesHandlerSuite) TestCreateRole_Success() {
	s.roleService.On("CreateRole", mock.Anything, mock.MatchedBy(func(req *role_service.CreateRoleRequest) bool {
		return req.OrganizationID == testOrgID && req.Name == "Editor"
	}), s.userAccess).Return(&models.Role{Name: "Editor"}, nil).Once()

	s.echo.POST("/api/v1/organization/:orgId/role", s.handler.CreateRole, s.withAccess)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/organization/"+testOrgID+"/role", strings.NewReader(`{"name":"Editor","permissions":["read"]}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusCreated, rec.Code)
	s.roleService.AssertExpectations(s.T())
}

func (s *RolesHandlerSuite) TestDeleteRole_Success() {
	s.roleService.On("DeleteRole", mock.Anything, mock.MatchedBy(func(req *role_service.DeleteRoleRequest) bool {
		return req.OrganizationID == testOrgID && req.RoleID == testRoleID
	}), s.userAccess).Return(nil).Once()

	s.echo.DELETE("/api/v1/organization/:orgId/role/:roleId", s.handler.DeleteRole, s.withAccess)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/organization/"+testOrgID+"/role/"+testRoleID, nil)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusNoContent, rec.Code)
	s.roleService.AssertExpectations(s.T())
}
