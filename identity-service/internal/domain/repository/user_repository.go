package repository

import (
	"context"
	"errors"

	"github.com/DoMinhHHung/go-app/identity-service/internal/domain/entity"
)

var (
	ErrEmailConflict = errors.New("domain: email already exists")
	ErrPhoneConflict = errors.New("domain: phone number already exists")
	ErrUserNotFound  = errors.New("domain: user not found")
)

type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	FindByID(ctx context.Context, id string) (*entity.User, error)
	ExistsActiveEmail(ctx context.Context, email string) (bool, error)
	DeleteByID(ctx context.Context, id string) error
	ActivateByEmail(ctx context.Context, email string) error
}
