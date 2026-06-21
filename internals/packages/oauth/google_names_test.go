package oauth_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	platform_oauth "github.com/yca-software/2chi-go-api/internals/packages/oauth"
	chi_google_oauth "github.com/yca-software/2chi-go-google/oauth"
)

func TestGoogleUserNames(t *testing.T) {
	t.Parallel()

	first, last := platform_oauth.GoogleUserNames(&chi_google_oauth.UserInfo{
		GivenName:  "Ada",
		FamilyName: "Lovelace",
	})
	require.Equal(t, "Ada", first)
	require.Equal(t, "Lovelace", last)

	first, last = platform_oauth.GoogleUserNames(&chi_google_oauth.UserInfo{
		Name: "Grace Hopper",
	})
	require.Equal(t, "Grace", first)
	require.Equal(t, "Hopper", last)

	first, last = platform_oauth.GoogleUserNames(&chi_google_oauth.UserInfo{
		Name: "Madonna",
	})
	require.Equal(t, "Madonna", first)
	require.Equal(t, "", last)

	first, last = platform_oauth.GoogleUserNames(&chi_google_oauth.UserInfo{})
	require.Equal(t, "", first)
	require.Equal(t, "", last)
}
