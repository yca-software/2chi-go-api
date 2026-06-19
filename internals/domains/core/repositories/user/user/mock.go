package user_repository

import (
	"context"

	"github.com/stretchr/testify/mock"
	chi_archive "github.com/yca-software/2chi-go-archive"
	"github.com/yca-software/2chi-go-api/internals/domains/core/models"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

type MockUsersRepository struct {
	mock.Mock
}

func (m *MockUsersRepository) WithTx(_ chi_repository.Tx) UsersRepository {
	return m
}

func (m *MockUsersRepository) CreateUser(ctx context.Context, user *models.User) error {
	return m.Called(ctx, user).Error(0)
}

func (m *MockUsersRepository) UpdateUser(ctx context.Context, user *models.User) error {
	return m.Called(ctx, user).Error(0)
}

func (m *MockUsersRepository) ArchiveUser(ctx context.Context, user *models.User) error {
	return m.Called(ctx, user).Error(0)
}

func (m *MockUsersRepository) RestoreUser(ctx context.Context, userID string) error {
	return m.Called(ctx, userID).Error(0)
}

func (m *MockUsersRepository) CleanupArchivedUsers(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}

func (m *MockUsersRepository) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUsersRepository) GetUserByIDIncludeArchived(ctx context.Context, id string) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUsersRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUsersRepository) SearchUsers(ctx context.Context, searchPhrase string, filter chi_archive.ArchiveFilter, limit, offset int) (*[]models.User, error) {
	args := m.Called(ctx, searchPhrase, filter, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.User), args.Error(1)
}
