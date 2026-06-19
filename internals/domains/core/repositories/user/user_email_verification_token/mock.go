package user_email_verification_token_repository

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/yca-software/2chi-go-api/internals/domains/core/models"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

type MockUserEmailVerificationTokenRepository struct {
	mock.Mock
}

func (m *MockUserEmailVerificationTokenRepository) WithTx(_ chi_repository.Tx) UserEmailVerificationTokenRepository {
	return m
}

func (m *MockUserEmailVerificationTokenRepository) CreateEmailVerificationToken(ctx context.Context, token *models.UserEmailVerificationToken) error {
	return m.Called(ctx, token).Error(0)
}

func (m *MockUserEmailVerificationTokenRepository) GetEmailVerificationTokenByHash(ctx context.Context, hash string) (*models.UserEmailVerificationToken, error) {
	args := m.Called(ctx, hash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserEmailVerificationToken), args.Error(1)
}

func (m *MockUserEmailVerificationTokenRepository) MarkEmailVerificationTokenAsUsed(ctx context.Context, tokenID string) error {
	return m.Called(ctx, tokenID).Error(0)
}

func (m *MockUserEmailVerificationTokenRepository) CleanupStaleUnusedEmailVerificationTokens(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}
