package user_service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/yca-software/2chi-go-api/internals/models"
	"github.com/yca-software/2chi-go-api/internals/repositories"
	organization_member_repository "github.com/yca-software/2chi-go-api/internals/repositories/org_member"
	admin_access_repository "github.com/yca-software/2chi-go-api/internals/repositories/admin_access"
	user_repository "github.com/yca-software/2chi-go-api/internals/repositories/user"
	user_email_verification_token_repository "github.com/yca-software/2chi-go-api/internals/repositories/user_email_verification_token"
	user_legal_document_acceptance_repository "github.com/yca-software/2chi-go-api/internals/repositories/user_legal_document_acceptance"
	user_password_reset_token_repository "github.com/yca-software/2chi-go-api/internals/repositories/user_password_reset_token"
	user_refresh_token_repository "github.com/yca-software/2chi-go-api/internals/repositories/user_refresh_token"
	user_service "github.com/yca-software/2chi-go-api/internals/services/user"
	"github.com/yca-software/2chi-go-api/internals/packages/authz"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_password "github.com/yca-software/2chi-go-password"
	chi_token "github.com/yca-software/2chi-go-token"
	chi_types "github.com/yca-software/2chi-go-types"
	chi_validator "github.com/yca-software/2chi-go-validator"
)

var testTokenHasher = chi_token.NewHasher("test-pepper")

type UserServiceSuite struct {
	suite.Suite
	ctx               context.Context
	now               time.Time
	userID            uuid.UUID
	usersRepo         *user_repository.MockUsersRepository
	refreshTokensRepo *user_refresh_token_repository.MockUserRefreshTokenRepository
	passwordResetRepo *user_password_reset_token_repository.MockUserPasswordResetTokenRepository
	emailVerifyRepo   *user_email_verification_token_repository.MockUserEmailVerificationTokenRepository
	legalAcceptRepo   *user_legal_document_acceptance_repository.MockUserLegalDocumentAcceptanceRepository
	adminAccessRepo   *admin_access_repository.MockAdminAccessRepository
	membersRepo       *organization_member_repository.MockOrganizationMembersRepository
	sessionCache      *authz.SessionCache
	svc               user_service.Service
}

func TestUserServiceSuite(t *testing.T) {
	suite.Run(t, new(UserServiceSuite))
}

func (s *UserServiceSuite) SetupTest() {
	s.ctx = context.Background()
	s.now = time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	s.userID = uuid.MustParse("018f1234-5678-7abc-8def-012345678901")
	s.usersRepo = &user_repository.MockUsersRepository{}
	s.refreshTokensRepo = &user_refresh_token_repository.MockUserRefreshTokenRepository{}
	s.passwordResetRepo = &user_password_reset_token_repository.MockUserPasswordResetTokenRepository{}
	s.emailVerifyRepo = &user_email_verification_token_repository.MockUserEmailVerificationTokenRepository{}
	s.legalAcceptRepo = &user_legal_document_acceptance_repository.MockUserLegalDocumentAcceptanceRepository{}
	s.adminAccessRepo = &admin_access_repository.MockAdminAccessRepository{}
	s.membersRepo = &organization_member_repository.MockOrganizationMembersRepository{}
	s.sessionCache = authz.NewTestSessionCache(s.T(), time.Hour)

	s.svc = user_service.New(user_service.Dependencies{
		GenerateID:     uuid.NewV7,
		Now:            func() time.Time { return s.now },
		Validator:      chi_validator.New(),
		Logger:         mockLogger(),
		PasswordHashFn: chi_password.Hash,
		HashToken:      testTokenHasher.Hash,
		Authorizer:     authz.NewAuthorizer(func() time.Time { return s.now }),
		Repositories: &repositories.Repositories{
			Users:                        s.usersRepo,
			UserRefreshTokens:            s.refreshTokensRepo,
			UserPasswordResetTokens:      s.passwordResetRepo,
			UserEmailVerificationTokens:  s.emailVerifyRepo,
			UserLegalDocumentAcceptances: s.legalAcceptRepo,
			AdminAccess:                  s.adminAccessRepo,
			OrganizationMembers:          s.membersRepo,
		},
		SessionCache: s.sessionCache,
		AppURL:       "https://app.example.com",
	})
}

func (s *UserServiceSuite) ownAccess() *chi_types.AccessInfo {
	return &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeUser,
		SubjectID: s.userID,
		Email:     "user@example.com",
	}
}

func (s *UserServiceSuite) adminAccess() *chi_types.AccessInfo {
	return &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeUser,
		SubjectID: uuid.New(),
		IsAdmin:   true,
		Email:     "admin@example.com",
	}
}

func (s *UserServiceSuite) TestAcceptTerms_ForbiddenForOtherUser() {
	_, err := s.svc.AcceptTerms(s.ctx, &user_service.AcceptTermsRequest{
		UserID:       uuid.New().String(),
		TermsVersion: "1.0.0",
	}, s.ownAccess())
	s.Error(err)
}

func (s *UserServiceSuite) TestAcceptTerms_Success() {
	s.usersRepo.On("GetUserByID", s.ctx, s.userID.String()).Return(&models.User{
		ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
			ModelBase: chi_types.ModelBase{ID: s.userID},
		},
		Email: "user@example.com",
	}, nil).Once()
	s.legalAcceptRepo.On("CreateUserLegalDocumentAcceptance", s.ctx, mock.MatchedBy(func(a *models.UserLegalDocumentAcceptance) bool {
		return a.DocumentVersion == "1.0.0"
	})).Return(nil).Once()

	updated, err := s.svc.AcceptTerms(s.ctx, &user_service.AcceptTermsRequest{
		UserID:       s.userID.String(),
		TermsVersion: "1.0.0",
	}, s.ownAccess())
	s.Require().NoError(err)
	s.Equal(s.userID, updated.ID)
}

func (s *UserServiceSuite) TestChangePassword_OldPasswordMismatch() {
	hash, err := chi_password.Hash("correct-old")
	s.Require().NoError(err)
	s.usersRepo.On("GetUserByID", s.ctx, s.userID.String()).Return(&models.User{
		ModelBaseWithArchive: chi_types.ModelBaseWithArchive{ModelBase: chi_types.ModelBase{ID: s.userID}},
		Password:             hash,
	}, nil).Once()

	err = s.svc.ChangePassword(s.ctx, &user_service.ChangePasswordRequest{
		UserID:      s.userID.String(),
		OldPassword: "wrong-old",
		NewPassword: "newpassword123",
	}, s.ownAccess())
	s.Error(err)
	if apiErr, ok := chi_error.AsError(err); ok {
		s.Equal("OldPasswordMismatch", apiErr.ErrorCode)
	}
}

func (s *UserServiceSuite) TestUpdateProfile_Success() {
	s.usersRepo.On("GetUserByID", s.ctx, s.userID.String()).Return(&models.User{
		ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
			ModelBase: chi_types.ModelBase{ID: s.userID},
		},
		FirstName: "Ada",
		LastName:  "Lovelace",
	}, nil).Once()
	s.usersRepo.On("UpdateUser", s.ctx, mock.MatchedBy(func(u *models.User) bool {
		return u.FirstName == "Grace" && u.LastName == "Hopper"
	})).Return(nil).Once()

	updated, err := s.svc.UpdateProfile(s.ctx, &user_service.UpdateProfileRequest{
		UserID:    s.userID.String(),
		FirstName: "Grace",
		LastName:  "Hopper",
	}, s.ownAccess())
	s.Require().NoError(err)
	s.Equal("Grace", updated.FirstName)
}

func (s *UserServiceSuite) TestGetUser_AdminCanReadAnyUser() {
	targetID := uuid.New()
	s.usersRepo.On("GetUserByID", s.ctx, targetID.String()).Return(&models.User{
		ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
			ModelBase: chi_types.ModelBase{ID: targetID},
		},
		Email: "other@example.com",
	}, nil).Once()
	s.membersRepo.On("ListByUserIDWithRole", s.ctx, targetID.String()).
		Return(&[]models.OrganizationMemberWithOrganizationAndRole{}, nil).Once()
	s.adminAccessRepo.On("GetAdminAccessByUserID", s.ctx, targetID.String()).
		Return(nil, chi_error.NewNotFoundError(nil, "NotFound", nil)).Once()

	resp, err := s.svc.GetUser(s.ctx, &user_service.GetUserRequest{UserID: targetID.String()}, s.adminAccess())
	s.Require().NoError(err)
	s.Equal(targetID, resp.User.ID)
}

func (s *UserServiceSuite) TestListUsers_RequiresAdmin() {
	_, err := s.svc.ListUsers(s.ctx, &user_service.ListUsersRequest{Limit: 20}, s.ownAccess())
	s.Error(err)
}

func (s *UserServiceSuite) TestCleanupArchivedUsers() {
	s.usersRepo.On("CleanupArchivedUsers", s.ctx).Return(nil).Once()
	s.NoError(s.svc.CleanupArchivedUsers(s.ctx))
}

func (s *UserServiceSuite) TestCleanupStaleUnusedUserTokens() {
	s.refreshTokensRepo.On("CleanupStaleUnusedRefreshTokens", mock.Anything).Return(nil).Once()
	s.passwordResetRepo.On("CleanupStaleUnusedPasswordResetTokens", mock.Anything).Return(nil).Once()
	s.emailVerifyRepo.On("CleanupStaleUnusedEmailVerificationTokens", mock.Anything).Return(nil).Once()
	s.NoError(s.svc.CleanupStaleUnusedUserTokens(s.ctx))
}

func (s *UserServiceSuite) TestUpdateLanguage_Success() {
	s.usersRepo.On("GetUserByID", s.ctx, s.userID.String()).Return(&models.User{
		ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
			ModelBase: chi_types.ModelBase{ID: s.userID},
		},
		Language: "en",
	}, nil).Once()
	s.usersRepo.On("UpdateUser", s.ctx, mock.MatchedBy(func(u *models.User) bool {
		return u.Language == "no"
	})).Return(nil).Once()

	updated, err := s.svc.UpdateLanguage(s.ctx, &user_service.UpdateLanguageRequest{
		UserID: s.userID.String(), Language: "no",
	}, s.ownAccess())
	s.Require().NoError(err)
	s.Equal("no", updated.Language)
}

func (s *UserServiceSuite) TestArchiveUser_Success() {
	user := &models.User{
		ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
			ModelBase: chi_types.ModelBase{ID: s.userID},
		},
	}
	s.usersRepo.On("GetUserByID", s.ctx, s.userID.String()).Return(user, nil).Once()
	s.usersRepo.On("ArchiveUser", s.ctx, user).Return(nil).Once()

	err := s.svc.ArchiveUser(s.ctx, &user_service.ArchiveUserRequest{UserID: s.userID.String()}, s.ownAccess())
	s.NoError(err)
}

func (s *UserServiceSuite) TestRestoreUser_Success() {
	s.usersRepo.On("GetUserByIDIncludeArchived", s.ctx, s.userID.String()).Return(&models.User{
		ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
			ModelBase: chi_types.ModelBase{ID: s.userID},
		},
	}, nil).Once()
	s.usersRepo.On("RestoreUser", s.ctx, s.userID.String()).Return(nil).Once()

	restored, err := s.svc.RestoreUser(s.ctx, &user_service.RestoreUserRequest{UserID: s.userID.String()}, s.ownAccess())
	s.Require().NoError(err)
	s.Equal(s.userID, restored.ID)
}

func (s *UserServiceSuite) TestRevokeUserRefreshToken_Success() {
	tokenID := uuid.New()
	s.refreshTokensRepo.On("RevokeRefreshTokenByID", s.ctx, s.userID.String(), tokenID.String()).Return(nil).Once()
	s.Require().NoError(s.sessionCache.Set(s.ctx, s.ownAccess()))

	err := s.svc.RevokeUserRefreshToken(s.ctx, &user_service.RevokeUserRefreshTokenRequest{
		UserID: s.userID.String(), RefreshTokenID: tokenID.String(),
	}, s.ownAccess())
	s.NoError(err)

	_, ok := s.sessionCache.Get(s.ctx, s.userID.String())
	s.False(ok)
}

func (s *UserServiceSuite) TestRevokeUserAdminAccess_InvalidatesSession() {
	targetID := uuid.MustParse("018f1234-5678-7abc-8def-012345678902")
	adminAccess := &models.AdminAccess{UserID: targetID, CreatedAt: s.now}
	cachedAccess := &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeUser,
		SubjectID: targetID,
		Email:     "admin-target@example.com",
		IsAdmin:   true,
	}
	s.Require().NoError(s.sessionCache.Set(s.ctx, cachedAccess))
	s.adminAccessRepo.On("GetAdminAccessByUserID", s.ctx, targetID.String()).Return(adminAccess, nil).Once()
	s.adminAccessRepo.On("DeleteAdminAccessByUserID", s.ctx, targetID.String()).Return(nil).Once()

	err := s.svc.RevokeUserAdminAccess(s.ctx, &user_service.RevokeUserAdminAccessRequest{
		UserID: targetID.String(),
	}, s.adminAccess())
	s.NoError(err)

	_, ok := s.sessionCache.Get(s.ctx, targetID.String())
	s.False(ok)
}

func (s *UserServiceSuite) TestRevokeUserAdminAccess_ForbidsSelfDemotion() {
	selfAdmin := &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeUser,
		SubjectID: s.userID,
		IsAdmin:   true,
		Email:     "admin@example.com",
	}
	err := s.svc.RevokeUserAdminAccess(s.ctx, &user_service.RevokeUserAdminAccessRequest{
		UserID: s.userID.String(),
	}, selfAdmin)
	s.Error(err)
	apiErr, ok := chi_error.AsError(err)
	s.Require().True(ok)
	s.Equal("CannotRevokeOwnAdminAccess", apiErr.ErrorCode)
}

func (s *UserServiceSuite) TestRevokeUserAllRefreshTokens_Success() {
	s.refreshTokensRepo.On("RevokeAllRefreshTokensByUserID", s.ctx, s.userID.String(), (*string)(nil)).Return(nil).Once()

	err := s.svc.RevokeUserAllRefreshTokens(s.ctx, &user_service.RevokeUserAllRefreshTokensRequest{
		UserID: s.userID.String(),
	}, s.ownAccess())
	s.NoError(err)
}

func (s *UserServiceSuite) TestListUsers_Success() {
	listUserID := uuid.New()
	users := []models.User{{
		ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
			ModelBase: chi_types.ModelBase{ID: listUserID},
		},
		Email: "a@example.com",
	}}
	s.usersRepo.On("SearchUsers", s.ctx, "", mock.Anything, 21, 0).Return(&users, nil).Once()

	resp, err := s.svc.ListUsers(s.ctx, &user_service.ListUsersRequest{Limit: 20, Offset: 0}, s.adminAccess())
	s.Require().NoError(err)
	s.Len(resp.Items, 1)
}

func (s *UserServiceSuite) TestListUserActiveRefreshTokens_Success() {
	tokens := []models.UserRefreshToken{{
		ModelBase: chi_types.ModelBase{ID: uuid.New()},
		UserID:    s.userID,
	}}
	s.refreshTokensRepo.On("GetActiveRefreshTokensByUserID", s.ctx, s.userID.String()).Return(&tokens, nil).Once()

	result, err := s.svc.ListUserActiveRefreshTokens(s.ctx, &user_service.ListUserActiveRefreshTokensRequest{
		UserID: s.userID.String(),
	}, s.ownAccess())
	s.Require().NoError(err)
	s.Len(*result, 1)
}

func (s *UserServiceSuite) TestResendVerificationEmail_AlreadyVerified() {
	verifiedAt := s.now
	s.usersRepo.On("GetUserByID", s.ctx, s.userID.String()).Return(&models.User{
		ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
			ModelBase: chi_types.ModelBase{ID: s.userID},
		},
		EmailVerifiedAt: &verifiedAt,
	}, nil).Once()

	err := s.svc.ResendVerificationEmail(s.ctx, &user_service.ResendVerificationEmailRequest{
		UserID: s.userID.String(), Language: "en",
	}, s.ownAccess())
	s.Error(err)
	if apiErr, ok := chi_error.AsError(err); ok {
		s.Equal("EmailAlreadyVerified", apiErr.ErrorCode)
	}
}

func (s *UserServiceSuite) TestGetUser_OwnUser() {
	s.usersRepo.On("GetUserByID", s.ctx, s.userID.String()).Return(&models.User{
		ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
			ModelBase: chi_types.ModelBase{ID: s.userID},
		},
	}, nil).Once()
	s.membersRepo.On("ListByUserIDWithRole", s.ctx, s.userID.String()).Return(&[]models.OrganizationMemberWithOrganizationAndRole{}, nil).Once()
	s.adminAccessRepo.On("GetAdminAccessByUserID", s.ctx, s.userID.String()).
		Return(nil, chi_error.NewNotFoundError(nil, "NotFound", nil)).Once()

	resp, err := s.svc.GetUser(s.ctx, &user_service.GetUserRequest{UserID: s.userID.String()}, s.ownAccess())
	s.Require().NoError(err)
	s.Equal(s.userID, resp.User.ID)
}

func mockLogger() chi_logger.Logger {
	m := new(chi_logger.MockLogger)
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
	return m
}
