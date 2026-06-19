package support_service

type SubmitSupportRequest struct {
	Subject   string `json:"subject"`
	Message   string `json:"message" validate:"required"`
	PageURL   string `json:"pageUrl,omitempty" validate:"omitempty,url"`
	UserAgent string `json:"userAgent,omitempty"`
}
