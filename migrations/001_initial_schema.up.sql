-- Clotho initial schema

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Tenants
CREATE TABLE tenants (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Users
CREATE TABLE users (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES tenants(id),
    email      TEXT NOT NULL UNIQUE,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Projects
CREATE TABLE projects (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Pipelines
CREATE TABLE pipelines (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Pipeline versions (immutable graph snapshots)
CREATE TABLE pipeline_versions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pipeline_id UUID NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
    version     INT NOT NULL,
    graph       JSONB NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (pipeline_id, version)
);

-- Agent presets
CREATE TABLE agent_presets (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID REFERENCES tenants(id),
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    category    TEXT NOT NULL,
    config      JSONB NOT NULL,
    icon        TEXT NOT NULL DEFAULT 'bot',
    is_built_in BOOL NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Credentials (plaintext for Phase 1)
CREATE TABLE credentials (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES tenants(id),
    provider   TEXT NOT NULL,
    api_key    TEXT NOT NULL,
    label      TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Executions
CREATE TABLE executions (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pipeline_version_id  UUID NOT NULL REFERENCES pipeline_versions(id),
    tenant_id            UUID NOT NULL REFERENCES tenants(id),
    status               TEXT NOT NULL DEFAULT 'pending'
                         CHECK (status IN ('pending','running','completed','failed','cancelled')),
    total_cost           NUMERIC(10,6),
    total_tokens         INT,
    error                TEXT,
    started_at           TIMESTAMPTZ,
    completed_at         TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Step results
CREATE TABLE step_results (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    node_id      TEXT NOT NULL,
    status       TEXT NOT NULL
                 CHECK (status IN ('pending','running','completed','failed','skipped')),
    input_data   JSONB,
    output_data  JSONB,
    error        TEXT,
    tokens_used  INT,
    cost_usd     NUMERIC(10,6),
    duration_ms  BIGINT,
    started_at   TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_step_results_execution ON step_results (execution_id);

-- Job queue (Postgres SKIP LOCKED pattern)
CREATE TABLE job_queue (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES executions(id),
    status       TEXT NOT NULL DEFAULT 'pending'
                 CHECK (status IN ('pending','running','completed','failed')),
    payload      JSONB,
    claimed_by   TEXT,
    claimed_at   TIMESTAMPTZ,
    last_ping    TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_job_queue_poll ON job_queue (created_at)
    WHERE status = 'pending';

-- Indexes for common queries
CREATE INDEX idx_projects_tenant ON projects (tenant_id);
CREATE INDEX idx_pipelines_project ON pipelines (project_id);
CREATE INDEX idx_executions_tenant ON executions (tenant_id, created_at DESC);
CREATE INDEX idx_executions_version ON executions (pipeline_version_id);
