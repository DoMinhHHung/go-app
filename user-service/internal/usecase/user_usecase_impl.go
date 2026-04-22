package usecase

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/DoMinhHHung/go-app/user-service/internal/domain/entity"
	domainRepo "github.com/DoMinhHHung/go-app/user-service/internal/domain/repository"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrForbidden    = errors.New("forbidden")
)

type userUsecaseImpl struct {
	userRepo domainRepo.UserRepository
	logger   *zap.Logger
}

func NewUserUsecase(userRepo domainRepo.UserRepository, logger *zap.Logger) AuthUsecase {
	return &userUsecaseImpl{userRepo: userRepo, logger: logger}
}

func (uc *userUsecaseImpl) GetProfile(ctx context.Context, userID string) (*ProfileOutput, error) {
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, domainRepo.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get profile: %w", err)
	}
	return toProfileOutput(user), nil
}

func (uc *userUsecaseImpl) UpdateProfile(ctx context.Context, userID string, input UpdateProfileInput) (*ProfileOutput, error) {
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, domainRepo.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("update profile: find: %w", err)
	}

	if input.FullName != nil {
		user.FullName = *input.FullName
	}
	if input.PhoneNumber != nil {
		user.PhoneNumber = input.PhoneNumber
	}

	if err := uc.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update profile: save: %w", err)
	}

	uc.logger.Info("user profile updated", zap.String("user_id", userID))
	return toProfileOutput(user), nil
}

func (uc *userUsecaseImpl) SoftDelete(ctx context.Context, userID string) error {
	if _, err := uc.userRepo.FindByID(ctx, userID); err != nil {
		if errors.Is(err, domainRepo.ErrUserNotFound) {
			return ErrUserNotFound
		}
		return fmt.Errorf("soft delete: find: %w", err)
	}

	if err := uc.userRepo.SoftDelete(ctx, userID); err != nil {
		return fmt.Errorf("soft delete: %w", err)
	}

	uc.logger.Info("user soft deleted", zap.String("user_id", userID))
	return nil
}

func (uc *userUsecaseImpl) ListUsers(ctx context.Context, limit, offset int) ([]*ProfileOutput, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	users, err := uc.userRepo.ListActive(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	result := make([]*ProfileOutput, len(users))
	for i, u := range users {
		result[i] = toProfileOutput(u)
	}
	return result, nil
}

func toProfileOutput(u *entity.User) *ProfileOutput {
	return &ProfileOutput{
		ID:           u.ID,
		EmailAddress: u.EmailAddress,
		PhoneNumber:  u.PhoneNumber,
		FullName:     u.FullName,
		Role:         string(u.Role),
		Status:       string(u.Status),
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}
