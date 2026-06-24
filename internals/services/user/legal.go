package user_service

import (
	"context"

	"github.com/google/uuid"
	"github.com/yca-software/2chi-go-api/internals/constants"
	"github.com/yca-software/2chi-go-api/internals/models"
	platform_repository "github.com/yca-software/2chi-go-api/internals/packages/repository"
	user_legal_document_acceptance_repository "github.com/yca-software/2chi-go-api/internals/repositories/user_legal_document_acceptance"
	chi_types "github.com/yca-software/2chi-go-types"
)

func (s *service) userProfileWithLegalAcceptances(ctx context.Context, user *models.User) (*UserProfile, error) {
	profile := &UserProfile{User: *user}

	terms, err := s.legalDocumentAcceptancesRepo.GetLatestByUserIDAndDocumentType(
		ctx, user.ID.String(), constants.LEGAL_DOCUMENT_TYPE_TERMS_OF_SERVICE,
	)
	if err != nil && !platform_repository.IsNotFound(err) {
		return nil, err
	}
	if terms != nil {
		profile.TermsVersion = terms.DocumentVersion
		acceptedAt := terms.CreatedAt
		profile.TermsAcceptedAt = &acceptedAt
	}

	privacy, err := s.legalDocumentAcceptancesRepo.GetLatestByUserIDAndDocumentType(
		ctx, user.ID.String(), constants.LEGAL_DOCUMENT_TYPE_PRIVACY_POLICY,
	)
	if err != nil && !platform_repository.IsNotFound(err) {
		return nil, err
	}
	if privacy != nil {
		profile.PrivacyPolicyVersion = privacy.DocumentVersion
		acceptedAt := privacy.CreatedAt
		profile.PrivacyPolicyAcceptedAt = &acceptedAt
	}

	return profile, nil
}

func (s *service) createLegalDocumentAcceptance(
	ctx context.Context,
	repo user_legal_document_acceptance_repository.Repository,
	userID uuid.UUID,
	documentType, documentVersion string,
) error {
	acceptanceID, err := s.generateID()
	if err != nil {
		return err
	}
	return repo.Create(ctx, &models.UserLegalDocumentAcceptance{
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
	repo user_legal_document_acceptance_repository.Repository,
	userID uuid.UUID,
	termsVersion, privacyPolicyVersion string,
) error {
	if err := s.createLegalDocumentAcceptance(ctx, repo, userID, constants.LEGAL_DOCUMENT_TYPE_TERMS_OF_SERVICE, termsVersion); err != nil {
		return err
	}
	return s.createLegalDocumentAcceptance(ctx, repo, userID, constants.LEGAL_DOCUMENT_TYPE_PRIVACY_POLICY, privacyPolicyVersion)
}
