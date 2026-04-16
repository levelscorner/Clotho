# Clotho — Architecture & Decision Record

> **This document is the single source of truth for Clotho's architecture, design decisions, and data flow.** Update it every time a significant change is made to the codebase.

**Last updated**: 2026-04-06 (Phase 1 complete)

---

## 1. Vision & Problem Statement

Clotho is an **AI-native visual workflow platform** for creative content production.

### The Problem
The creator (Abhinav Rana) works across OpenArt, Higgsfield (video), NanoBanan/Gemini/ChatGPT (images), and various LLMs (scripts) — all in scattered browser windows. Weavy.ai has the right workflow UX but unaffordable pricing. n8n is general-purpose automation, not AI-native.

### The Solution
A drag-and-drop workspace where AI **Agents** with configurable **Roles** chain together into executable **Pipelines** that produce creative content (scripts, image prompts, video prompts, stories).

### Strategic Position
- n8n has 10+ critical CVEs (late 2025–2026), including unauthenticated RCE chains
- Windmill is developer-only, no visual builder for creatives
- Activepieces is immature (~644 integrations, UI glitches)
- **Clotho differentiates by making AI a first-class citizen**, not a bolted-on node

---

## 2. Core Domain Model

### The Agent / Role / Task Triad

This is the conceptual heart of Clotho, derived from the vision document (`docs/Scribble.md`).

```
[Agent] — An LLM/companion (the box you drag onto the canvas)
[Role]  — A personality or job injected into the Agent (e.g., "Screenwriter")
[Task]  — What the Agent produces (script, image prompt, story, etc.)
```

**Key design decision**: All specialized agents from the vision (ScriptPromptAgent, ImagePromptAgent, StoryAgent, etc.) are **presets** of one atomic `AgentNode` type — NOT separate implementations. This directly implements the user's insight: *"at some level we need to go atomic and create a base component [Agent] which can be used to create other components."*

New agent types are **data** (a row in `agent_presets`), not code.

### Entity Hierarchy

```
Tenant
  └── User
  └── Project
        └── Pipeline
              └── PipelineVersion (immutable graph snapshot)
                    ├── NodeInstance[] (AgentNode or ToolNode)
                    └── Edge[] (typed connections between ports)
              └── Execution[]
                    └── StepResult[] (per-node I/O, tokens, cost, timing)
```

### Node Types

| Type | Purpose | Has LLM Call? | Example |
|------|---------|---------------|---------|
| **AgentNode** | LLM-backed creative AI | Yes | Script Writer, Prompt Enhancer |
| **ToolNode** | Static data source | No | TextBox, ImageBox, VideoBox |

### Port Type System

Nine types with subtyping rules. Enforced at design-time (frontend) and save-time (backend).

```
text ←── image_prompt   (image_prompt IS text, connects to text inputs)
     ←── video_prompt
     ←── audio_prompt

image, video, audio      (strict binary/URL — NOT compatible with text)
json                     (structured data)
any                      (universal wildcard, connects to anything)
```

**Why typed ports?** Prevents nonsensical connections (e.g., feeding an image binary into a text prompt input) and makes the creative content pipeline semantically meaningful.

### Built-in Agent Presets (7)

| # | Name | Persona | Task Type | Output Type | Temperature |
|---|------|---------|-----------|-------------|-------------|
| 1 | Script Writer | Screenwriter | script | text | 0.8 |
| 2 | Image Prompt Crafter | Prompt Engineer | image_prompt | image_prompt | 0.7 |
| 3 | Video Prompt Writer | Visual Director | video_prompt | video_prompt | 0.7 |
| 4 | Character Designer | Character Artist | character_prompt | text | 0.8 |
| 5 | Prompt Enhancer | Prompt Optimizer | prompt_enhancement | text | 0.6 |
| 6 | Story Writer | Story Narrator | story | text | 0.9 |
| 7 | Story-to-Prompt | Adaptation Specialist | story_to_prompt | image_prompt | 0.7 |

---

## 3. Architecture Overview

### Three-Plane Design

```
┌─────────────────────────────────────────────────────────┐
│  DESIGN-TIME PLANE (Frontend)                           │
│  React 18 + TypeScript + Vite                           │
│  React Flow v12 — canvas, custom nodes, typed handles   │
│  Zustand — pipelineStore (editing), executionStore (SSE) │
│  SSE (EventSource) for real-time execution streaming     │
└──────────────────────┬──────────────────────────────────┘
                       │ REST API + SSE
┌──────────────────────▼──────────────────────────────────┐
│  CONTROL PLANE (Go / Chi router)                        │
│  API handlers: projects, pipelines, presets, executions  │
│  Middleware: request-id, CORS, tenant injection          │
│  SSE streaming endpoint (/api/executions/:id/stream)    │
└──────────────────────┬──────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────┐
│  DATA PLANE                                             │
│  Postgres job queue (SKIP LOCKED, ~5K jobs/sec)         │
│  Worker pool — polls, claims, executes, heartbeats      │
│  Engine — validates DAG, topo-sorts, runs step-by-step  │
│  LLM providers — OpenAI (Phase 1), Anthropic/Gemini (P2)│
│  PostgreSQL 16 — JSONB graphs, execution logs, presets  │
└─────────────────────────────────────────────────────────┘
```

### Tech Stack

| Layer | Choice | Rationale |
|-------|--------|-----------|
| LLM providers | **OpenAI + Gemini + OpenRouter + Ollama** | 4 providers via ProviderRegistry. Gemini is free. OpenRouter gives 200+ models. Ollama is free local. |
| Backend language | **Go 1.22+** | Goroutine concurrency for parallel execution, orchestration ecosystem (Temporal, K8s, NATS all in Go), 2-4 week onboarding for new devs |
| HTTP framework | **Chi v5** | stdlib-compatible (`net/http` handlers), zero lock-in, used by Cloudflare/Heroku |
| Database | **PostgreSQL 16** | RLS, JSONB, SKIP LOCKED for queuing, all competitors use it |
| Execution queue | **Postgres SKIP LOCKED** | Zero additional infra (no Redis), ACID guarantees, Windmill proves this at 26M jobs/month |
| Frontend framework | **React 18 + TypeScript** | Required by React Flow |
| Graph editor | **React Flow v12 (@xyflow/react)** | MIT, used by Stripe, industry standard |
| State management | **Zustand 4** | React Flow uses it internally, selector-based re-renders |
| Build tool | **Vite 5** | Fast HMR, proxy for dev API |
| Real-time | **SSE (Server-Sent Events)** | Unidirectional (server→client), auto-reconnect, works with HTTP/2 |
| Deployment | **Single Go binary** | `go:embed` serves frontend, `MODE=server/worker/all` |
| LLM client | **sashabaranov/go-openai** | Most popular Go OpenAI client |

### Decisions NOT Taken (and why)

| Rejected | Why |
|----------|-----|
| **Temporal** for execution | Requires 4+ services, too complex for self-hosting. Postgres queue is sufficient for years. |
| **Redis/BullMQ** for queue | Extra infra. Postgres SKIP LOCKED handles ~5K jobs/sec. |
| **Gin** instead of Chi | Gin uses custom `gin.Context`, creates coupling. Chi is stdlib-compatible. |
| **Redux** instead of Zustand | Boilerplate without benefit. React Flow already uses Zustand. |
| **WebSocket** instead of SSE | Bidirectional not needed. SSE has built-in reconnection. |
| **Separate frontend service** | Embedding via `go:embed` means `docker compose up` = fully working app. |

---

## 4. Data Flow: End-to-End Execution

This is the complete flow from "user clicks Run" to "output appears on canvas."

### Step-by-step

```
1. USER clicks "Run" button
   ↓
2. FRONTEND: executionStore.startExecution(pipelineId)
   → Saves pipeline (POST /api/pipelines/{id}/versions)
   → Creates execution (POST /api/pipelines/{id}/execute)
   → Receives {execution_id} in response
   → Opens EventSource to /api/executions/{execution_id}/stream
   ↓
3. API HANDLER: ExecutionHandler.Execute()
   → Gets latest PipelineVersion
   → Creates Execution record (status: "pending") in Postgres
   → Enqueues Job in job_queue table
   → Returns Execution to frontend
   ↓
4. JOB QUEUE (Postgres):
   job_queue row: status="pending", execution_id=xxx
   ↓
5. WORKER: Polls every 500ms with SKIP LOCKED
   → Claims job (atomically sets status="running", claimed_by=workerID)
   → Starts heartbeat goroutine (pings every 10s to prevent zombie)
   → Loads Execution + PipelineVersion from store
   ↓
6. ENGINE: ExecuteWorkflow(ctx, execution, graph)
   a. ValidateGraph() — checks nodes, edges, ports, types, cycles
   b. TopoSort() — Kahn's algorithm → ordered []NodeInstance
   c. UpdateStatus(execution, "running")
   d. Build edge lookup: targetNodeID.targetPortID → sourceNodeID.sourcePortID
   ↓
7. FOR EACH NODE (in topological order):
   a. Collect inputs from upstream node outputs
   b. Create StepResult record (status: "running")
   c. Publish event: step_started {node_id}
   d. Get executor from registry (AgentExecutor or ToolExecutor)
   e. Execute:
      ↓
      IF AgentNode:
        - Parse AgentNodeConfig from JSON
        - Build system prompt: role.SystemPrompt + persona
        - Build user prompt: render task.Template with {{input}} replaced
        - Call llm.Provider.Complete(model, systemPrompt, userPrompt, temp, maxTokens)
        - OpenAI API call → response with content + token usage
        - CalculateCost(model, usage) → cost in USD
        - Return StepOutput{data, tokens_used, cost_usd}
      ↓
      IF ToolNode:
        - Return content or media_url as-is (zero tokens, zero cost)
      ↓
   f. Store output in nodeOutputs map
   g. Update StepResult (status: "completed", output_data, tokens, cost, duration)
   h. Publish event: step_completed {node_id, data}
   i. Accumulate totalCost, totalTokens
   ↓
8. ENGINE: Complete(execution, totalCost, totalTokens)
   → Single atomic UPDATE: status="completed", cost, tokens, completed_at
   → Publish event: execution_completed
   ↓
9. EVENT BUS → SSE HANDLER:
   Events flow through EventBus (buffered channels per execution)
   → SSE handler writes: "event: step_completed\ndata: {...}\n\n"
   → Flushes immediately
   ↓
10. FRONTEND: EventSource receives events
    → executionStore.updateStep() for each step event
    → React re-renders: node status overlay (running→success), output in inspector
    → executionStore totalCost updates in real-time
    → On execution_completed: close EventSource, show final state
```

### Error Handling Flow (Phase A — structured failures)

The pre-Phase-A path persisted only `error *string` and surfaced it as a red pill. Phase A replaced that with a typed flow so the FailureDrawer can render class-coded badges, hints, and retry CTAs.

```
Provider call returns err
  ↓
1. ClassifyProviderError(err, provider, model) →
     domain.StepFailure{
       Class:     network|timeout|rate_limit|auth|provider_5xx|...
       Stage:     provider_call (default)
       Retryable: true for transient classes only
       Message:   redact.Secrets(humanMessage(class, err))
       Cause:     redact.Secrets(err.Error())
       Hint:      hintFor[class]    // "Verify the API key in Settings."
       Attempts:  N                 // bumped by retry loop
     }
  ↓
2. AgentExecutor returns *FailureError{Failure: ...} to errCh
  ↓
3. Engine: AsFailure(execErr) recovers the structured payload (or
   classifies the raw error as fallback)
  ↓
4. Per-node OnFailure policy routes the next step:
   - "abort" (default): persist + fail execution + return
   - "skip":            persist failure, continue with no upstream value
   - "continue":        persist failure, pipe failure JSON downstream
  ↓
5. Persist:
   - step_results.error TEXT    (1-line summary, back-compat)
   - step_results.failure_json  (structured StepFailure)
   - executions.failure_json    (FIRST failed step, indexed by class)
  ↓
6. Publish step_failed event with payload {failure: {...}, error: "..."}
  ↓
7. Frontend SSE → executionStore.updateStep() → StepResult.failure
  ↓
8. UI surfaces:
   - AgentNode: red border + hover tooltip {Class · provider · attempt N}
   - Inspector: error pill click → opens FailureDrawer
   - Top bar: "N failures — why?" CTA → opens FailureDrawer
   - Executions page: "Why?" button per failed row → opens FailureDrawer
```

### Reliability Wrappers (Phase A)

Each provider call goes through a stack:

```
agent_executor.go
  ↓
retryWithBackoff(ctx, policy, retryable, fn)   // 3 attempts, 500ms→5s
  ↓
BreakerProvider                                // Allow → call → record
  ↓
llm.Provider                                   // OpenAI / Gemini / Ollama / OpenRouter
```

- **Timeout**: per-step `context.WithTimeout` (default 120s, override via `cfg.StepTimeoutSec`).
- **Retry**: only when `failure.Retryable=true`. Auth failures short-circuit immediately so a wrong API key doesn't burn 3 attempts.
- **Breaker**: per-`(provider, model)` 4-state (Closed→Degraded→Open→HalfOpen). Auth failures and other non-retryable classes don't count toward the open threshold.
- **Output validation**: post-stream, magic-byte sniff for media ports + JSON Schema for json ports. Failures land as `output_shape` / `validation` classes.

### Pin / On-Failure Short-Circuits (Phase B)

The engine's per-node loop checks node-level state BEFORE dispatching to the executor:

```
for node in sortedNodes:
  if node.Pinned and node.PinnedOutput != nil:
      nodeOutputs[node.ID] = node.PinnedOutput
      emit step_completed{pinned: true}
      continue                  // skip executor entirely

  ... normal execution ...

  if execErr != nil:
      switch node.OnFailure:
        case "skip":     continue
        case "continue": nodeOutputs[node.ID] = failureBytes; continue
        default:         return failExecution(...)
```

Both features are creator-facing in the inspector's Reliability section.

### Zombie Recovery Flow

```
Worker crashes mid-execution:
  → Job stuck with status="running", last_ping goes stale
  → Every 30s, zombie reaper checks: last_ping < now() - 60 seconds
  → Re-enqueues job: status="pending", clears claimed_by
  → Another worker picks it up
```

---

## 5. Database Schema

### Core Tables

| Table | Purpose | Key Columns |
|-------|---------|-------------|
| `tenants` | Multi-tenancy root | id, name |
| `users` | User accounts | id, tenant_id, email, name |
| `projects` | Workspace containers | id, tenant_id, name, description |
| `pipelines` | Named workflows | id, project_id, name |
| `pipeline_versions` | Immutable graph snapshots | id, pipeline_id, version, **graph JSONB** |
| `agent_presets` | Reusable agent templates | id, tenant_id (null=built-in), config JSONB |
| `credentials` | LLM API keys | id, tenant_id, provider, api_key |

### Execution Tables

| Table | Purpose | Key Columns |
|-------|---------|-------------|
| `executions` | Pipeline run records | id, pipeline_version_id, status, total_cost, total_tokens, **failure_json** (Phase A), **trace_id** (Phase A) |
| `step_results` | Per-node execution data | id, execution_id, node_id, input/output JSONB, tokens_used, cost_usd, **failure_json** (Phase A) |
| `job_queue` | Postgres-backed work queue | id, execution_id, status, claimed_by, last_ping |

### Key Indexes

- `idx_job_queue_poll` — partial index on `(created_at) WHERE status='pending'` for SKIP LOCKED
- `idx_step_results_execution` — for fetching all steps of an execution
- `idx_executions_tenant` — for listing user's executions
- `idx_executions_failure_class` — partial JSONB index on `(failure_json->>'class') WHERE failure_json IS NOT NULL`, powers `?status=failed&class=auth` queries on the executions page

### JSONB Usage

The `pipeline_versions.graph` column stores the complete `PipelineGraph`:

```json
{
  "nodes": [
    {
      "id": "node_1",
      "type": "agent",
      "label": "Story Writer",
      "position": {"x": 100, "y": 200},
      "ports": [
        {"id": "in", "name": "Input", "type": "text", "direction": "input"},
        {"id": "out", "name": "Output", "type": "text", "direction": "output"}
      ],
      "config": {
        "provider": "openai",
        "model": "gpt-4o",
        "role": {"system_prompt": "You are a story narrator...", "persona": "Story Narrator"},
        "task": {"task_type": "story", "output_type": "text", "template": "Write a story about: {{input}}"},
        "temperature": 0.9,
        "max_tokens": 4096
      }
    }
  ],
  "edges": [
    {"id": "e1", "source": "node_1", "source_port": "out", "target": "node_2", "target_port": "in"}
  ],
  "viewport": {"x": 0, "y": 0, "zoom": 1}
}
```

---

## 6. Go Package Structure

```
clotho/
├── cmd/clotho/main.go              # Entry point: config → DB → stores → engine → API → server/worker
├── internal/
│   ├── domain/                      # Pure types, ZERO external deps
│   │   ├── agent.go                 # AgentNodeConfig, RoleConfig, TaskConfig, TaskType
│   │   ├── tool.go                  # ToolNodeConfig, ToolType, DefaultToolPorts
│   │   ├── node.go                  # NodeInstance, NodeType, PortType, Port, Position
│   │   ├── edge.go                  # Edge, CanConnect() compatibility matrix
│   │   ├── pipeline.go              # Pipeline, PipelineVersion, PipelineGraph
│   │   ├── execution.go             # Execution, StepResult, ExecutionStatus (+ FailureJSON, TraceID Phase A)
│   │   ├── failure.go               # StepFailure, FailureClass (12), FailureStage (5) — Phase A
│   │   ├── preset.go                # AgentPreset (7 built-in)
│   │   ├── project.go               # Project
│   │   ├── credential.go            # Credential (plaintext Phase 1)
│   │   └── user.go                  # Tenant, User
│   ├── engine/                      # Workflow execution
│   │   ├── graph.go                 # ValidateGraph, TopoSort (Kahn's algorithm)
│   │   ├── executor.go              # StepExecutor interface, StepOutput, ExecutorRegistry
│   │   ├── engine.go                # ExecuteWorkflow loop (pin short-circuit + on-failure branching)
│   │   ├── agent_executor.go        # AgentExecutor: role+task → prompt → retry → breaker → LLM → output → validate
│   │   ├── tool_executor.go         # ToolExecutor: passthrough content/URL
│   │   ├── failure.go               # ClassifyProviderError + FailureError wrapper (Phase A)
│   │   ├── breaker.go               # 4-state circuit breaker per (provider, model) (Phase A)
│   │   ├── breaker_provider.go      # BreakerProvider wraps llm.Provider (Phase A)
│   │   ├── retry.go                 # retryWithBackoff helper, no external dep (Phase A)
│   │   ├── timeout.go               # stepTimeoutFor: per-node deadline (Phase A)
│   │   ├── validate_output.go       # Magic-byte + JSON-schema output validation (Phase A)
│   │   ├── agent_output.go          # writeAgentOutputFile + clotho://file URL minting
│   │   ├── pipeline_patterns_test.go         # B1-B12 contract tests
│   │   ├── pipeline_patterns_b13_b14_test.go # Retry recovery + breaker trip
│   │   ├── pipeline_patterns_b15_b16_test.go # Pin + on-failure skip
│   │   └── events.go                # EventBus (pub/sub), Event types (failure rides in Data payload)
│   ├── llm/                         # LLM provider abstraction
│   │   ├── provider.go              # Provider interface (Complete, Stream)
│   │   ├── registry.go              # ProviderRegistry (maps name → Provider)
│   │   ├── openai.go                # OpenAI + OpenAI-compatible base (reused by OpenRouter, Ollama)
│   │   ├── gemini.go                # Google AI Studio (raw HTTP, no SDK)
│   │   ├── openrouter.go            # OpenRouter (OpenAI-compatible, custom baseURL)
│   │   ├── ollama.go                # Ollama local (OpenAI-compatible, no key)
│   │   └── cost.go                  # Token pricing table for all providers
│   ├── queue/                       # Postgres SKIP LOCKED job queue
│   │   ├── queue.go                 # Queue wrapper (Submit)
│   │   ├── worker.go                # Worker poll loop + heartbeat
│   │   └── zombie.go                # Stale job reaper (30s interval)
│   ├── store/                       # Repository layer
│   │   ├── interfaces.go            # 8 store interfaces + Job struct
│   │   └── postgres/                # PostgreSQL implementations
│   │       ├── postgres.go          # Pool creation + migration runner
│   │       ├── project.go, pipeline.go, execution.go, preset.go, credential.go, job.go
│   │       └── ...
│   ├── api/                         # HTTP layer
│   │   ├── router.go                # Chi router, middleware, route mounting
│   │   ├── middleware/               # tenant.go (hardcoded P1), requestid.go
│   │   ├── handler/                  # project, pipeline, execution, preset, credential, stream (SSE), provider
│   │   └── dto/                      # Request/response types
│   └── config/                      # Config loading from env vars
├── migrations/                      # Embedded SQL (go:embed)
│   ├── 001_initial_schema.up/down.sql
│   ├── 002_seed_presets.up/down.sql
│   └── migrations.go                # embed.FS
└── web/                             # React frontend
    └── src/
        ├── lib/types.ts             # TypeScript domain types (mirrors Go) + StepFailure + OnFailurePolicy
        ├── lib/portCompatibility.ts  # Port type matrix + PORT_COLORS (text family unified)
        ├── lib/llmCapabilities.ts    # Per-provider knob support (Phase 1)
        ├── lib/failureSchema.ts      # coerceStepFailure defensive validator (Phase A)
        ├── lib/api.ts               # Typed fetch wrapper + executions/credentials/testNode helpers
        ├── stores/                   # Zustand
        │   ├── pipelineStore.ts      # Canvas state (nodes, edges, save/load, setNodePin, setNodeOnFailure)
        │   ├── executionStore.ts     # Execution + SSE streaming + structured failure parsing
        │   └── projectStore.ts       # Project/pipeline listing
        ├── components/
        │   ├── canvas/PipelineCanvas.tsx   # React Flow wrapper + DnD
        │   ├── canvas/ValidationModal.tsx  # Save-time validation modal (Phase B B8)
        │   ├── canvas/nodes/AgentNode.tsx  # Agent renderer + failure tooltip (React.memo)
        │   ├── canvas/nodes/ToolNode.tsx   # Tool renderer (React.memo)
        │   ├── canvas/nodes/MediaNode.tsx  # Media renderer
        │   ├── canvas/nodes/BaseNode.tsx   # Shared handles + status overlay
        │   ├── sidebar/NodePalette.tsx     # Drag source (4 modalities + tools)
        │   ├── inspector/                  # Config panels — AgentInspector composes:
        │   │   ├── sections/VariablesSection.tsx
        │   │   ├── sections/SamplingSection.tsx
        │   │   ├── sections/ReliabilitySection.tsx  # Pin + on-failure (Phase B)
        │   │   └── TestStepButton.tsx              # POST /api/nodes/test (Phase B B4)
        │   ├── execution/
        │   │   ├── RunButton.tsx
        │   │   ├── ExecutionStatus.tsx     # + failure-count CTA opens drawer (Phase A)
        │   │   ├── FailureDrawer.tsx       # Class badge, hint, copy diagnostic, rerun (Phase A)
        │   │   └── OpenFolderButton.tsx
        │   └── settings/SettingsPanel.tsx  # + per-credential Test button (Phase B B1)
        ├── pages/
        │   ├── DevNodes.tsx          # Dev-only node renderer playground
        │   └── ExecutionsPage.tsx    # /executions list + retry + drawer (Phase B B5+B6)
        ├── hooks/useSSE.ts           # EventSource lifecycle hook
        └── styles/                   # global.css, nodes.css
```

### Dependency Rules

```
domain (ZERO deps) ← engine ← api
                   ← llm
                   ← store/postgres
                   ← queue
```

`internal/domain/` has NO external imports (not even uuid in types that use it — wait, it does import uuid, but zero framework deps). Engine defines interfaces; `llm/` and `store/` implement them.

---

## 7. Frontend Architecture

### React Flow Custom Nodes

| Node | Visual | Color | Handles |
|------|--------|-------|---------|
| AgentNode | Robot icon + persona + model badge + task tag | Blue/purple gradient | Input (left) + Output (right), typed |
| ToolNode | Content preview or media icon | Gray | Output only |

All nodes wrapped in `React.memo()` — critical for React Flow performance (without memo, dragging drops to 2 FPS with complex nodes; with memo, sustains 60 FPS).

### Port Handle Colors

| Type | Color | Hex |
|------|-------|-----|
| text | Gray | #94a3b8 |
| image_prompt | Blue | #3b82f6 |
| video_prompt | Purple | #a855f7 |
| audio_prompt | Amber | #f59e0b |
| image | Green | #22c55e |
| video | Orange | #f97316 |
| audio | Pink | #ec4899 |
| json | Yellow | #eab308 |
| any | Dark gray | #6b7280 |

### Zustand Store Split

Two separate stores prevent execution SSE updates from causing editor re-renders:

- **pipelineStore** — nodes, edges, selection, isDirty, save/load (editing concern)
- **executionStore** — executionId, status, stepResults Map, totalCost, SSE (runtime concern)
- **projectStore** — project/pipeline listing (navigation concern)

### Node Palette (Sidebar)

Two sections:
1. **Agents** (top, prominent): "Blank Agent" + 7 built-in presets (fetched from `/api/presets`)
2. **Tools** (bottom): TextBox, ImageBox, VideoBox (hardcoded)

Items are draggable via HTML5 DnD. Drop on canvas creates node with preset config pre-filled.

---

## 8. API Routes

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| GET | `/health` | inline | Health check |
| POST | `/api/projects` | ProjectHandler | Create project |
| GET | `/api/projects` | ProjectHandler | List projects (by tenant) |
| GET | `/api/projects/{id}` | ProjectHandler | Get project |
| PUT | `/api/projects/{id}` | ProjectHandler | Update project |
| DELETE | `/api/projects/{id}` | ProjectHandler | Delete project |
| POST | `/api/projects/{projectID}/pipelines` | PipelineHandler | Create pipeline |
| GET | `/api/projects/{projectID}/pipelines` | PipelineHandler | List pipelines |
| GET | `/api/pipelines/{id}` | PipelineHandler | Get pipeline |
| PUT | `/api/pipelines/{id}` | PipelineHandler | Update pipeline |
| DELETE | `/api/pipelines/{id}` | PipelineHandler | Delete pipeline |
| POST | `/api/pipelines/{id}/versions` | PipelineHandler | Save graph version |
| GET | `/api/pipelines/{id}/versions/latest` | PipelineHandler | Get latest version |
| GET | `/api/pipelines/{id}/versions/{version}` | PipelineHandler | Get specific version |
| POST | `/api/pipelines/{id}/execute` | ExecutionHandler | Run pipeline |
| GET | `/api/executions/{id}` | ExecutionHandler | Get execution + steps |
| GET | `/api/executions` | ExecutionHandler | List executions |
| GET | `/api/executions/{id}/stream` | StreamHandler | SSE stream |
| GET | `/api/presets` | PresetHandler | List presets |
| POST | `/api/presets` | PresetHandler | Create custom preset |
| GET | `/api/presets/{id}` | PresetHandler | Get preset |
| PUT | `/api/presets/{id}` | PresetHandler | Update preset |
| DELETE | `/api/presets/{id}` | PresetHandler | Delete preset |
| POST | `/api/credentials` | CredentialHandler | Store API key |
| GET | `/api/credentials` | CredentialHandler | List (masked) |
| DELETE | `/api/credentials/{id}` | CredentialHandler | Delete |

---

## 9. LLM Integration

### Provider Architecture

```
ProviderRegistry (maps name → Provider)
  ├── "openai"      → OpenAIProvider (sashabaranov/go-openai)
  ├── "gemini"       → GeminiProvider (raw HTTP to generativelanguage.googleapis.com)
  ├── "openrouter"   → OpenAIProvider (reused, baseURL=openrouter.ai/api/v1)
  └── "ollama"       → OpenAIProvider (reused, baseURL=localhost:11434/v1)
```

Providers are registered conditionally at startup based on configured API keys. Ollama is always registered (no key needed). The `AgentExecutor` looks up the provider by `config.Provider` name at execution time.

### Provider Interface

```go
type Provider interface {
    Complete(ctx, CompletionRequest) (CompletionResponse, error)
    Stream(ctx, CompletionRequest) (<-chan StreamChunk, error)
}
```

### Provider Details

| Provider | API | Auth | Free Tier | Notes |
|----------|-----|------|-----------|-------|
| **OpenAI** | `api.openai.com/v1` | `OPENAI_API_KEY` | No | GPT-4o, GPT-4o-mini |
| **Gemini** | `generativelanguage.googleapis.com/v1beta` | `GEMINI_API_KEY` | **Yes** (1500 req/day) | Raw HTTP, no Go SDK needed |
| **OpenRouter** | `openrouter.ai/api/v1` (OpenAI-compatible) | `OPENROUTER_API_KEY` | No | 200+ models, single key |
| **Ollama** | `localhost:11434/v1` (OpenAI-compatible) | None | **Yes** (local) | Requires Ollama running locally |

### Prompt Assembly (AgentExecutor)

```
System Prompt = role.SystemPrompt + "\n\nPersona: " + role.Persona
User Prompt   = task.Template with {{input}} replaced by concatenated upstream outputs
```

If `task.Template` is empty, the raw concatenated inputs become the user prompt.

### Cost Tracking

Per-model pricing table (per 1M tokens):

| Model | Input | Output |
|-------|-------|--------|
| gpt-4o | $2.50 | $10.00 |
| gpt-4o-mini | $0.15 | $0.60 |
| gemini-2.0-flash | **$0.00** | **$0.00** |
| gemini-1.5-pro | $1.25 | $5.00 |
| llama3, mistral, phi3 (Ollama) | **$0.00** | **$0.00** |
| anthropic/claude-sonnet-4 (OpenRouter) | $3.00 | $15.00 |

Cost is tracked at three levels:
1. **StepResult**: per-node tokens_used + cost_usd
2. **Execution**: aggregated total_cost + total_tokens
3. **Frontend**: real-time totalCost display from SSE events

---

## 10. Phased Roadmap

### Phase 1 (COMPLETE) — Foundation + Core AI Pipeline
- Go backend: domain types, engine, stores, API, queue, worker
- React frontend: canvas, agent/tool nodes, palette, inspector, SSE
- OpenAI LLM provider
- 7 built-in agent presets
- Docker Compose deployment

### Phase 1.5 (COMPLETE) — Multi-Provider + Critique Fixes
- ProviderRegistry: OpenAI, Gemini (free), OpenRouter (200+ models), Ollama (local free)
- AgentExecutor resolves provider per-node from config.Provider
- Credential store wired to executor (per-node API key selection)
- `/api/providers` endpoint for frontend provider discovery
- Frontend: dynamic provider/model selector, provider availability warnings
- Input validation on all API handlers
- Viewport persistence on save/reload
- React error boundary

### Phase 2 (NEXT) — Auth + Enhanced UX
- JWT authentication (login/register)
- Envelope encryption (AES-256-GCM) for API keys
- Model router with failover
- Undo/redo, pipeline versioning UI

### Phase 3 — Generation Nodes + Research Nodes + Enhanced UX
- ImageGeneratorAgent (DALL-E, Stability AI)
- VideoGeneratorAgent (Runway, Pika APIs)
- **Research nodes** (inspired by last30days): WebSearchNode, RedditNode, YouTubeNode
- **ScoringNode**: multi-factor ranking (relevance × recency × engagement)
- **ConvergenceNode**: detect shared themes across multiple source outputs
- S3/MinIO for binary asset storage
- Prompt template system with variable injection
- Auto-layout (ELKjs), minimap, copy/paste
- Pipeline templates: Topic Research, Competitive Analysis, Prompt Research

### Phase 4 — Production Hardening + Scheduling
- PostgreSQL RLS enforcement
- nsjail sandboxing for code execution
- OIDC SSO, RBAC
- Webhook + cron triggers
- **Scheduled pipelines** (watchlist pattern from last30days): run daily/weekly, accumulate results
- **Auto-save research library**: execution outputs browsable and exportable
- OpenTelemetry + Grafana dashboards
- Helm chart

### Phase 5 — Differentiation Bets
- Event sourcing → time-travel debugging
- Human-in-the-loop approval nodes
- Declarative pipeline language (from Scribble.md vision)
- Custom preset marketplace
- Local model support (Ollama)

---

## 11. Key Design Decisions Log

| # | Decision | Choice | Why | Date |
|---|----------|--------|-----|------|
| 1 | Agent subtypes | Presets, not code | User's insight: "go atomic." New agents = data, not implementations. | 2026-04-06 |
| 2 | Port types | 9 types with subtyping | Creative semantics: image_prompt IS text, but image binary is NOT. | 2026-04-06 |
| 3 | Execution queue | Postgres SKIP LOCKED | Zero extra infra. Proven by Windmill at 26M jobs/month. | 2026-04-06 |
| 4 | State management | Two Zustand stores | Execution SSE must not cause editor re-renders. | 2026-04-06 |
| 5 | LLM cost tracking | First-class domain concept | AI-native platform MUST make cost visible. Per-step and per-run. | 2026-04-06 |
| 6 | Frontend embedding | go:embed in binary | `docker compose up` = fully working app. No separate service. | 2026-04-06 |
| 7 | Sequential execution | Phase 1 only | Creative pipelines are typically linear chains. Parallel in Phase 3+. | 2026-04-06 |
| 8 | No auth in Phase 1 | Hardcoded tenant/user | Ship the creative loop first, auth second. | 2026-04-06 |
| 9 | Atomic Complete | Single UPDATE for cost+status | Code review found split UPDATE race condition. Fixed with `Complete()`. | 2026-04-06 |
| 10 | Remove CloseExecution | Deleted from EventBus | Double-close panic risk. Unsubscribe per-handler is sufficient. | 2026-04-06 |
| 11 | Multi-provider via ProviderRegistry | Registry pattern | Each agent node picks its own provider. Extensible without code changes. | 2026-04-06 |
| 12 | Gemini as default preset model | gemini-2.0-flash | Free tier means app works out-of-the-box with no paid API key. | 2026-04-06 |
| 13 | OpenRouter + Ollama reuse OpenAI client | Custom baseURL | Both expose OpenAI-compatible APIs. Zero new dependencies. | 2026-04-06 |
| 14 | Gemini uses raw HTTP, not SDK | net/http | Avoids heavy google-cloud-go dependency for a simple REST call. | 2026-04-06 |
| 15 | Credential store wired to executor | Per-node API keys | Users can store multiple keys and assign per agent node. | 2026-04-06 |
| 16 | Input validation on all handlers | 400 with specific errors | Prevents cryptic downstream failures from malformed requests. | 2026-04-06 |
| 17 | Structured `domain.StepFailure` | Replace `error *string` with class+stage+hint+cause+attempts JSON | Frontend FailureDrawer needs class for badge, hint for CTA, cause for diagnostic — string-only path made polished UX impossible. | 2026-04-16 |
| 18 | Per-`(provider,model)` circuit breaker | Hand-rolled 4-state (Closed/Degraded/Open/HalfOpen) | research1.md spec needs Degraded state that sony/gobreaker doesn't have. Auth failures don't trip the breaker — they need a human, not a wait. | 2026-04-16 |
| 19 | Retry only initial Stream call | 3 attempts, 500ms→5s backoff, retryable-only | Mid-stream retry would replay chunks the user already saw. Initial-fail retry covers transient 503s without UX confusion. | 2026-04-16 |
| 20 | Output validation: structural + semantic | h2non/filetype magic-bytes + santhosh-tekuri/jsonschema | Catches the canonical "TTS returned text" failure mode at the engine boundary, not in the user's downloaded .txt. | 2026-04-16 |
| 21 | `Pinned + PinnedOutput` on NodeInstance | Top-level fields, not in per-type config | Pinning is universal across node types; engine consults before dispatch. Saves $$ during downstream prompt iteration. | 2026-04-16 |
| 22 | Per-node `OnFailure` policy | abort (default) / skip / continue enum | Fan-out pipelines need branch-level resilience. `continue` pipes the StepFailure JSON downstream so an error-handler agent can react. | 2026-04-16 |
| 23 | Save-time `ValidateGraph` enforcement | 400 with `validation_errors[]` array | Silent edge-drop in `pipelineStore.onConnect` was the worst UX regression in the app. Backend rejects bad graphs at save time; frontend opens a "Click to fix" modal. | 2026-04-16 |
| 24 | Credential test endpoint | 1-token ping completion via stored key | Catches FailureAuth at config time, not the first run. Returns 200 with structured `failure` JSON for the SettingsPanel badge. | 2026-04-16 |
| 25 | "Test step in isolation" handler | `POST /api/nodes/test`, no DB writes | Cuts iteration loop from minutes to seconds. Reuses the live executor stack so the failure surface matches a real run. | 2026-04-16 |
| 26 | TDD baseline locked | `feedback_tdd` memory + `superpowers:test-driven-development` skill | After founder request 2026-04-17. Going forward: failing test first, then implementation. | 2026-04-17 |

---

## 12. Research Documents

These documents informed the architecture:

| Document | Location | Focus |
|----------|----------|-------|
| **Scribble.md** | `docs/Scribble.md` | User's vision: Agent/Role/Task, component types, pipeline concept |
| **Claude Research** | `docs/Clotho_calude_research.md` | Technical blueprint: Go+Postgres+ReactFlow, competitive analysis, 6-month plan |
| **Deep Research Report** | `docs/deep-research-report-for-clotho.md` | Broader landscape: n8n/Windmill/Activepieces comparison, security gaps, architecture patterns |

---

## 13. File Counts (Phase 1)

| Category | Files | Lines |
|----------|-------|-------|
| Go backend | 48 | ~3,980 |
| TypeScript/CSS frontend | 22 | ~2,550 |
| SQL migrations | 4 | ~200 |
| Config/Docker/Make | 7 | ~100 |
| **Total** | **81** | **~6,830** |
