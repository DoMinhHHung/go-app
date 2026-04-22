CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS users (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    email_address VARCHAR(255) NOT NULL,
    phone_number  VARCHAR(20),
    full_name     VARCHAR(255) NOT NULL,
    password      VARCHAR(255) NOT NULL,
    role          VARCHAR(20)  NOT NULL DEFAULT 'user'
                               CHECK (role IN ('admin', 'user')),
    status        VARCHAR(20)  NOT NULL DEFAULT 'active'
                               CHECK (status IN ('active', 'banned')),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_users_email_active
    ON users(email_address)
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX idx_users_phone_active
    ON users(phone_number)
    WHERE deleted_at IS NULL AND phone_number IS NOT NULL;

CREATE INDEX idx_users_email      ON users(email_address);
CREATE INDEX idx_users_created_at ON users(created_at);