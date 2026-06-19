package user_legal_document_acceptance_repository

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/yca-software/2chi-go-api/internals/domains/core/models"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

type MockUserLegalDocumentAcceptanceRepository struct {
	mock.Mock
}

func (m *MockUserLegalDocumentAcceptanceRepository) WithTx(_ chi_repository.Tx) UserLegalDocumentAcceptanceRepository {
	return m
}

func (m *MockUserLegalDocumentAcceptanceRepository) CreateUserLegalDocumentAcceptance(ctx context.Context, acceptance *models.UserLegalDocumentAcceptance) error {
	return m.Called(ctx, acceptance).Error(0)
}

func (m *MockUserLegalDocumentAcceptanceRepository) ListUserLegalDocumentAcceptancesByUserID(ctx context.Context, userID string) (*[]models.UserLegalDocumentAcceptance, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.UserLegalDocumentAcceptance), args.Error(1)
}

func (m *MockUserLegalDocumentAcceptanceRepository) GetLatestUserLegalDocumentAcceptanceByUserIDAndDocumentType(ctx context.Context, userID, documentType string) (*models.UserLegalDocumentAcceptance, error) {
	args := m.Called(ctx, userID, documentType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserLegalDocumentAcceptance), args.Error(1)
}

func (m *MockUserLegalDocumentAcceptanceRepository) ListLatestUserLegalDocumentAcceptancesByUserID(ctx context.Context, userID string) (*[]models.UserLegalDocumentAcceptance, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.UserLegalDocumentAcceptance), args.Error(1)
}
