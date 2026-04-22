package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DoMinhHHung/go-app/user-service/internal/domain/entity"
	domainRepo "github.com/DoMinhHHung/go-app/user-service/internal/domain/repository"
)

type userRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *userRepo {
	return &userRepo{pool: pool}
}

func (r *userRepo) Upsert(ctx context.Context, u *entity.User) error {
	query := `
		INSERT INTO users (id, email_address, phone_number, full_name, role, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE
		SET email_address = EXCLUDED.email_address,
			phone_number = EXCLUDED.phone_number,
			full_name = EXCLUDED.full_name,
			role = EXCLUDED.role,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at
	`
	_, err := r.pool.Exec(ctx, query,
		u.ID,
		u.EmailAddress,
		u.PhoneNumber,
		u.FullName,
		u.Role,
		u.Status,
		u.CreatedAt,
		u.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("user_repo: upsert: %w", err)
	}
	return nil
}

func (r *userRepo) FindByID(ctx context.Context, id string) (*entity.User, error) {
	query := `
        SELECT id, email_address, phone_number, full_name, role, status, created_at, updated_at, deleted_at
        FROM users
        WHERE id = $1 AND deleted_at IS NULL
    `
	return scanUser(r.pool.QueryRow(ctx, query, id))
}

func (r *userRepo) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	query := `
        SELECT id, email_address, phone_number, full_name, role, status, created_at, updated_at, deleted_at
        FROM users
        WHERE email_address = $1 AND deleted_at IS NULL
    `
	return scanUser(r.pool.QueryRow(ctx, query, email))
}

func (r *userRepo) Update(ctx context.Context, u *entity.User) error {
	query := `
        UPDATE users
        SET full_name = $1, phone_number = $2, updated_at = NOW()
        WHERE id = $3 AND deleted_at IS NULL
    `
	_, err := r.pool.Exec(ctx, query, u.FullName, u.PhoneNumber, u.ID)
	if err != nil {
		return fmt.Errorf("user_repo: update: %w", err)
	}
	return nil
}

func (r *userRepo) SoftDelete(ctx context.Context, id string) error {
	query := `UPDATE users SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("user_repo: soft delete: %w", err)
	}
	return nil
}

func (r *userRepo) ListActive(ctx context.Context, limit, offset int) ([]*entity.User, error) {
	query := `
        SELECT id, email_address, phone_number, full_name, role, status, created_at, updated_at, deleted_at
        FROM users
        WHERE deleted_at IS NULL AND status = 'active'
        ORDER BY created_at DESC
        LIMIT $1 OFFSET $2
    `
	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("user_repo: list: %w", err)
	}
	defer rows.Close()

	var users []*entity.User
	for rows.Next() {
		var u entity.User
		if err := rows.Scan(
			&u.ID, &u.EmailAddress, &u.PhoneNumber, &u.FullName,
			&u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("user_repo: scan: %w", err)
		}
		users = append(users, &u)
	}
	return users, rows.Err()
}

func scanUser(row pgx.Row) (*entity.User, error) {
	var u entity.User
	err := row.Scan(
		&u.ID, &u.EmailAddress, &u.PhoneNumber, &u.FullName,
		&u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domainRepo.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("user_repo: scan: %w", err)
	}
	return &u, nil
}
