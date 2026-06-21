package services

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/yca-software/2chi-go-api/internals/config"
	"github.com/yca-software/2chi-go-api/internals/constants"
	"github.com/yca-software/2chi-go-api/internals/packages/authz"
	"github.com/yca-software/2chi-go-api/internals/packages/datastores"
	"github.com/yca-software/2chi-go-api/internals/repositories"
	api_key_service "github.com/yca-software/2chi-go-api/internals/services/api_key"
	audit_service "github.com/yca-software/2chi-go-api/internals/services/audit"
	auth_service "github.com/yca-software/2chi-go-api/internals/services/auth"
	billing_service "github.com/yca-software/2chi-go-api/internals/services/billing"
	invitation_service "github.com/yca-software/2chi-go-api/internals/services/invitation"
	location_service "github.com/yca-software/2chi-go-api/internals/services/location"
	organization_service "github.com/yca-software/2chi-go-api/internals/services/organization"
	role_service "github.com/yca-software/2chi-go-api/internals/services/role"
	support_service "github.com/yca-software/2chi-go-api/internals/services/support"
	team_service "github.com/yca-software/2chi-go-api/internals/services/team"
	user_service "github.com/yca-software/2chi-go-api/internals/services/user"
	chi_aws "github.com/yca-software/2chi-go-aws"
	chi_aws_ses "github.com/yca-software/2chi-go-aws/ses"
	chi_google "github.com/yca-software/2chi-go-google"
	chi_google_maps "github.com/yca-software/2chi-go-google/maps"
	chi_google_oauth "github.com/yca-software/2chi-go-google/oauth"
	chi_localizer "github.com/yca-software/2chi-go-localizer"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_paddle "github.com/yca-software/2chi-go-paddle"
	chi_password "github.com/yca-software/2chi-go-password"
	chi_template "github.com/yca-software/2chi-go-template"
	chi_token "github.com/yca-software/2chi-go-token"
	chi_validator "github.com/yca-software/2chi-go-validator"
)

type Services struct {
	Audit        audit_service.Service
	Auth         auth_service.Service
	APIKey       api_key_service.Service
	Billing      billing_service.Service
	Invitation   invitation_service.Service
	Location     location_service.Service
	Organization organization_service.Service
	Role         role_service.Service
	Support      support_service.Service
	Team         team_service.Service
	User         user_service.Service
}

func NewServices(
	datastores *datastores.Datastores,
	cfg *config.Config,
	logger chi_logger.Logger,
	repos *repositories.Repositories,
	sessionCache *authz.SessionCache,
) *Services {
	now := time.Now
	authorizer := authz.NewAuthorizer(now)

	appValidator := chi_validator.New()
	appLocalizer := chi_localizer.New(
		constants.SUPPORTED_LANGUAGES,
		constants.DEFAULT_LANGUAGE,
		cfg.App.LocalesPath,
	)

	emailTemplates := chi_template.NewHTML(cfg.App.TemplatesPath)
	tokenHasher := chi_token.NewHasher(cfg.Auth.TokenHashPepper)

	ctx := context.Background()
	awsModule, err := chi_aws.New(ctx, chi_aws.Config{
		Region:   cfg.AWS.DefaultRegion,
		Endpoint: cfg.AWS.DefaultEndpoint,
		SES: &chi_aws_ses.Config{
			FromEmail: cfg.AWS.Ses.FromEmail,
			FromName:  cfg.AWS.Ses.FromName,
		},
	})
	if err != nil {
		log.Fatalf("failed to create aws module: %v", err)
	}

	httpClient := &http.Client{Timeout: 10 * time.Second}
	googleModule := chi_google.New(chi_google.Config{
		OAuth: &chi_google_oauth.OAuthConfig{
			ClientID:     cfg.Google.OAuth.ClientID,
			ClientSecret: cfg.Google.OAuth.ClientSecret,
			RedirectURL:  cfg.Google.OAuth.RedirectURL,
		},
		Maps: &chi_google_maps.MapsConfig{
			APIKey: cfg.Google.Maps.APIKey,
		},
		Logger:     logger,
		HTTPClient: httpClient,
	})

	paddleModule, err := chi_paddle.New(chi_paddle.Config{
		APIKey:      cfg.Paddle.APIKey,
		Environment: chi_paddle.PaddleEnvironment(cfg.Paddle.Environment),
	})
	if err != nil {
		log.Fatalf("failed to create paddle module: %v", err)
	}

	locationSrv := location_service.New(location_service.Dependencies{
		Logger: logger,
		Maps:   googleModule.Maps,
	})

	auditSrv := audit_service.New(audit_service.Dependencies{
		GenerateID:   uuid.NewV7,
		Now:          now,
		Validator:    appValidator,
		Logger:       logger,
		Authorizer:   authorizer,
		Repositories: repos,
	})

	billingSrv := billing_service.New(billing_service.Dependencies{
		Validator:           appValidator,
		Logger:              logger,
		Authorizer:          authorizer,
		Repositories:        repos,
		PaddleCustomer:      paddleModule.Customer,
		PaddleSubscription:  paddleModule.Subscription,
		PaddleTransaction:   paddleModule.Transaction,
		PaddleWebhookSecret: cfg.Paddle.WebhookSecret,
		PriceCatalog: billing_service.PriceCatalog{
			PriceIDs: billing_service.PriceIDs{
				BasicMonthly: constants.PRICE_ID_BASIC_MONTHLY,
				BasicAnnual:  constants.PRICE_ID_BASIC_ANNUAL,
				ProMonthly:   constants.PRICE_ID_PRO_MONTHLY,
				ProAnnual:    constants.PRICE_ID_PRO_ANNUAL,
			},
		},
		AuditService: auditSrv,
	})

	invitationSrv := invitation_service.New(invitation_service.Dependencies{
		InvitationTTL:  constants.INVITATION_TOKEN_TTL,
		AppURL:         cfg.App.WebURL,
		GenerateID:     uuid.NewV7,
		Now:            now,
		Validator:      appValidator,
		Logger:         logger,
		GenerateToken:  chi_token.GenerateOpaqueToken,
		HashToken:      tokenHasher.Hash,
		Authorizer:     authorizer,
		Repositories:   repos,
		RunInTx:        repos.RunInTx,
		SessionCache:   sessionCache,
		Localizer:      appLocalizer,
		EmailSender:    awsModule.SES,
		EmailTemplates: emailTemplates,
	})

	return &Services{
		Audit: auditSrv,
		Auth: auth_service.New(auth_service.Dependencies{
			GenerateID:        uuid.NewV7,
			Now:               now,
			Validator:         appValidator,
			Logger:            logger,
			PasswordHashFn:    chi_password.Hash,
			GenerateToken:     chi_token.GenerateOpaqueToken,
			HashToken:         tokenHasher.Hash,
			Authorizer:        authorizer,
			Repositories:      repos,
			RunInTx:           repos.RunInTx,
			SessionCache:      sessionCache,
			AccessTokenSecret: cfg.Auth.AccessTokenSecret,
			AppURL:            cfg.App.WebURL,
			Localizer:         appLocalizer,
			EmailSender:       awsModule.SES,
			EmailTemplates:    emailTemplates,
			GoogleOAuth:       googleModule.OAuth,
		}),
		APIKey: api_key_service.New(api_key_service.Dependencies{
			GenerateID:   uuid.NewV7,
			Now:          now,
			Validator:    appValidator,
			Logger:       logger,
			Authorizer:   authorizer,
			Repositories: repos,
			AuditService: auditSrv,
		}),
		Billing:    billingSrv,
		Invitation: invitationSrv,
		Location:   locationSrv,
		Organization: organization_service.New(organization_service.Dependencies{
			GenerateID:         uuid.NewV7,
			Now:                now,
			Validator:          appValidator,
			Logger:             logger,
			Authorizer:         authorizer,
			Repositories:       repos,
			SessionCache:       sessionCache,
			RunInTx:            repos.RunInTx,
			AuditService:       auditSrv,
			BillingService:     billingSrv,
			LocationService:    locationSrv,
			InvitationsService: invitationSrv,
		}),
		Role: role_service.New(role_service.Dependencies{
			GenerateID:   uuid.NewV7,
			Now:          now,
			Validator:    appValidator,
			Logger:       logger,
			Authorizer:   authorizer,
			Repositories: repos,
			AuditService: auditSrv,
			SessionCache: sessionCache,
		}),
		Support: support_service.New(support_service.Dependencies{
			Validator:         appValidator,
			Logger:            logger,
			EmailSender:       awsModule.SES,
			EmailTemplates:    emailTemplates,
			SupportInboxEmail: cfg.Support.InboxEmail,
		}),
		Team: team_service.New(team_service.Dependencies{
			GenerateID:   uuid.NewV7,
			Now:          now,
			Validator:    appValidator,
			Logger:       logger,
			Authorizer:   authorizer,
			Repositories: repos,
			AuditService: auditSrv,
		}),
		User: user_service.New(user_service.Dependencies{
			GenerateID:     uuid.NewV7,
			Now:            now,
			Validator:      appValidator,
			Logger:         logger,
			PasswordHashFn: chi_password.Hash,
			GenerateToken:  chi_token.GenerateOpaqueToken,
			HashToken:      tokenHasher.Hash,
			Authorizer:     authorizer,
			Repositories:   repos,
			SessionCache:   sessionCache,
			AppURL:         cfg.App.WebURL,
			Localizer:      appLocalizer,
			EmailSender:    awsModule.SES,
			EmailTemplates: emailTemplates,
		}),
	}
}
