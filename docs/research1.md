# Typed pipelines and model introspection for AI-native workflow orchestration

Clotho's core engineering challenge — chaining heterogeneous AI models (LLMs, TTS, STT, image gen, video gen) in validated pipelines — has well-established solutions scattered across at least a dozen frameworks and emerging academic work. **The central insight is that every pipeline connection needs a typed contract enforced at runtime, not just at design time.** The specific failure mode described (a TTS agent returning text advice instead of audio bytes) is a solved problem when you combine three techniques: forcing tool-call invocation over free-text responses, validating output MIME types via magic-byte detection, and implementing retry-with-error-feedback loops. This report synthesizes findings across model metadata standards, validation frameworks, academic research, practical implementation patterns, and web crawling integration to provide an actionable architecture for Clotho.

---

## Every AI model type has discoverable, schema-definable interfaces

The good news is that **every major model type has a well-defined input/output contract** already documented via OpenAPI specifications, and existing platforms already auto-generate configuration UIs from these schemas.

**LLMs** accept messages arrays (with roles and multimodal content parts) and return text, tool calls, or structured JSON. Their configuration surface includes **temperature** (0–2), **top_p** (0–1), **max_tokens**, **stop sequences**, **presence/frequency penalties** (-2.0 to 2.0), **response_format** (text/json_object/json_schema), and **seed** for reproducibility. Anthropic adds **top_k**; Google adds **safety_settings**.

**TTS models** accept text strings plus voice identifiers, returning binary audio streams. ElevenLabs exposes the richest parameter set: **stability** (0–1), **similarity_boost** (0–1), **style** exaggeration (0–1), **speed**, **output_format** (mp3_44100_128, pcm_16000, etc.), and SSML parsing toggles. OpenAI's TTS is simpler: six voice options, speed (0.25–4.0), five output formats.

**Image generation models** take text prompts (optionally weighted), negative prompts, and init/mask images for img2img. Stable Diffusion exposes the largest parameter surface: **cfg_scale** (1–20), **steps** (1–150), **sampler** (DDIM, DPM++2M, Euler, etc.), **scheduler** (karras, exponential), **style_preset** (photographic, anime, digital-art), and dimension constraints (multiples of 64, 512–2048). DALL-E simplifies to quality/style enums and fixed size presets. Flux adds LoRA configuration arrays.

**STT, video generation, and multimodal models** follow similar patterns with domain-specific parameters — Whisper adds timestamp granularity and vocabulary hints; video models add duration, fps, and motion bucket IDs; vision models add image detail level controls.

### Existing standards already solve schema discovery

Five complementary standards provide the foundation for a model inspector:

**Replicate's Cog framework** is the closest existing solution to what Clotho needs. Model authors write typed Python `predict()` methods with `Input()` decorators specifying description, default, min/max, choices, and regex constraints. Cog auto-generates an **OpenAPI schema** accessible via API (`GET /v1/models/{owner}/{name}/versions/{id}` returns `openapi_schema`). Every parameter includes an `x-order` field for UI ordering. Replicate's web UI renders interactive forms directly from these schemas — **this is the exact pattern Clotho should replicate**.

**Fal.ai's approach** demonstrates runtime schema discovery. Their LOPs operator "fetches the API schema and dynamically creates input parameters, so the interface always matches the selected model's requirements." This includes smart type detection (identifying image URL parameters, converting to appropriate input types) and intelligent cleanup that only recreates UI elements when definitions actually change.

**HuggingFace model cards** use YAML front-matter with a critical **`pipeline_tag`** field that maps directly to inference pipeline type (text-generation, text-to-image, automatic-speech-recognition, text-to-speech). The Hub infers library type from file presence and maps tasks to standardized I/O schemas defined in `huggingface.js/packages/tasks/`.

**MLflow's Model Signature** provides a three-part schema (inputs, outputs, params) supporting both column-based (tabular) and tensor-based (images/audio) types with numpy dtype specifications and variable-dimension shapes. As of MLflow 2.20+, Python type annotations and dataclasses automatically generate schemas.

**OpenAI publishes a full OpenAPI specification** on GitHub, and **OpenRouter** normalizes schemas across 300+ models into a unified spec. Both use standard JSON Schema for structured output and tool definitions.

---

## Pipeline integrity requires layered validation, not just type checking

The described failure — a TTS agent responding with text advice instead of generating audio — exposes a fundamental gap: **API-level success (HTTP 200) does not mean task-level success**. Addressing this requires validation at three layers: structural (did we get the right data type?), semantic (did the model actually perform the task?), and quality (is the output good enough?).

### Six frameworks tackle structured output enforcement

**Outlines** (by dottxt) is the only framework providing **guaranteed valid output** with zero retries. It uses constrained decoding — modifying token logits during generation via finite state machines compiled from JSON schemas. This means O(1) valid token lookup per step, reportedly **5x faster** than unconstrained generation. The critical limitation: it requires access to model logits, so it works with self-hosted models but not API-only providers like OpenAI or Anthropic.

**OpenAI and Anthropic's native structured outputs** use similar constrained decoding on their servers, achieving **100% schema compliance** on evaluations. OpenAI's `response_format` with JSON Schema and Anthropic's `strict: true` tool definitions both compile grammars internally. Anthropic caches compiled grammars for 24 hours but has complexity limits (max 24 optional parameters across all strict tools).

**Instructor** patches LLM client libraries to add a `response_model` parameter enforcing Pydantic schemas. Its killer feature is **automatic retry-with-feedback**: when Pydantic validation fails, Instructor retries with the validation error embedded in the prompt, teaching the model to self-correct. It supports 15+ providers across Python, TypeScript, Go, Ruby, and Rust.

**Guardrails AI** wraps LLM calls with a `Guard` object and provides 100+ pre-built validators on a community hub. Its `OnFailAction.REASK` pattern re-prompts with error details. Uniquely, it can run as a **standalone Flask service** acting as an OpenAI-compatible proxy, making it deployable as infrastructure rather than application code.

**Pydantic AI** (by the Pydantic team) introduces `Agent[DepsType, OutputType]` — agents generic in both dependency type and output type. This is the most directly applicable pattern for Clotho's pipeline stages, as each component becomes a typed agent with compile-time checking via IDE autocomplete and static analysis. Its `@agent.output_validator` decorator with `ModelRetry` exception provides clean retry semantics.

**LangChain's output parsers** include `OutputFixingParser` (uses another LLM to fix invalid output) and `RetryOutputParser` (retries with enhanced instructions), but the framework is shifting toward native `model.with_structured_output(schema)` that leverages provider-level structured outputs.

### Binary output validation solves the TTS failure mode directly

For the specific TTS failure, the solution is straightforward: **validate output bytes against expected MIME types using magic-byte detection**. Libraries like `python-magic` (wraps libmagic) or `filetype` (pure Python, checks first 261 bytes) can verify that audio output is actual MP3 (magic bytes `FF FB` or ID3 header `49 44 33`) rather than UTF-8 text. The `file-type` npm package needs at most **4,100 bytes** to detect all supported formats.

The detection strategy layers four checks: (1) verify `stop_reason` equals `tool_use` not `end_turn` — if the model returned text instead of calling the generation tool, reject immediately; (2) run magic-byte MIME detection against expected format; (3) apply size heuristics (text advice is ~500–2000 bytes, actual audio is much larger); (4) check if output is valid UTF-8 when binary was expected (a definitive indicator of task failure).

---

## Academic research converges on typed, declarative pipeline contracts

### The Open Agent Specification defines the target architecture

The most directly relevant academic work is the **Open Agent Specification (Agent Spec)** from Oracle (arXiv:2510.04173, October 2025). It introduces a declarative, framework-agnostic specification language where every component has **typed Inputs/Outputs conforming to JSON Schema standards**. Workflow connectivity uses explicit **DataFlowEdges** and **ControlFlowEdges** enabling both static and runtime validation. The paper demonstrates interoperability across LangGraph, CrewAI, AutoGen, and WayFlow runtimes, with a PyAgentSpec SDK providing serialization with round-trip guarantees.

**Agent Behavioral Contracts (ABC)** (arXiv:2602.22302) formalizes Design-by-Contract for AI agents using the tuple **C = (P, I, G, R)** — preconditions, invariants, governance policies, and recovery mechanisms. It introduces **(p, δ, k)-satisfaction**, a probabilistic contract compliance measure accounting for LLM non-determinism, and proves a **Drift Bounds Theorem**: contracts with recovery rate γ > natural drift rate α bound behavioral drift to D* = α/γ. The associated AgentAssert library detected 5.2–6.8 soft violations per session that uncontracted baselines missed entirely.

**DSPy** (Stanford NLP, ICLR 2024) pioneered **natural-language typed signatures** as declarative function declarations for pipeline modules. A signature like `"question: str -> answer: str, confidence: float"` defines both the interface contract and the optimization target. DSPy's compiler optimizes entire pipelines to maximize metrics, with Pydantic model support for complex structured types.

### Open-source frameworks provide implementation blueprints

Several production frameworks demonstrate typed pipeline patterns directly applicable to Clotho:

- **Flyte** (Linux Foundation) enforces **strongly typed interfaces** at every step using FlyteFile/FlyteDirectory for typed file transfers and Structured Dataset for column-level checking. Used by LinkedIn, Spotify, and Freenome.
- **Hamilton** (Apache/DAGWorks) builds DAGs from Python functions where function names define outputs and parameter types define dependencies — the type system *is* the pipeline definition.
- **ZenML** stores each step's outputs as **typed artifacts** with full data lineage, and recently added first-class support for agentic frameworks (CrewAI, AutoGen).
- **Google ADK** provides SequentialAgent, ParallelAgent, and LoopAgent with communication via typed shared session state using explicit `output_key` contracts.

A key cross-cutting theme from the GitHub Engineering Blog crystallizes the approach: "When agents return typed, validated data instead of free-form text, the rest of your application becomes much more predictable. Treat schema violations like contract failures — retry, repair, or escalate."

---

## A concrete architecture for Clotho's pipeline validation system

### Discriminated unions as the universal output type

Following the Vercel AI SDK's approach (separate versioned interfaces per modality — `LanguageModelV4`, `ImageModelV2`, `SpeechModelV1`, `TranscriptionModelV1`), Clotho should use a discriminated union for all pipeline outputs:

```typescript
type PipelineOutput =
  | { type: 'text'; content: string; metadata: TextMetadata }
  | { type: 'audio'; data: Uint8Array; mimeType: 'audio/mp3' | 'audio/wav'; metadata: AudioMetadata }
  | { type: 'image'; data: Uint8Array; mimeType: 'image/png' | 'image/jpeg'; metadata: ImageMetadata }
  | { type: 'video'; data: Uint8Array; mimeType: 'video/mp4'; metadata: VideoMetadata }
  | { type: 'structured'; data: Record<string, unknown>; schema: JSONSchema }
  | { type: 'embedding'; vector: number[] };
```

Each pipeline stage declares its input and output types as Zod schemas. The platform validates at every connection point, rejecting mismatches before execution. TypeScript's exhaustive checking via `never` in switch defaults forces handling of every output variant.

### Three-layer validation at every pipeline boundary

**Layer 1 — Structural validation**: Check that the output matches the declared discriminated union variant. For binary outputs, use magic-byte detection (`file-type` or `puremagic`) to verify actual content matches declared MIME type. This catches the TTS text-instead-of-audio failure instantly.

**Layer 2 — Semantic validation**: For LLM-powered stages, verify `stop_reason === "tool_use"` (not `end_turn`) to confirm the model called the generation tool rather than generating free text. For structured outputs, run Pydantic/Zod schema validation. For media, check minimum size thresholds (audio < 1KB is almost certainly an error).

**Layer 3 — Quality validation**: Use Guardrails AI validators for content quality checks (toxicity, PII, length bounds). Implement quality scoring where outputs below a threshold trigger retry or fallback to alternative models.

### Dynamic configuration panels from JSON Schema

The recommended approach for model-agnostic property panels combines **Replicate's Cog pattern** (define parameters via typed code → auto-generate JSON Schema) with **React JSON Schema Form** (`@rjsf/core` + `@rjsf/mui`) for rendering. When a user selects a model, Clotho fetches or looks up the model's parameter schema and renders the configuration panel dynamically. Constraints (min/max, enum, regex) become form validation rules; descriptions become help text; defaults pre-fill fields; `x-order` controls layout.

For the model registry itself, adopt the Fal.ai pattern of runtime schema fetching with intelligent caching — only regenerate UI elements when the schema actually changes. Store model descriptors with capability metadata (supported modalities, streaming support, tool calling, context window size) to enable connection-time validation in the pipeline editor.

### Circuit breakers adapted for AI-specific failure modes

Classic circuit breakers assume binary success/failure, but AI failures often look like successes (HTTP 200 with hallucinated content). Clotho needs a **four-state circuit breaker**: CLOSED (normal), DEGRADED (works but at reduced quality), OPEN (all requests fail-fast), and HALF-OPEN (graduated re-enablement with multiple probe samples). Five failure categories require separate tracking: network errors, rate limits, timeouts, semantic failures (wrong output type), and quality degradation.

---

## OpenCrawl is a crawling library, not the dataset you might expect

A critical clarification: **Common Crawl** and **OpenCrawl** are entirely different projects. **Common Crawl** (commoncrawl.org) is the massive nonprofit web archive containing **300+ billion pages** spanning 18+ years — the dataset that made GPT-3, LLaMA, and most modern LLMs possible. It's accessible via AWS S3 in WARC (raw HTTP responses), WAT (metadata JSON), and WET (extracted text) formats.

**OpenCrawl** (github.com/janhq/OpenCrawl) is an open-source Python crawling library by Jan.ai, built for ethical, high-performance active crawling with **LLM-powered content analysis** and Apache Kafka for parallel processing. It's a tool for real-time crawling, not a static dataset.

### Firecrawl is the strongest choice for a pipeline crawl node

For integrating web crawling into Clotho's pipeline, **Firecrawl** (firecrawl.dev, YC S22) is the most production-ready option. It provides five core operations — scrape (single URL → markdown/JSON), crawl (async full-site crawl), map (URL discovery), extract (LLM-powered structured extraction), and search (web search with full page content) — all behind a clean API. It handles JavaScript rendering, anti-bot bypasses, and rate limiting transparently. Firecrawl is already a **native n8n integration** and has official LangChain/LlamaIndex loaders and an MCP server.

**Crawl4AI** is the best open-source alternative (Apache 2.0, no API keys required) with async architecture, clean markdown output using BM25-based noise removal, and built-in chunking strategies. **Tavily** excels for real-time agent search with SOC 2 certification and built-in prompt injection prevention. **Jina Reader API** offers the simplest integration — just prepend `https://r.jina.ai/` to any URL.

A web crawl node in Clotho should abstract provider selection behind a unified interface: URL(s) + mode + depth/limit + selectors + output format as inputs; an array of pages with markdown content, metadata, links, and images as output. The node connects downstream to a transform node (clean/chunk/embed) feeding into a vector store index, completing the RAG pipeline.

---

## Conclusion

Clotho's architecture should rest on five pillars. First, adopt **Replicate's Cog pattern** for model schema discovery — every model exposes an OpenAPI-compatible JSON Schema that drives both API validation and UI generation via React JSON Schema Form. Second, use **discriminated union output types** (following Vercel AI SDK's per-modality interfaces) with magic-byte MIME validation at every pipeline boundary to catch the TTS-returns-text failure class. Third, layer **Instructor-style retry-with-feedback** for LLM stages, **Outlines constrained decoding** for self-hosted models, and **native structured outputs** for OpenAI/Anthropic to maximize schema compliance. Fourth, implement **Agent Behavioral Contracts** (preconditions, invariants, governance, recovery) as the formal framework for pipeline stage contracts, using Pydantic AI's `Agent[DepsType, OutputType]` generic pattern for type-safe composition. Fifth, build the web crawl node around **Firecrawl** (production) or **Crawl4AI** (open-source) with a provider-agnostic interface feeding standard RAG chunking and embedding pipelines. The Open Agent Specification from Oracle provides the most complete reference architecture for the overall system, with its JSON Schema-typed components and explicit DataFlowEdge/ControlFlowEdge connectivity enabling both static analysis in the visual editor and runtime validation during execution.
