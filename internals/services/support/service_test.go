package support_service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/yca-software/2chi-go-api/internals/packages/testutil"
	support_service "github.com/yca-software/2chi-go-api/internals/services/support"
	chi_aws_ses "github.com/yca-software/2chi-go-aws/ses"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_template "github.com/yca-software/2chi-go-template"
	chi_types "github.com/yca-software/2chi-go-types"
	chi_validator "github.com/yca-software/2chi-go-validator"
)

type SupportServiceSuite struct {
	suite.Suite
	ctx         context.Context
	emailSender *chi_aws_ses.MockSES
	svc         support_service.Service
	access      *chi_types.AccessInfo
	logger      chi_logger.Logger
}

func TestSupportServiceSuite(t *testing.T) {
	suite.Run(t, new(SupportServiceSuite))
}

func (s *SupportServiceSuite) SetupTest() {
	s.ctx = context.Background()
	s.emailSender = &chi_aws_ses.MockSES{}
	s.logger = &chi_logger.MockLogger{}
	s.svc = support_service.New(support_service.Dependencies{
		Validator:         chi_validator.New(),
		Logger:            s.logger,
		EmailSender:       s.emailSender,
		EmailTemplates:    chi_template.NewHTML(testutil.TemplatesDir()),
		SupportInboxEmail: "support@example.com",
	})
	s.access = &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeUser,
		SubjectID: uuid.New(),
		Email:     "user@example.com",
		IPAddress: "127.0.0.1",
	}
}

func (s *SupportServiceSuite) TestSubmit_Validation_MissingMessage() {
	err := s.svc.Submit(s.ctx, &support_service.SubmitRequest{
		Message: "",
	}, s.access)
	s.Error(err)
}

func (s *SupportServiceSuite) TestSubmit_RequiresUserIdentity() {
	err := s.svc.Submit(s.ctx, &support_service.SubmitRequest{
		Message: "help",
	}, &chi_types.AccessInfo{Type: chi_types.AccessTypeAPIKey, SubjectID: uuid.New()})
	s.Error(err)
}

func (s *SupportServiceSuite) TestSubmit_NotConfigured() {
	svc := support_service.New(support_service.Dependencies{
		Validator: chi_validator.New(),
		Logger:    s.logger,
	})
	err := svc.Submit(s.ctx, &support_service.SubmitRequest{Message: "help"}, s.access)
	s.Error(err)
}

func (s *SupportServiceSuite) TestSubmit_Success() {
	s.emailSender.On("Send", s.ctx, mock.Anything).Return(nil).Once()

	err := s.svc.Submit(s.ctx, &support_service.SubmitRequest{
		Subject:   "Billing",
		Message:   "Need help",
		PageURL:   "https://app.example.com/settings",
		UserAgent: "test",
	}, s.access)
	s.NoError(err)
	s.emailSender.AssertExpectations(s.T())
}
