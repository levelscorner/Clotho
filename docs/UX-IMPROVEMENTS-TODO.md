# UX & Product Improvements — Deferred Work

Consolidated list of work deliberately deferred out of recent sprints. Organized by
priority × effort so a future contributor (or future-self 3 months from now) can
pick items up without re-deriving the motivation.

- **Priority** — P0 = unblocks a user, P1 = sharpens a stated goal, P2 = nice when demand arrives.
- **Effort** — S (<1 day), M (1-3 days), L (week+), XL (multi-week design-then-build).

Last updated: 2026-04-15 — after Stage B (disk storage for media artifacts) wave 5.

---

## P0 — Ships before external users

### Fix refresh-token bug in auth handler — S

The refresh-token endpoint validates the expired access token and rejects it.
Normal refresh scenarios (expired access + valid refresh) always return 401.

- File: `internal/api/handler/auth.go`
- Fix direction: validate ONLY the refresh token in `/auth/refresh`; generate a
  new access token from the refresh claims without requiring the old access
  token to be valid.
- Current state: bypassed via `NO_AUTH=true` for solo dev. The bug doesn't bite
  until multi-user mode returns.
- Cross-model agreement (Claude + Codex, 2026-04-07) confidence 9/10.

### User-defined agent personalities (major refactor) — L

Rip out the hardcoded 7 built-in agent presets. Replace with a "Create Agent"
flow where users define their own persona, prompt, model, and optional visual
treatment. The paper-rule / LCD / matte materials become user-selectable themes
rather than preset-driven styling.

Founder, during live testing 2026-04-15:

> I don't like the personalities … it's like they just catered to me. What if
> someone else wants their own personalities or agent, in that case they should
> be totally different.

The current preset system is opinionated ABOUT taste rather than enabling taste.

Touches: `internal/domain/preset.go`, migrations, frontend NodePalette,
AgentInspector preset-category dispatch, `DESIGN.md`, onboarding.

Plan in a dedicated `/office-hours` session before starting.

---

## P1 — Sharpens product direction

### "Made with Clotho" output identity — M

Identity signature on content produced by Clotho — watermark, branded export
container, signature animation, or aesthetic preset baked into prompts.

Why: every user-produced video/image/audio becomes a marketing artifact.
Distribution is the marketing surface. Even when users bring their own model
(Ollama, etc), output should carry Clotho's brand.

Ambiguous direction — multiple valid implementations (watermark vs signature
vs preset). Start with a dedicated `/office-hours` to pick direction.

### Pluggable-model UX (full provider marketplace) — XL

Out-of-scope from the Stage B storage sprint; captured here for the record.
Current state supports OpenAI, Anthropic, Ollama, Replicate, Kokoro, ComfyUI
— but adding one is a code change. A real marketplace would need:

- Per-user/per-project credential storage (exists for OpenAI/Anthropic keys;
  scope it broader).
- Dynamic model discovery at config time (already stubbed for Ollama).
- Cost-preview UI per provider × model.
- Fallback model policies ("if OpenAI is down, use Anthropic").
- Model × task compatibility metadata (some models don't do image prompts).

### Connection-craft polish (port glow, snap animation) — M

When dragging out a port, compatible target ports glow in their port-type
color. On approach, the connector snaps to the valid target with a brief
animation.

Why: building a pipeline is the primary craft action in Clotho. Right now
it's functional but mute. Port-type colors would teach the compatibility
system faster.

Logic already exists (`web/src/lib/portCompatibility.ts`), `PORT_COLORS`
is defined. Rendering-layer enhancement only — touches React Flow edge
rendering internals, so take care not to regress drag-drop reliability.

### ToolNode visual personality — S

Give `web/src/components/canvas/nodes/ToolNode.tsx` (currently 62 lines,
generic styling) a distinct visual identity matching the AgentNode/MediaNode
personality system. Suggested direction: terminal/plumbing aesthetic — small
monospace "command" look with a different accent (not warm amber — maybe
cyan or steel, to signal "utility" not "creative").

### Animated oscilloscope for AudioNode — S

Replace the static SVG waveform + CSS pulse with a per-sample animated
oscilloscope driven by stream progress (or a fake driver if the model
doesn't stream). Reuse a SINGLE `requestAnimationFrame` loop across all
audio nodes to avoid an N-loops footgun at scale.

### Output viewer route — S

A dedicated `/executions/{id}/outputs` route that paginates every artifact
the run produced. Useful when a pipeline fans out 20 shots and the canvas
preview isn't the best shape for reviewing them.

### Per-project running cost totals — S

Aggregate `cost_usd` across every execution in a project; show it in the
project header ("$12.44 across 83 runs"). Already have the numbers per
execution; this is a query + a UI pill.

---

## P2 — Nice when demand arrives

### Build user management service (IAM) — L

Full multi-user auth/IAM layer: signup, login with working refresh tokens,
password reset, organization/tenant model, per-user project scoping. Auth
scaffolding already exists (`internal/api/handler/auth.go`, migration
`004_add_auth.*`) but the refresh bug means the flow doesn't fully work.

Blocked by: external demand. At least one non-founder user asking for access.

### Theme-able node styling — M

Allow user to swap the per-preset visual metaphor (e.g., "manuscript" vs
"terminal" for Script Writer). Creator taste varies.

Surface-area increase. Not asked for. YAGNI until someone complains.

### Phone/tablet touch interaction model for canvas — XL

Current sprint shipped phone/tablet layouts but the core canvas metaphor
(drag nodes, drag to connect ports) was designed for mouse + keyboard.
On touch: port-to-port drag is imprecise, node selection vs canvas pan
fight for the same gesture, inspector + canvas + sidebar won't all fit.

This is a design problem, not an implementation problem. Needs a dedicated
`/office-hours` to pick the interaction model. Could mean a radically
different editor mode (e.g., "list mode" — node pipeline as a scrollable
list on phone).

Don't invest until there's a real use case (creator wants to do this on a
commute? show a WIP to a friend in a coffee shop?).

### Migrate historical base64 outputs to on-disk format — S

Stage B introduced disk storage for NEW executions. Historical executions
still carry multi-MB base64 payloads in `step_results.output_data`. A
one-shot migration would rewrite those to `clotho://file/` refs and write
the bytes to disk. Only worth running if DB size becomes a practical
concern — otherwise leave them.

### Share / export a run folder as zip — S

Given an execution ID, bundle the artifacts directory (`~/.gstack/clotho/
executions/{id}/`) plus `manifest.json` into a downloadable zip. Useful
for handing a run to a collaborator, archiving a favorite, or attaching
to a bug report.

### iCloud sync considerations for `~/Documents/Clotho/` — S

Storage currently writes under `~/.gstack/clotho/` (hidden, not synced).
Founder may eventually want outputs under `~/Documents/Clotho/` so iCloud
syncs them automatically. Needs a config knob and a one-time migration;
`.gstack` path remains as the ephemeral/cache fallback.

---

## Reference

### Plans + sprint docs

- Branch: `design/warm-amber-v2`
- Stage B (disk storage) plan: `/Users/level/.claude/plans/bright-wondering-kitten.md`
- Design doc: `DESIGN.md`
- Architecture: `docs/ARCHITECTURE.md`

### How to add to this list

Add items under the priority × effort tier that matches. Keep the "Why"
short but real — motivation rots fastest when the sprint that generated
the item is forgotten. Link to the file(s) where the change will land so
the next contributor doesn't have to re-discover the surface area.
