-- Add authentication columns to users and create refresh_tokens table.

ALTER TABLE users ADD COLUMN password_hash TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE users ADD COLUMN last_login_at TIMESTAMPTZ;

-- Note: created_at already exists from 001_initial_schema.up.sql, so we skip it.

CREATE TABLE refresh_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_refresh_tokens_user ON refresh_tokens (user_id);

-- Seed: set default dev user password to bcrypt hash of "clotho123"
UPDATE users SET password_hash = '$2a$10$iUxo3ZRU09xm5htjvRSraO2.Lx9IwbyHNiqTklmQtREKavK/1i/d2'
WHERE email = 'admin@clotho.dev';
