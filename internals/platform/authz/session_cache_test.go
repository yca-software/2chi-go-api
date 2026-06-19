package authz_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
	"github.com/yca-software/2chi-go-api/internals/platform/authz"
	chi_types "github.com/yca-software/2chi-go-types"
)

type SessionCacheSuite struct {
	suite.Suite
	mr    *miniredis.Miniredis
	redis *redis.Client
	cache *authz.SessionCache
	ctx   context.Context
}

func TestSessionCacheSuite(t *testing.T) {
	suite.Run(t, new(SessionCacheSuite))
}

func (s *SessionCacheSuite) SetupTest() {
	s.ctx = context.Background()
	s.mr = miniredis.RunT(s.T())
	s.redis = redis.NewClient(&redis.Options{Addr: s.mr.Addr()})
	s.cache = authz.NewSessionCache(s.redis, 15*time.Minute)
}

func (s *SessionCacheSuite) TearDownTest() {
	if s.redis != nil {
		_ = s.redis.Close()
	}
}

func (s *SessionCacheSuite) TestSetGetAndInvalidate() {
	userID := uuid.MustParse("11111111-1111-4111-8111-111111111101")
	access := &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeUser,
		SubjectID: userID,
		Email:     "alice@example.com",
		Roles: []chi_types.JWTAccessTokenPermissionData{{
			OrganizationID: uuid.MustParse("22222222-2222-4222-8222-222222222201"),
			Permissions:    []string{"org:read"},
		}},
	}

	s.NoError(s.cache.Set(s.ctx, access))

	got, ok := s.cache.Get(s.ctx, userID.String())
	s.True(ok)
	s.Require().NotNil(got)
	s.Len(got.Roles, 1)
	s.Equal("org:read", got.Roles[0].Permissions[0])

	s.NoError(s.cache.InvalidateSession(s.ctx, userID.String()))
	_, ok = s.cache.Get(s.ctx, userID.String())
	s.False(ok)
}

func (s *SessionCacheSuite) TestSetGet_PreservesImpersonation() {
	userID := uuid.MustParse("11111111-1111-4111-8111-111111111102")
	adminID := uuid.MustParse("33333333-3333-4333-8333-333333333301")
	access := &chi_types.AccessInfo{
		Type:                chi_types.AccessTypeUser,
		SubjectID:           userID,
		Email:               "impersonated@example.com",
		ImpersonatedBy:      uuid.NullUUID{UUID: adminID, Valid: true},
		ImpersonatedByEmail: "admin@example.com",
	}

	s.NoError(s.cache.Set(s.ctx, access))

	got, ok := s.cache.Get(s.ctx, userID.String())
	s.True(ok)
	s.True(got.ImpersonatedBy.Valid)
	s.Equal(adminID, got.ImpersonatedBy.UUID)
	s.Equal("admin@example.com", got.ImpersonatedByEmail)
}

func (s *SessionCacheSuite) TestInvalidateOnRoleChange_ClearsCachedPermissions() {
	userID := uuid.MustParse("11111111-1111-4111-8111-111111111101")
	access := &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeUser,
		SubjectID: userID,
		Email:     "alice@example.com",
		Roles: []chi_types.JWTAccessTokenPermissionData{{
			OrganizationID: uuid.MustParse("22222222-2222-4222-8222-222222222201"),
			Permissions:    []string{"members:read"},
		}},
	}
	s.NoError(s.cache.Set(s.ctx, access))

	s.NoError(s.cache.InvalidateSession(s.ctx, userID.String()))

	_, ok := s.cache.Get(s.ctx, userID.String())
	s.False(ok)
}

func (s *SessionCacheSuite) TestInvalidateMarksRevokedUntilFreshLogin() {
	userID := uuid.MustParse("11111111-1111-4111-8111-111111111103")
	access := &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeUser,
		SubjectID: userID,
		Email:     "bob@example.com",
	}

	s.NoError(s.cache.Set(s.ctx, access))
	s.False(s.cache.IsRevoked(s.ctx, userID.String()))

	s.NoError(s.cache.InvalidateSession(s.ctx, userID.String()))
	s.True(s.cache.IsRevoked(s.ctx, userID.String()))

	s.NoError(s.cache.Set(s.ctx, access))
	s.False(s.cache.IsRevoked(s.ctx, userID.String()))
}
