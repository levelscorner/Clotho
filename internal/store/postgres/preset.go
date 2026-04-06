package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/user/clotho/internal/domain"
)

// PresetStore implements store.PresetStore.
type PresetStore struct {
	pool *pgxpool.Pool
}

func NewPresetStore(pool *pgxpool.Pool) *PresetStore {
	return &PresetStore{pool: pool}
}

func (s *PresetStore) List(ctx context.Context, tenantID uuid.UUID) ([]domain.AgentPreset, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, tenant_id, name, description, category, config, icon, is_built_in, created_at
		 FROM agent_presets
		 WHERE tenant_id = $1 OR is_built_in = true
		 ORDER BY is_built_in DESC, name ASC`, tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("preset list: %w", err)
	}
	defer rows.Close()

	var presets []domain.AgentPreset
	for rows.Next() {
		p, err := scanPreset(rows)
		if err != nil {
			return nil, fmt.Errorf("preset list scan: %w", err)
		}
		presets = append(presets, p)
	}
	return presets, rows.Err()
}

func (s *PresetStore) Get(ctx context.Context, id uuid.UUID) (domain.AgentPreset, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, category, config, icon, is_built_in, created_at
		 FROM agent_presets WHERE id = $1`, id,
	)

	var p domain.AgentPreset
	var configJSON []byte
	err := row.Scan(
		&p.ID, &p.TenantID, &p.Name, &p.Description,
		&p.Category, &configJSON, &p.Icon, &p.IsBuiltIn, &p.CreatedAt,
	)
	if err != nil {
		return domain.AgentPreset{}, fmt.Errorf("preset get: %w", err)
	}
	if err := json.Unmarshal(configJSON, &p.Config); err != nil {
		return domain.AgentPreset{}, fmt.Errorf("preset get: unmarshal config: %w", err)
	}
	return p, nil
}

func (s *PresetStore) Create(ctx context.Context, p domain.AgentPreset) (domain.AgentPreset, error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}

	configJSON, err := json.Marshal(p.Config)
	if err != nil {
		return domain.AgentPreset{}, fmt.Errorf("preset create: marshal config: %w", err)
	}

	err = s.pool.QueryRow(ctx,
		`INSERT INTO agent_presets (id, tenant_id, name, description, category, config, icon, is_built_in)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING created_at`,
		p.ID, p.TenantID, p.Name, p.Description, p.Category, configJSON, p.Icon, p.IsBuiltIn,
	).Scan(&p.CreatedAt)
	if err != nil {
		return domain.AgentPreset{}, fmt.Errorf("preset create: %w", err)
	}
	return p, nil
}

func (s *PresetStore) Update(ctx context.Context, p domain.AgentPreset) error {
	configJSON, err := json.Marshal(p.Config)
	if err != nil {
		return fmt.Errorf("preset update: marshal config: %w", err)
	}

	tag, err := s.pool.Exec(ctx,
		`UPDATE agent_presets
		 SET name = $1, description = $2, category = $3, config = $4, icon = $5
		 WHERE id = $6 AND is_built_in = false`,
		p.Name, p.Description, p.Category, configJSON, p.Icon, p.ID,
	)
	if err != nil {
		return fmt.Errorf("preset update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("preset update: not found or is built-in")
	}
	return nil
}

func (s *PresetStore) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM agent_presets WHERE id = $1 AND is_built_in = false`, id,
	)
	if err != nil {
		return fmt.Errorf("preset delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("preset delete: not found or is built-in")
	}
	return nil
}

// scannable abstracts pgx.Row and pgx.Rows for reuse.
type scannable interface {
	Scan(dest ...any) error
}

func scanPreset(row scannable) (domain.AgentPreset, error) {
	var p domain.AgentPreset
	var configJSON []byte
	err := row.Scan(
		&p.ID, &p.TenantID, &p.Name, &p.Description,
		&p.Category, &configJSON, &p.Icon, &p.IsBuiltIn, &p.CreatedAt,
	)
	if err != nil {
		return domain.AgentPreset{}, err
	}
	if err := json.Unmarshal(configJSON, &p.Config); err != nil {
		return domain.AgentPreset{}, fmt.Errorf("unmarshal config: %w", err)
	}
	return p, nil
}
