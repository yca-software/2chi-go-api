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
	organization_handlers "github.com/yca-software/2chi-go-api/internals/handlers/organization"
	"github.com/yca-software/2chi-go-api/internals/models"
	audit_service "github.com/yca-software/2chi-go-api/internals/services/audit"
	billing_service "github.com/yca-software/2chi-go-api/internals/services/billing"
	invitation_service "github.com/yca-software/2chi-go-api/internals/services/invitation"
	organization_service "github.com/yca-software/2chi-go-api/internals/services/organization"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_types "github.com/yca-software/2chi-go-types"
)

const testOrgID = "33333333-3333-4333-8333-333333333303"
const testMemberID = "88888888-8888-4888-8888-888888888801"

type OrganizationsHandlerSuite struct {
	suite.Suite
	echo         *echo.Echo
	orgService   *organization_service.MockService
	billingSvc   *billing_service.MockService
	auditSvc     *audit_service.MockService
	handler      *organization_handlers.OrganizationsHandler
	userAccess   *chi_types.AccessInfo
}

func TestOrganizationsHandlerSuite(t *testing.T) {
	suite.Run(t, new(OrganizationsHandlerSuite))
}

func (s *OrganizationsHandlerSuite) SetupTest() {
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
	s.billingSvc = new(billing_service.MockService)
	s.auditSvc = new(audit_service.MockService)
	s.handler = organization_handlers.NewOrganizationsHandler(
		s.orgService,
		s.billingSvc,
		s.auditSvc,
		nil, nil, nil, nil,
		&chi_logger.MockLogger{},
	)
	s.userAccess = &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeUser,
		SubjectID: uuid.MustParse("11111111-1111-4111-8111-111111111101"),
		Email:     "user@example.com",
	}
}

func (s *OrganizationsHandlerSuite) withAccess(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Set("accessInfo", s.userAccess)
		return next(c)
	}
}

func (s *OrganizationsHandlerSuite) postJSON(path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)
	return rec
}

func (s *OrganizationsHandlerSuite) get(path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)
	return rec
}

func (s *OrganizationsHandlerSuite) patchJSON(path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPatch, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)
	return rec
}

func (s *OrganizationsHandlerSuite) delete(path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodDelete, path, nil)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)
	return rec
}

func (s *OrganizationsHandlerSuite) TestCreateOrganization_Success() {
	s.orgService.On("CreateOrganization", mock.Anything, mock.Anything, s.userAccess).
		Return(&organization_service.CreateOrganizationResponse{}, nil).Once()

	s.echo.POST("/api/v1/organization", s.handler.CreateOrganization, s.withAccess)
	rec := s.postJSON("/api/v1/organization", `{"name":"Acme"}`)

	s.Equal(http.StatusCreated, rec.Code)
	s.orgService.AssertExpectations(s.T())
}

func (s *OrganizationsHandlerSuite) TestGetOrganization_Success() {
	s.orgService.On("GetOrganization", mock.Anything, mock.MatchedBy(func(req *organization_service.GetOrganizationRequest) bool {
		return req.OrganizationID == testOrgID
	}), s.userAccess).Return(&models.Organization{}, nil).Once()

	s.echo.GET("/api/v1/organization/:orgId", s.handler.GetOrganization, s.withAccess)
	rec := s.get("/api/v1/organization/" + testOrgID)

	s.Equal(http.StatusOK, rec.Code)
	s.orgService.AssertExpectations(s.T())
}

func (s *OrganizationsHandlerSuite) TestListAuditLogs_InvalidStartDate() {
	s.echo.GET("/api/v1/organization/:orgId/audit-log", s.handler.ListAuditLogs, s.withAccess)
	rec := s.get("/api/v1/organization/" + testOrgID + "/audit-log?startDate=bad")

	s.Equal(http.StatusUnprocessableEntity, rec.Code)
}

func (s *OrganizationsHandlerSuite) TestListAuditLogs_Success() {
	start := time.Now().UTC().Format(time.RFC3339)
	s.auditSvc.On("ListForOrganization", mock.Anything, mock.MatchedBy(func(req *audit_service.ListForOrganizationRequest) bool {
		return req.OrganizationID == testOrgID
	}), s.userAccess).Return(&audit_service.ListForOrganizationResponse{}, nil).Once()

	s.echo.GET("/api/v1/organization/:orgId/audit-log", s.handler.ListAuditLogs, s.withAccess)
	rec := s.get("/api/v1/organization/" + testOrgID + "/audit-log?startDate=" + start)

	s.Equal(http.StatusOK, rec.Code)
	s.auditSvc.AssertExpectations(s.T())
}

func (s *OrganizationsHandlerSuite) TestCreateCheckoutSession_Success() {
	s.billingSvc.On("CreateCheckoutSession", mock.Anything, mock.MatchedBy(func(req *billing_service.CreateCheckoutSessionRequest) bool {
		return req.OrganizationID == testOrgID
	}), s.userAccess).Return(&billing_service.CheckoutSessionResponse{TransactionID: "txn_123"}, nil).Once()

	s.echo.POST("/api/v1/organization/:orgId/subscription/checkout", s.handler.CreateCheckoutSession, s.withAccess)
	rec := s.postJSON("/api/v1/organization/"+testOrgID+"/subscription/checkout", `{"planId":"basic_monthly"}`)

	s.Equal(http.StatusOK, rec.Code)
	s.billingSvc.AssertExpectations(s.T())
}

func (s *OrganizationsHandlerSuite) TestListOrganizationMembers_Success() {
	s.orgService.On("ListOrganizationMembers", mock.Anything, mock.MatchedBy(func(req *organization_service.ListOrganizationMembersRequest) bool {
		return req.OrganizationID == testOrgID
	}), s.userAccess).Return(&[]models.OrganizationMemberWithUser{}, nil).Once()

	s.echo.GET("/api/v1/organization/:orgId/member", s.handler.ListOrganizationMembers, s.withAccess)
	rec := s.get("/api/v1/organization/" + testOrgID + "/member")

	s.Equal(http.StatusOK, rec.Code)
	s.orgService.AssertExpectations(s.T())
}

func (s *OrganizationsHandlerSuite) TestHandlerWithInvitationService() {
	h := organization_handlers.NewOrganizationsHandler(
		s.orgService, s.billingSvc, s.auditSvc,
		nil, nil, new(invitation_service.MockService), nil,
		&chi_logger.MockLogger{},
	)
	s.NotNil(h)
}

func (s *OrganizationsHandlerSuite) TestGetOrganizationBillingAccount_Success() {
	s.orgService.On("GetOrganizationBillingAccount", mock.Anything, mock.MatchedBy(func(req *organization_service.GetOrganizationBillingAccountRequest) bool {
		return req.OrganizationID == testOrgID && !req.IncludeArchived
	}), s.userAccess).Return(&models.OrganizationBillingAccount{}, nil).Once()

	s.echo.GET("/api/v1/organization/:orgId/billing", s.handler.GetOrganizationBillingAccount, s.withAccess)
	rec := s.get("/api/v1/organization/" + testOrgID + "/billing")

	s.Equal(http.StatusOK, rec.Code)
	s.orgService.AssertExpectations(s.T())
}

func (s *OrganizationsHandlerSuite) TestUpdateOrganization_Success() {
	s.orgService.On("UpdateOrganization", mock.Anything, mock.MatchedBy(func(req *organization_service.UpdateOrganizationRequest) bool {
		return req.OrganizationID == testOrgID && req.Name == "Renamed"
	}), s.userAccess).Return(&models.Organization{Name: "Renamed"}, nil).Once()

	s.echo.PATCH("/api/v1/organization/:orgId", s.handler.UpdateOrganization, s.withAccess)
	rec := s.patchJSON("/api/v1/organization/"+testOrgID, `{"name":"Renamed","placeId":"place_1"}`)

	s.Equal(http.StatusOK, rec.Code)
	s.orgService.AssertExpectations(s.T())
}

func (s *OrganizationsHandlerSuite) TestArchiveOrganization_Success() {
	s.orgService.On("ArchiveOrganization", mock.Anything, mock.MatchedBy(func(req *organization_service.ArchiveOrganizationRequest) bool {
		return req.OrganizationID == testOrgID
	}), s.userAccess).Return(nil).Once()

	s.echo.POST("/api/v1/organization/:orgId/archive", s.handler.ArchiveOrganization, s.withAccess)
	rec := s.postJSON("/api/v1/organization/"+testOrgID+"/archive", `{}`)

	s.Equal(http.StatusNoContent, rec.Code)
	s.orgService.AssertExpectations(s.T())
}

func (s *OrganizationsHandlerSuite) TestUpdateOrganizationMemberRole_Success() {
	s.orgService.On("UpdateOrganizationMember", mock.Anything, mock.MatchedBy(func(req *organization_service.UpdateOrganizationMemberRequest) bool {
		return req.OrganizationID == testOrgID && req.MemberID == testMemberID
	}), s.userAccess).Return(&models.OrganizationMemberWithUser{}, nil).Once()

	s.echo.PATCH("/api/v1/organization/:orgId/member/:memberId/role", s.handler.UpdateOrganizationMemberRole, s.withAccess)
	rec := s.patchJSON("/api/v1/organization/"+testOrgID+"/member/"+testMemberID+"/role", `{"roleId":"55555555-5555-4555-8555-555555555501"}`)

	s.Equal(http.StatusOK, rec.Code)
	s.orgService.AssertExpectations(s.T())
}

func (s *OrganizationsHandlerSuite) TestRemoveOrganizationMember_Success() {
	s.orgService.On("DeleteOrganizationMember", mock.Anything, mock.MatchedBy(func(req *organization_service.DeleteOrganizationMemberRequest) bool {
		return req.OrganizationID == testOrgID && req.MemberID == testMemberID
	}), s.userAccess).Return(nil).Once()

	s.echo.DELETE("/api/v1/organization/:orgId/member/:memberId", s.handler.RemoveOrganizationMember, s.withAccess)
	rec := s.delete("/api/v1/organization/" + testOrgID + "/member/" + testMemberID)

	s.Equal(http.StatusNoContent, rec.Code)
	s.orgService.AssertExpectations(s.T())
}

func (s *OrganizationsHandlerSuite) TestChangePlan_Success() {
	s.billingSvc.On("ChangePlan", mock.Anything, mock.MatchedBy(func(req *billing_service.ChangePlanRequest) bool {
		return req.OrganizationID == testOrgID
	}), s.userAccess).Return(&billing_service.ChangePlanResponse{}, nil).Once()

	s.echo.POST("/api/v1/organization/:orgId/subscription/change-plan", s.handler.ChangePlan, s.withAccess)
	rec := s.postJSON("/api/v1/organization/"+testOrgID+"/subscription/change-plan", `{"planId":"basic_monthly"}`)

	s.Equal(http.StatusOK, rec.Code)
	s.billingSvc.AssertExpectations(s.T())
}

func (s *OrganizationsHandlerSuite) TestCreateCustomerPortalSession_Success() {
	s.billingSvc.On("CreateCustomerPortalSession", mock.Anything, mock.MatchedBy(func(req *billing_service.CreateCustomerPortalSessionRequest) bool {
		return req.OrganizationID == testOrgID
	}), s.userAccess).Return(&billing_service.CustomerPortalSessionResponse{PortalURL: "https://portal.example.com"}, nil).Once()

	s.echo.POST("/api/v1/organization/:orgId/subscription/portal", s.handler.CreateCustomerPortalSession, s.withAccess)
	rec := s.postJSON("/api/v1/organization/"+testOrgID+"/subscription/portal", `{}`)

	s.Equal(http.StatusOK, rec.Code)
	s.billingSvc.AssertExpectations(s.T())
}

func (s *OrganizationsHandlerSuite) TestProcessTransaction_Success() {
	s.billingSvc.On("ProcessTransaction", mock.Anything, mock.MatchedBy(func(req *billing_service.ProcessTransactionRequest) bool {
		return req.OrganizationID == testOrgID
	}), s.userAccess).Return(&models.OrganizationBillingAccount{}, nil).Once()

	s.echo.POST("/api/v1/organization/:orgId/subscription/process-transaction", s.handler.ProcessTransaction, s.withAccess)
	rec := s.postJSON("/api/v1/organization/"+testOrgID+"/subscription/process-transaction", `{"transactionId":"txn_123","priceId":"price_123"}`)

	s.Equal(http.StatusOK, rec.Code)
	s.billingSvc.AssertExpectations(s.T())
}
