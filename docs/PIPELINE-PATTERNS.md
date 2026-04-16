# Clotho Pipeline Patterns

Authoritative catalog of **what can be wired to what**, **which shapes creators actually build**, and **where the output of each pattern lands** (DB rows, disk files, manifest entries, SSE events, UI surfaces).

This doc is the source of truth the pipeline-pattern test suite checks its assertions against. If you change pipeline behaviour in the engine, update this doc AND the matching test in `internal/engine/pipeline_patterns_test.go`.

---

## 1. Port types (the vocabulary)

Source: `web/src/lib/types.ts` `PortType` union + `internal/domain/node.go` constants. Identical across surfaces.

| Type | Produced by | Accepted by | Notes |
|---|---|---|---|
| `text` | Agent (default output), Tool/text_box | `text`, `any` | Plain text; LLM output |
| `image_prompt` | Agent (task_type = image_prompt) | `image_prompt`, `text`, `any` | Subtype of text — degrades to `text` |
| `video_prompt` | Agent (task_type = video_prompt) | `video_prompt`, `text`, `any` | Subtype of text |
| `audio_prompt` | Agent (task_type = audio_prompt) | `audio_prompt`, `text`, `any` | Subtype of text |
| `image` | Media/image, Tool/image_box | `image`, `any` | Generated or fixed image asset |
| `video` | Media/video, Tool/video_box | `video`, `any` | Video asset |
| `audio` | Media/audio | `audio`, `any` | Audio asset |
| `json` | Agent (custom task output) | `json`, `any` | Structured passthrough |
| `any` | (palette-default agent input, Media reference inputs) | anything | Universal sink; never produced by runtime output, only declared as input |

### Compatibility matrix (output row → input column)

```
                  TARGET
                  text  img_p  vid_p  aud_p  image  video  audio  json   any
         text  │   ✓      ✓      ✓      ✓      ✗      ✗      ✗      ✗     ✓
SOURCE  img_p  │   ✓      ✓      ✓      ✓      ✗      ✗      ✗      ✗     ✓
        vid_p  │   ✓      ✓      ✓      ✓      ✗      ✗      ✗      ✗     ✓
        aud_p  │   ✓      ✓      ✓      ✓      ✗      ✗      ✗      ✗     ✓
        image  │   ✗      ✗      ✗      ✗      ✓      ✗      ✗      ✗     ✓
        video  │   ✗      ✗      ✗      ✗      ✗      ✓      ✗      ✗     ✓
        audio  │   ✗      ✗      ✗      ✗      ✗      ✗      ✓      ✗     ✓
         json  │   ✗      ✗      ✗      ✗      ✗      ✗      ✗      ✓     ✓
          any  │   ✓      ✓      ✓      ✓      ✓      ✓      ✓      ✓     ✓
```

Rule summary:

- **Text family is one interchangeable group.** `text`, `image_prompt`, `video_prompt`, and `audio_prompt` all carry strings; a Prompt agent's `text` output can feed any media node's prompt input, and prompt specializations can flow into generic text agents. Downstream agents route behavior from their own `task_type`, not from the incoming specialization.
- **Media outputs are hermetic.** `image → video` is rejected; cross-media remixing is an explicit executor's job, not free piping.
- **`any` accepts everything.** Used for optional reference inputs on Media nodes and the palette-default agent input.
- **`json` is pair-only.** It's a structured-data channel; connect only json→json or json→any. If you need to flatten JSON into text, set the upstream agent's `output_type` to `text`.

Enforcement:

- Design-time: `web/src/stores/pipelineStore.ts::onConnect` silently drops incompatible edges. (Current UX gap: no user feedback. Locked-in-test so a future toast doesn't break coverage.)
- Save-time: `internal/engine/graph.go::ValidateGraph` returns a structured error per edge.
- Execute-time: same `ValidateGraph` is called at execution start; the engine then resolves inputs by direct node-output lookup with no coercion. A type mismatch that slipped past validation surfaces as an executor-side unmarshal error, not silent corruption.

---

## 2. Node-kind port shapes (what you get when you drag from the palette)

### Agent (`NodeTypeAgent`)

Palette default (`web/src/components/sidebar/NodePalette.tsx::defaultAgentPorts`):

```
in_text   (Input,   type=any,  required=false)
out_text  (Output,  type=text, required=false)
```

The output type mutates when the user switches `config.task.output_type` in the inspector (e.g., to `image_prompt` for a prompt-crafter agent). The palette default for the four agent modalities:

| Palette tile | Input | Output |
|---|---|---|
| Prompt | `any` | `text` |
| Image | `image_prompt` required | `image` |
| Audio | `audio_prompt` required | `audio` |
| Video | `video_prompt` required | `video` |

Note: "Image/Audio/Video" tiles in the Agent rail section actually create **Media** nodes (the payload sets `nodeType: 'media'`). Only "Prompt" creates an Agent node.

### Media (`NodeTypeMedia`)

Per `internal/domain/media.go::DefaultMediaPorts`:

- `Image`: `in_prompt` (image_prompt, required) + `ref` (any, optional) → `out_image` (image)
- `Video`: `in_prompt` (video_prompt, required) + `ref` (any, optional) → `out_video` (video)
- `Audio`: `in_prompt` (audio_prompt, required) + `ref` (any, optional) → `out_audio` (audio)

The optional `ref` port is how you pipe a fixed asset or a prior media output into a generation call (image → video reference, e.g.).

### Tool (`NodeTypeTool`)

Per `internal/domain/tool.go::DefaultToolPorts` — data sources, no inputs:

- `text_box`: `out_text` (text)
- `image_box`: `out_image` (image)
- `video_box`: `out_video` (video)

---

## 3. The 12 creator workflow patterns

For each: the graph, why someone builds it, the edges exercised, and the landing table.

### B1. Text-only chain (outline → polish)

```
Tool(text_box) ──text──▶ Agent(script) ──text──▶ Agent(script)
```

**Why:** Two-stage LLM refinement. The text_box is the seed ("write a 3-sentence scene about a lighthouse at dawn"), the first agent outlines, the second polishes.

**Edges:** `text → any` (text_box into first agent), `text → any` (agent-to-agent).

**Landing:**

| Node | output_data | Disk | Manifest | SSE |
|---|---|---|---|---|
| text_box | inline text (the seed) | — | `nodes[0].output` inline | `step_started` + `step_completed` (no chunks, tool is synchronous) |
| agent #1 | inline text (outline) | — | `nodes[1].output` inline, plus `tokens_used`, `cost_usd` | `step_started`, N×`step_chunk`, `step_completed` |
| agent #2 | inline text (polished draft) | — | `nodes[2].output` inline | same as agent #1 |
| (execution) | totals roll up | — | top-level `total_cost_usd`, `total_tokens` | `execution_completed` |

---

### B2. Script → single image (the canonical sample pipeline)

```
Agent(script) ──text──▶ Agent(image_prompt) ──image_prompt──▶ Media(image)
```

**Why:** The default "click LOAD SAMPLE PIPELINE" flow. Generates a short scene, turns it into a visual prompt, renders an image.

**Edges:** `text → any` (script into crafter), `image_prompt → image_prompt` (crafter into media).

**Landing:**

| Node | output_data | Disk | Manifest | SSE |
|---|---|---|---|---|
| script agent | inline text | — | `nodes[0].output` inline | `step_started`, chunks, `step_completed` |
| image_prompt agent | inline text (polished prompt) | — | `nodes[1].output` inline | same |
| media/image | `"clotho://file/{project}/{pipe}/{exec}/image-*.png"` | `{DataDir}/{slug}/{slug}/{exec_id}/image-*.png` | `nodes[2].output_file = image-*.png` | `step_started`, `step_completed` (no chunks — media providers don't stream) |
| (execution) | — | — | top-level totals | `execution_completed` |

**Failure case (B11 — see below):** the script agent fails → `step_failed` for node 0, downstream nodes never execute, `execution_failed` fires.

---

### B3. Script → image → video with reference

```
Agent(image_prompt) ──image_prompt─▶ Media(image) ──image─▶ (Media/video.ref: any)
                                                             ▲
Agent(video_prompt) ──video_prompt─────────────────────────▶ Media(video).in_prompt
```

**Why:** Generate a still that anchors the scene, then use it as a reference image for the video generator — gives the video continuity with the still.

**Edges:** `image_prompt → image_prompt`, `image → any` (the new test case — image output piped into a video node's reference input), `video_prompt → video_prompt`.

**Landing:** each media node writes its own file; manifest has both. This is the first pattern that exercises the `any`-typed reference port on a Media node.

---

### B4. Script → TTS narration

```
Agent(script) ──text─▶ Agent(audio_prompt) ──audio_prompt─▶ Media(audio) [Kokoro]
```

**Why:** Narration track for the scene. Kokoro is the local TTS provider; manifest carries `audio-*.mp3`.

**Landing:** same shape as B2 but swapping image for audio.

---

### B5. Full story fan-out

```
Agent(script) ─┬─▶ Agent(image_prompt) ─▶ Media(image)
               │
               ├─▶ Agent(video_prompt) ─▶ Media(video)
               │
               └─▶ Agent(audio_prompt) ─▶ Media(audio)
```

**Why:** One script, three assets. The whole reason Clotho exists.

**Edges:** three outbound edges from the same `out_text` port; exercises the fan-out code path in `engine.ExecuteWorkflow`'s input resolution.

**Landing:** one execution writes seven rows to `step_results` (1 script + 3 prompt agents + 3 media nodes). Manifest lists all seven. Three files on disk.

**Invariant to verify in tests:** the script node's output is delivered to all three downstream agents — not consumed by the first, vanished for the others.

---

### B6. Reference-guided image

```
Tool(image_box) ──image──▶ Media(image).ref  (any)
Agent(image_prompt) ──image_prompt──▶ Media(image).in_prompt
```

**Why:** Inject a fixed image (a character sheet, a style guide) as ComfyUI's reference alongside the text prompt. Tests the "two inputs to one node" pattern — required + optional.

**Landing:** Media receives both inputs in its `inputs` map; executor is responsible for using the reference. Manifest carries just one output file.

---

### B7. Tool text_box as sole prompt source

```
Tool(text_box) ──text──▶ Agent(image_prompt) ──image_prompt──▶ Media(image)
```

**Why:** Skip the scriptwriter step — user provides the prompt directly via a text_box. Fastest way to test a new model/workflow.

**Edges:** `text → any` (the agent input port), `image_prompt → image_prompt`.

---

### B8. Agent-to-agent prompt handoff

```
Agent(image_prompt) ──image_prompt──▶ Agent(image_prompt) ──image_prompt──▶ Media(image)
```

**Why:** A polish step. The first agent takes the user's rough idea and the second tightens/embellishes before the image generator sees it.

**Edges:** `image_prompt → any` (into second agent — any-typed input accepts the prompt subtype), `image_prompt → image_prompt` (typed agent-to-media).

**Note:** In the current frontend, the second agent's input is declared `any`, so the edge is `image_prompt → any`. The matrix makes that valid. For a stricter workflow the inspector can switch the second agent's input port to `image_prompt`.

---

### B9. Video from audio + image (multi-input media)

```
Tool(image_box) ──image──▶ Media(video).ref         (any)
Agent(video_prompt) ──video_prompt──▶ Media(video).in_prompt
```

**Why:** Static image + motion description → short clip. Exercises multi-input Media more aggressively than B3 (asset source is Tool, not Media).

---

### B10. Re-run from a specific node

Start from any completed pattern (say B2). User clicks the play button in the middle node.

**Engine behaviour (`engine.RerunFromNode`):**

- Loads prior step_results for upstream nodes (the script agent).
- Deletes or marks-stale the step_results for the re-run node and everything downstream.
- Topologically executes from the target node using the cached upstream outputs.

**Landing:** `step_results` for upstream nodes unchanged; re-run node and downstream are fresh rows (or updated). Manifest is re-written after completion (same filename, new content).

**Invariant to verify:** upstream outputs are read from the cache, not re-requested from providers; cost/token counters only reflect the re-executed steps.

---

### B11. Failure propagation

B2 with the first agent's provider returning a 4xx.

**Expected:**

- Script step: status=`failed`, `error` is the scrubbed provider message (redact.Secrets applied).
- Downstream steps: never executed; no step_results row created.
- Execution: `status=failed`, `error` set to the step error.
- SSE: `step_failed` for node 0, then `execution_failed`. No further events.
- UI: error inspector shows the remediation hint from `errorRemediation.ts` matching the scrubbed message.

---

### B12. Cancellation mid-stream

B5 (fan-out) with cancellation fired while the script agent is still streaming.

**Expected (after plumbing from Phase 4):**

- Script agent's stream is interrupted within ~200ms of the cancel.
- Script step_result: `status=cancelled` with partial output preserved (chunks received so far).
- All downstream steps: never created.
- Execution: `status=cancelled`.
- SSE: no `execution_completed`; the stream closes cleanly.

**Note:** Cancellation plumbing is per the Phase 4 note in the refactor sweep — there's a `context.CancelFunc` available to the executor but the Cancel button wiring across stores→engine isn't fully proven in tests. This pattern is the regression guard for that work.

---

## 4. What each surface sees (the landing cheatsheet)

Concise mapping of "where does each piece of output land":

- `step_results.output_data` (Postgres JSONB) — every successful step. For text/json nodes: the output inline. For media nodes: a `clotho://file/{rel}` URL string.
- `step_results.error` (TEXT) — scrubbed error message on failure. Shape is free-form; matched against `errorRemediation.ts` catalog in the UI.
- `step_results.tokens_used`, `cost_usd`, `duration_ms` — per-step metrics; populated from StepOutput returned by the executor.
- `executions.total_cost`, `total_tokens` — rolled up on `Complete`.
- Disk: `{DataDir}/{project_slug}/{pipeline_slug}/{execution_id}/{node-*.ext}` via `storage.LocalStore`. `DataDir` defaults to `~/Documents/Clotho/`.
- `manifest.json` — one per execution, dropped in the same directory. Top-level totals + one `nodes[]` entry per step with provider, model, prompt, output_file OR output (inline), metrics.
- SSE envelope — six event types: `step_started`, `step_chunk`, `step_completed`, `step_failed`, `execution_completed`, `execution_failed`. Frontend `sseParse.parseEnvelope` validates shape before dispatch.
- Frontend store: `useExecutionStore.stepResults` (Map keyed by nodeId) + `totalCost` + `totalTokens`.
- UI surface: node body preview, inspector output block, top-bar execution status chip + cost/tokens display, completion toast.

---

## 5. Known gaps / deferred

- **`onConnect` rejection UX** — incompatible edges are silently dropped; no toast or inline message. Tracked in `pipelineStore.onConnect.test.ts` as a locked-in behaviour.
- **B12 cancellation** — fully plumbed for the Run button but per-node cancel (mid-stream stop on a specific agent) is untested; B12 in the test suite establishes the baseline.
- **Cross-media remixing** (e.g., `image + audio → video`) — needs a new Media executor type. Out of scope for this test sweep; noted in Out-of-scope in the plan.
- **Pipeline cost prediction** — walking the graph pre-run to surface a cost estimate. Separate feature, separate plan.
- **Backend vs frontend default-port drift** — `internal/domain/agent.go::DefaultAgentPorts` uses `text` input while the palette creates `any`. Runtime is governed by the payload the palette builds. Worth reconciling so a future backend-seeded pipeline follows the same convention.

---

## 6. How to add a new pattern

1. Describe the shape in Section 3 (diagram + why + landing table).
2. Add a case to `internal/engine/pipeline_patterns_test.go` that builds the graph, registers fake executors, runs the engine, and asserts the landing table.
3. If the pattern reveals an engine bug, fix it inline before the next pattern goes in. Reference the bug in the test comment.
4. If the pattern is creator-facing (not just a matrix-coverage test), add an `@live`-tagged Playwright spec in `pipeline-patterns.spec.ts`.

---

_This doc locks the contract. Engine changes that break a pattern should either update the doc + test together, or fail CI until one of the two is brought in line._
