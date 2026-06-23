package user_legal_document_acceptance_repository

import (
	"context"

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

func (m *MockRepository) Create(ctx context.Context, acceptance *models.UserLegalDocumentAcceptance) error {
	return m.Called(ctx, acceptance).Error(0)
}

func (m *MockRepository) ListByUserID(ctx context.Context, userID string) (*[]models.UserLegalDocumentAcceptance, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.UserLegalDocumentAcceptance), args.Error(1)
}

func (m *MockRepository) GetLatestByUserIDAndDocumentType(ctx context.Context, userID, documentType string) (*models.UserLegalDocumentAcceptance, error) {
	args := m.Called(ctx, userID, documentType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserLegalDocumentAcceptance), args.Error(1)
}

func (m *MockRepository) ListLatestByUserID(ctx context.Context, userID string) (*[]models.UserLegalDocumentAcceptance, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.UserLegalDocumentAcceptance), args.Error(1)
}
