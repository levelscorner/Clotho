# Clotho Phase Plan (Post Eng Review)

Updated: 2026-04-07
Status: APPROVED + ENG REVIEWED
Total Timeline: ~12 weeks

## Phase 1A — Core Product Works (weeks 1-3)

Priority: Get a pipeline running with streaming output and cost safety.

### Streaming Engine Refactor
- Redesign `StepExecutor` interface: return `(<-chan StreamChunk, error)` instead of `(StepOutput, error)`
- Wire `Provider.Stream()` through engine loop
- Publish `EventStepChunk` events from engine during execution
- Frontend: render streaming chunks in nodes via existing SSE + `appendChunk()` in executionStore
- Increase EventBus buffer from 64 to 1024, add `log.Warn` on dropped events

### Security Fixes
- Fix refresh token endpoint: don't validate expired access token, validate refresh token only
- Add `WHERE tenant_id = $N` to `ExecutionStore.Get()`, `CredentialStore.Get()`, `CredentialStore.GetDecrypted()`
- Move admin user creation from migration 004 to app startup code, read password from `ADMIN_PASSWORD` env var

### Cost Safety
- Enforce `AgentNodeConfig.CostCap` in `agent_executor.go`: compare `response.CostUSD` against cap, fail step if exceeded
- Add `POST /api/executions/{id}/cancel` endpoint: set status to cancelled, cancel context
- Check `ctx.Done()` between node executions in engine loop

### Determinism Fix
- Sort map keys in `agent_executor.go:concatenateInputs()` before iteration (Go maps have random order)

### Tests
- `internal/crypto/envelope_test.go`: encrypt/decrypt round-trip, wrong key, corrupted data, empty input
- `internal/auth/jwt_test.go`: generate/validate round-trip, expired token rejection, invalid token
- `internal/auth/password_test.go`: hash/compare round-trip
- `web/src/stores/__tests__/pipelineStore.test.ts`: addNode, removeNodes, updateNodeConfig, onConnect (port compat), save/load serialization
- `web/src/stores/__tests__/historyStore.test.ts`: push/undo/redo/clear, MAX_HISTORY enforcement
- `web/src/lib/__tests__/portCompatibility.test.ts`: all port type combinations

### Parallelization
```
Lane A: Streaming refactor (internal/engine/, internal/llm/)
Lane B: Security fixes + backend tests (internal/auth/, internal/crypto/, internal/store/postgres/)
Lane C: Frontend tests (web/src/stores/, web/src/lib/)

Launch A + B + C in parallel. Merge all.
```

## Phase 1B — Should-Haves (weeks 3-5)

### Re-run From Any Node
- Add `StepResultStore.GetByExecution(ctx, executionID) -> []StepResult` store method
- Engine: reconstruct `nodeOutputs` map from prior StepResults, skip completed nodes, execute from selected node onward
- Frontend: "Re-run from here" button on each node
- Invalidate cached results when pipeline graph is modified (nodes/edges changed since last run)

### Error Recovery UX
- Node error state: show error message, offer 3 actions:
  - (a) Retry with same inputs
  - (b) Edit prompt/config in inspector, re-run just this node
  - (c) Select fallback provider (dropdown in error state)
- No automatic retry — creator should see and decide

### API Key Management UI
- Settings page with per-provider credential CRUD
- Form: provider selector, label, API key input
- Backend: existing credential store + envelope encryption handles storage
- AgentInspector: credential selector dropdown (populated from credential store)

### Frontend Auth
- Login page with email/password form
- Token storage in localStorage (access token) + httpOnly cookie (refresh token) or localStorage
- `api.ts`: inject `Authorization: Bearer <token>` header on every request
- Refresh logic: intercept 401, call refresh endpoint, retry original request
- Logout: clear tokens, redirect to login

### Gemini Polish
- Complete model listing endpoint for Gemini provider
- Test Gemini streaming end-to-end
- Add Gemini-specific cost table entries for newer models

## Phase 2 — Media Integration (weeks 5-8)

### MediaProvider Interface
```go
type MediaProvider interface {
    Submit(ctx context.Context, req MediaRequest) (jobID string, error)
    Poll(ctx context.Context, jobID string) (MediaStatus, error)
}

type MediaStatus struct {
    State   string // pending, processing, succeeded, failed
    Output  []byte // result data (image bytes, video URL, audio bytes)
    Error   string
}
```
- New package: `internal/media/`
- Provider registry (mirrors `internal/llm/registry.go` pattern)
- MediaExecutor: Submit, poll with exponential backoff, timeout after configurable duration

### Providers
- **Image gen**: Replicate API (Flux, SDXL) — primary. OpenAI DALL-E — secondary (synchronous).
- **Video gen**: Replicate-hosted models (Stable Video Diffusion, AnimateDiff) — primary. Runway ML as stretch (UNVERIFIED). Higgsfield deferred (UNVERIFIED).
- **Audio/TTS**: OpenAI TTS API — primary (synchronous). ElevenLabs — secondary.

### Frontend
- New node types: `ImageGeneratorNode`, `VideoGeneratorNode`, `AudioGeneratorNode`
- These are distinct from AgentNode (use MediaProvider, not LLMProvider)
- Each has provider selector + model config in inspector panel
- Media preview: image thumbnails inline, audio waveform player, video preview with play button

### Execution
- Sequential for now (design supports concurrent upgrade later)
- Media polling: submit job, poll every 2s with exponential backoff, timeout after 5 minutes
- Progress events: publish polling status updates via EventBus

## Phase 3 — Pipeline Export/Import (week 8-9)

### Export
- Serialize current canvas pipeline to JSON file
- Includes: nodes, edges, agent configs, provider settings, viewport
- Excludes: API keys, execution results, user data

### Import
- Load JSON pipeline file onto canvas
- Create all nodes and connections
- Validate: port compatibility, required fields, no duplicate IDs
- Handle: malformed JSON (validation error, not crash)

### Version Snapshots
- Build on existing `pipeline_versions` table and store
- Save immutable version before each execution
- Version history already has UI in versionStore

## Phase 4 — Templates + Landing (weeks 9-12)

### Pipeline Templates
- Templates are JSON pipeline files bundled with the app
- Built on existing preset system: agent presets (7 exist) define individual agents, templates compose them
- 5 initial templates:
  1. **YouTube Story**: Script Writer -> Character Designer -> Scene Director -> Image Gen -> Video Gen
  2. **Instagram Reel**: Prompt Enhancer -> Image Gen -> Short Video Gen
  3. **Character Sheet**: Character Designer -> Image Gen (multiple outputs)
  4. **Script-to-Storyboard**: Script Writer -> Scene Director -> Image Gen grid
  5. **Prompt Enhancer Chain**: Text Input -> Prompt Enhancer -> enhanced output

### Template Gallery UI
- Browse templates with preview (node graph thumbnail)
- Fork-to-canvas: creates a copy on the user's canvas that they can modify
- Category filtering (video, image, text)

### Landing Page
- Product positioning: "One canvas for your AI creative pipeline"
- Demo video showing a real pipeline in action
- Waitlist signup
- Deploy as static page or within the app

### Canvas Polish
- Keyboard shortcuts (Delete, Ctrl+C/V, Ctrl+Z/Shift+Z already partial)
- Auto-layout via ELKjs
- Helper lines for alignment
- Copy/paste nodes

## Deferred (Not In Scope)

| Item | Rationale | Revisit When |
|------|-----------|-------------|
| Custom pipeline DSL | JSON export covers sharing | Users find JSON insufficient |
| Community sharing | No external users | Post-launch, users exist |
| Goroutine-per-node | Sequential sufficient for linear pipelines | Fan-out pipelines needed |
| nsjail sandboxing | Single user, no untrusted code | Multi-tenant cloud deployment |
| RLS multi-tenancy | Basic tenant scoping sufficient | Multiple orgs on same instance |
| SAML SSO | Enterprise feature | Enterprise customers exist |
| Connector marketplace | All integrations are first-party | Third-party demand |
| Video timeline editor | Clotho produces assets | Creators request it |
| Observability (OTEL) | slog sufficient | Production multi-user |
| Variable interpolation | Scope undefined | Phase 1.5 or later |
| Visual polish (dark luxury) | Functional first | Phase 1.5 or Phase 4 |
