package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	admin_organization_handlers "github.com/yca-software/2chi-go-api/internals/handlers/admin/organization"
	"github.com/yca-software/2chi-go-api/internals/models"
	audit_service "github.com/yca-software/2chi-go-api/internals/services/audit"
	organization_service "github.com/yca-software/2chi-go-api/internals/services/organization"
	chi_archive "github.com/yca-software/2chi-go-archive"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_types "github.com/yca-software/2chi-go-types"
)

const testOrgID = "33333333-3333-4333-8333-333333333303"

type AdminOrganizationsHandlerSuite struct {
	suite.Suite
	echo         *echo.Echo
	orgService   *organization_service.MockService
	auditService *audit_service.MockService
	handler      *admin_organization_handlers.OrganizationsHandler
	adminAccess  *chi_types.AccessInfo
}

func TestAdminOrganizationsHandlerSuite(t *testing.T) {
	suite.Run(t, new(AdminOrganizationsHandlerSuite))
}

func (s *AdminOrganizationsHandlerSuite) SetupTest() {
	s.echo = echo.New()
	s.echo.HTTPErrorHandler = func(err error, c echo.Context) {
		if apiErr, ok := chi_error.AsError(err); ok {
			_ = c.JSON(apiErr.StatusCode, apiErr)
			return
		}
		if httpErr, ok := err.(*echo.HTTPError); ok {
			_ = c.JSON(httpErr.Code, map[string]any{"message": httpErr.Message})
			return
		}
		_ = c.JSON(http.StatusInternalServerError, map[string]any{"message": err.Error()})
	}

	s.orgService = new(organization_service.MockService)
	s.auditService = new(audit_service.MockService)

	s.handler = admin_organization_handlers.NewOrganizationsHandler(
		s.orgService,
		s.auditService,
		&chi_logger.MockLogger{},
	)

	s.adminAccess = &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeUser,
		SubjectID: uuid.MustParse("11111111-1111-4111-8111-111111111101"),
		Email:     "admin@example.com",
		IsAdmin:   true,
		IPAddress: "127.0.0.1",
	}
}

func (s *AdminOrganizationsHandlerSuite) withAccess(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Set("accessInfo", s.adminAccess)
		return next(c)
	}
}

func (s *AdminOrganizationsHandlerSuite) postJSON(path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)
	return rec
}

func (s *AdminOrganizationsHandlerSuite) patchJSON(path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPatch, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)
	return rec
}

func (s *AdminOrganizationsHandlerSuite) get(path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)
	return rec
}

func (s *AdminOrganizationsHandlerSuite) TestListOrganizations_Success() {
	s.orgService.On("ListOrganizations", mock.Anything, mock.MatchedBy(func(req *organization_service.ListOrganizationsRequest) bool {
		return req.SearchPhrase == "" &&
			req.ArchiveFilter == chi_archive.ArchiveFilterActive &&
			req.Limit == 20 &&
			req.Offset == 0
	}), s.adminAccess).Return(&organization_service.ListOrganizationsResponse{}, nil).Once()

	s.echo.GET("/api/v1/admin/organization", s.handler.ListOrganizations, s.withAccess)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/organization", nil)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)
	s.orgService.AssertExpectations(s.T())
}

func (s *AdminOrganizationsHandlerSuite) TestListOrganizationAuditLogs_Success() {
	s.auditService.On("ListForOrganization", mock.Anything, mock.MatchedBy(func(req *audit_service.ListForOrganizationRequest) bool {
		return req.OrganizationID == testOrgID &&
			req.Limit == 50 &&
			req.Offset == 0
	}), s.adminAccess).Return(&audit_service.ListForOrganizationResponse{}, nil).Once()

	s.echo.GET("/api/v1/admin/organization/:orgId/audit-log", s.handler.ListOrganizationAuditLogs, s.withAccess)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/organization/"+testOrgID+"/audit-log", nil)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)
	s.auditService.AssertExpectations(s.T())
}

func (s *AdminOrganizationsHandlerSuite) TestListArchivedOrganizations_Success() {
	s.orgService.On("ListOrganizations", mock.Anything, mock.MatchedBy(func(req *organization_service.ListOrganizationsRequest) bool {
		return req.ArchiveFilter == chi_archive.ArchiveFilterArchived
	}), s.adminAccess).Return(&organization_service.ListOrganizationsResponse{}, nil).Once()

	s.echo.GET("/api/v1/admin/organization/archived", s.handler.ListArchivedOrganizations, s.withAccess)
	rec := s.get("/api/v1/admin/organization/archived")

	s.Equal(http.StatusOK, rec.Code)
	s.orgService.AssertExpectations(s.T())
}

func (s *AdminOrganizationsHandlerSuite) TestGetOrganization_Success() {
	s.orgService.On("GetOrganization", mock.Anything, mock.MatchedBy(func(req *organization_service.GetOrganizationRequest) bool {
		return req.OrganizationID == testOrgID
	}), s.adminAccess).Return(&models.Organization{}, nil).Once()

	s.echo.GET("/api/v1/admin/organization/:orgId", s.handler.GetOrganization, s.withAccess)
	rec := s.get("/api/v1/admin/organization/" + testOrgID)

	s.Equal(http.StatusOK, rec.Code)
	s.orgService.AssertExpectations(s.T())
}

func (s *AdminOrganizationsHandlerSuite) TestGetOrganizationBillingAccount_Success() {
	s.orgService.On("GetOrganizationBillingAccount", mock.Anything, mock.MatchedBy(func(req *organization_service.GetOrganizationBillingAccountRequest) bool {
		return req.OrganizationID == testOrgID && req.IncludeArchived
	}), s.adminAccess).Return(&models.OrganizationBillingAccount{}, nil).Once()

	s.echo.GET("/api/v1/admin/organization/:orgId/billing", s.handler.GetOrganizationBillingAccount, s.withAccess)
	rec := s.get("/api/v1/admin/organization/" + testOrgID + "/billing")

	s.Equal(http.StatusOK, rec.Code)
	s.orgService.AssertExpectations(s.T())
}

func (s *AdminOrganizationsHandlerSuite) TestGetArchivedOrganization_Success() {
	s.orgService.On("GetArchivedOrganization", mock.Anything, mock.MatchedBy(func(req *organization_service.GetOrganizationRequest) bool {
		return req.OrganizationID == testOrgID
	}), s.adminAccess).Return(&models.Organization{}, nil).Once()

	s.echo.GET("/api/v1/admin/organization/archived/:orgId", s.handler.GetArchivedOrganization, s.withAccess)
	rec := s.get("/api/v1/admin/organization/archived/" + testOrgID)

	s.Equal(http.StatusOK, rec.Code)
	s.orgService.AssertExpectations(s.T())
}

func (s *AdminOrganizationsHandlerSuite) TestCreateOrganization_Success() {
	s.orgService.On("AdminCreateOrganization", mock.Anything, mock.Anything, s.adminAccess).
		Return(&models.Organization{}, nil).Once()

	s.echo.POST("/api/v1/admin/organization", s.handler.CreateOrganization, s.withAccess)
	rec := s.postJSON("/api/v1/admin/organization", `{
		"name":"Acme",
		"placeId":"place_1",
		"billingEmail":"billing@example.com",
		"ownerEmail":"owner@example.com",
		"subscriptionType":"basic",
		"subscriptionSeats":5,
		"language":"en"
	}`)

	s.Equal(http.StatusCreated, rec.Code)
	s.orgService.AssertExpectations(s.T())
}

func (s *AdminOrganizationsHandlerSuite) TestRestoreOrganization_Success() {
	s.orgService.On("RestoreOrganization", mock.Anything, mock.MatchedBy(func(req *organization_service.RestoreOrganizationRequest) bool {
		return req.OrganizationID == testOrgID
	}), s.adminAccess).Return(&models.Organization{}, nil).Once()

	s.echo.POST("/api/v1/admin/organization/archived/:orgId/restore", s.handler.RestoreOrganization, s.withAccess)
	rec := s.postJSON("/api/v1/admin/organization/archived/"+testOrgID+"/restore", `{}`)

	s.Equal(http.StatusOK, rec.Code)
	s.orgService.AssertExpectations(s.T())
}

func (s *AdminOrganizationsHandlerSuite) TestUpdateOrganizationSubscription_Success() {
	expiresAt := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	s.orgService.On("UpdateOrganizationSubscription", mock.Anything, mock.MatchedBy(func(req *organization_service.UpdateOrganizationSubscriptionRequest) bool {
		return req.OrganizationID == testOrgID && req.SubscriptionType == "basic"
	}), s.adminAccess).Return(&models.OrganizationBillingAccount{}, nil).Once()

	s.echo.PATCH("/api/v1/admin/organization/:orgId/subscription", s.handler.UpdateOrganizationSubscription, s.withAccess)
	rec := s.patchJSON("/api/v1/admin/organization/"+testOrgID+"/subscription", `{
		"customSubscription":true,
		"subscriptionType":"basic",
		"subscriptionSeats":5,
		"subscriptionExpiresAt":"`+expiresAt.Format(time.RFC3339)+`"
	}`)

	s.Equal(http.StatusOK, rec.Code)
	s.orgService.AssertExpectations(s.T())
}
