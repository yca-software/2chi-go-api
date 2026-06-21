package oauth

import (
	"strings"

	chi_google_oauth "github.com/yca-software/2chi-go-google/oauth"
)

func GoogleUserNames(googleUser *chi_google_oauth.UserInfo) (firstName, lastName string) {
	firstName = strings.TrimSpace(googleUser.GivenName)
	lastName = strings.TrimSpace(googleUser.FamilyName)
	if firstName != "" || lastName != "" {
		return firstName, lastName
	}

	name := strings.TrimSpace(googleUser.Name)
	if name == "" {
		return "", ""
	}

	parts := strings.SplitN(name, " ", 2)
	firstName = parts[0]
	if len(parts) > 1 {
		lastName = strings.TrimSpace(parts[1])
	}
	return firstName, lastName
}
