package auth_service

type AuthenticateResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type AuthenticateWithPasswordRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required"`
	IPAddress string `json:"ipAddress" validate:"required,ip"`
	UserAgent string `json:"userAgent" validate:"required"`
}

type AuthenticateWithGoogleRequest struct {
	Code            string `json:"code" validate:"required"`
	TermsVersion    string `json:"termsVersion" validate:"required,semver"`
	InvitationToken string `json:"invitationToken"`
	IPAddress       string `json:"ipAddress" validate:"required,ip"`
	UserAgent       string `json:"userAgent" validate:"required"`
	Language        string `json:"language" validate:"required,len=2"`
}

type ForgotPasswordRequest struct {
	Language string `json:"language" validate:"required,len=2"`
	Email    string `json:"email" validate:"required,email"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
}

type RefreshAccessTokenRequest struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
	IPAddress    string `json:"ipAddress" validate:"required,ip"`
	UserAgent    string `json:"userAgent" validate:"required"`
}

type RefreshAccessTokenResponse struct {
	AccessToken string `json:"accessToken"`
}

type ResetPasswordRequest struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}

type SignUpRequest struct {
	Email           string `json:"email" validate:"required,email"`
	Password        string `json:"password" validate:"required"`
	FirstName       string `json:"firstName" validate:"required"`
	LastName        string `json:"lastName" validate:"required"`
	Language        string `json:"-" validate:"required,len=2"`
	IPAddress       string `json:"ipAddress" validate:"required,ip"`
	UserAgent       string `json:"userAgent" validate:"required"`
	TermsVersion    string `json:"termsVersion" validate:"required,semver"`
	InvitationToken string `json:"invitationToken"`
}

type SignUpResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type VerifyEmailRequest struct {
	Token string `json:"token" validate:"required"`
}

type ImpersonateRequest struct {
	UserID    string `json:"-" validate:"required,uuid"`
	IPAddress string `json:"ipAddress" validate:"required,ip"`
	UserAgent string `json:"userAgent" validate:"required"`
	RequestID string `json:"-"`
}
