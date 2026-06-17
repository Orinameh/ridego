-- +goose Up
-- +goose StatementBegin

-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT        NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    role          TEXT        NOT NULL CHECK (role IN ('rider','driver','admin')),
    full_name     TEXT        NOT NULL DEFAULT '',
    phone         TEXT        NOT NULL DEFAULT '',
    avatar_url    TEXT,
    rating        NUMERIC(3,2) NOT NULL DEFAULT 5.0
                               CHECK (rating BETWEEN 1.0 AND 5.0),
    is_active     BOOLEAN     NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users (email);
CREATE INDEX idx_users_role  ON users (role);

CREATE TABLE refresh_tokens (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT        NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_user_id   ON refresh_tokens (user_id);
CREATE INDEX idx_refresh_token_hash ON refresh_tokens (token_hash);

-- Automatically remove expired tokens (pg_cron or periodic cleanup job)
-- Alternatively add a partial index to speed up expiry checks:
CREATE INDEX idx_refresh_expires ON refresh_tokens (expires_at);

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS refresh_tokens;

-- +goose StatementEnd
