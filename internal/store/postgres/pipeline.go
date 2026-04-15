package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/user/clotho/internal/domain"
)

// PipelineStore implements store.PipelineStore.
type PipelineStore struct {
	pool *pgxpool.Pool
}

func NewPipelineStore(pool *pgxpool.Pool) *PipelineStore {
	return &PipelineStore{pool: pool}
}

func (s *PipelineStore) Create(ctx context.Context, p domain.Pipeline) (domain.Pipeline, error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO pipelines (id, project_id, name, description)
		 VALUES ($1, $2, $3, $4)
		 RETURNING created_at, updated_at`,
		p.ID, p.ProjectID, p.Name, p.Description,
	).Scan(&p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return domain.Pipeline{}, fmt.Errorf("pipeline create: %w", err)
	}
	return p, nil
}

// Get returns a pipeline only if it belongs to the given tenant (via the
// pipeline → project → tenant chain). Used by HTTP handlers to enforce
// tenant isolation. Callers inside the system that already hold a trusted
// pipeline ID should use GetByID instead.
func (s *PipelineStore) Get(ctx context.Context, id, tenantID uuid.UUID) (domain.Pipeline, error) {
	var p domain.Pipeline
	err := s.pool.QueryRow(ctx,
		`SELECT p.id, p.project_id, p.name, p.description, p.created_at, p.updated_at
		 FROM pipelines p
		 JOIN projects pr ON pr.id = p.project_id
		 WHERE p.id = $1 AND pr.tenant_id = $2`, id, tenantID,
	).Scan(&p.ID, &p.ProjectID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return domain.Pipeline{}, fmt.Errorf("pipeline get: %w", err)
	}
	return p, nil
}

// GetByID returns a pipeline by ID without tenant scoping. Only for system-
// internal callers (queue worker, engine) that already hold a trusted ID.
func (s *PipelineStore) GetByID(ctx context.Context, id uuid.UUID) (domain.Pipeline, error) {
	var p domain.Pipeline
	err := s.pool.QueryRow(ctx,
		`SELECT id, project_id, name, description, created_at, updated_at
		 FROM pipelines WHERE id = $1`, id,
	).Scan(&p.ID, &p.ProjectID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return domain.Pipeline{}, fmt.Errorf("pipeline get by id: %w", err)
	}
	return p, nil
}

func (s *PipelineStore) ListByProject(ctx context.Context, projectID uuid.UUID) ([]domain.Pipeline, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, project_id, name, description, created_at, updated_at
		 FROM pipelines WHERE project_id = $1 ORDER BY created_at DESC`, projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("pipeline list: %w", err)
	}
	defer rows.Close()

	var pipelines []domain.Pipeline
	for rows.Next() {
		var p domain.Pipeline
		if err := rows.Scan(&p.ID, &p.ProjectID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("pipeline list scan: %w", err)
		}
		pipelines = append(pipelines, p)
	}
	return pipelines, rows.Err()
}

// Update mutates a pipeline only if it belongs to the given tenant.
func (s *PipelineStore) Update(ctx context.Context, p domain.Pipeline, tenantID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE pipelines SET name = $1, description = $2, updated_at = now()
		 WHERE id = $3
		   AND project_id IN (SELECT id FROM projects WHERE tenant_id = $4)`,
		p.Name, p.Description, p.ID, tenantID,
	)
	if err != nil {
		return fmt.Errorf("pipeline update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("pipeline update: not found")
	}
	return nil
}

// Delete removes a pipeline only if it belongs to the given tenant.
func (s *PipelineStore) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM pipelines
		 WHERE id = $1
		   AND project_id IN (SELECT id FROM projects WHERE tenant_id = $2)`,
		id, tenantID,
	)
	if err != nil {
		return fmt.Errorf("pipeline delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("pipeline delete: not found")
	}
	return nil
}

// PipelineVersionStore implements store.PipelineVersionStore.
type PipelineVersionStore struct {
	pool *pgxpool.Pool
}

func NewPipelineVersionStore(pool *pgxpool.Pool) *PipelineVersionStore {
	return &PipelineVersionStore{pool: pool}
}

func (s *PipelineVersionStore) Create(ctx context.Context, pv domain.PipelineVersion) (domain.PipelineVersion, error) {
	if pv.ID == uuid.Nil {
		pv.ID = uuid.New()
	}

	graphJSON, err := json.Marshal(pv.Graph)
	if err != nil {
		return domain.PipelineVersion{}, fmt.Errorf("pipeline version create: marshal graph: %w", err)
	}

	err = s.pool.QueryRow(ctx,
		`INSERT INTO pipeline_versions (id, pipeline_id, version, graph)
		 VALUES ($1, $2, $3, $4)
		 RETURNING created_at`,
		pv.ID, pv.PipelineID, pv.Version, graphJSON,
	).Scan(&pv.CreatedAt)
	if err != nil {
		return domain.PipelineVersion{}, fmt.Errorf("pipeline version create: %w", err)
	}
	return pv, nil
}

func (s *PipelineVersionStore) Get(ctx context.Context, id uuid.UUID) (domain.PipelineVersion, error) {
	var pv domain.PipelineVersion
	var graphJSON []byte
	err := s.pool.QueryRow(ctx,
		`SELECT id, pipeline_id, version, graph, created_at
		 FROM pipeline_versions WHERE id = $1`, id,
	).Scan(&pv.ID, &pv.PipelineID, &pv.Version, &graphJSON, &pv.CreatedAt)
	if err != nil {
		return domain.PipelineVersion{}, fmt.Errorf("pipeline version get: %w", err)
	}
	if err := json.Unmarshal(graphJSON, &pv.Graph); err != nil {
		return domain.PipelineVersion{}, fmt.Errorf("pipeline version get: unmarshal graph: %w", err)
	}
	return pv, nil
}

func (s *PipelineVersionStore) GetLatest(ctx context.Context, pipelineID uuid.UUID) (domain.PipelineVersion, error) {
	var pv domain.PipelineVersion
	var graphJSON []byte
	err := s.pool.QueryRow(ctx,
		`SELECT id, pipeline_id, version, graph, created_at
		 FROM pipeline_versions
		 WHERE pipeline_id = $1
		 ORDER BY version DESC LIMIT 1`, pipelineID,
	).Scan(&pv.ID, &pv.PipelineID, &pv.Version, &graphJSON, &pv.CreatedAt)
	if err != nil {
		return domain.PipelineVersion{}, fmt.Errorf("pipeline version get latest: %w", err)
	}
	if err := json.Unmarshal(graphJSON, &pv.Graph); err != nil {
		return domain.PipelineVersion{}, fmt.Errorf("pipeline version get latest: unmarshal graph: %w", err)
	}
	return pv, nil
}

func (s *PipelineVersionStore) GetByVersion(ctx context.Context, pipelineID uuid.UUID, version int) (domain.PipelineVersion, error) {
	var pv domain.PipelineVersion
	var graphJSON []byte
	err := s.pool.QueryRow(ctx,
		`SELECT id, pipeline_id, version, graph, created_at
		 FROM pipeline_versions
		 WHERE pipeline_id = $1 AND version = $2`, pipelineID, version,
	).Scan(&pv.ID, &pv.PipelineID, &pv.Version, &graphJSON, &pv.CreatedAt)
	if err != nil {
		return domain.PipelineVersion{}, fmt.Errorf("pipeline version get by version: %w", err)
	}
	if err := json.Unmarshal(graphJSON, &pv.Graph); err != nil {
		return domain.PipelineVersion{}, fmt.Errorf("pipeline version get by version: unmarshal graph: %w", err)
	}
	return pv, nil
}

func (s *PipelineVersionStore) ListByPipeline(ctx context.Context, pipelineID uuid.UUID) ([]domain.PipelineVersion, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, pipeline_id, version, graph, created_at
		 FROM pipeline_versions
		 WHERE pipeline_id = $1
		 ORDER BY version DESC`, pipelineID,
	)
	if err != nil {
		return nil, fmt.Errorf("pipeline version list: %w", err)
	}
	defer rows.Close()

	var versions []domain.PipelineVersion
	for rows.Next() {
		var pv domain.PipelineVersion
		var graphJSON []byte
		if err := rows.Scan(&pv.ID, &pv.PipelineID, &pv.Version, &graphJSON, &pv.CreatedAt); err != nil {
			return nil, fmt.Errorf("pipeline version list scan: %w", err)
		}
		if err := json.Unmarshal(graphJSON, &pv.Graph); err != nil {
			return nil, fmt.Errorf("pipeline version list unmarshal: %w", err)
		}
		versions = append(versions, pv)
	}
	return versions, rows.Err()
}
