package handlers_test

import (
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	core_handlers "github.com/yca-software/2chi-go-api/internals/domains/core/handlers"
	"github.com/yca-software/2chi-go-api/internals/domains/core/models"
	api_key_service "github.com/yca-software/2chi-go-api/internals/domains/core/services/api_key"
)

type APIKeysHandlerSuite struct {
	suite.Suite
	echo           *echo.Echo
	apiKeysService *api_key_service.MockService
	handler        *core_handlers.APIKeysHandler
}

func TestAPIKeysHandlerSuite(t *testing.T) {
	suite.Run(t, new(APIKeysHandlerSuite))
}

func (s *APIKeysHandlerSuite) SetupTest() {
	s.echo = newEchoWithAPIErrors()
	s.apiKeysService = new(api_key_service.MockService)
	s.handler = core_handlers.NewAPIKeysHandler(s.apiKeysService, testLogger())
}

func (s *APIKeysHandlerSuite) TestListAPIKeys_Success() {
	keys := []models.APIKey{{Name: "CI"}}
	s.apiKeysService.On("ListAPIKeys", mock.Anything, mock.MatchedBy(func(req *api_key_service.ListAPIKeysRequest) bool {
		return req.OrganizationID == testOrgID.String()
	}), mock.Anything).Return(&keys, nil).Once()

	s.echo.GET("/api/v1/organization/:orgId/api-key", s.handler.ListAPIKeys, injectAccess(testUserAccess()))
	rec := serve(s.echo, jsonRequest(http.MethodGet, "/api/v1/organization/"+testOrgID.String()+"/api-key", nil))

	s.Equal(http.StatusOK, rec.Code)
	s.apiKeysService.AssertExpectations(s.T())
}

func (s *APIKeysHandlerSuite) TestCreateAPIKey_Success() {
	s.apiKeysService.On("CreateAPIKey", mock.Anything, mock.MatchedBy(func(req *api_key_service.CreateAPIKeyRequest) bool {
		return req.OrganizationID == testOrgID.String() && req.Name == "CI"
	}), mock.Anything).Return(&api_key_service.CreateAPIKeyResponse{Secret: "secret"}, nil).Once()

	s.echo.POST("/api/v1/organization/:orgId/api-key", s.handler.CreateAPIKey, injectAccess(testUserAccess()))
	rec := serve(s.echo, jsonRequest(http.MethodPost, "/api/v1/organization/"+testOrgID.String()+"/api-key", map[string]any{
		"name":        "CI",
		"permissions": []string{"read"},
	}))

	s.Equal(http.StatusCreated, rec.Code)
	s.apiKeysService.AssertExpectations(s.T())
}

func (s *APIKeysHandlerSuite) TestDeleteAPIKey_Success() {
	s.apiKeysService.On("DeleteAPIKey", mock.Anything, mock.MatchedBy(func(req *api_key_service.DeleteAPIKeyRequest) bool {
		return req.OrganizationID == testOrgID.String() && req.APIKeyID == testAPIKeyID.String()
	}), mock.Anything).Return(nil).Once()

	s.echo.DELETE("/api/v1/organization/:orgId/api-key/:apiKeyId", s.handler.DeleteAPIKey, injectAccess(testUserAccess()))
	rec := serve(s.echo, jsonRequest(http.MethodDelete, "/api/v1/organization/"+testOrgID.String()+"/api-key/"+testAPIKeyID.String(), nil))

	s.Equal(http.StatusNoContent, rec.Code)
	s.apiKeysService.AssertExpectations(s.T())
}
