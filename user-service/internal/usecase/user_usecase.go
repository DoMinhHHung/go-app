package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type AuthUsecase interface {
	GetProfile(ctx context.Context, userID string) (*ProfileOutput, error)
	UpdateProfile(ctx context.Context, userID string, input UpdateProfileInput) (*ProfileOutput, error)
	SoftDelete(ctx context.Context, userID string) error
	ListUsers(ctx context.Context, limit, offset int) ([]*ProfileOutput, error)
}

type ProfileOutput struct {
	ID           uuid.UUID `json:"id"            example:"01932b4a-dead-7abc-8def-123456789abc"`
	EmailAddress string    `json:"email_address" example:"user@example.com"`
	PhoneNumber  *string   `json:"phone_number"  example:"+84912345678"`
	FullName     string    `json:"full_name"     example:"Nguyen Van A"`
	Role         string    `json:"role"          example:"user"`
	Status       string    `json:"status"        example:"active"`
	CreatedAt    time.Time `json:"created_at"    example:"2025-01-01T00:00:00Z"`
	UpdatedAt    time.Time `json:"updated_at"    example:"2025-01-01T00:00:00Z"`
}

type UpdateProfileInput struct {
	FullName    *string
	PhoneNumber *string
}
