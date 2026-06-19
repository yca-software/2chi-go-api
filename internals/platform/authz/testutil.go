package authz

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// NewTestSessionCache returns a SessionCache backed by miniredis for unit tests.
func NewTestSessionCache(t *testing.T, ttl time.Duration) *SessionCache {
	t.Helper()
	mr := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })
	return NewSessionCache(redisClient, ttl)
}
