package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/user/clotho/internal/domain"
)

type ProjectStore struct {
	pool *pgxpool.Pool
}

func NewProjectStore(pool *pgxpool.Pool) *ProjectStore {
	return &ProjectStore{pool: pool}
}

func (s *ProjectStore) Create(ctx context.Context, p domain.Project) (domain.Project, error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO projects (id, tenant_id, name, description)
		 VALUES ($1, $2, $3, $4)
		 RETURNING created_at, updated_at`,
		p.ID, p.TenantID, p.Name, p.Description,
	).Scan(&p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return domain.Project{}, fmt.Errorf("project create: %w", err)
	}
	return p, nil
}

// Get returns a project only if it belongs to the given tenant. Used by
// HTTP handlers. Callers inside the system should use GetByID instead.
func (s *ProjectStore) Get(ctx context.Context, id, tenantID uuid.UUID) (domain.Project, error) {
	var p domain.Project
	err := s.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, created_at, updated_at
		 FROM projects WHERE id = $1 AND tenant_id = $2`, id, tenantID,
	).Scan(&p.ID, &p.TenantID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return domain.Project{}, fmt.Errorf("project get: %w", err)
	}
	return p, nil
}

// GetByID returns a project by ID without tenant scoping. Only for system-
// internal callers (queue worker) that already hold a trusted ID.
func (s *ProjectStore) GetByID(ctx context.Context, id uuid.UUID) (domain.Project, error) {
	var p domain.Project
	err := s.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, created_at, updated_at
		 FROM projects WHERE id = $1`, id,
	).Scan(&p.ID, &p.TenantID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return domain.Project{}, fmt.Errorf("project get by id: %w", err)
	}
	return p, nil
}

func (s *ProjectStore) List(ctx context.Context, tenantID uuid.UUID) ([]domain.Project, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, tenant_id, name, description, created_at, updated_at
		 FROM projects WHERE tenant_id = $1 ORDER BY created_at DESC`, tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("project list: %w", err)
	}
	defer rows.Close()

	var projects []domain.Project
	for rows.Next() {
		var p domain.Project
		if err := rows.Scan(&p.ID, &p.TenantID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("project list scan: %w", err)
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

// Update mutates a project only if it belongs to the given tenant.
func (s *ProjectStore) Update(ctx context.Context, p domain.Project, tenantID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE projects SET name = $1, description = $2, updated_at = now()
		 WHERE id = $3 AND tenant_id = $4`,
		p.Name, p.Description, p.ID, tenantID,
	)
	if err != nil {
		return fmt.Errorf("project update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("project update: not found")
	}
	return nil
}

// Delete removes a project only if it belongs to the given tenant.
func (s *ProjectStore) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM projects WHERE id = $1 AND tenant_id = $2`, id, tenantID,
	)
	if err != nil {
		return fmt.Errorf("project delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("project delete: not found")
	}
	return nil
}
