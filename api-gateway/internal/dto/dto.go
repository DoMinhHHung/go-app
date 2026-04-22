package dto

// SuccessResponse là response khi request thành công.
type SuccessResponse struct {
	Success bool   `json:"success" example:"true"`
	Message string `json:"message" example:"operation successful"`
	Data    any    `json:"data,omitempty"`
}

// ErrorResponse là response khi có lỗi.
type ErrorResponse struct {
	Success bool   `json:"success" example:"false"`
	Message string `json:"message" example:"error description"`
	Code    string `json:"code"    example:"ERROR_CODE"`
}

// HealthResponse là response của health check endpoint.
type HealthResponse struct {
	Status  string `json:"status"  example:"ok"`
	Service string `json:"service" example:"api-gateway"`
	Env     string `json:"env"     example:"development"`
}

// — Auth DTOs (mirrors identity-service) ——————————————————————————————————————

type RegisterRequest struct {
	EmailAddress string  `json:"email_address" example:"user@example.com"`
	FullName     string  `json:"full_name"     example:"Nguyen Van A"`
	Password     string  `json:"password"      example:"Secret123"`
	PhoneNumber  *string `json:"phone_number"  example:"+84912345678"`
}

type VerifyOTPRequest struct {
	EmailAddress string `json:"email_address" example:"user@example.com"`
	OTPCode      string `json:"otp_code"      example:"123456"`
}

type ResendOTPRequest struct {
	EmailAddress string `json:"email_address" example:"user@example.com"`
}

type LoginRequest struct {
	EmailAddress string `json:"email_address" example:"user@example.com"`
	Password     string `json:"password"      example:"Secret123"`
}

type LoginData struct {
	AccessToken  string `json:"access_token"  example:"eyJhbGci..."`
	RefreshToken string `json:"refresh_token" example:"eyJhbGci..."`
	ExpiresIn    int64  `json:"expires_in"    example:"900"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" example:"eyJhbGci..."`
}

type RefreshData struct {
	AccessToken string `json:"access_token" example:"eyJhbGci..."`
	ExpiresIn   int64  `json:"expires_in"   example:"900"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" example:"eyJhbGci..."`
}
