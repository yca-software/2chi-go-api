package subscription

import (
	"errors"
	"time"

	"github.com/yca-software/2chi-go-api/internals/constants"
	chi_error "github.com/yca-software/2chi-go-error"
)

// AuditRetentionDays returns how far back audit logs may be queried for a plan tier.
// Must match react-spa getAuditRetentionDays in subscriptionCapabilities.ts.
func AuditRetentionDays(subscriptionType string) int {
	switch subscriptionType {
	case constants.TIER_BASIC:
		return 180
	case constants.TIER_PRO:
		return 365
	case constants.TIER_ENTERPRISE:
		return 365 * 3
	default:
		return 30
	}
}

// AuditLogMinStartDate returns the earliest created_at filter for audit log queries.
func AuditLogMinStartDate(subscriptionType string, now time.Time) time.Time {
	return now.AddDate(0, 0, -AuditRetentionDays(subscriptionType))
}

// IsUnlimitedSubscriptionSeats reports whether the organization has no seat cap (enterprise).
func IsUnlimitedSubscriptionSeats(subscriptionSeats int) bool {
	return subscriptionSeats == constants.SUBSCRIPTION_TYPE_SEATS_INCLUDED_ENTERPRISE
}

// ValidateSubscriptionSeats rejects invalid seat counts for a subscription tier.
func ValidateSubscriptionSeats(subscriptionSeats int, subscriptionType string) error {
	if subscriptionSeats == 0 {
		return chi_error.NewUnprocessableEntityError(errors.New("subscription seats must be positive or unlimited"), "InvalidSubscriptionSeats", nil)
	}
	if IsUnlimitedSubscriptionSeats(subscriptionSeats) && subscriptionType != constants.TIER_ENTERPRISE {
		return chi_error.NewUnprocessableEntityError(errors.New("unlimited seats require enterprise plan"), "InvalidSubscriptionSeats", nil)
	}
	return nil
}

// OrganizationAtSeatLimit reports whether adding another member would exceed subscription seats.
func OrganizationAtSeatLimit(memberCount, subscriptionSeats int) bool {
	if IsUnlimitedSubscriptionSeats(subscriptionSeats) {
		return false
	}
	return memberCount >= subscriptionSeats
}

// IsAccessBlocked reports whether API access should be denied due to subscription expiry.
// customSubscription orgs (enterprise deals) are never blocked by expiry.
func IsAccessBlocked(subscriptionType string, customSubscription bool, expiresAt *time.Time, now time.Time) bool {
	if customSubscription {
		return false
	}
	if subscriptionType == constants.TIER_ENTERPRISE {
		return false
	}
	if subscriptionType == constants.TIER_FREE {
		return false
	}
	if expiresAt == nil || !expiresAt.Before(now) {
		return false
	}
	if subscriptionType != constants.TIER_BASIC && subscriptionType != constants.TIER_PRO {
		return true
	}
	graceEnd := expiresAt.Add(time.Duration(constants.SUBSCRIPTION_PAST_DUE_GRACE_DAYS) * 24 * time.Hour)
	return !now.Before(graceEnd)
}
