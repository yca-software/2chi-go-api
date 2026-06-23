package organization_service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yca-software/2chi-go-api/internals/constants"
	"github.com/yca-software/2chi-go-api/internals/models"
	"github.com/yca-software/2chi-go-api/internals/packages/audit"
	"github.com/yca-software/2chi-go-api/internals/packages/authz"
	platform_subscription "github.com/yca-software/2chi-go-api/internals/packages/subscription"
	"github.com/yca-software/2chi-go-api/internals/repositories"
	billing_account_repository "github.com/yca-software/2chi-go-api/internals/repositories/billing_account"
	organization_member_repository "github.com/yca-software/2chi-go-api/internals/repositories/org_member"
	organization_repository "github.com/yca-software/2chi-go-api/internals/repositories/organization"
	role_repository "github.com/yca-software/2chi-go-api/internals/repositories/role"
	user_repository "github.com/yca-software/2chi-go-api/internals/repositories/user"
	audit_service "github.com/yca-software/2chi-go-api/internals/services/audit"
	billing_service "github.com/yca-software/2chi-go-api/internals/services/billing"
	invitation_service "github.com/yca-software/2chi-go-api/internals/services/invitation"
	location_service "github.com/yca-software/2chi-go-api/internals/services/location"
	chi_archive "github.com/yca-software/2chi-go-archive"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_repository "github.com/yca-software/2chi-go-repository"
	chi_types "github.com/yca-software/2chi-go-types"
	chi_validator "github.com/yca-software/2chi-go-validator"
)

type Dependencies struct {
	GenerateID         func() (uuid.UUID, error)
	Now                func() time.Time
	Validator          chi_validator.Validator
	Logger             chi_logger.Logger
	Authorizer         *authz.Authorizer
	Repositories       *repositories.Repositories
	RunInTx            repositories.TxRunner
	SessionCache       *authz.SessionCache
	AuditService       audit_service.Service
	BillingService     billing_service.Service
	LocationService    location_service.Service
	InvitationsService invitation_service.Service
}

type Service interface {
	AdminCreateOrganization(ctx context.Context, req *AdminCreateOrganizationRequest, access *chi_types.AccessInfo) (*models.Organization, error)
	CreateOrganization(ctx context.Context, req *CreateOrganizationRequest, access *chi_types.AccessInfo) (*CreateOrganizationResponse, error)
	UpdateOrganization(ctx context.Context, req *UpdateOrganizationRequest, access *chi_types.AccessInfo) (*models.Organization, error)
	UpdateOrganizationSubscription(ctx context.Context, req *UpdateOrganizationSubscriptionRequest, access *chi_types.AccessInfo) (*models.OrganizationBillingAccount, error)
	ArchiveOrganization(ctx context.Context, req *ArchiveOrganizationRequest, access *chi_types.AccessInfo) error
	RestoreOrganization(ctx context.Context, req *RestoreOrganizationRequest, access *chi_types.AccessInfo) (*models.Organization, error)
	CleanupArchivedOrganizations(ctx context.Context) error

	GetOrganization(ctx context.Context, req *GetOrganizationRequest, access *chi_types.AccessInfo) (*models.Organization, error)
	GetArchivedOrganization(ctx context.Context, req *GetOrganizationRequest, access *chi_types.AccessInfo) (*models.Organization, error)
	ListOrganizations(ctx context.Context, req *ListOrganizationsRequest, access *chi_types.AccessInfo) (*ListOrganizationsResponse, error)

	UpdateOrganizationMember(ctx context.Context, req *UpdateOrganizationMemberRequest, access *chi_types.AccessInfo) (*models.OrganizationMemberWithUser, error)
	DeleteOrganizationMember(ctx context.Context, req *DeleteOrganizationMemberRequest, access *chi_types.AccessInfo) error
	ListOrganizationMembers(ctx context.Context, req *ListOrganizationMembersRequest, access *chi_types.AccessInfo) (*[]models.OrganizationMemberWithUser, error)
	ListOrganizationRolesForUser(ctx context.Context, req *ListOrganizationRolesForUserRequest, access *chi_types.AccessInfo) (*[]models.OrganizationMemberWithOrganizationAndRole, error)
}

type service struct {
	generateID              func() (uuid.UUID, error)
	now                     func() time.Time
	validator               chi_validator.Validator
	logger                  chi_logger.Logger
	runInTx                 repositories.TxRunner
	authorizer              *authz.Authorizer
	organizationsRepo       organization_repository.Repository
	organizationMembersRepo organization_member_repository.Repository
	billingAccountsRepo     billing_account_repository.Repository
	rolesRepo               role_repository.Repository
	usersRepo               user_repository.Repository
	sessionCache            *authz.SessionCache
	auditService            audit_service.Service
	billingService          billing_service.Service
	locationService         location_service.Service
	invitationsService      invitation_service.Service
}

func New(deps Dependencies) Service {
	runInTx := deps.RunInTx
	if runInTx == nil && deps.Repositories != nil {
		runInTx = deps.Repositories.RunInTx
	}
	return &service{
		generateID:              deps.GenerateID,
		now:                     deps.Now,
		validator:               deps.Validator,
		logger:                  deps.Logger,
		runInTx:                 runInTx,
		authorizer:              deps.Authorizer,
		organizationsRepo:       deps.Repositories.Organizations,
		organizationMembersRepo: deps.Repositories.OrganizationMembers,
		billingAccountsRepo:     deps.Repositories.OrganizationBillingAccounts,
		rolesRepo:               deps.Repositories.Roles,
		usersRepo:               deps.Repositories.Users,
		sessionCache:            deps.SessionCache,
		auditService:            deps.AuditService,
		billingService:          deps.BillingService,
		locationService:         deps.LocationService,
		invitationsService:      deps.InvitationsService,
	}
}

func (s *service) CreateOrganization(ctx context.Context, req *CreateOrganizationRequest, access *chi_types.AccessInfo) (*CreateOrganizationResponse, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	if access == nil {
		return nil, chi_error.NewForbiddenError(errors.New("access required"), "Forbidden", nil)
	}
	if access.Type != chi_types.AccessTypeUser {
		return nil, chi_error.NewForbiddenError(errors.New("api keys cannot create organizations"), "APICannotCreateOrganization", nil)
	}

	now := s.now()
	orgID, err := s.generateID()
	if err != nil {
		return nil, err
	}

	locationData, err := s.locationService.GetLocationData(ctx, req.PlaceID)
	if err != nil {
		return nil, err
	}

	org := &models.Organization{
		ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
			ModelBase: chi_types.ModelBase{
				ID:        orgID,
				CreatedAt: now,
			},
		},
		Name:     strings.TrimSpace(req.Name),
		Address:  locationData.Address,
		City:     locationData.City,
		Zip:      locationData.Zip,
		Country:  locationData.Country,
		PlaceID:  locationData.PlaceID,
		Geo:      chi_types.Point{Lat: locationData.Geo.Lat, Lng: locationData.Geo.Lng},
		Timezone: locationData.Timezone,
	}

	billingAccount := &models.OrganizationBillingAccount{
		OrganizationID:              orgID,
		BillingEmail:                req.BillingEmail,
		Provider:                    constants.BILLING_PROVIDER_PADDLE,
		SubscriptionTier:            constants.TIER_FREE,
		SubscriptionSeats:           constants.SUBSCRIPTION_TYPE_SEATS_INCLUDED_FREE,
		SubscriptionPaymentInterval: constants.PAYMENT_INTERVAL_MONTHLY,
	}

	providerCustomerID, paddleErr := s.billingService.CreateCustomer(ctx, &billing_service.CreateCustomerInput{
		OrganizationID:   orgID.String(),
		OrganizationName: org.Name,
		BillingEmail:     req.BillingEmail,
		Address:          org.Address,
		City:             org.City,
		Zip:              org.Zip,
		Country:          org.Country,
		Timezone:         org.Timezone,
	})
	if paddleErr != nil {
		return nil, paddleErr
	}
	billingAccount.ProviderCustomerID = providerCustomerID

	roles := make([]models.Role, 0, len(DefaultRolesToCreateForOrganization))
	var ownerRoleID uuid.UUID

	for i, roleTemplate := range DefaultRolesToCreateForOrganization {
		roleID, genErr := s.generateID()
		if genErr != nil {
			s.rollbackPaddleCustomer(ctx, orgID.String(), billingAccount)
			return nil, genErr
		}

		role := roleTemplate
		role.ID = roleID
		role.CreatedAt = now
		role.OrganizationID = orgID

		if i == 0 {
			ownerRoleID = roleID
		}
		roles = append(roles, role)
	}

	membershipID, genErr := s.generateID()
	if genErr != nil {
		s.rollbackPaddleCustomer(ctx, orgID.String(), billingAccount)
		return nil, genErr
	}

	membership := models.OrganizationMember{
		ModelBase: chi_types.ModelBase{
			ID:        membershipID,
			CreatedAt: now,
		},
		OrganizationID: orgID,
		UserID:         access.SubjectID,
		RoleID:         ownerRoleID,
	}

	if txErr := s.runInTx(ctx, func(tx chi_repository.Tx) error {
		orgRepo := s.organizationsRepo.WithTx(tx)
		if err := orgRepo.Create(ctx, org); err != nil {
			return err
		}

		if err := s.billingAccountsRepo.WithTx(tx).Create(ctx, billingAccount); err != nil {
			return err
		}

		rolesRepo := s.rolesRepo.WithTx(tx)
		if err := rolesRepo.CreateMany(ctx, &roles); err != nil {
			return err
		}

		return s.organizationMembersRepo.WithTx(tx).Create(ctx, &membership)
	}); txErr != nil {
		s.rollbackPaddleCustomer(ctx, orgID.String(), billingAccount)
		return nil, txErr
	}

	if err := s.sessionCache.InvalidateSession(ctx, access.SubjectID.String()); err != nil {
		s.logger.WithContext(ctx).Error("failed to invalidate session", "error", err, "organizationId", org.ID.String())
	}

	s.logOrganizationAudit(ctx, access, org.ID.String(), constants.AUDIT_ACTION_TYPE_CREATE, org.Name, audit.CreatePayload(map[string]any{
		"name":         org.Name,
		"placeId":      org.PlaceID,
		"billingEmail": billingAccount.BillingEmail,
	}))

	return &CreateOrganizationResponse{
		Organization: org,
		Roles:        &roles,
		Member:       &membership,
	}, nil
}

func (s *service) UpdateOrganization(ctx context.Context, req *UpdateOrganizationRequest, access *chi_types.AccessInfo) (*models.Organization, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	org, err := s.organizationsRepo.GetByID(ctx, req.OrganizationID)
	if err != nil {
		return nil, err
	}

	billingAccount, err := s.billingAccountsRepo.GetByOrganizationID(ctx, req.OrganizationID)
	if err != nil {
		return nil, err
	}

	if err := s.authorizer.CheckOrganizationPermissionWithSubscription(access, billingAccount, constants.PERMISSION_ORG_WRITE); err != nil {
		return nil, err
	}

	previousOrg := *org

	updatedOrg := *org
	updatedOrg.Name = strings.TrimSpace(req.Name)
	if req.PlaceID != org.PlaceID {
		locationData, locErr := s.locationService.GetLocationData(ctx, req.PlaceID)
		if locErr != nil {
			return nil, locErr
		}
		updatedOrg.Address = locationData.Address
		updatedOrg.City = locationData.City
		updatedOrg.Zip = locationData.Zip
		updatedOrg.Country = locationData.Country
		updatedOrg.PlaceID = locationData.PlaceID
		updatedOrg.Geo = chi_types.Point{Lat: locationData.Geo.Lat, Lng: locationData.Geo.Lng}
		updatedOrg.Timezone = locationData.Timezone
	}

	if paddleErr := s.billingService.UpdateCustomer(ctx, &billing_service.UpdateCustomerInput{
		OrganizationID:   req.OrganizationID,
		OrganizationName: updatedOrg.Name,
		BillingAccount:   billingAccount,
		Address:          updatedOrg.Address,
		City:             updatedOrg.City,
		Zip:              updatedOrg.Zip,
		Country:          updatedOrg.Country,
		Timezone:         updatedOrg.Timezone,
	}); paddleErr != nil {
		s.logger.WithContext(ctx).Error("failed to update paddle customer", "error", paddleErr, "organizationId", req.OrganizationID)
		return nil, paddleErr
	}

	if txErr := s.runInTx(ctx, func(tx chi_repository.Tx) error {
		if err := s.organizationsRepo.WithTx(tx).Update(ctx, &updatedOrg); err != nil {
			return err
		}
		return nil
	}); txErr != nil {
		_ = s.billingService.UpdateCustomer(ctx, &billing_service.UpdateCustomerInput{
			OrganizationID:   req.OrganizationID,
			OrganizationName: previousOrg.Name,
			BillingAccount:   billingAccount,
			Address:          previousOrg.Address,
			City:             previousOrg.City,
			Zip:              previousOrg.Zip,
			Country:          previousOrg.Country,
			Timezone:         previousOrg.Timezone,
		})
		return nil, txErr
	}

	s.logOrganizationAudit(ctx, access, org.ID.String(), constants.AUDIT_ACTION_TYPE_UPDATE, updatedOrg.Name, audit.UpdatePayload(
		map[string]any{
			"name":    previousOrg.Name,
			"placeId": previousOrg.PlaceID,
		},
		map[string]any{
			"name":    updatedOrg.Name,
			"placeId": updatedOrg.PlaceID,
		},
	))

	return &updatedOrg, nil
}

func (s *service) UpdateOrganizationSubscription(ctx context.Context, req *UpdateOrganizationSubscriptionRequest, access *chi_types.AccessInfo) (*models.OrganizationBillingAccount, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	if err := s.authorizer.CheckPlatformAdmin(access); err != nil {
		return nil, err
	}

	if _, err := s.organizationsRepo.GetByID(ctx, req.OrganizationID); err != nil {
		return nil, err
	}

	if err := platform_subscription.ValidateSubscriptionSeats(req.SubscriptionSeats, req.SubscriptionType); err != nil {
		return nil, err
	}

	if !platform_subscription.IsUnlimitedSubscriptionSeats(req.SubscriptionSeats) {
		members, listErr := s.organizationMembersRepo.ListByOrganizationID(ctx, req.OrganizationID)
		if listErr != nil {
			return nil, listErr
		}
		if len(*members) > req.SubscriptionSeats {
			return nil, chi_error.NewUnprocessableEntityError(
				errors.New("subscription seats below member count"),
				"OrganizationSeatsBelowMemberCount",
				map[string]any{"memberCount": len(*members), "subscriptionSeats": req.SubscriptionSeats},
			)
		}
	}

	account, err := s.billingAccountsRepo.GetByOrganizationID(ctx, req.OrganizationID)
	if err != nil {
		return nil, err
	}

	provider := constants.BILLING_PROVIDER_PADDLE
	if req.CustomSubscription {
		provider = constants.BILLING_PROVIDER_CUSTOM
	}

	updatedAccount := *account
	updatedAccount.Provider = provider
	updatedAccount.SubscriptionTier = req.SubscriptionType
	updatedAccount.SubscriptionSeats = req.SubscriptionSeats
	expiresAt := req.SubscriptionExpiresAt
	updatedAccount.SubscriptionExpiresAt = &expiresAt

	if txErr := s.runInTx(ctx, func(tx chi_repository.Tx) error {
		return s.billingAccountsRepo.WithTx(tx).Update(ctx, &updatedAccount)
	}); txErr != nil {
		return nil, txErr
	}

	return &updatedAccount, nil
}

func (s *service) UpdateOrganizationMember(ctx context.Context, req *UpdateOrganizationMemberRequest, access *chi_types.AccessInfo) (*models.OrganizationMemberWithUser, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	if _, err := s.organizationsRepo.GetByID(ctx, req.OrganizationID); err != nil {
		return nil, err
	}

	billingAccount, err := s.billingAccountsRepo.GetByOrganizationID(ctx, req.OrganizationID)
	if err != nil {
		return nil, err
	}

	if err := s.authorizer.CheckOrganizationPermissionWithSubscription(access, billingAccount, constants.PERMISSION_MEMBERS_WRITE); err != nil {
		return nil, err
	}

	member, err := s.organizationMembersRepo.GetByMemberID(ctx, req.OrganizationID, req.MemberID)
	if err != nil {
		return nil, err
	}

	if access != nil && access.Type == chi_types.AccessTypeUser && member.UserID == access.SubjectID {
		return nil, chi_error.NewForbiddenError(errors.New("cannot update own membership"), "UserCannotUpdateOwnMember", nil)
	}

	currentRole, err := s.rolesRepo.GetByID(ctx, req.OrganizationID, member.RoleID.String())
	if err != nil {
		return nil, err
	}

	newRole, err := s.rolesRepo.GetByID(ctx, req.OrganizationID, req.RoleID)
	if err != nil {
		return nil, err
	}

	member.RoleID = newRole.ID
	if err := s.organizationMembersRepo.Update(ctx, member); err != nil {
		return nil, err
	}

	if err := s.sessionCache.InvalidateSession(ctx, member.UserID.String()); err != nil {
		s.logger.WithContext(ctx).Error("failed to invalidate session", "error", err, "organizationId", req.OrganizationID)
	}

	memberWithUser, err := s.organizationMembersRepo.GetByMemberIDWithUser(ctx, req.OrganizationID, req.MemberID)
	if err != nil {
		return nil, err
	}

	changes, _ := json.Marshal(audit.UpdatePayload(
		map[string]any{"roleId": currentRole.ID, "roleName": currentRole.Name},
		map[string]any{"roleId": newRole.ID, "roleName": newRole.Name},
	))
	changesRaw := json.RawMessage(changes)
	resourceName := "Organization member"
	s.createMemberAudit(ctx, access, req.OrganizationID, constants.AUDIT_ACTION_TYPE_UPDATE, member.ID.String(), resourceName, &changesRaw)

	return memberWithUser, nil
}

func (s *service) ArchiveOrganization(ctx context.Context, req *ArchiveOrganizationRequest, access *chi_types.AccessInfo) error {
	if err := s.validator.ValidateStruct(req); err != nil {
		return chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	org, err := s.organizationsRepo.GetByID(ctx, req.OrganizationID)
	if err != nil {
		return err
	}

	if err := s.authorizer.CheckOrganizationPermission(access, org.ID.String(), constants.PERMISSION_ORG_DELETE); err != nil {
		return err
	}

	if err := s.organizationsRepo.Archive(ctx, org); err != nil {
		return err
	}

	billingAccount, billingErr := s.billingAccountsRepo.GetByOrganizationID(ctx, req.OrganizationID)
	if billingErr == nil && billingAccount.ProviderSubscriptionID != "" {
		if cancelErr := s.billingService.CancelSubscription(ctx, billingAccount.ProviderSubscriptionID); cancelErr != nil {
			s.logger.WithContext(ctx).Error("failed to cancel paddle subscription on archive", "error", cancelErr, "organizationId", org.ID.String())
		}
	}

	s.logOrganizationAudit(ctx, access, org.ID.String(), constants.AUDIT_ACTION_TYPE_ARCHIVE, org.Name, map[string]any{})

	return nil
}

func (s *service) RestoreOrganization(ctx context.Context, req *RestoreOrganizationRequest, access *chi_types.AccessInfo) (*models.Organization, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	if err := s.authorizer.CheckPlatformAdmin(access); err != nil {
		return nil, err
	}

	org, err := s.organizationsRepo.GetByIDIncludeArchived(ctx, req.OrganizationID)
	if err != nil {
		return nil, err
	}
	if org.DeletedAt == nil {
		return nil, chi_error.NewNotFoundError(errors.New("organization is not archived"), "NotFound", nil)
	}

	if err := s.organizationsRepo.Restore(ctx, req.OrganizationID); err != nil {
		return nil, err
	}

	restored, err := s.organizationsRepo.GetByID(ctx, req.OrganizationID)
	if err != nil {
		return nil, err
	}

	s.logOrganizationAudit(ctx, access, org.ID.String(), constants.AUDIT_ACTION_TYPE_RESTORE, restored.Name, map[string]any{})

	return restored, nil
}

func (s *service) DeleteOrganizationMember(ctx context.Context, req *DeleteOrganizationMemberRequest, access *chi_types.AccessInfo) error {
	if err := s.validator.ValidateStruct(req); err != nil {
		return chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	if _, err := s.organizationsRepo.GetByID(ctx, req.OrganizationID); err != nil {
		return err
	}

	billingAccount, err := s.billingAccountsRepo.GetByOrganizationID(ctx, req.OrganizationID)
	if err != nil {
		return err
	}

	if err := s.authorizer.CheckOrganizationPermissionWithSubscription(access, billingAccount, constants.PERMISSION_MEMBERS_DELETE); err != nil {
		return err
	}

	member, err := s.organizationMembersRepo.GetByMemberID(ctx, req.OrganizationID, req.MemberID)
	if err != nil {
		return err
	}

	if access.Type == chi_types.AccessTypeUser && member.UserID == access.SubjectID {
		return chi_error.NewForbiddenError(errors.New("cannot remove own membership"), "UserCannotRemoveOwnMember", nil)
	}

	user, err := s.usersRepo.GetByID(ctx, member.UserID.String())
	if err != nil {
		return err
	}

	role, err := s.rolesRepo.GetByID(ctx, req.OrganizationID, member.RoleID.String())
	if err != nil {
		return err
	}

	if err := s.organizationMembersRepo.DeleteByMemberID(ctx, req.OrganizationID, req.MemberID); err != nil {
		return err
	}

	if err := s.sessionCache.InvalidateSession(ctx, member.UserID.String()); err != nil {
		s.logger.WithContext(ctx).Error("failed to invalidate session", "error", err, "organizationId", req.OrganizationID)
	}

	data, _ := json.Marshal(audit.DeletePayload(map[string]any{
		"userId":    user.ID,
		"userEmail": user.Email,
		"roleId":    role.ID,
		"roleName":  role.Name,
	}))
	dataRaw := json.RawMessage(data)
	resourceName := "Organization member"
	s.createMemberAudit(ctx, access, req.OrganizationID, constants.AUDIT_ACTION_TYPE_DELETE, member.ID.String(), resourceName, &dataRaw)

	return nil
}

func (s *service) GetOrganization(ctx context.Context, req *GetOrganizationRequest, access *chi_types.AccessInfo) (*models.Organization, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	org, err := s.organizationsRepo.GetByID(ctx, req.OrganizationID)
	if err != nil {
		return nil, err
	}

	if err := s.authorizer.CheckOrganizationPermission(access, org.ID.String(), constants.PERMISSION_ORG_READ); err != nil {
		if subErr := s.authorizer.CheckOrganizationPermission(access, org.ID.String(), constants.PERMISSION_SUBSCRIPTION_READ); subErr != nil {
			return nil, err
		}
	}
	return org, nil
}

func (s *service) GetArchivedOrganization(ctx context.Context, req *GetOrganizationRequest, access *chi_types.AccessInfo) (*models.Organization, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}
	if err := s.authorizer.CheckPlatformAdmin(access); err != nil {
		return nil, err
	}
	return s.organizationsRepo.GetByIDIncludeArchived(ctx, req.OrganizationID)
}

func (s *service) ListOrganizations(ctx context.Context, req *ListOrganizationsRequest, access *chi_types.AccessInfo) (*ListOrganizationsResponse, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	if err := s.authorizer.CheckPlatformAdmin(access); err != nil {
		return nil, err
	}

	filter := req.ArchiveFilter
	if filter == "" {
		filter = chi_archive.ArchiveFilterActive
	}

	orgs, err := s.organizationsRepo.Search(ctx, req.SearchPhrase, filter, req.Limit+1, req.Offset)
	if err != nil {
		return nil, err
	}

	hasNext := len(*orgs) > req.Limit
	if hasNext {
		items := (*orgs)[:req.Limit]
		orgs = &items
	}

	return &ListOrganizationsResponse{
		Items:   *orgs,
		HasNext: hasNext,
	}, nil
}

func (s *service) ListOrganizationMembers(ctx context.Context, req *ListOrganizationMembersRequest, access *chi_types.AccessInfo) (*[]models.OrganizationMemberWithUser, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	if _, err := s.organizationsRepo.GetByID(ctx, req.OrganizationID); err != nil {
		return nil, err
	}

	if err := s.authorizer.CheckOrganizationPermission(access, req.OrganizationID, constants.PERMISSION_MEMBERS_READ); err != nil {
		return nil, err
	}

	return s.organizationMembersRepo.ListByOrganizationID(ctx, req.OrganizationID)
}

func (s *service) ListOrganizationRolesForUser(ctx context.Context, req *ListOrganizationRolesForUserRequest, access *chi_types.AccessInfo) (*[]models.OrganizationMemberWithOrganizationAndRole, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	if err := s.authorizer.CheckOwnResource(access, req.UserID); err != nil {
		return nil, err
	}

	return s.organizationMembersRepo.ListByUserID(ctx, req.UserID)
}

func (s *service) AdminCreateOrganization(ctx context.Context, req *AdminCreateOrganizationRequest, access *chi_types.AccessInfo) (*models.Organization, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}
	if err := s.authorizer.CheckPlatformAdmin(access); err != nil {
		return nil, err
	}

	now := s.now()
	orgID, err := s.generateID()
	if err != nil {
		return nil, err
	}

	locationData, err := s.locationService.GetLocationData(ctx, req.PlaceID)
	if err != nil {
		return nil, err
	}

	org := &models.Organization{
		ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
			ModelBase: chi_types.ModelBase{
				ID:        orgID,
				CreatedAt: now,
			},
		},
		Name:     strings.TrimSpace(req.Name),
		Address:  locationData.Address,
		City:     locationData.City,
		Zip:      locationData.Zip,
		Country:  locationData.Country,
		PlaceID:  locationData.PlaceID,
		Geo:      chi_types.Point{Lat: locationData.Geo.Lat, Lng: locationData.Geo.Lng},
		Timezone: locationData.Timezone,
	}

	billingAccount := &models.OrganizationBillingAccount{
		OrganizationID:              orgID,
		BillingEmail:                req.BillingEmail,
		Provider:                    constants.BILLING_PROVIDER_CUSTOM,
		SubscriptionTier:            req.SubscriptionType,
		SubscriptionSeats:           req.SubscriptionSeats,
		SubscriptionPaymentInterval: constants.PAYMENT_INTERVAL_MONTHLY,
		SubscriptionExpiresAt:       req.SubscriptionExpiresAt,
	}

	providerCustomerID, paddleErr := s.billingService.CreateCustomer(ctx, &billing_service.CreateCustomerInput{
		OrganizationID:   orgID.String(),
		OrganizationName: org.Name,
		BillingEmail:     req.BillingEmail,
		Address:          org.Address,
		City:             org.City,
		Zip:              org.Zip,
		Country:          org.Country,
		Timezone:         org.Timezone,
	})
	if paddleErr != nil {
		return nil, paddleErr
	}
	billingAccount.ProviderCustomerID = providerCustomerID

	roles := make([]models.Role, 0, len(DefaultRolesToCreateForOrganization))
	var ownerRoleID uuid.UUID
	for i, roleTemplate := range DefaultRolesToCreateForOrganization {
		roleID, genErr := s.generateID()
		if genErr != nil {
			s.rollbackPaddleCustomer(ctx, orgID.String(), billingAccount)
			return nil, genErr
		}
		role := roleTemplate
		role.ID = roleID
		role.CreatedAt = now
		role.OrganizationID = orgID
		if i == 0 {
			ownerRoleID = roleID
		}
		roles = append(roles, role)
	}

	emailLower := strings.ToLower(strings.TrimSpace(req.OwnerEmail))
	existingUser, userErr := s.usersRepo.GetByEmail(ctx, emailLower)
	if userErr != nil {
		if apiErr, ok := userErr.(*chi_error.Error); !ok || apiErr.StatusCode != http.StatusNotFound {
			s.rollbackPaddleCustomer(ctx, orgID.String(), billingAccount)
			return nil, userErr
		}
		existingUser = nil
	}

	var membership *models.OrganizationMember
	if existingUser != nil {
		membershipID, genErr := s.generateID()
		if genErr != nil {
			s.rollbackPaddleCustomer(ctx, orgID.String(), billingAccount)
			return nil, genErr
		}
		membership = &models.OrganizationMember{
			ModelBase: chi_types.ModelBase{
				ID:        membershipID,
				CreatedAt: now,
			},
			OrganizationID: orgID,
			UserID:         existingUser.ID,
			RoleID:         ownerRoleID,
		}
	}

	if txErr := s.runInTx(ctx, func(tx chi_repository.Tx) error {
		orgRepo := s.organizationsRepo.WithTx(tx)
		if err := orgRepo.Create(ctx, org); err != nil {
			return err
		}
		if err := s.billingAccountsRepo.WithTx(tx).Create(ctx, billingAccount); err != nil {
			return err
		}
		rolesRepo := s.rolesRepo.WithTx(tx)
		if err := rolesRepo.CreateMany(ctx, &roles); err != nil {
			return err
		}
		if membership != nil {
			return s.organizationMembersRepo.WithTx(tx).Create(ctx, membership)
		}
		return nil
	}); txErr != nil {
		s.rollbackPaddleCustomer(ctx, orgID.String(), billingAccount)
		return nil, txErr
	}

	if existingUser == nil {
		_, invErr := s.invitationsService.Create(ctx, &invitation_service.CreateRequest{
			Email:          req.OwnerEmail,
			OrganizationID: org.ID.String(),
			RoleID:         ownerRoleID.String(),
			InvitedByID:    access.SubjectID.String(),
			InvitedByEmail: access.Email,
			Language:       req.Language,
		}, access)
		if invErr != nil {
			if archiveErr := s.organizationsRepo.Archive(ctx, org); archiveErr != nil {
				s.logger.WithContext(ctx).Error("failed to archive org after owner invitation failure", "error", archiveErr, "organizationId", org.ID.String())
			}
			s.logger.WithContext(ctx).Error("failed to send owner invitation", "error", invErr, "organizationId", org.ID.String())
			return nil, invErr
		}
	} else if existingUser != nil && s.sessionCache != nil {
		if err := s.sessionCache.InvalidateSession(ctx, existingUser.ID.String()); err != nil {
			s.logger.WithContext(ctx).Error("failed to invalidate session", "error", err, "organizationId", org.ID.String())
		}
	}

	return org, nil
}

func (s *service) CleanupArchivedOrganizations(ctx context.Context) error {
	return s.organizationsRepo.CleanupArchived(ctx)
}

func (s *service) logOrganizationAudit(ctx context.Context, access *chi_types.AccessInfo, orgID, action, resourceName string, payload map[string]any) {
	changes, err := json.Marshal(payload)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to marshal organization audit payload", "error", err, "organizationId", orgID)
		return
	}
	changesRaw := json.RawMessage(changes)

	if _, err := s.auditService.Create(ctx, &audit_service.CreateRequest{
		OrganizationID: orgID,
		Action:         action,
		ResourceType:   constants.RESOURCE_TYPE_ORGANIZATION,
		ResourceID:     orgID,
		ResourceName:   resourceName,
		Data:           &changesRaw,
	}, access); err != nil {
		s.logger.WithContext(ctx).Error("failed to create organization audit log", "error", err, "organizationId", orgID)
	}
}

func (s *service) createMemberAudit(ctx context.Context, access *chi_types.AccessInfo, orgID, action, resourceID, resourceName string, data *json.RawMessage) {
	if _, err := s.auditService.Create(ctx, &audit_service.CreateRequest{
		OrganizationID: orgID,
		Action:         action,
		ResourceType:   constants.RESOURCE_TYPE_MEMBER,
		ResourceID:     resourceID,
		ResourceName:   resourceName,
		Data:           data,
	}, access); err != nil {
		s.logger.WithContext(ctx).Error("failed to create member audit log", "error", err, "organizationId", orgID)
	}
}

func (s *service) rollbackPaddleCustomer(ctx context.Context, organizationID string, billingAccount *models.OrganizationBillingAccount) {
	if billingAccount == nil || billingAccount.ProviderCustomerID == "" {
		return
	}

	if err := s.billingService.ReleaseProvisionedCustomer(ctx, organizationID, billingAccount); err != nil {
		s.logger.WithContext(ctx).Error(
			"failed to release paddle customer after org provision rollback",
			"error", err,
			"organizationId", organizationID,
			"paddleCustomerId", billingAccount.ProviderCustomerID,
		)
	}
}
