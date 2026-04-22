package repository

import (
	"context"

	"github.com/DoMinhHHung/go-app/identity-service/internal/domain/entity"
)

type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	FindByID(ctx context.Context, id string) (*entity.User, error)
	ExistsActiveEmail(ctx context.Context, email string) (bool, error)
}
