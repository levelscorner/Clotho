# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Clotho — an AI-native visual workflow platform for creative content production. Users drag Agent nodes onto a React Flow canvas, configure them with Roles and Tasks, connect them into pipelines, and execute them against LLMs.

**Full architecture, decisions, and data flow**: see `docs/ARCHITECTURE.md` (must be updated with every significant change).

## Build & Run

```bash
# Backend (Go)
go build ./cmd/clotho                    # build binary
go test -race ./...                      # run all tests with race detection
go test ./internal/engine/...            # run single package tests
go vet ./...                             # static analysis

# Frontend (React/TypeScript)
cd web && npm run dev                    # dev server on :3000
cd web && npm run build                  # production build
cd web && npx tsc --noEmit               # type-check only

# Full stack development
make dev-backend                         # Go server on :8080
make dev-frontend                        # Vite dev server on :3000 (proxies /api)

# Docker
docker compose up                        # Postgres + Clotho on :8080
```

## Architecture

Single Go binary (`cmd/clotho/main.go`) runs as server, worker, or both via `MODE` env var.

```
cmd/clotho/          Entry point — config → DB → stores → engine → API → server/worker
internal/
  domain/            Pure types, zero dependencies (Agent, Pipeline, Execution, etc.)
  engine/            Workflow execution: DAG validation, topo-sort, step runners, event bus
  llm/               LLM provider abstraction (OpenAI impl)
  queue/             Postgres SKIP LOCKED job queue + worker
  store/             Repository interfaces (8 interfaces)
  store/postgres/    PostgreSQL implementations
  api/               Chi HTTP handlers, middleware, DTOs
  config/            Configuration loading from env vars
migrations/          SQL migrations (embedded via go:embed)
web/                 React + TypeScript + Vite frontend
  src/stores/        Zustand stores (pipeline, execution, project)
  src/components/    React Flow canvas, node palette, inspector, execution UI
  src/lib/           API client, domain types, port compatibility
  src/hooks/         SSE hook for real-time execution streaming
docs/
  ARCHITECTURE.md    Full architecture, decisions, data flow (KEEP UPDATED)
  Scribble.md        Original vision document
  Clotho_calude_research.md    Technical research
  deep-research-report-for-clotho.md   Competitive analysis
```

## Key Concepts

- **Agent/Role/Task triad**: An AgentNode has a Role (personality/system prompt) and Task (what to produce). Specialized agents (Script Writer, Image Prompt Crafter, etc.) are presets of one base AgentNode type — NOT separate implementations.
- **Port types**: text, image_prompt, video_prompt, audio_prompt, image, video, audio, json, any. Prompt types are subtypes of text (image_prompt connects to text inputs). Compatibility checked at design-time (frontend) and save-time (backend).
- **Pipeline execution**: Graphs are validated (cycle detection, port compatibility), topologically sorted, then executed sequentially. Results stream via SSE.
- **Job queue**: Postgres `FOR UPDATE SKIP LOCKED` pattern. Worker polls every 500ms. Heartbeat every 10s. Zombie reaper every 30s.
- **Cost tracking**: First-class concept. Every StepResult records tokens_used + cost_usd. Execution aggregates total. Frontend shows real-time.

## Execution Data Flow

```
User clicks Run → POST /api/pipelines/{id}/execute
  → Create Execution (pending) + Enqueue Job
  → Worker dequeues (SKIP LOCKED) → Engine.ExecuteWorkflow()
  → Validate graph → TopoSort → For each node:
      → Collect upstream outputs → Execute (LLM call or passthrough)
      → Save StepResult → Publish event via EventBus
  → SSE stream → Frontend updates node status + output in real-time
```

## Conventions

- `internal/domain/` has ZERO framework dependencies — pure Go types and validation
- Store methods accept `context.Context` first, return domain types
- All SQL uses parameterized `$N` placeholders — no string concatenation
- `go:embed` used for migrations (migrations package) and frontend (cmd/clotho in production)
- Frontend custom nodes MUST be wrapped in `React.memo()` for React Flow performance
- `nodeTypes` object defined OUTSIDE React components (not recreated on render)
- Zustand stores split by concern: pipelineStore (editing) vs executionStore (runtime)
- **After making changes**: update `docs/ARCHITECTURE.md` sections 4 (data flow), 6 (package structure), or 11 (decisions) as appropriate

## Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `MODE` | `all` | `server`, `worker`, or `all` |
| `PORT` | `8080` | HTTP server port |
| `DATABASE_URL` | `postgres://clotho:clotho@localhost:5432/clotho?sslmode=disable` | Postgres connection |
| `OPENAI_API_KEY` | (none) | OpenAI API key for LLM calls |
| `LOG_LEVEL` | `info` | debug, info, warn, error |
