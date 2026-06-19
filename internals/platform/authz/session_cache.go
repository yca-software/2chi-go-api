package authz

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	chi_types "github.com/yca-software/2chi-go-types"
)

// SessionCache stores serialized AccessInfo in Redis for JWT auth.
type SessionCache struct {
	redis *redis.Client
	ttl   time.Duration
}

func NewSessionCache(rdb *redis.Client, ttl time.Duration) *SessionCache {
	return &SessionCache{redis: rdb, ttl: ttl}
}

func SessionKey(userID string) string {
	return "session:" + userID
}

func RevokedSessionKey(userID string) string {
	return "session:revoked:" + userID
}

func (c *SessionCache) TTL() time.Duration {
	return c.ttl
}

func (c *SessionCache) Get(ctx context.Context, userID string) (*chi_types.AccessInfo, bool) {
	if c == nil || c.redis == nil {
		return nil, false
	}
	cached, err := c.redis.Get(ctx, SessionKey(userID)).Result()
	if err != nil || cached == "" {
		return nil, false
	}
	var access chi_types.AccessInfo
	if err := json.Unmarshal([]byte(cached), &access); err != nil {
		return nil, false
	}
	return &access, true
}

func (c *SessionCache) Set(ctx context.Context, access *chi_types.AccessInfo) error {
	if c == nil || c.redis == nil || access == nil {
		return nil
	}
	sessionBytes, err := json.Marshal(access)
	if err != nil {
		return err
	}
	userID := access.SubjectID.String()
	if err := c.redis.Set(ctx, SessionKey(userID), sessionBytes, c.ttl).Err(); err != nil {
		return err
	}
	return c.ClearRevoked(ctx, userID)
}

func (c *SessionCache) InvalidateSession(ctx context.Context, userID string) error {
	if c == nil || c.redis == nil || userID == "" {
		return nil
	}
	if err := c.redis.Del(ctx, SessionKey(userID)).Err(); err != nil {
		return err
	}
	return c.MarkRevoked(ctx, userID)
}

// MarkRevoked blocks JWT bootstrap until access token TTL expires (logout / password reset).
func (c *SessionCache) MarkRevoked(ctx context.Context, userID string) error {
	if c == nil || c.redis == nil || userID == "" {
		return nil
	}
	return c.redis.Set(ctx, RevokedSessionKey(userID), "1", c.ttl).Err()
}

func (c *SessionCache) ClearRevoked(ctx context.Context, userID string) error {
	if c == nil || c.redis == nil || userID == "" {
		return nil
	}
	return c.redis.Del(ctx, RevokedSessionKey(userID)).Err()
}

func (c *SessionCache) IsRevoked(ctx context.Context, userID string) bool {
	if c == nil || c.redis == nil || userID == "" {
		return false
	}
	n, err := c.redis.Exists(ctx, RevokedSessionKey(userID)).Result()
	return err == nil && n > 0
}
