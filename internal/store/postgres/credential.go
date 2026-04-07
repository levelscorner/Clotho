package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/user/clotho/internal/crypto"
	"github.com/user/clotho/internal/domain"
)

// CredentialStore implements store.CredentialStore with optional envelope encryption.
type CredentialStore struct {
	pool     *pgxpool.Pool
	envelope *crypto.Envelope // nil means plaintext fallback (dev mode)
}

// NewCredentialStore creates a CredentialStore. If envelope is nil, credentials are stored
// without encryption (dev mode with a log warning on each write).
func NewCredentialStore(pool *pgxpool.Pool, envelope *crypto.Envelope) *CredentialStore {
	return &CredentialStore{pool: pool, envelope: envelope}
}

func (s *CredentialStore) Create(ctx context.Context, c domain.Credential) (domain.Credential, error) {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}

	if s.envelope != nil && c.PlaintextKey != "" {
		encVal, encDEK, nonce, err := s.envelope.Encrypt([]byte(c.PlaintextKey))
		if err != nil {
			return domain.Credential{}, fmt.Errorf("credential create: encrypt: %w", err)
		}
		c.EncryptedValue = encVal
		c.EncryptedDEK = encDEK
		c.Nonce = nonce
	} else if c.PlaintextKey != "" {
		slog.Warn("storing credential without encryption (CLOTHO_MASTER_KEY not set)")
		// Store plaintext in encrypted_value as a fallback
		c.EncryptedValue = []byte(c.PlaintextKey)
	}

	err := s.pool.QueryRow(ctx,
		`INSERT INTO credentials (id, tenant_id, provider, encrypted_value, encrypted_dek, nonce, label)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING created_at`,
		c.ID, c.TenantID, c.Provider, c.EncryptedValue, c.EncryptedDEK, c.Nonce, c.Label,
	).Scan(&c.CreatedAt)
	if err != nil {
		return domain.Credential{}, fmt.Errorf("credential create: %w", err)
	}
	c.PlaintextKey = "" // clear transient value
	return c, nil
}

func (s *CredentialStore) Get(ctx context.Context, id uuid.UUID) (domain.Credential, error) {
	var c domain.Credential
	err := s.pool.QueryRow(ctx,
		`SELECT id, tenant_id, provider, encrypted_value, encrypted_dek, nonce, label, created_at
		 FROM credentials WHERE id = $1`, id,
	).Scan(&c.ID, &c.TenantID, &c.Provider, &c.EncryptedValue, &c.EncryptedDEK, &c.Nonce, &c.Label, &c.CreatedAt)
	if err != nil {
		return domain.Credential{}, fmt.Errorf("credential get: %w", err)
	}
	return c, nil
}

// GetDecrypted retrieves a credential and decrypts its API key for executor use.
func (s *CredentialStore) GetDecrypted(ctx context.Context, id uuid.UUID) (string, error) {
	c, err := s.Get(ctx, id)
	if err != nil {
		return "", err
	}

	if s.envelope != nil && c.EncryptedDEK != nil && c.Nonce != nil {
		plaintext, err := s.envelope.Decrypt(c.EncryptedValue, c.EncryptedDEK, c.Nonce)
		if err != nil {
			return "", fmt.Errorf("credential get decrypted: %w", err)
		}
		return string(plaintext), nil
	}

	// Fallback: plaintext was stored in encrypted_value (dev mode)
	if c.EncryptedValue != nil {
		return string(c.EncryptedValue), nil
	}

	return "", fmt.Errorf("credential get decrypted: no encrypted value found")
}

func (s *CredentialStore) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]domain.Credential, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, tenant_id, provider, label, created_at
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
		if err := rows.Scan(&c.ID, &c.TenantID, &c.Provider, &c.Label, &c.CreatedAt); err != nil {
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
