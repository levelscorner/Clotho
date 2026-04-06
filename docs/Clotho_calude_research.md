# Clotho: production architecture and implementation blueprint

**Clotho should be built on Go + PostgreSQL + React Flow — three components for day one — with a Postgres-based execution queue, nsjail sandboxing, and envelope encryption.** This stack is not theoretical: Windmill proves the Postgres-queue model at **26M jobs/month per worker**, React Flow powers Stripe's workflow builder, and Go is the language of the orchestration ecosystem (Temporal, Kubernetes, etcd). The competitive window is real: n8n has disclosed **10+ critical CVEs since late 2025** including unauthenticated RCE chains, with CISA confirming active exploitation. Windmill serves developers only. Activepieces is still maturing. The differentiation opportunity lies in three compounding bets: time-travel debugging, AI-native workflow primitives, and first-class human-in-the-loop — capabilities that require architectural decisions made at the foundation, not bolted on later.

---

## Dimension 1: Architecture and tech stack decisions

### Go wins the backend language decision by elimination

Go is the correct choice for Clotho's core API and execution orchestration layer. The reasoning is structural, not preferential.

**Concurrency model** is the decisive factor. Workflow orchestration means managing thousands of parallel HTTP calls, database queries, and execution contexts simultaneously. Go's goroutines — **2KB stack, multiplexed onto OS threads** — are purpose-built for this pattern. Node.js/TypeScript's single-threaded event loop requires worker threads or process clustering for CPU-bound orchestration work, adding accidental complexity. Rust delivers superior raw performance, but Windmill's CTO has publicly stated that their Postgres queue speed "does not really benefit from Rust and could be implemented in any language."

**Ecosystem alignment** seals it. Temporal (the gold standard for durable execution) is written in Go with a first-class Go SDK. The entire cloud-native stack — Kubernetes, Docker, Terraform, etcd, NATS — is Go. gRPC support is canonical via `google.golang.org/grpc`. PostgreSQL drivers (`pgx`) are excellent. Hiring is practical: Go's **25-keyword language** means **2–4 week onboarding** for experienced backend engineers, versus 2–4 months for Rust.

**The framework choice is Chi** (or Go 1.22+ standard library with Chi middleware). Chi is a thin layer over `net/http`, making all standard middleware compatible. Gin has 48% market share but uses its own `gin.Context`, creating coupling. Fiber uses `fasthttp` instead of `net/http`, breaking ecosystem compatibility. Chi handlers are standard `http.Handler` — zero lock-in, identical performance to Gin in benchmarks, and used by Cloudflare and Heroku in production.

| Component | Choice | Trade-off |
|-----------|--------|-----------|
| Language | **Go** | Slower iteration than TypeScript, but the concurrency model and ecosystem alignment outweigh this |
| HTTP Framework | **Chi** | Smaller community than Gin, but stdlib-compatible and zero lock-in |
| gRPC | **google.golang.org/grpc** | Canonical implementation, no alternative needed |

### Build a Postgres-based execution engine, not Temporal

This is Clotho's most consequential architectural decision. **Build a Postgres-queue execution engine modeled after Windmill's approach.** Do not integrate Temporal for v1.

Windmill's execution model uses PostgreSQL's `UPDATE ... FOR UPDATE SKIP LOCKED` as a job queue. Workers poll the `queue` table, atomically claim jobs, execute them, and move results to a `completed_job` table. Every state transition is a **single atomic Postgres statement** — no distributed consensus, no eventual consistency. The actual SQL:

```sql
UPDATE queue
SET running = true, started_at = coalesce(started_at, now()), last_ping = now()
WHERE id = (
    SELECT id FROM queue
    WHERE running = false AND scheduled_for <= now() AND tag = ANY($1)
    ORDER BY priority DESC NULLS LAST, scheduled_for, created_at
    FOR UPDATE SKIP LOCKED
    LIMIT 1
) RETURNING *
```

This scales to **~5,000 jobs/second** on a standard Postgres instance — sufficient for years of growth. The architecture uses just two tables: `queue` for active/pending jobs and `completed_job` for finished results. Flow state transitions are stored as JSONB, and zombie detection works via periodic `last_ping` checks.

**Why not Temporal?** Temporal requires **4+ services** (Frontend, History, Matching, Worker) plus Postgres/Cassandra plus Elasticsearch plus a monitoring stack. Datadog's engineering team described their Temporal self-hosting journey as "marked by hard lessons and high-stakes incidents." Self-hosted users need **significant distributed systems expertise**. For a platform that promises `docker compose up` simplicity, Temporal is architecturally incompatible with the self-hosting story.

Temporal also solves the wrong problem. It provides durable execution of *your own deterministic code*. Clotho needs to execute *arbitrary user code in sandboxed environments* across multiple language runtimes, manage flow state transitions, and provide a visual graph editor — none of which Temporal addresses. You'd still build everything around it.

**When to reconsider:** Only if Clotho pivots to offering SDK-level durable execution as a product feature, or if concurrent workflow executions exceed 10K sustained. For the foreseeable roadmap, Postgres-as-queue is the correct abstraction.

**Skip Redis/BullMQ, NATS, and RabbitMQ for the queue.** If a message broker is needed later (for real-time events, webhook fanout, inter-service communication), choose **NATS JetStream**: a single Go binary, first-class Go SDK (NATS is written in Go), CNCF incubating, with at-least-once delivery and built-in RAFT clustering.

### React Flow is the only serious option for the graph editor

The xyflow ecosystem (React Flow / Svelte Flow / VueFlow) dominates the workflow editor space. n8n uses VueFlow, Windmill uses Svelte Flow, and Stripe's workflow builder uses React Flow directly. Activepieces explicitly regretted not having it when they were on Angular — their migration to React cited React Flow as a key motivator.

React Flow (`@xyflow/react` v12) is **MIT licensed** — the library itself is fully open-source. The "Pro" subscription provides code examples and templates, not gated features. The library renders nodes as actual React components (SVG/HTML DOM, not Canvas2D), which means the full React component model applies for custom nodes.

**Performance with 100+ nodes** requires one critical optimization: `React.memo()` on all custom node components. Synergy Codes benchmarked this: without memo, dragging drops to **2 FPS** for complex nodes; with memo, it sustains **60 FPS**. Additional optimizations include `onlyRenderVisibleElements` for viewport culling, `useCallback` for event handlers, and fine-grained Zustand selectors.

**State management should use Zustand** (separate store from React Flow's internal store). React Flow already uses Zustand internally, so you get a consistent mental model. The architecture splits clearly: React Flow manages viewport, selection, and drag state; your Zustand store manages workflow metadata, node configurations, execution status, and undo/redo history. Redux Toolkit adds boilerplate without benefit. Jotai's atomic model doesn't match the inherently interconnected graph structure.

### PostgreSQL is the only viable database choice

All three open-source competitors — n8n, Windmill, and Activepieces — use PostgreSQL. CockroachDB lacks full RLS support and adds latency for single-region deployments. PlanetScale is MySQL-based, has no RLS equivalent, and cannot self-host. PostgreSQL provides native Row-Level Security for tenant isolation, mature JSONB support for workflow definitions, and declarative partitioning for execution logs.

**Schema strategy: versioned JSON in shared tables with RLS.** Store workflow definitions as JSONB with version numbers (following Activepieces' `flow_version` pattern — never mutate published flows). Use shared tables with `tenant_id` column, not separate schemas per tenant. Enforce isolation via RLS:

```sql
CREATE OR REPLACE FUNCTION current_tenant_id() RETURNS UUID AS $$
  SELECT current_setting('app.current_tenant', true)::UUID;
$$ LANGUAGE SQL STABLE;

ALTER TABLE workflows ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON workflows
  USING (tenant_id = current_tenant_id());
```

Execution logs should use **time-based partitioning** via `pg_partman` (monthly partitions). Old partitions detach, export to S3 as Parquet, and drop — instant deletion versus row-by-row DELETE. Store large execution payloads (input/output data) in S3; keep only metadata and pointers in Postgres.

### Observability: OpenTelemetry plus Grafana LGTM

The observability stack is **OpenTelemetry + Grafana (Loki + Tempo + Prometheus/Mimir)**. The Go OTel SDK is production-ready: traces and metrics are GA stable. This stack is vendor-neutral (apps instrument once, export anywhere), self-hostable (critical for self-hosted customers), and used by Windmill in production. Auto-instrumentation packages exist for net/http (`otelhttp`), gRPC (`otelgrpc`), SQL (`otelsql`), and Chi (`otelchi`).

Workflow-level tracing should use a span-per-node hierarchy: a root `workflow.execution` span contains child `workflow.node.*` spans, each with attributes for node type, input/output size, duration, and status. Correlate with infrastructure traces via shared `trace_id` propagated as baggage.

### Secrets: envelope encryption with a pluggable backend

n8n uses a single `N8N_ENCRYPTION_KEY` that directly encrypts all credentials — losing the key means all credentials are unrecoverable. This is inadequate.

Clotho should implement **envelope encryption** with a pluggable KEK (Key Encryption Key) backend:

1. **Default (self-hosted):** Self-managed AES-256-GCM. Master key from `CLOTHO_MASTER_KEY` env var. Per-workspace DEKs (Data Encryption Keys).
2. **Cloud option:** AWS KMS or GCP KMS as the KEK provider.
3. **Enterprise option:** HashiCorp Vault Transit engine integration.

The key hierarchy: Master Key → per-workspace DEK → encrypted credential value. DEKs are encrypted with the master key and stored alongside the ciphertext. This enables per-tenant key isolation and master key rotation without re-encrypting all data.

**Credential scoping uses three tiers:** Platform-level (admin-managed, shared across all workspaces), Workspace-level (team-shared), and Personal (individual user). Workflows reference credentials by ID — values are injected at execution time only, never embedded in workflow definitions.

---

## Dimension 2: MVP implementation blueprint

### The graph editor MVP requires seven node types and twelve interaction patterns

Based on what n8n, Windmill, and Activepieces ship as table stakes, the minimum viable editor needs these **node types**: Trigger (webhook, cron, event — distinct green color, always first), Action (API call with integration icon), Conditional/Branch (two output handles for true/false), Code (embedded Monaco editor), Loop/Iterator (for-each over arrays), HTTP Request (generic API call), and Delay/Wait (timer-based pause).

**Table-stakes interactions**: drag-and-drop from sidebar palette, click-to-connect between handles, zoom/pan/scroll, minimap overview, undo/redo (Ctrl+Z), delete nodes/edges (Backspace), multi-select (Shift+click or lasso), copy/paste (Ctrl+C/V), auto-layout ("tidy up" button via ELKjs), node inspector panel (right sidebar on selection), execution visualization (green/red borders for success/failure), and save/load to backend.

The **React Flow Pro subscription is worth the investment**. The undo/redo, helper lines, auto-layout, and copy/paste examples save 2–3 weeks of development each. You get perpetual access to any examples accessed during the subscription.

### The execution engine MVP fits in a single Go function

The minimum viable execution engine is surprisingly simple when built on Postgres-as-queue:

```go
func (e *Engine) ExecuteWorkflow(ctx context.Context, wf *Workflow, input json.RawMessage) {
    results := make(map[string]*StepResult)
    for _, step := range wf.TopologicalOrder() {
        stepCtx, cancel := context.WithTimeout(ctx, step.Timeout)
        result := e.executeStepWithRetry(stepCtx, step, results)
        results[step.ID] = result
        cancel()
        if result.Status == "failed" && step.OnError == "stop" { break }
    }
}
```

Sequential node execution with DAG support via topological sort, per-step timeouts via `context.WithTimeout`, configurable retry with exponential backoff, and step results stored as JSONB in Postgres. Each step runs in a goroutine. No subprocess spawning needed for Go-native connectors.

**Webhooks** should follow n8n's two-URL pattern: a test URL (active during editor debugging, 120s TTL) and a production URL (active when workflow is published). Route incoming webhooks via `POST /api/v1/webhooks/{webhook_id}` → database lookup → enqueue execution job. For fanout (one webhook triggering multiple workflows), enqueue one independent job per matching workflow.

**Real-time log streaming should use SSE (Server-Sent Events)**, not WebSockets. SSE is unidirectional (server → client, exactly right for log streaming), has built-in browser reconnection with `Last-Event-ID`, works with standard HTTP infrastructure, and multiplexes over HTTP/2. The Go implementation is straightforward: set `Content-Type: text/event-stream`, flush after each event, subscribe to execution updates via Postgres `LISTEN/NOTIFY` or Redis Pub/Sub as backplane.

### The connector SDK should follow Activepieces' TypeScript model

The connector interface defines three core concepts: **TriggerDefinition** (webhook or polling), **ActionDefinition** (execute an operation), and **AuthDefinition** (OAuth2, API key, or basic). Connectors are TypeScript npm packages using a framework SDK. The minimal interface:

```typescript
interface ConnectorDefinition {
  name: string;
  displayName: string;
  version: string;
  auth: AuthDefinition;
  triggers: TriggerDefinition[];
  actions: ActionDefinition[];
}

interface ActionDefinition {
  name: string;
  props: PropDefinition[];
  run: (ctx: ActionContext) => Promise<ActionOutput>;
  rateLimit?: { requests: number; windowMs: number };
}

interface ActionContext {
  auth: Record<string, string>;   // injected credentials — never raw
  props: Record<string, any>;
  http: HttpClient;                // pre-authenticated HTTP client
  logger: Logger;
}
```

**OAuth follows the proxy pattern**: the platform registers one OAuth app per provider. Users authorize through Clotho's app. Clotho stores the refresh token (encrypted), auto-generates fresh access tokens, and injects them into connector HTTP calls server-side. The connector code never sees raw credentials — this is the critical defense against supply-chain attacks.

### Multi-tenancy: RLS from day one, three-level RBAC

Implement PostgreSQL RLS with `SET LOCAL app.current_tenant` in every request's middleware. Performance overhead is minimal for simple equality checks on indexed columns. Use `SET LOCAL` for transaction-scoped settings, which works correctly with connection pooling (PgBouncer in transaction mode).

The RBAC model synthesized from all three competitors: **Instance-level** (Superadmin, Platform Admin), **Organization/Workspace-level** (Owner, Admin, Member), and **Project-level** (Admin, Editor/Developer, Operator, Viewer, Custom Roles). Permission scopes cover workflows (create, read, update, execute, delete), credentials (create, read, use, delete), and executions (read, retry, delete).

### Self-hosting: three files, one command

Follow Windmill's proven pattern: a **single Go binary** that runs as either server or worker based on a `MODE` environment variable. The minimum Docker Compose is three services: API (server mode), Worker (worker mode, multiple replicas), and PostgreSQL. No Redis needed when using Postgres-as-queue. Database migrations run automatically on server startup, embedded in the binary.

```yaml
services:
  api:
    image: clotho/server:latest
    environment:
      - MODE=server
      - DATABASE_URL=postgres://clotho:secret@db:5432/clotho
      - ENCRYPTION_KEY=${ENCRYPTION_KEY}
    ports: ["8080:8080"]
  worker:
    image: clotho/server:latest
    environment:
      - MODE=worker
      - DATABASE_URL=postgres://clotho:secret@db:5432/clotho
    deploy: { replicas: 2 }
  db:
    image: postgres:16-alpine
    volumes: [pgdata:/var/lib/postgresql/data]
```

For Kubernetes, the Helm chart needs: API Deployment + Service, Worker Deployment (with HPA), ConfigMap, Secret, Ingress, and a **pre-install/pre-upgrade migration Job** that runs `clotho migrate up` before the new pods start.

---

## Dimension 3: Competitive gaps and differentiation

### n8n's security crisis is Clotho's strategic opening

n8n has disclosed **10+ critical/maximum-severity CVEs** in a compressed timeframe (late 2025 through early 2026). The most alarming:

- **CVE-2026-21858 (CVSS 10.0, "Ni8mare")**: Unauthenticated file access via webhook content-type confusion, enabling full RCE without any credentials.
- **CVE-2025-68613 (CVSS 9.9)**: Expression injection through insufficient sandbox isolation — **added to CISA's Known Exploited Vulnerabilities catalog**, confirming active in-the-wild exploitation.
- **CVE-2025-68668 (CVSS 9.9, "N8scape")**: Sandbox bypass enabling arbitrary command execution on the host.

Censys identified **26,512 exposed n8n hosts** (another count found 103,000+). These CVEs can be **chained**: unauthenticated entry via CVE-2026-21858 combined with sandbox escape via CVE-2025-68613 gives full RCE from zero access. The Canadian Cyber Centre issued a specific alert (AL26-001) for n8n — rare for a workflow tool.

The architectural root cause: **n8n has no code sandboxing**. Community nodes run with the same access level as n8n itself, including decrypted API keys and OAuth tokens. In January 2026, attackers published **8+ malicious npm packages** disguised as n8n community nodes that exfiltrated OAuth tokens to attacker-controlled servers using n8n's own master encryption key to decrypt credentials during workflow execution.

Beyond security, real user complaints consistently cite: **memory leaks** in webhook-triggered workflows (GitHub #16862, #17154 — container crashes despite low memory usage), **execution time regressions** from updates (0–3 seconds to 60+ seconds in v1.105–1.106), **poor debugging** (no step-by-step replay, cryptic error messages), and **licensing friction** (execution limits on self-hosted enterprise, SSO locked behind paywall, "Fair Code" versus actual open source).

### Windmill is developer-only, Activepieces is still early

**Windmill's gap** is accessibility. It's explicitly code-first — far fewer pre-built integrations than n8n, no drag-and-drop visual builder for non-technical users. GitHub Issue #5014 ("An open letter to the Windmill team") catalogs licensing frustrations: alerting on failures requires Enterprise Edition, Git sync and RBAC are paywalled, and per-worker pricing creates unpredictable costs.

**Activepieces' gap** is maturity. With **~644 integrations versus Zapier's 7,000+**, limited integrations is the #1 complaint. Users report UI glitches with large flows, confusing error handling, and pricing plan changes that eroded trust (users moved from paid plans to trial plans without notification).

### The three differentiation bets that compound

These aren't incremental improvements — they require architectural decisions that competitors cannot easily retrofit.

**Bet #1: Durable execution with time-travel debugging.** No open-source workflow tool offers deterministic replay. n8n has no real debugger. Temporal offers replay but requires code. The innovation is combining Temporal-grade event sourcing with a visual step-through debugger: play/pause/scrub through any execution on the canvas, fork from any step with modified inputs, set breakpoints that pause execution for inspection, and compare two executions side-by-side to understand why one succeeded and another failed. This requires recording every node's input/output as immutable events — an architectural decision that must be made at the foundation. **Estimated effort: 4–6 months.**

**Bet #2: AI-native workflow engine.** Current platforms bolt AI nodes onto traditional automation. n8n has ~70 LangChain nodes but no prompt versioning, no model routing with circuit breakers, no token cost tracking, and no evaluation nodes. Zapier offers single-shot LLM calls without streaming or tool-use orchestration. The opportunity is a platform where AI is a first-class primitive: Prompt Template nodes with Jinja2 variable injection, Model Router nodes with automatic failover, Tool Use nodes where LLMs call other workflow nodes, Evaluation nodes for quality scoring, and per-workflow cost tracking with budget caps. **Estimated effort: 3–4 months.**

**Bet #3: First-class human-in-the-loop.** n8n's HITL is built on Wait nodes and resume URLs — functional but crude, with no approval dashboard, no escalation, and no audit trail. The opportunity is a comprehensive approval system: configurable Approval Nodes with reviewer assignment, timeout, escalation chains, and conditional routing (e.g., "if amount > $10K, require VP approval"), plus a standalone approval inbox UI showing all pending approvals across workflows. This builds directly on durable execution from Bet #1. **Estimated effort: 2–3 months.**

**These three bets compound.** Durable execution enables proper HITL and reliable AI workflows. Time-travel debugging makes AI workflows trustworthy. The approval inbox makes AI-generated outputs safe for production. Together, they position Clotho as "Temporal for visual workflows, with AI as a first-class citizen" — a space no competitor currently occupies.

### Error handling is a stealth differentiator

An audit of one client's n8n instance found **847 workflows running 14 months with zero error handling**, causing a measurable "4.7% lead leak." n8n provides no built-in error handling by default. Clotho should ship with per-node retry policies (exponential backoff with jitter, retryable vs. fatal error classification), circuit breakers for external APIs (trip after N consecutive failures, half-open testing after cooldown), dead letter queues (permanently failed executions go to a review queue, never dropped), and partial execution recovery (resume from the exact failed step, preserving all prior step outputs).

---

## Dimension 4: Security and sandboxing

### Start with nsjail, design for Firecracker

Clotho should implement a **tiered sandboxing model** with a swappable execution boundary.

**Tier 1 (MVP/self-hosted): nsjail.** This matches Windmill's proven approach. nsjail is a lightweight Linux process isolation tool from Google that combines kernel namespaces (PID, mount, network, user), cgroups (memory, CPU, PID limits), and seccomp-bpf (syscall filtering). Startup overhead is **1–5ms** versus 125ms for Firecracker. Windmill recently open-sourced nsjail sandboxing in their Community Edition (v1.634.0+). The limitation is shared-kernel isolation: a kernel vulnerability could theoretically escape the sandbox.

**Tier 2 (cloud/enterprise): Firecracker microVMs.** Each workflow step executes in its own Firecracker microVM with hardware-enforced KVM isolation. Boot time is **<125ms** (sub-10ms with snapshot/restore). Memory overhead is **<5 MiB per VM**. This is the approach used by AWS Lambda (trillions of executions monthly) and Fly.io. Use pre-built root filesystem images per language runtime, pass code and inputs via virtio-vsock, and leverage snapshotting for warm pools.

**gVisor** serves as a middle ground for cloud deployments without KVM access. It's a user-space kernel (written in Go) that intercepts syscalls in the Sentry process — the host kernel never sees raw application syscalls. Drop-in OCI runtime (`runsc`) compatible with Docker and Kubernetes. Ant Group reports 70% of applications run with <1% overhead after optimization.

| Sandbox | Isolation | Startup | Self-host ease | Best for |
|---------|-----------|---------|----------------|----------|
| **nsjail** | Namespace + seccomp (shared kernel) | 1–5ms | Easy | Default/self-hosted |
| **gVisor** | User-space kernel (reduced kernel surface) | 50–100ms | Medium | Cloud without KVM |
| **Firecracker** | Hardware virtualization (KVM) | 100–200ms (<10ms warm) | Hard | Multi-tenant cloud |

### The credential proxy pattern prevents n8n-style attacks

The single most important security decision is: **connectors must never receive raw credentials.** Instead, implement a server-side credential proxy:

```
Connector code: clotho.http.get("https://api.google.com/ads", { auth: "credential_id" })
Clotho runtime: Injects OAuth token into request header server-side
```

This eliminates the attack vector that compromised n8n: malicious community nodes decrypting and exfiltrating stored OAuth tokens. The connector SDK's `ActionContext.http` is a pre-authenticated HTTP client — the connector declares which credential it needs, and the runtime injects auth headers transparently.

### Connector marketplace security requires a three-tier trust model

The n8n supply-chain attack (January 2026) is the canonical cautionary tale. Attackers published 8+ malicious npm packages posing as n8n community nodes, impersonating integrations like Google Ads, Stripe, and Salesforce. The packages presented legitimate-looking OAuth configuration screens, collected credentials, and exfiltrated them using n8n's master key.

Clotho's marketplace should enforce three trust tiers:

- **Official** (Clotho-built): Internal code review + security audit, full API access, nsjail sandbox.
- **Verified** (partner/community-reviewed): Automated Semgrep scan + manual security review, nsjail sandbox, restricted network to declared endpoints only.
- **Community** (unverified): Automated scan only, strict nsjail or gVisor/Firecracker isolation, no credential access beyond declared scopes, **credential proxy enforced** (connector never sees tokens).

All published connectors must require **npm provenance via Sigstore** — verifying that the published artifact matches a specific commit in a verified CI/CD pipeline. Combined with a **permission manifest** (declared network endpoints, credential types, resource limits) and automated static analysis (Semgrep custom rules for credential exfiltration patterns, `eval()`, dynamic `require()`).

### Auth: start custom, migrate to Ory Kratos

For the auth stack, the pragmatic path is: **custom JWT auth for MVP** (using `golang-jwt` + session management), then **migrate to Ory Kratos + Hydra** for production. Ory is the only open-source solution that's cloud-native, written in Go, and supports both SaaS and self-hosted deployments natively. Unlike Keycloak, it doesn't require a JVM. Unlike Auth0/Clerk, it self-hosts without enterprise pricing.

This matches what Windmill and Activepieces did — custom auth core with SSO layered on top. Both use environment-variable encryption keys (`N8N_ENCRYPTION_KEY`, `AP_ENCRYPTION_KEY`) for credential storage.

**SSO implementation order**: OIDC first using `coreos/go-oidc` + `golang.org/x/oauth2` (supports Google, Microsoft, Okta — **2–3 weeks of work**), then SAML 2.0 using `crewjam/saml` as an enterprise-plan feature (**4–6 additional weeks**). Gate SAML behind the enterprise tier, following industry standard practice.

---

## Architecture summary and open decisions

### The day-one stack

| Layer | Choice | Why | Main trade-off |
|-------|--------|-----|----------------|
| Language | **Go** | Goroutine concurrency, orchestration ecosystem, hiring | Slower initial velocity than TypeScript |
| Framework | **Chi** | stdlib-compatible, zero lock-in | Smaller community than Gin |
| Execution queue | **PostgreSQL (SKIP LOCKED)** | Zero additional infra, ACID guarantees | ~5K jobs/sec ceiling per instance |
| Graph editor | **React Flow (@xyflow/react)** | MIT, used by Stripe, industry standard | DOM-based rendering caps at ~500–1000 nodes |
| State management | **Zustand** | React Flow uses it internally, selector-based re-renders | Less structured than Redux for large teams |
| Database | **PostgreSQL 16** | RLS, JSONB, partitioning, competitor-proven | Single-node by default |
| Observability | **OpenTelemetry + Grafana LGTM** | Vendor-neutral, self-hostable, Go SDK stable | Higher setup complexity than Datadog |
| Secrets | **Envelope encryption (AES-256-GCM)** | Pluggable backend (KMS/Vault/self-managed) | Self-managed key rotation is manual |
| Sandboxing | **nsjail** (default) → **Firecracker** (cloud) | Windmill-proven, tiered by deployment model | nsjail shares host kernel |
| Auth | **Custom JWT → Ory Kratos** | Go-native, OSS, supports SaaS + self-host | Ory SAML support still maturing |
| Log streaming | **SSE** | Unidirectional, auto-reconnect, HTTP-native | Not bidirectional (use WebSocket if needed later) |
| Connector SDK | **TypeScript npm packages** | Largest ecosystem, type-safe, community-contributable | Requires Node.js runtime for connector execution |

### Open product decisions that require deliberation

**Event sourcing depth.** Full event sourcing (Temporal-style immutable event history per execution) enables time-travel debugging and deterministic replay but adds storage costs and implementation complexity. The alternative is point-in-time snapshots (store input/output per step) — simpler but no deterministic replay. **Recommendation: full event sourcing.** It's foundational to Differentiation Bet #1 and cannot be retrofitted.

**Connector runtime isolation.** Connectors are TypeScript packages, but the core engine is Go. Options: (a) embed a JS runtime in Go (V8 via `nicholasgasior/gowv8` or Deno), (b) spawn a sidecar Node.js process, (c) use Firecracker microVMs for connector execution. Option (b) is simplest for MVP; option (c) is most secure for multi-tenant cloud. **This requires a prototype to validate performance.**

**Pricing model.** The market's biggest complaint about Zapier is per-task pricing. n8n's execution limits on self-hosted enterprise generated significant backlash. Per-workflow pricing (unlimited executions per workflow) would be a genuine differentiator. **This is a product decision, not a technical one, but it should be decided before the billing schema is designed.**

**AI model hosting.** Should Clotho proxy all LLM calls through its own infrastructure (enabling cost tracking, caching, rate limiting) or let users bring their own API keys? **Recommendation: both.** Default to user-provided API keys with optional Clotho-proxied calls for cost tracking and caching. This mirrors the credential proxy pattern.

### Build order for the first six months

**Month 1–2:** Core engine (Postgres queue, sequential execution, step timeout/retry), basic API (Chi + CRUD for workflows), database schema with RLS, single Go binary with server/worker mode. **Month 2–3:** React Flow graph editor (custom nodes, inspector panel, save/load), webhook trigger infrastructure, basic auth (JWT), Docker Compose packaging. **Month 3–4:** Connector SDK (TypeScript), first 10 connectors (HTTP Request, Slack, Gmail, GitHub, OpenAI, Postgres, webhook, cron, code, conditional), OAuth proxy, credential encryption. **Month 4–5:** Execution visualization (SSE streaming, node status), undo/redo, nsjail sandboxing, execution history UI. **Month 5–6:** Event sourcing foundation (immutable execution events), basic replay/debugging, AI nodes (Prompt Template, LLM Call), Helm chart, first public beta.