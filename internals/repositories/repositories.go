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
	organization_repository "github.com/yca-software/2chi-go-api/internals/repositories/organization"
	organization_location_repository "github.com/yca-software/2chi-go-api/internals/repositories/location"
	organization_member_repository "github.com/yca-software/2chi-go-api/internals/repositories/org_member"
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

	AdminAccess                  admin_access_repository.AdminAccessRepository
	APIKeys                      api_key_repository.APIKeysRepository
	AuditLogs                    audit_log_repository.AuditLogsRepository
	ImpersonationSessions        impersonation_session_repository.ImpersonationSessionsRepository
	Invitations                  invitation_repository.InvitationsRepository
	OrganizationBillingAccounts  billing_account_repository.OrganizationBillingAccountsRepository
	OrganizationLocations        organization_location_repository.OrganizationLocationsRepository
	OrganizationMembers          organization_member_repository.OrganizationMembersRepository
	Organizations                organization_repository.OrganizationsRepository
	Roles                        role_repository.RolesRepository
	TeamMembers                  team_member_repository.TeamMembersRepository
	Teams                        team_repository.TeamsRepository
	UserEmailVerificationTokens  user_email_verification_token_repository.UserEmailVerificationTokenRepository
	UserIdentities               user_identity_repository.UserIdentityRepository
	UserLegalDocumentAcceptances user_legal_document_acceptance_repository.UserLegalDocumentAcceptanceRepository
	UserPasswordResetTokens      user_password_reset_token_repository.UserPasswordResetTokenRepository
	UserRefreshTokens            user_refresh_token_repository.UserRefreshTokenRepository
	Users                        user_repository.UsersRepository
}

func NewRepositories(db *sqlx.DB, metricsHook chi_observer.QueryMetricsHook) *Repositories {
	return &Repositories{
		db:          db,
		metricsHook: metricsHook,

		AdminAccess:                  admin_access_repository.NewAdminAccessRepository(db, metricsHook),
		APIKeys:                      api_key_repository.NewAPIKeysRepository(db, metricsHook),
		AuditLogs:                    audit_log_repository.NewAuditLogsRepository(db, metricsHook),
		ImpersonationSessions:        impersonation_session_repository.NewImpersonationSessionsRepository(db, metricsHook),
		Invitations:                  invitation_repository.NewInvitationsRepository(db, metricsHook),
		OrganizationBillingAccounts:  billing_account_repository.NewOrganizationBillingAccountsRepository(db, metricsHook),
		OrganizationLocations:        organization_location_repository.NewOrganizationLocationsRepository(db, metricsHook),
		OrganizationMembers:          organization_member_repository.NewOrganizationMembersRepository(db, metricsHook),
		Organizations:                organization_repository.NewOrganizationsRepository(db, metricsHook),
		Roles:                        role_repository.NewRolesRepository(db, metricsHook),
		TeamMembers:                  team_member_repository.NewTeamMembersRepository(db, metricsHook),
		Teams:                        team_repository.NewTeamsRepository(db, metricsHook),
		UserEmailVerificationTokens:  user_email_verification_token_repository.NewUserEmailVerificationTokenRepository(db, metricsHook),
		UserIdentities:               user_identity_repository.NewUserIdentityRepository(db, metricsHook),
		UserLegalDocumentAcceptances: user_legal_document_acceptance_repository.NewUserLegalDocumentAcceptanceRepository(db, metricsHook),
		UserPasswordResetTokens:      user_password_reset_token_repository.NewUserPasswordResetTokenRepository(db, metricsHook),
		UserRefreshTokens:            user_refresh_token_repository.NewUserRefreshTokenRepository(db, metricsHook),
		Users:                        user_repository.NewUsersRepository(db, metricsHook),
	}
}

func (r *Repositories) RunInTx(ctx context.Context, fn func(tx chi_repository.Tx) error) error {
	return chi_repository.RunInTx(ctx, r.db, r.metricsHook, fn)
}
