package auth_service

import (
	"context"

	"github.com/google/uuid"
	"github.com/yca-software/2chi-go-api/internals/constants"
	"github.com/yca-software/2chi-go-api/internals/models"
	user_legal_document_acceptance_repository "github.com/yca-software/2chi-go-api/internals/repositories/user_legal_document_acceptance"
	chi_types "github.com/yca-software/2chi-go-types"
)

func (s *service) createLegalDocumentAcceptance(
	ctx context.Context,
	repo user_legal_document_acceptance_repository.UserLegalDocumentAcceptanceRepository,
	userID uuid.UUID,
	documentType, documentVersion string,
) error {
	acceptanceID, err := s.generateID()
	if err != nil {
		return err
	}
	return repo.CreateUserLegalDocumentAcceptance(ctx, &models.UserLegalDocumentAcceptance{
		ModelBase: chi_types.ModelBase{
			ID: acceptanceID,
		},
		UserID:          userID,
		DocumentType:    documentType,
		DocumentVersion: documentVersion,
	})
}

func (s *service) createLegalDocumentAcceptances(
	ctx context.Context,
	repo user_legal_document_acceptance_repository.UserLegalDocumentAcceptanceRepository,
	userID uuid.UUID,
	termsVersion, privacyPolicyVersion string,
) error {
	if err := s.createLegalDocumentAcceptance(ctx, repo, userID, constants.LEGAL_DOCUMENT_TYPE_TERMS_OF_SERVICE, termsVersion); err != nil {
		return err
	}
	return s.createLegalDocumentAcceptance(ctx, repo, userID, constants.LEGAL_DOCUMENT_TYPE_PRIVACY_POLICY, privacyPolicyVersion)
}
