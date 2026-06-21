package subscription_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yca-software/2chi-go-api/internals/constants"
	platform_subscription "github.com/yca-software/2chi-go-api/internals/packages/subscription"
	chi_error "github.com/yca-software/2chi-go-error"
)

func TestAuditRetentionDays(t *testing.T) {
	t.Parallel()

	require.Equal(t, 30, platform_subscription.AuditRetentionDays(constants.TIER_FREE))
	require.Equal(t, 30, platform_subscription.AuditRetentionDays("unknown"))
	require.Equal(t, 180, platform_subscription.AuditRetentionDays(constants.TIER_BASIC))
	require.Equal(t, 365, platform_subscription.AuditRetentionDays(constants.TIER_PRO))
	require.Equal(t, 365*3, platform_subscription.AuditRetentionDays(constants.TIER_ENTERPRISE))
}

func TestAuditLogMinStartDate(t *testing.T) {
	t.Parallel()

	now := time.Date(2024, 6, 10, 12, 0, 0, 0, time.UTC)

	require.Equal(t, now.AddDate(0, 0, -30), platform_subscription.AuditLogMinStartDate(constants.TIER_FREE, now))
	require.Equal(t, now.AddDate(0, 0, -180), platform_subscription.AuditLogMinStartDate(constants.TIER_BASIC, now))
	require.Equal(t, now.AddDate(0, 0, -365), platform_subscription.AuditLogMinStartDate(constants.TIER_PRO, now))
	require.Equal(t, now.AddDate(0, 0, -365*3), platform_subscription.AuditLogMinStartDate(constants.TIER_ENTERPRISE, now))
}

func TestOrganizationAtSeatLimit(t *testing.T) {
	t.Parallel()

	require.False(t, platform_subscription.OrganizationAtSeatLimit(0, 5))
	require.False(t, platform_subscription.OrganizationAtSeatLimit(4, 5))
	require.True(t, platform_subscription.OrganizationAtSeatLimit(5, 5))
	require.False(t, platform_subscription.OrganizationAtSeatLimit(100, constants.SUBSCRIPTION_TYPE_SEATS_INCLUDED_ENTERPRISE))
}

func TestIsUnlimitedSubscriptionSeats(t *testing.T) {
	t.Parallel()

	require.True(t, platform_subscription.IsUnlimitedSubscriptionSeats(constants.SUBSCRIPTION_TYPE_SEATS_INCLUDED_ENTERPRISE))
	require.False(t, platform_subscription.IsUnlimitedSubscriptionSeats(10))
}

func TestValidateSubscriptionSeats(t *testing.T) {
	t.Parallel()

	require.NoError(t, platform_subscription.ValidateSubscriptionSeats(5, constants.TIER_BASIC))
	require.NoError(t, platform_subscription.ValidateSubscriptionSeats(constants.SUBSCRIPTION_TYPE_SEATS_INCLUDED_ENTERPRISE, constants.TIER_ENTERPRISE))

	err := platform_subscription.ValidateSubscriptionSeats(0, constants.TIER_BASIC)
	require.Error(t, err)
	var apiErr *chi_error.Error
	require.True(t, errors.As(err, &apiErr))
	require.Equal(t, "InvalidSubscriptionSeats", apiErr.ErrorCode)

	err = platform_subscription.ValidateSubscriptionSeats(constants.SUBSCRIPTION_TYPE_SEATS_INCLUDED_ENTERPRISE, constants.TIER_PRO)
	require.Error(t, err)
	require.True(t, errors.As(err, &apiErr))
	require.Equal(t, "InvalidSubscriptionSeats", apiErr.ErrorCode)
}

func TestIsAccessBlocked(t *testing.T) {
	t.Parallel()

	now := time.Date(2024, 6, 10, 12, 0, 0, 0, time.UTC)
	expired := now.Add(-24 * time.Hour)
	active := now.Add(24 * time.Hour)

	require.False(t, platform_subscription.IsAccessBlocked(constants.TIER_FREE, false, &expired, now))
	require.False(t, platform_subscription.IsAccessBlocked(constants.TIER_BASIC, true, &expired, now))
	require.False(t, platform_subscription.IsAccessBlocked(constants.TIER_ENTERPRISE, false, &expired, now))

	require.False(t, platform_subscription.IsAccessBlocked(constants.TIER_BASIC, false, &active, now))
	require.False(t, platform_subscription.IsAccessBlocked(constants.TIER_BASIC, false, &expired, now))
	require.False(t, platform_subscription.IsAccessBlocked(constants.TIER_PRO, false, &expired, now))

	beyondGrace := expired.Add(-time.Duration(constants.SUBSCRIPTION_PAST_DUE_GRACE_DAYS) * 24 * time.Hour)
	require.True(t, platform_subscription.IsAccessBlocked(constants.TIER_BASIC, false, &beyondGrace, now))
	require.True(t, platform_subscription.IsAccessBlocked(constants.TIER_PRO, false, &beyondGrace, now))
}
