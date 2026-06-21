package user_password_reset_token_repository

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/yca-software/2chi-go-api/internals/models"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

type MockUserPasswordResetTokenRepository struct {
	mock.Mock
}

func (m *MockUserPasswordResetTokenRepository) WithTx(_ chi_repository.Tx) UserPasswordResetTokenRepository {
	return m
}

func (m *MockUserPasswordResetTokenRepository) CreatePasswordResetToken(ctx context.Context, token *models.UserPasswordResetToken) error {
	return m.Called(ctx, token).Error(0)
}

func (m *MockUserPasswordResetTokenRepository) GetPasswordResetTokenByHash(ctx context.Context, hash string) (*models.UserPasswordResetToken, error) {
	args := m.Called(ctx, hash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserPasswordResetToken), args.Error(1)
}

func (m *MockUserPasswordResetTokenRepository) MarkPasswordResetTokenAsUsed(ctx context.Context, tokenID string) error {
	return m.Called(ctx, tokenID).Error(0)
}

func (m *MockUserPasswordResetTokenRepository) CleanupStaleUnusedPasswordResetTokens(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}
