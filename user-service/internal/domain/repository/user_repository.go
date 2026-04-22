package repository

import (
	"context"
	"errors"

	"github.com/DoMinhHHung/go-app/user-service/internal/domain/entity"
)

var (
	ErrUserNotFound  = errors.New("domain: user not found")
	ErrEmailConflict = errors.New("domain: email already exists")
)

type UserRepository interface {
	Upsert(ctx context.Context, user *entity.User) error
	FindByID(ctx context.Context, id string) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	Update(ctx context.Context, user *entity.User) error
	SoftDelete(ctx context.Context, id string) error
	ListActive(ctx context.Context, limit, offset int) ([]*entity.User, error)
}
