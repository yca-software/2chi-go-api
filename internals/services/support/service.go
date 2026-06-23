package support_service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	chi_aws_ses "github.com/yca-software/2chi-go-aws/ses"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_template "github.com/yca-software/2chi-go-template"
	chi_types "github.com/yca-software/2chi-go-types"
	chi_validator "github.com/yca-software/2chi-go-validator"
)

type Dependencies struct {
	Validator         chi_validator.Validator
	Logger            chi_logger.Logger
	EmailSender       chi_aws_ses.SES
	EmailTemplates    *chi_template.HTML
	SupportInboxEmail string
}

type Service interface {
	Submit(ctx context.Context, req *SubmitRequest, access *chi_types.AccessInfo) error
}

type service struct {
	validator         chi_validator.Validator
	logger            chi_logger.Logger
	emailSender       chi_aws_ses.SES
	emailTemplates    *chi_template.HTML
	supportInboxEmail string
}

func New(deps Dependencies) Service {
	return &service{
		validator:         deps.Validator,
		logger:            deps.Logger,
		emailSender:       deps.EmailSender,
		emailTemplates:    deps.EmailTemplates,
		supportInboxEmail: deps.SupportInboxEmail,
	}
}

func (s *service) Submit(ctx context.Context, req *SubmitRequest, access *chi_types.AccessInfo) error {
	if err := s.validator.ValidateStruct(req); err != nil {
		return chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}
	if access == nil || access.Type != chi_types.AccessTypeUser {
		return chi_error.NewForbiddenError(errors.New("user identity required"), "UserIdentityRequired", nil)
	}
	if s.supportInboxEmail == "" || s.emailSender == nil || s.emailTemplates == nil {
		return chi_error.NewInternalServerError(errors.New("support inbox not configured"), "InternalServerError", nil)
	}

	subject := strings.TrimSpace(req.Subject)
	if subject == "" {
		subject = fmt.Sprintf("Support request from %s", access.Email)
	} else {
		subject = "[Support] " + subject
	}

	body, err := s.emailTemplates.Render("support", map[string]any{
		"FromEmail": access.Email,
		"UserID":    access.SubjectID.String(),
		"Subject":   strings.TrimSpace(req.Subject),
		"Message":   req.Message,
		"PageURL":   req.PageURL,
		"UserAgent": req.UserAgent,
		"RequestIP": access.IPAddress,
	})
	if err != nil {
		return err
	}

	return s.emailSender.Send(ctx, chi_aws_ses.SESEmailDataPayload{
		To:      s.supportInboxEmail,
		Subject: subject,
		HTML:    body,
	})
}
