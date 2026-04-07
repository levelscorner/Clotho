package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/user/clotho/internal/domain"
)

// UserStore implements store.UserStore.
type UserStore struct {
	pool *pgxpool.Pool
}

func NewUserStore(pool *pgxpool.Pool) *UserStore {
	return &UserStore{pool: pool}
}

func (s *UserStore) Create(ctx context.Context, u domain.User) (domain.User, error) {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO users (id, tenant_id, email, name, password_hash, is_active)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING created_at`,
		u.ID, u.TenantID, u.Email, u.Name, u.PasswordHash, u.IsActive,
	).Scan(&u.CreatedAt)
	if err != nil {
		return domain.User{}, fmt.Errorf("user create: %w", err)
	}
	return u, nil
}

func (s *UserStore) GetByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	var u domain.User
	err := s.pool.QueryRow(ctx,
		`SELECT id, tenant_id, email, name, password_hash, is_active, last_login_at, created_at
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.TenantID, &u.Email, &u.Name, &u.PasswordHash, &u.IsActive, &u.LastLoginAt, &u.CreatedAt)
	if err != nil {
		return domain.User{}, fmt.Errorf("user get by id: %w", err)
	}
	return u, nil
}

func (s *UserStore) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	var u domain.User
	err := s.pool.QueryRow(ctx,
		`SELECT id, tenant_id, email, name, password_hash, is_active, last_login_at, created_at
		 FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.TenantID, &u.Email, &u.Name, &u.PasswordHash, &u.IsActive, &u.LastLoginAt, &u.CreatedAt)
	if err != nil {
		return domain.User{}, fmt.Errorf("user get by email: %w", err)
	}
	return u, nil
}

func (s *UserStore) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE users SET last_login_at = now() WHERE id = $1`, id,
	)
	if err != nil {
		return fmt.Errorf("user update last login: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user update last login: not found")
	}
	return nil
}

func (s *UserStore) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE users SET password_hash = $1 WHERE id = $2`, passwordHash, id,
	)
	if err != nil {
		return fmt.Errorf("user update password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user update password: not found")
	}
	return nil
}
