package auth_service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/yca-software/2chi-go-api/internals/constants"
	"github.com/yca-software/2chi-go-api/internals/models"
	"github.com/yca-software/2chi-go-api/internals/packages/authz"
	"github.com/yca-software/2chi-go-api/internals/repositories"
	impersonation_session_repository "github.com/yca-software/2chi-go-api/internals/repositories/impersonation_session"
	invitation_repository "github.com/yca-software/2chi-go-api/internals/repositories/invitation"
	organization_member_repository "github.com/yca-software/2chi-go-api/internals/repositories/org_member"
	organization_repository "github.com/yca-software/2chi-go-api/internals/repositories/organization"
	admin_access_repository "github.com/yca-software/2chi-go-api/internals/repositories/admin_access"
	user_repository "github.com/yca-software/2chi-go-api/internals/repositories/user"
	user_email_verification_token_repository "github.com/yca-software/2chi-go-api/internals/repositories/user_email_verification_token"
	user_password_reset_token_repository "github.com/yca-software/2chi-go-api/internals/repositories/user_password_reset_token"
	user_refresh_token_repository "github.com/yca-software/2chi-go-api/internals/repositories/user_refresh_token"
	auth_service "github.com/yca-software/2chi-go-api/internals/services/auth"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_password "github.com/yca-software/2chi-go-password"
	chi_repository "github.com/yca-software/2chi-go-repository"
	chi_token "github.com/yca-software/2chi-go-token"
	chi_types "github.com/yca-software/2chi-go-types"
	chi_validator "github.com/yca-software/2chi-go-validator"
)

var testTokenHasher = chi_token.NewHasher("test-pepper")

type AuthServiceSuite struct {
	suite.Suite
	ctx               context.Context
	now               time.Time
	usersRepo         *user_repository.MockUsersRepository
	refreshTokensRepo *user_refresh_token_repository.MockUserRefreshTokenRepository
	passwordResetRepo *user_password_reset_token_repository.MockUserPasswordResetTokenRepository
	emailVerifyRepo   *user_email_verification_token_repository.MockUserEmailVerificationTokenRepository
	impersonationRepo *impersonation_session_repository.MockImpersonationSessionsRepository
	invitationsRepo   *invitation_repository.MockInvitationsRepository
	orgsRepo          *organization_repository.MockOrganizationsRepository
	adminAccessRepo   *admin_access_repository.MockAdminAccessRepository
	membersRepo       *organization_member_repository.MockOrganizationMembersRepository
	logger            *chi_logger.MockLogger
	svc               auth_service.Service
}

func TestAuthServiceSuite(t *testing.T) {
	suite.Run(t, new(AuthServiceSuite))
}

func (s *AuthServiceSuite) SetupTest() {
	s.ctx = context.Background()
	s.now = fixedNow()
	s.usersRepo = &user_repository.MockUsersRepository{}
	s.refreshTokensRepo = &user_refresh_token_repository.MockUserRefreshTokenRepository{}
	s.passwordResetRepo = &user_password_reset_token_repository.MockUserPasswordResetTokenRepository{}
	s.emailVerifyRepo = &user_email_verification_token_repository.MockUserEmailVerificationTokenRepository{}
	s.impersonationRepo = &impersonation_session_repository.MockImpersonationSessionsRepository{}
	s.invitationsRepo = &invitation_repository.MockInvitationsRepository{}
	s.orgsRepo = &organization_repository.MockOrganizationsRepository{}
	s.adminAccessRepo = &admin_access_repository.MockAdminAccessRepository{}
	s.membersRepo = &organization_member_repository.MockOrganizationMembersRepository{}
	s.logger = &chi_logger.MockLogger{}
	configureMockLogger(s.logger)

	s.svc = auth_service.New(auth_service.Dependencies{
		GenerateID:     uuid.NewV7,
		Now:            func() time.Time { return s.now },
		Validator:      chi_validator.New(),
		Logger:         s.logger,
		PasswordHashFn: chi_password.Hash,
		GenerateToken:  chi_token.GenerateOpaqueToken,
		HashToken:      testTokenHasher.Hash,
		Authorizer:     authz.NewAuthorizer(func() time.Time { return s.now }),
		Repositories: &repositories.Repositories{
			Users:                       s.usersRepo,
			UserRefreshTokens:           s.refreshTokensRepo,
			UserPasswordResetTokens:     s.passwordResetRepo,
			UserEmailVerificationTokens: s.emailVerifyRepo,
			ImpersonationSessions:       s.impersonationRepo,
			Invitations:                 s.invitationsRepo,
			Organizations:               s.orgsRepo,
			AdminAccess:                 s.adminAccessRepo,
			OrganizationMembers:         s.membersRepo,
		},
		RunInTx:           inlineRunInTx,
		AccessTokenSecret: "test-secret-key-at-least-32-bytes-long",
		AppURL:            "https://app.example.com",
	})
}

func (s *AuthServiceSuite) TestSignUp_Validation_InvalidEmail() {
	resp, err := s.svc.SignUp(s.ctx, &auth_service.SignUpRequest{
		FirstName:    "Ada",
		LastName:     "Lovelace",
		Email:        "not-an-email",
		Password:     "password123",
		TermsVersion: "1.0.0",
		Language:     "en",
		IPAddress:    "127.0.0.1",
		UserAgent:    "test",
	})
	s.Error(err)
	s.Nil(resp)
}

func (s *AuthServiceSuite) TestSignUp_EmailAlreadyInUse() {
	existingUserID := uuid.New()
	s.usersRepo.On("GetUserByEmail", s.ctx, "taken@example.com").
		Return(&models.User{
			ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
				ModelBase: chi_types.ModelBase{ID: existingUserID},
			},
			Email: "taken@example.com",
		}, nil).Once()

	resp, err := s.svc.SignUp(s.ctx, &auth_service.SignUpRequest{
		FirstName:    "Ada",
		LastName:     "Lovelace",
		Email:        "taken@example.com",
		Password:     "password123",
		TermsVersion: "1.0.0",
		Language:     "en",
		IPAddress:    "127.0.0.1",
		UserAgent:    "test",
	})
	s.Error(err)
	s.Nil(resp)
	if apiErr, ok := chi_error.AsError(err); ok {
		s.Equal("EmailAlreadyInUse", apiErr.ErrorCode)
	}
}

func (s *AuthServiceSuite) TestAuthenticateWithPassword_Validation_MissingEmail() {
	resp, err := s.svc.AuthenticateWithPassword(s.ctx, &auth_service.AuthenticateWithPasswordRequest{
		Email:     "",
		Password:  "password123",
		IPAddress: "127.0.0.1",
		UserAgent: "test",
	})
	s.Error(err)
	s.Nil(resp)
}

func (s *AuthServiceSuite) TestAuthenticateWithGoogle_OAuthNotConfigured() {
	resp, err := s.svc.AuthenticateWithGoogle(s.ctx, &auth_service.AuthenticateWithGoogleRequest{
		Code:         "code",
		TermsVersion: "1.0.0",
		IPAddress:    "127.0.0.1",
		UserAgent:    "test",
		Language:     "en",
	})
	s.Error(err)
	s.Nil(resp)
	if apiErr, ok := chi_error.AsError(err); ok {
		s.Equal("InternalServerError", apiErr.ErrorCode)
	}
}

func (s *AuthServiceSuite) TestForgotPassword_Validation_InvalidEmail() {
	err := s.svc.ForgotPassword(s.ctx, &auth_service.ForgotPasswordRequest{
		Email:    "bad",
		Language: "en",
	})
	s.Error(err)
}

func (s *AuthServiceSuite) TestForgotPassword_UserNotFound_ReturnsNil() {
	s.usersRepo.On("GetUserByEmail", s.ctx, "missing@example.com").
		Return(nil, chi_error.NewNotFoundError(nil, "NotFound", nil)).Once()

	err := s.svc.ForgotPassword(s.ctx, &auth_service.ForgotPasswordRequest{
		Email:    "missing@example.com",
		Language: "en",
	})
	s.NoError(err)
}

func (s *AuthServiceSuite) TestAuthenticateWithPassword_UserNotFound() {
	s.usersRepo.On("GetUserByEmail", s.ctx, "missing@example.com").
		Return(nil, chi_error.NewNotFoundError(nil, "NotFound", nil)).Once()

	resp, err := s.svc.AuthenticateWithPassword(s.ctx, &auth_service.AuthenticateWithPasswordRequest{
		Email:     "missing@example.com",
		Password:  "password123",
		IPAddress: "127.0.0.1",
		UserAgent: "test",
	})
	s.Error(err)
	s.Nil(resp)
	if apiErr, ok := chi_error.AsError(err); ok {
		s.Equal("PasswordMismatch", apiErr.ErrorCode)
	}
}

func (s *AuthServiceSuite) TestLogout_Validation() {
	err := s.svc.Logout(s.ctx, &auth_service.LogoutRequest{RefreshToken: ""}, &chi_types.AccessInfo{
		Type: chi_types.AccessTypeUser, SubjectID: uuid.New(),
	})
	s.Error(err)
}

func (s *AuthServiceSuite) TestLogout_InvalidRefreshToken() {
	s.refreshTokensRepo.On("GetRefreshTokenByHash", s.ctx, mock.Anything).
		Return(nil, chi_error.NewNotFoundError(nil, "NotFound", nil)).Once()

	userID := uuid.New()
	err := s.svc.Logout(s.ctx, &auth_service.LogoutRequest{RefreshToken: "rt"}, &chi_types.AccessInfo{
		Type: chi_types.AccessTypeUser, SubjectID: userID,
	})
	s.Error(err)
	if apiErr, ok := chi_error.AsError(err); ok {
		s.Equal("InvalidToken", apiErr.ErrorCode)
	}
}

func (s *AuthServiceSuite) TestRefreshAccessToken_Validation() {
	resp, err := s.svc.RefreshAccessToken(s.ctx, &auth_service.RefreshAccessTokenRequest{
		RefreshToken: "",
		IPAddress:    "127.0.0.1",
		UserAgent:    "test",
	})
	s.Error(err)
	s.Nil(resp)
}

func (s *AuthServiceSuite) TestRefreshAccessToken_InvalidToken() {
	s.refreshTokensRepo.On("GetRefreshTokenByHash", s.ctx, mock.Anything).
		Return(nil, chi_error.NewNotFoundError(nil, "NotFound", nil)).Once()

	resp, err := s.svc.RefreshAccessToken(s.ctx, &auth_service.RefreshAccessTokenRequest{
		RefreshToken: "rt",
		IPAddress:    "127.0.0.1",
		UserAgent:    "test",
	})
	s.Error(err)
	s.Nil(resp)
}

func (s *AuthServiceSuite) TestResetPassword_Validation() {
	err := s.svc.ResetPassword(s.ctx, &auth_service.ResetPasswordRequest{Token: "", Password: "short"})
	s.Error(err)
}

func (s *AuthServiceSuite) TestResetPassword_InvalidToken() {
	s.passwordResetRepo.On("GetPasswordResetTokenByHash", s.ctx, mock.Anything).
		Return(nil, chi_error.NewNotFoundError(nil, "NotFound", nil)).Once()

	err := s.svc.ResetPassword(s.ctx, &auth_service.ResetPasswordRequest{
		Token:    "token",
		Password: "newpassword123",
	})
	s.Error(err)
	if apiErr, ok := chi_error.AsError(err); ok {
		s.Equal("InvalidPasswordResetToken", apiErr.ErrorCode)
	}
}

func (s *AuthServiceSuite) TestVerifyEmail_Validation() {
	err := s.svc.VerifyEmail(s.ctx, &auth_service.VerifyEmailRequest{Token: ""})
	s.Error(err)
}

func (s *AuthServiceSuite) TestVerifyEmail_InvalidToken() {
	s.emailVerifyRepo.On("GetEmailVerificationTokenByHash", s.ctx, mock.Anything).
		Return(nil, chi_error.NewNotFoundError(nil, "NotFound", nil)).Once()

	err := s.svc.VerifyEmail(s.ctx, &auth_service.VerifyEmailRequest{Token: "token"})
	s.Error(err)
	if apiErr, ok := chi_error.AsError(err); ok {
		s.Equal("InvalidVerificationToken", apiErr.ErrorCode)
	}
}

func (s *AuthServiceSuite) TestImpersonate_RequiresAdmin() {
	resp, err := s.svc.Impersonate(s.ctx, &auth_service.ImpersonateRequest{
		UserID:    uuid.New().String(),
		IPAddress: "127.0.0.1",
		UserAgent: "test",
	}, &chi_types.AccessInfo{Type: chi_types.AccessTypeUser, SubjectID: uuid.New()})
	s.Error(err)
	s.Nil(resp)
}

func (s *AuthServiceSuite) TestImpersonate_Validation() {
	admin := &chi_types.AccessInfo{Type: chi_types.AccessTypeUser, SubjectID: uuid.New(), IsAdmin: true}
	resp, err := s.svc.Impersonate(s.ctx, &auth_service.ImpersonateRequest{
		UserID:    "not-a-uuid",
		IPAddress: "127.0.0.1",
		UserAgent: "test",
	}, admin)
	s.Error(err)
	s.Nil(resp)
}

func (s *AuthServiceSuite) TestImpersonate_CreatesSession() {
	adminID := uuid.New()
	targetID := uuid.New()
	admin := &chi_types.AccessInfo{
		Type: chi_types.AccessTypeUser, SubjectID: adminID, Email: "admin@example.com", IsAdmin: true,
	}
	s.usersRepo.On("GetUserByID", s.ctx, targetID.String()).
		Return(&models.User{
			ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
				ModelBase: chi_types.ModelBase{ID: targetID},
			},
			Email: "target@example.com",
		}, nil).Once()
	s.mockAuthSessionLoad(targetID, "target@example.com")
	s.refreshTokensRepo.On("CreateRefreshToken", s.ctx, mock.AnythingOfType("*models.UserRefreshToken")).Return(nil).Once()
	s.impersonationRepo.On("CreateSession", s.ctx, mock.MatchedBy(func(session *models.ImpersonationSession) bool {
		return session.AdminID == adminID &&
			session.TargetUserID == targetID &&
			session.AdminEmail == "admin@example.com" &&
			session.TargetUserEmail == "target@example.com"
	})).Return(nil).Once()

	resp, err := s.svc.Impersonate(s.ctx, &auth_service.ImpersonateRequest{
		UserID:    targetID.String(),
		IPAddress: "127.0.0.1",
		UserAgent: "test",
		RequestID: "req-123",
	}, admin)
	s.Require().NoError(err)
	s.NotEmpty(resp.AccessToken)
	s.NotEmpty(resp.RefreshToken)
}

func (s *AuthServiceSuite) TestVerifyEmail_Success() {
	userID := uuid.New()
	tokenID := uuid.New()
	s.emailVerifyRepo.On("GetEmailVerificationTokenByHash", s.ctx, mock.Anything).
		Return(&models.UserEmailVerificationToken{
			ModelBase: chi_types.ModelBase{ID: tokenID},
			UserID:    userID,
			ExpiresAt: s.now.Add(time.Hour),
		}, nil).Once()
	s.usersRepo.On("GetUserByID", s.ctx, userID.String()).
		Return(&models.User{
			ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
				ModelBase: chi_types.ModelBase{ID: userID},
			},
		}, nil).Once()
	s.emailVerifyRepo.On("MarkEmailVerificationTokenAsUsed", s.ctx, tokenID.String()).Return(nil).Once()

	err := s.svc.VerifyEmail(s.ctx, &auth_service.VerifyEmailRequest{Token: "verify-token"})
	s.NoError(err)
}

func (s *AuthServiceSuite) TestResetPassword_Success() {
	userID := uuid.New()
	tokenID := uuid.New()
	s.passwordResetRepo.On("GetPasswordResetTokenByHash", s.ctx, mock.Anything).
		Return(&models.UserPasswordResetToken{
			ModelBase: chi_types.ModelBase{ID: tokenID},
			UserID:    userID,
			ExpiresAt: s.now.Add(time.Hour),
		}, nil).Once()
	s.usersRepo.On("GetUserByID", s.ctx, userID.String()).
		Return(&models.User{
			ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
				ModelBase: chi_types.ModelBase{ID: userID},
			},
		}, nil).Once()
	s.passwordResetRepo.On("MarkPasswordResetTokenAsUsed", s.ctx, tokenID.String()).Return(nil).Once()
	s.usersRepo.On("UpdateUser", s.ctx, mock.AnythingOfType("*models.User")).Return(nil).Once()

	err := s.svc.ResetPassword(s.ctx, &auth_service.ResetPasswordRequest{
		Token:    "reset-token",
		Password: "newpassword123",
	})
	s.NoError(err)
}

func (s *AuthServiceSuite) TestLogout_Success() {
	userID := uuid.New()
	refreshPlain := "refresh-token-plain"
	tokenHash := testTokenHasher.Hash(refreshPlain)
	s.refreshTokensRepo.On("GetRefreshTokenByHash", s.ctx, tokenHash).
		Return(&models.UserRefreshToken{
			UserID: userID, ExpiresAt: s.now.Add(time.Hour),
			TokenHash: tokenHash,
		}, nil).Once()
	s.refreshTokensRepo.On("RevokeRefreshTokenByHash", s.ctx, tokenHash).Return(nil).Once()

	err := s.svc.Logout(s.ctx, &auth_service.LogoutRequest{RefreshToken: refreshPlain}, &chi_types.AccessInfo{
		Type: chi_types.AccessTypeUser, SubjectID: userID,
	})
	s.NoError(err)
}

func (s *AuthServiceSuite) TestLogout_ClosesImpersonationSession() {
	userID := uuid.New()
	tokenID := uuid.New()
	adminID := uuid.New()
	refreshPlain := "refresh-token-plain"
	tokenHash := testTokenHasher.Hash(refreshPlain)
	s.refreshTokensRepo.On("GetRefreshTokenByHash", s.ctx, tokenHash).
		Return(&models.UserRefreshToken{
			ModelBase:      chi_types.ModelBase{ID: tokenID},
			UserID:         userID,
			ExpiresAt:      s.now.Add(time.Hour),
			ImpersonatedBy: uuid.NullUUID{UUID: adminID, Valid: true},
			TokenHash:      tokenHash,
		}, nil).Once()
	s.refreshTokensRepo.On("RevokeRefreshTokenByHash", s.ctx, tokenHash).Return(nil).Once()
	s.impersonationRepo.On(
		"EndSessionByRefreshTokenID", s.ctx, tokenID, s.now, constants.IMPERSONATION_END_REASON_LOGOUT,
	).Return(nil).Once()

	err := s.svc.Logout(s.ctx, &auth_service.LogoutRequest{RefreshToken: refreshPlain}, &chi_types.AccessInfo{
		Type: chi_types.AccessTypeUser, SubjectID: userID,
	})
	s.NoError(err)
}

func (s *AuthServiceSuite) mockAuthSessionLoad(userID uuid.UUID, email string) {
	emptyRoles := []models.OrganizationMemberWithOrganizationAndRole{}
	s.usersRepo.On("GetUserByID", s.ctx, userID.String()).Return(&models.User{
		ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
			ModelBase: chi_types.ModelBase{ID: userID},
		},
		Email: email,
	}, nil)
	s.membersRepo.On("ListByUserIDWithRole", s.ctx, userID.String()).Return(&emptyRoles, nil)
	s.adminAccessRepo.On("GetAdminAccessByUserID", s.ctx, userID.String()).
		Return(nil, chi_error.NewNotFoundError(nil, "NotFound", nil))
	s.refreshTokensRepo.On("GetActiveImpersonationRefreshTokenByUserID", s.ctx, userID.String()).
		Return(nil, chi_error.NewNotFoundError(nil, "NotFound", nil))
}

func (s *AuthServiceSuite) TestRefreshAccessToken_Success() {
	userID := uuid.New()
	refreshPlain := "refresh-token-plain"
	s.refreshTokensRepo.On("GetRefreshTokenByHash", s.ctx, testTokenHasher.Hash(refreshPlain)).
		Return(&models.UserRefreshToken{
			UserID: userID, ExpiresAt: s.now.Add(time.Hour),
		}, nil).Once()
	s.mockAuthSessionLoad(userID, "user@example.com")

	resp, err := s.svc.RefreshAccessToken(s.ctx, &auth_service.RefreshAccessTokenRequest{
		RefreshToken: refreshPlain,
		IPAddress:    "127.0.0.1",
		UserAgent:    "test",
	})
	s.Require().NoError(err)
	s.NotEmpty(resp.AccessToken)
}

func (s *AuthServiceSuite) TestAuthenticateWithPassword_Success() {
	userID := uuid.New()
	hashed, hashErr := chi_password.Hash("password123")
	s.Require().NoError(hashErr)
	s.usersRepo.On("GetUserByEmail", s.ctx, "user@example.com").
		Return(&models.User{
			ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
				ModelBase: chi_types.ModelBase{ID: userID},
			},
			Email:    "user@example.com",
			Password: hashed,
		}, nil).Once()
	s.mockAuthSessionLoad(userID, "user@example.com")
	s.refreshTokensRepo.On("CreateRefreshToken", s.ctx, mock.AnythingOfType("*models.UserRefreshToken")).Return(nil).Once()

	resp, err := s.svc.AuthenticateWithPassword(s.ctx, &auth_service.AuthenticateWithPasswordRequest{
		Email:     "user@example.com",
		Password:  "password123",
		IPAddress: "127.0.0.1",
		UserAgent: "test",
	})
	s.Require().NoError(err)
	s.NotEmpty(resp.AccessToken)
	s.NotEmpty(resp.RefreshToken)
}

func fixedNow() time.Time {
	return time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
}

func inlineRunInTx(_ context.Context, fn func(chi_repository.Tx) error) error {
	return fn(nil)
}

func configureMockLogger(m *chi_logger.MockLogger) {
	for n := 0; n <= 8; n++ {
		args := make([]any, n)
		for i := range args {
			args[i] = mock.Anything
		}
		if n == 0 {
			m.On("With").Return(m).Maybe()
			continue
		}
		m.On("With", args...).Return(m).Maybe()
	}
	m.On("WithContext", mock.Anything).Return(m).Maybe()
	for _, method := range []string{"Debug", "Info", "Warn", "Error"} {
		for n := 0; n <= 8; n++ {
			args := make([]any, n+1)
			for i := range args {
				args[i] = mock.Anything
			}
			m.On(method, args...).Return().Maybe()
		}
	}
}
