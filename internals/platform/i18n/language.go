package i18n

import (
	"slices"
	"strings"

	"github.com/yca-software/2chi-go-api/internals/constants"
)

func NormalizeLanguage(language string) string {
	lang := strings.ToLower(strings.TrimSpace(language))
	if lang == "" || !slices.Contains(constants.SUPPORTED_LANGUAGES, lang) {
		return constants.DEFAULT_LANGUAGE
	}
	return lang
}
