package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RefreshTokenStore implements store.RefreshTokenStore.
type RefreshTokenStore struct {
	pool *pgxpool.Pool
}

func NewRefreshTokenStore(pool *pgxpool.Pool) *RefreshTokenStore {
	return &RefreshTokenStore{pool: pool}
}

func (s *RefreshTokenStore) Create(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		 VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("refresh token create: %w", err)
	}
	return nil
}

func (s *RefreshTokenStore) Validate(ctx context.Context, userID uuid.UUID, tokenHash string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM refresh_tokens
			WHERE user_id = $1 AND token_hash = $2 AND expires_at > now()
		)`,
		userID, tokenHash,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("refresh token validate: %w", err)
	}
	return exists, nil
}

func (s *RefreshTokenStore) DeleteByUser(ctx context.Context, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM refresh_tokens WHERE user_id = $1`, userID,
	)
	if err != nil {
		return fmt.Errorf("refresh token delete by user: %w", err)
	}
	return nil
}

func (s *RefreshTokenStore) DeleteExpired(ctx context.Context) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM refresh_tokens WHERE expires_at <= now()`,
	)
	if err != nil {
		return fmt.Errorf("refresh token delete expired: %w", err)
	}
	return nil
}
