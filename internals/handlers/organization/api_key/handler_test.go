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
	api_key_handlers "github.com/yca-software/2chi-go-api/internals/handlers/organization/api_key"
	"github.com/yca-software/2chi-go-api/internals/models"
	api_key_service "github.com/yca-software/2chi-go-api/internals/services/api_key"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_types "github.com/yca-software/2chi-go-types"
)

const (
	testOrgID    = "33333333-3333-4333-8333-333333333303"
	testAPIKeyID = "66666666-6666-4666-8666-666666666601"
)

type APIKeysHandlerSuite struct {
	suite.Suite
	echo           *echo.Echo
	apiKeysService *api_key_service.MockService
	handler        *api_key_handlers.APIKeysHandler
	userAccess     *chi_types.AccessInfo
}

func TestAPIKeysHandlerSuite(t *testing.T) {
	suite.Run(t, new(APIKeysHandlerSuite))
}

func (s *APIKeysHandlerSuite) SetupTest() {
	s.echo = echo.New()
	s.echo.HTTPErrorHandler = func(err error, c echo.Context) {
		if apiErr, ok := chi_error.AsError(err); ok {
			_ = c.JSON(apiErr.StatusCode, apiErr)
			return
		}
		_ = c.JSON(http.StatusInternalServerError, map[string]any{"message": err.Error()})
	}

	s.apiKeysService = new(api_key_service.MockService)
	s.handler = api_key_handlers.NewAPIKeysHandler(s.apiKeysService, &chi_logger.MockLogger{})
	s.userAccess = &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeUser,
		SubjectID: uuid.MustParse("11111111-1111-4111-8111-111111111101"),
		Email:     "user@example.com",
	}
}

func (s *APIKeysHandlerSuite) withAccess(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Set("accessInfo", s.userAccess)
		return next(c)
	}
}

func (s *APIKeysHandlerSuite) TestListAPIKeys_Success() {
	keys := []models.APIKey{{Name: "CI"}}
	s.apiKeysService.On("List", mock.Anything, mock.MatchedBy(func(req *api_key_service.ListRequest) bool {
		return req.OrganizationID == testOrgID
	}), s.userAccess).Return(&keys, nil).Once()

	s.echo.GET("/api/v1/organization/:orgId/api-key", s.handler.ListAPIKeys, s.withAccess)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/organization/"+testOrgID+"/api-key", nil)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)
	s.apiKeysService.AssertExpectations(s.T())
}

func (s *APIKeysHandlerSuite) TestCreateAPIKey_Success() {
	s.apiKeysService.On("Create", mock.Anything, mock.MatchedBy(func(req *api_key_service.CreateRequest) bool {
		return req.OrganizationID == testOrgID && req.Name == "CI"
	}), s.userAccess).Return(&api_key_service.CreateResponse{Secret: "secret"}, nil).Once()

	s.echo.POST("/api/v1/organization/:orgId/api-key", s.handler.CreateAPIKey, s.withAccess)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/organization/"+testOrgID+"/api-key", strings.NewReader(`{"name":"CI","permissions":["read"]}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusCreated, rec.Code)
	s.apiKeysService.AssertExpectations(s.T())
}

func (s *APIKeysHandlerSuite) TestDeleteAPIKey_Success() {
	s.apiKeysService.On("Delete", mock.Anything, mock.MatchedBy(func(req *api_key_service.DeleteRequest) bool {
		return req.OrganizationID == testOrgID && req.APIKeyID == testAPIKeyID
	}), s.userAccess).Return(nil).Once()

	s.echo.DELETE("/api/v1/organization/:orgId/api-key/:apiKeyId", s.handler.DeleteAPIKey, s.withAccess)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/organization/"+testOrgID+"/api-key/"+testAPIKeyID, nil)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusNoContent, rec.Code)
	s.apiKeysService.AssertExpectations(s.T())
}
