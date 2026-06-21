package user_refresh_token_repository

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/yca-software/2chi-go-api/internals/models"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

type MockUserRefreshTokenRepository struct {
	mock.Mock
}

func (m *MockUserRefreshTokenRepository) WithTx(_ chi_repository.Tx) UserRefreshTokenRepository {
	return m
}

func (m *MockUserRefreshTokenRepository) CreateRefreshToken(ctx context.Context, token *models.UserRefreshToken) error {
	return m.Called(ctx, token).Error(0)
}

func (m *MockUserRefreshTokenRepository) GetRefreshTokenByHash(ctx context.Context, hash string) (*models.UserRefreshToken, error) {
	args := m.Called(ctx, hash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserRefreshToken), args.Error(1)
}

func (m *MockUserRefreshTokenRepository) GetActiveRefreshTokensByUserID(ctx context.Context, userID string) (*[]models.UserRefreshToken, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.UserRefreshToken), args.Error(1)
}

func (m *MockUserRefreshTokenRepository) GetActiveImpersonationRefreshTokenByUserID(ctx context.Context, userID string) (*models.UserRefreshToken, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserRefreshToken), args.Error(1)
}

func (m *MockUserRefreshTokenRepository) CleanupStaleUnusedRefreshTokens(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}

func (m *MockUserRefreshTokenRepository) RevokeRefreshTokenByID(ctx context.Context, userID, tokenID string) error {
	return m.Called(ctx, userID, tokenID).Error(0)
}

func (m *MockUserRefreshTokenRepository) RevokeRefreshTokenByHash(ctx context.Context, hash string) error {
	return m.Called(ctx, hash).Error(0)
}

func (m *MockUserRefreshTokenRepository) RevokeAllRefreshTokensByUserID(ctx context.Context, userID string, excludeTokenID *string) error {
	return m.Called(ctx, userID, excludeTokenID).Error(0)
}
