package dto

type RegisterRequest struct {
	EmailAddress string  `json:"email_address" binding:"required,email"`
	FullName     string  `json:"full_name"     binding:"required,min=2,max=100"`
	Password     string  `json:"password"      binding:"required,min=8"`
	PhoneNumber  *string `json:"phone_number"`
}

type VerifyOTPRequest struct {
	EmailAddress string `json:"email_address" binding:"required,email"`
	OTPCode      string `json:"otp_code"      binding:"required,len=6"`
}

type ResendOTPRequest struct {
	EmailAddress string `json:"email_address" binding:"required,email"`
}
