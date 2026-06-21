package constants

import "time"

const (
	ACCESS_TOKEN_TTL                = 15 * time.Minute
	REFRESH_TOKEN_TTL               = 30 * 24 * time.Hour
	PASSWORD_RESET_TOKEN_TTL        = 1 * time.Hour
	EMAIL_VERIFICATION_TOKEN_TTL    = 24 * time.Hour
	INVITATION_TOKEN_TTL            = 7 * 24 * time.Hour
	IMPERSONATION_END_REASON_LOGOUT = "logout"

	LEGAL_DOCUMENT_TYPE_TERMS_OF_SERVICE = "terms_of_service"
	LEGAL_DOCUMENT_TYPE_PRIVACY_POLICY   = "privacy_policy"

	USER_IDENTITY_PROVIDER_GOOGLE = "google"
)
