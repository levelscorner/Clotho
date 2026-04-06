package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/user/clotho/internal/domain"
)

// CredentialStore implements store.CredentialStore.
type CredentialStore struct {
	pool *pgxpool.Pool
}

func NewCredentialStore(pool *pgxpool.Pool) *CredentialStore {
	return &CredentialStore{pool: pool}
}

func (s *CredentialStore) Create(ctx context.Context, c domain.Credential) (domain.Credential, error) {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO credentials (id, tenant_id, provider, api_key, label)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING created_at`,
		c.ID, c.TenantID, c.Provider, c.APIKey, c.Label,
	).Scan(&c.CreatedAt)
	if err != nil {
		return domain.Credential{}, fmt.Errorf("credential create: %w", err)
	}
	return c, nil
}

func (s *CredentialStore) Get(ctx context.Context, id uuid.UUID) (domain.Credential, error) {
	var c domain.Credential
	err := s.pool.QueryRow(ctx,
		`SELECT id, tenant_id, provider, api_key, label, created_at
		 FROM credentials WHERE id = $1`, id,
	).Scan(&c.ID, &c.TenantID, &c.Provider, &c.APIKey, &c.Label, &c.CreatedAt)
	if err != nil {
		return domain.Credential{}, fmt.Errorf("credential get: %w", err)
	}
	return c, nil
}

func (s *CredentialStore) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]domain.Credential, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, tenant_id, provider, api_key, label, created_at
		 FROM credentials
		 WHERE tenant_id = $1
		 ORDER BY created_at DESC`, tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("credential list: %w", err)
	}
	defer rows.Close()

	var credentials []domain.Credential
	for rows.Next() {
		var c domain.Credential
		if err := rows.Scan(&c.ID, &c.TenantID, &c.Provider, &c.APIKey, &c.Label, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("credential list scan: %w", err)
		}
		credentials = append(credentials, c)
	}
	return credentials, rows.Err()
}

func (s *CredentialStore) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM credentials WHERE id = $1`, id,
	)
	if err != nil {
		return fmt.Errorf("credential delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("credential delete: not found")
	}
	return nil
}
