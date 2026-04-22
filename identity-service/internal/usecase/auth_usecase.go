package usecase

import "context"

type AuthUsecase interface {
	Register(ctx context.Context, input RegisterInput) error
	VerifyOTP(ctx context.Context, email, code string) error
	ResendOTP(ctx context.Context, email string) error
	Login(ctx context.Context, input LoginInput) (LoginOutput, error)
	RefreshToken(ctx context.Context, refreshToken string) (RefreshOutput, error)
	Logout(ctx context.Context, refreshToken string) error
}

type RegisterInput struct {
	EmailAddress string
	FullName     string
	Password     string
	PhoneNumber  *string
}

type LoginInput struct {
	EmailAddress string
	Password     string
}

type LoginOutput struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

type RefreshOutput struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}
