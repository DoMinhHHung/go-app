package usecase

import "context"

type AuthUsecase interface {
	Register(ctx context.Context, input RegisterInput) error
	VerifyOTP(ctx context.Context, email, code string) error
	ResendOTP(ctx context.Context, email string) error
}

type RegisterInput struct {
	EmailAddress string
	FullName     string
	Password     string
	PhoneNumber  *string
}
