package repositories

import (
	"context"

	"github.com/jmoiron/sqlx"
	admin_access_repository "github.com/yca-software/2chi-go-api/internals/repositories/admin_access"
	api_key_repository "github.com/yca-software/2chi-go-api/internals/repositories/api_key"
	audit_log_repository "github.com/yca-software/2chi-go-api/internals/repositories/audit_log"
	billing_account_repository "github.com/yca-software/2chi-go-api/internals/repositories/billing_account"
	impersonation_session_repository "github.com/yca-software/2chi-go-api/internals/repositories/impersonation_session"
	invitation_repository "github.com/yca-software/2chi-go-api/internals/repositories/invitation"
	organization_member_repository "github.com/yca-software/2chi-go-api/internals/repositories/org_member"
	organization_repository "github.com/yca-software/2chi-go-api/internals/repositories/organization"
	role_repository "github.com/yca-software/2chi-go-api/internals/repositories/role"
	team_repository "github.com/yca-software/2chi-go-api/internals/repositories/team"
	team_member_repository "github.com/yca-software/2chi-go-api/internals/repositories/team_member"
	user_repository "github.com/yca-software/2chi-go-api/internals/repositories/user"
	user_email_verification_token_repository "github.com/yca-software/2chi-go-api/internals/repositories/user_email_verification_token"
	user_identity_repository "github.com/yca-software/2chi-go-api/internals/repositories/user_identity"
	user_legal_document_acceptance_repository "github.com/yca-software/2chi-go-api/internals/repositories/user_legal_document_acceptance"
	user_password_reset_token_repository "github.com/yca-software/2chi-go-api/internals/repositories/user_password_reset_token"
	user_refresh_token_repository "github.com/yca-software/2chi-go-api/internals/repositories/user_refresh_token"
	chi_observer "github.com/yca-software/2chi-go-observer"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

type TxRunner func(ctx context.Context, fn func(tx chi_repository.Tx) error) error

type Repositories struct {
	db          *sqlx.DB
	metricsHook chi_observer.QueryMetricsHook

	AdminAccess                  admin_access_repository.Repository
	APIKeys                      api_key_repository.Repository
	AuditLogs                    audit_log_repository.Repository
	ImpersonationSessions        impersonation_session_repository.Repository
	Invitations                  invitation_repository.Repository
	OrganizationBillingAccounts  billing_account_repository.Repository
	OrganizationMembers          organization_member_repository.Repository
	Organizations                organization_repository.Repository
	Roles                        role_repository.Repository
	TeamMembers                  team_member_repository.Repository
	Teams                        team_repository.Repository
	UserEmailVerificationTokens  user_email_verification_token_repository.Repository
	UserIdentities               user_identity_repository.Repository
	UserLegalDocumentAcceptances user_legal_document_acceptance_repository.Repository
	UserPasswordResetTokens      user_password_reset_token_repository.Repository
	UserRefreshTokens            user_refresh_token_repository.Repository
	Users                        user_repository.Repository
}

func NewRepositories(db *sqlx.DB, metricsHook chi_observer.QueryMetricsHook) *Repositories {
	return &Repositories{
		db:          db,
		metricsHook: metricsHook,

		AdminAccess:                  admin_access_repository.NewRepository(db, metricsHook),
		APIKeys:                      api_key_repository.NewRepository(db, metricsHook),
		AuditLogs:                    audit_log_repository.NewRepository(db, metricsHook),
		ImpersonationSessions:        impersonation_session_repository.NewRepository(db, metricsHook),
		Invitations:                  invitation_repository.NewRepository(db, metricsHook),
		OrganizationBillingAccounts:  billing_account_repository.NewRepository(db, metricsHook),
		OrganizationMembers:          organization_member_repository.NewRepository(db, metricsHook),
		Organizations:                organization_repository.NewRepository(db, metricsHook),
		Roles:                        role_repository.NewRepository(db, metricsHook),
		TeamMembers:                  team_member_repository.NewRepository(db, metricsHook),
		Teams:                        team_repository.NewRepository(db, metricsHook),
		UserEmailVerificationTokens:  user_email_verification_token_repository.NewRepository(db, metricsHook),
		UserIdentities:               user_identity_repository.NewRepository(db, metricsHook),
		UserLegalDocumentAcceptances: user_legal_document_acceptance_repository.NewRepository(db, metricsHook),
		UserPasswordResetTokens:      user_password_reset_token_repository.NewRepository(db, metricsHook),
		UserRefreshTokens:            user_refresh_token_repository.NewRepository(db, metricsHook),
		Users:                        user_repository.NewRepository(db, metricsHook),
	}
}

func (r *Repositories) RunInTx(ctx context.Context, fn func(tx chi_repository.Tx) error) error {
	return chi_repository.RunInTx(ctx, r.db, r.metricsHook, fn)
}
