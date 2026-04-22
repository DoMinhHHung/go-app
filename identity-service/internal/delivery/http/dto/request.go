package dto

type RegisterRequest struct {
	EmailAddress string  `json:"email_address" binding:"required,email"`
	FullName     string  `json:"full_name"     binding:"required,min=2,max=100"`
	Password     string  `json:"password"      binding:"required,min=8"`
	PhoneNumber  *string `json:"phone_number"  binding:"omitempty,min=7,max=15"`
}

type VerifyOTPRequest struct {
	EmailAddress string `json:"email_address" binding:"required,email"`
	OTPCode      string `json:"otp_code"      binding:"required,len=6"`
}

type ResendOTPRequest struct {
	EmailAddress string `json:"email_address" binding:"required,email"`
}

type LoginRequest struct {
	EmailAddress string `json:"email_address" binding:"required,email"`
	Password     string `json:"password"      binding:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}
