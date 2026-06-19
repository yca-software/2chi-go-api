package i18n_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yca-software/2chi-go-api/internals/constants"
	platform_i18n "github.com/yca-software/2chi-go-api/internals/platform/i18n"
)

func TestNormalizeLanguage(t *testing.T) {
	t.Parallel()

	require.Equal(t, constants.DEFAULT_LANGUAGE, platform_i18n.NormalizeLanguage(""))
	require.Equal(t, constants.DEFAULT_LANGUAGE, platform_i18n.NormalizeLanguage("  "))
	require.Equal(t, constants.DEFAULT_LANGUAGE, platform_i18n.NormalizeLanguage("fr"))
	require.Equal(t, "en", platform_i18n.NormalizeLanguage("EN"))
	require.Equal(t, "en", platform_i18n.NormalizeLanguage(" en "))
}
