package impersonation_session_repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/yca-software/2chi-go-api/internals/models"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) WithTx(_ chi_repository.Tx) Repository {
	return m
}

func (m *MockRepository) Create(ctx context.Context, session *models.ImpersonationSession) error {
	return m.Called(ctx, session).Error(0)
}

func (m *MockRepository) EndByRefreshTokenID(ctx context.Context, refreshTokenID uuid.UUID, endedAt time.Time, reason string) error {
	return m.Called(ctx, refreshTokenID, endedAt, reason).Error(0)
}

func (m *MockRepository) EndExpired(ctx context.Context, now time.Time, expiredReason string) (int64, error) {
	args := m.Called(ctx, now, expiredReason)
	return args.Get(0).(int64), args.Error(1)
}
