package impersonation_session_repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/yca-software/2chi-go-api/internals/models"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

type MockImpersonationSessionsRepository struct {
	mock.Mock
}

func (m *MockImpersonationSessionsRepository) WithTx(_ chi_repository.Tx) ImpersonationSessionsRepository {
	return m
}

func (m *MockImpersonationSessionsRepository) CreateSession(ctx context.Context, session *models.ImpersonationSession) error {
	return m.Called(ctx, session).Error(0)
}

func (m *MockImpersonationSessionsRepository) EndSessionByRefreshTokenID(ctx context.Context, refreshTokenID uuid.UUID, endedAt time.Time, reason string) error {
	return m.Called(ctx, refreshTokenID, endedAt, reason).Error(0)
}

func (m *MockImpersonationSessionsRepository) EndExpiredSessions(ctx context.Context, now time.Time, expiredReason string) (int64, error) {
	args := m.Called(ctx, now, expiredReason)
	return args.Get(0).(int64), args.Error(1)
}
