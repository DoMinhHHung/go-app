package infra

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// migrations is the ordered list of schema changes.
// Never modify an existing entry — always append a new one.
var migrations = []struct {
	version int
	sql     string
}{
	{1, `
		CREATE EXTENSION IF NOT EXISTS "pgcrypto";

		CREATE TABLE IF NOT EXISTS users (
			id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
			email_address VARCHAR(255) NOT NULL,
			phone_number  VARCHAR(20),
			full_name     VARCHAR(255) NOT NULL,
			password      VARCHAR(255) NOT NULL,
			role          VARCHAR(20)  NOT NULL DEFAULT 'user'
			                           CHECK (role IN ('admin', 'user')),
			status        VARCHAR(20)  NOT NULL DEFAULT 'pending'
			                           CHECK (status IN ('pending', 'active', 'banned')),
			created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
			updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
			deleted_at    TIMESTAMPTZ
		);

		CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_active
			ON users(email_address) WHERE deleted_at IS NULL;

		CREATE UNIQUE INDEX IF NOT EXISTS idx_users_phone_active
			ON users(phone_number)
			WHERE deleted_at IS NULL AND phone_number IS NOT NULL;

		CREATE INDEX IF NOT EXISTS idx_users_email      ON users(email_address);
		CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at);
	`},
	{2, `
    CREATE INDEX IF NOT EXISTS idx_users_email_status 
        ON users(email_address, status) 
        WHERE deleted_at IS NULL;
`},
}

func RunMigrations(ctx context.Context, pool *pgxpool.Pool, logger *zap.Logger) error {
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    INTEGER     PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return fmt.Errorf("migrate: init tracking table: %w", err)
	}

	for _, m := range migrations {
		var applied bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`,
			m.version,
		).Scan(&applied); err != nil {
			return fmt.Errorf("migrate: check v%d: %w", m.version, err)
		}
		if applied {
			continue
		}

		if _, err := pool.Exec(ctx, m.sql); err != nil {
			return fmt.Errorf("migrate: apply v%d: %w", m.version, err)
		}

		if _, err := pool.Exec(ctx,
			`INSERT INTO schema_migrations (version) VALUES ($1)`, m.version,
		); err != nil {
			return fmt.Errorf("migrate: record v%d: %w", m.version, err)
		}

		logger.Info("migration applied", zap.Int("version", m.version))
	}

	return nil
}
