package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DoMinhHHung/go-app/identity-service/internal/domain/entity"
	domainRepo "github.com/DoMinhHHung/go-app/identity-service/internal/domain/repository"
)

type userRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *userRepo {
	return &userRepo{pool: pool}
}

func (r *userRepo) Create(ctx context.Context, u *entity.User) error {
	query := `
		INSERT INTO users (id, email_address, phone_number, full_name, password, role, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
	`
	_, err := r.pool.Exec(ctx, query,
		u.ID,
		u.EmailAddress,
		u.PhoneNumber,
		u.FullName,
		u.Password,
		u.Role,
		u.Status,
	)
	if err != nil {
		if pgErr, ok := getUniqueViolation(err); ok {
			if pgErr.ConstraintName == "idx_users_phone_active" {
				return domainRepo.ErrPhoneConflict
			}
			return domainRepo.ErrEmailConflict
		}
		return fmt.Errorf("user_repo: create: %w", err)
	}
	return nil
}

func (r *userRepo) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	query := `
		SELECT id, email_address, phone_number, full_name, password, role, status, created_at, updated_at, deleted_at
		FROM users
		WHERE email_address = $1 AND deleted_at IS NULL
		LIMIT 1
	`
	row := r.pool.QueryRow(ctx, query, email)
	return scanUser(row)
}

func (r *userRepo) FindByID(ctx context.Context, id string) (*entity.User, error) {
	query := `
		SELECT id, email_address, phone_number, full_name, password, role, status, created_at, updated_at, deleted_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
		LIMIT 1
	`
	row := r.pool.QueryRow(ctx, query, id)
	return scanUser(row)
}

func (r *userRepo) ExistsActiveEmail(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email_address = $1 AND deleted_at IS NULL)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, email).Scan(&exists)
	return exists, err
}

func (r *userRepo) DeleteByID(ctx context.Context, id string) error {
	query := `UPDATE users SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("user_repo: delete: %w", err)
	}
	return nil
}

func (r *userRepo) ActivateByEmail(ctx context.Context, email string) error {
	query := `UPDATE users SET status = $1, updated_at = NOW() WHERE email_address = $2 AND deleted_at IS NULL`
	_, err := r.pool.Exec(ctx, query, entity.StatusActive, email)
	if err != nil {
		return fmt.Errorf("user_repo: activate: %w", err)
	}
	return nil
}

func scanUser(row pgx.Row) (*entity.User, error) {
	var u entity.User
	err := row.Scan(
		&u.ID,
		&u.EmailAddress,
		&u.PhoneNumber,
		&u.FullName,
		&u.Password,
		&u.Role,
		&u.Status,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domainRepo.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("user_repo: scan: %w", err)
	}
	return &u, nil
}

func getUniqueViolation(err error) (*pgconn.PgError, bool) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return pgErr, true
	}
	return nil, false
}
