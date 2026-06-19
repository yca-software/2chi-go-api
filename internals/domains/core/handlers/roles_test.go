package handlers_test

import (
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	core_handlers "github.com/yca-software/2chi-go-api/internals/domains/core/handlers"
	"github.com/yca-software/2chi-go-api/internals/domains/core/models"
	role_service "github.com/yca-software/2chi-go-api/internals/domains/core/services/role"
)

type RolesHandlerSuite struct {
	suite.Suite
	echo        *echo.Echo
	roleService *role_service.MockService
	handler     *core_handlers.RolesHandler
}

func TestRolesHandlerSuite(t *testing.T) {
	suite.Run(t, new(RolesHandlerSuite))
}

func (s *RolesHandlerSuite) SetupTest() {
	s.echo = newEchoWithAPIErrors()
	s.roleService = new(role_service.MockService)
	s.handler = core_handlers.NewRolesHandler(s.roleService, testLogger())
}

func (s *RolesHandlerSuite) TestListRoles_Success() {
	roles := []models.Role{{Name: "Admin"}}
	s.roleService.On("ListRoles", mock.Anything, mock.MatchedBy(func(req *role_service.ListRolesRequest) bool {
		return req.OrganizationID == testOrgID.String()
	}), mock.Anything).Return(&roles, nil).Once()

	s.echo.GET("/api/v1/organization/:orgId/role", s.handler.ListRoles, injectAccess(testUserAccess()))
	rec := serve(s.echo, jsonRequest(http.MethodGet, "/api/v1/organization/"+testOrgID.String()+"/role", nil))

	s.Equal(http.StatusOK, rec.Code)
	s.roleService.AssertExpectations(s.T())
}

func (s *RolesHandlerSuite) TestCreateRole_Success() {
	s.roleService.On("CreateRole", mock.Anything, mock.MatchedBy(func(req *role_service.CreateRoleRequest) bool {
		return req.OrganizationID == testOrgID.String() && req.Name == "Editor"
	}), mock.Anything).Return(&models.Role{Name: "Editor"}, nil).Once()

	s.echo.POST("/api/v1/organization/:orgId/role", s.handler.CreateRole, injectAccess(testUserAccess()))
	rec := serve(s.echo, jsonRequest(http.MethodPost, "/api/v1/organization/"+testOrgID.String()+"/role", map[string]any{
		"name":        "Editor",
		"permissions": []string{"read"},
	}))

	s.Equal(http.StatusCreated, rec.Code)
	s.roleService.AssertExpectations(s.T())
}

func (s *RolesHandlerSuite) TestDeleteRole_Success() {
	s.roleService.On("DeleteRole", mock.Anything, mock.MatchedBy(func(req *role_service.DeleteRoleRequest) bool {
		return req.OrganizationID == testOrgID.String() && req.RoleID == testRoleID.String()
	}), mock.Anything).Return(nil).Once()

	s.echo.DELETE("/api/v1/organization/:orgId/role/:roleId", s.handler.DeleteRole, injectAccess(testUserAccess()))
	rec := serve(s.echo, jsonRequest(http.MethodDelete, "/api/v1/organization/"+testOrgID.String()+"/role/"+testRoleID.String(), nil))

	s.Equal(http.StatusNoContent, rec.Code)
	s.roleService.AssertExpectations(s.T())
}
