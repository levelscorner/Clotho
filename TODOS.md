# TODOS

Captured during /plan-eng-review on 2026-04-13 (branch: design/warm-amber-v2).

## Auth & User Management

### Fix refresh-token bug in auth handler

**What:** The refresh-token endpoint validates the expired access token and rejects it. Normal refresh scenarios (expired access token + valid refresh token) always return 401.

**Why:** The current sprint (`design/warm-amber-v2`) bypasses auth entirely via `NO_AUTH=true` so the bug doesn't bite during solo development. When multi-user IAM returns, this bug blocks login flow entirely.

**Pros:** Removes the latent regression. Restores auth as a usable mechanism for when user management returns. Probably 30-60 min of work.

**Cons:** None relevant to this sprint — deferred specifically because solo + Playwright MCP dev flow does not need login.

**Context:**
- File: `internal/api/handler/auth.go`
- Logged as a learning on 2026-04-07 with confidence 9/10 (cross-model agreement between Claude and Codex).
- The bug: refresh handler calls the same access-token validation that the normal request path uses, so an expired access token always fails validation even though the refresh token alone should be sufficient.
- Fix direction: validate only the refresh token in the refresh endpoint; generate a new access token from the refresh claims without requiring the old access token to be valid.

**Depends on / blocked by:** None. Can be picked up the moment multi-user mode comes back.

### Build user management service (IAM)

**What:** Full multi-user auth/IAM layer: signup, login with working refresh tokens, password reset, organization/tenant model, per-user project scoping.

**Why:** Clotho currently has auth code but no product use case for it — solo dev is the only user. When external users arrive, a real IAM layer is required.

**Pros:** Ships the product to more than one person. Unlocks paid tiers, sharing, collaboration.

**Cons:** Large scope (days, not hours). Premature until there is demand.

**Context:** Current auth code is scaffolding from Phase 1A. Handlers exist (`internal/api/handler/auth.go`), DB tables exist (`migrations/004_add_auth.*`), but the refresh-token bug means the flow doesn't fully work. The existing JWT approach is fine as a foundation.

**Depends on / blocked by:** External demand. At least one user who is not the founder asking for access.

## Nodes & Visual Design

### ToolNode visual personality treatment

**What:** Give `ToolNode.tsx` (currently 62 lines, generic styling) a distinct visual identity matching the AgentNode/MediaNode personality system.

**Why:** The `design/warm-amber-v2` sprint differentiates Agent presets and Media types but ignores the fourth node kind (Tool). The canvas will have three expressive node families and one plain one. Inconsistent.

**Pros:** Visual consistency across the four node kinds. Tools feel like gear, not boxes.

**Cons:** Scope creep for the current sprint. The user's stated gripe was about agent nodes; tools weren't mentioned.

**Context:** `web/src/components/canvas/nodes/ToolNode.tsx`. Current styling is minimal. Suggested direction: terminal/plumbing aesthetic — small monospace "command" look with different accent (not warm amber — maybe cyan or steel, to signal "utility" not "creative").

**Depends on / blocked by:** AgentNode + MediaNode personality landing first so the design pattern is established.

### Animated oscilloscope tied to audio-generation stream progress

**What:** Replace the static SVG waveform + CSS pulse on AudioNode with a per-sample animated oscilloscope driven by stream progress (or a fake driver if the model doesn't stream).

**Why:** Makes the audio node feel alive during generation. Currently the CSS pulse is a minimum-viable signal.

**Pros:** Highest-delight polish on the media nodes.

**Cons:** ~0.5 day on its own. Requires reuse of a single requestAnimationFrame loop to avoid N-animation-loops footgun at scale.

**Context:** Deferred from `design/warm-amber-v2` sprint per eng review. Add after the base media CSS lands.

### "Made with Clotho" output identity

**What:** Identity signature on content produced by Clotho — watermark, branded export container, signature animation, or aesthetic preset baked into prompts.

**Why:** Even when users bring their own model (Ollama, etc), the output should carry Clotho's brand into the world. Distribution is the marketing surface.

**Pros:** Every user-produced video/image/audio becomes a marketing artifact.

**Cons:** Ambiguous — multiple valid implementations (watermark vs signature vs preset). Needs its own design conversation to pick.

**Context:** Flagged during 2026-04-13 office hours, deferred from the current sprint. Start with a dedicated `/office-hours` session to pick direction.

**Depends on / blocked by:** None blocking, but worth doing before user base grows so every user produces branded output from day one.

### Connection-craft polish (port glow, snap animation)

**What:** When dragging out a port, compatible target ports glow in their port-type color. On approach, the connector snaps to the valid target with a brief animation.

**Why:** Building a pipeline is the primary craft action in Clotho. Right now it's functional but mute. Adding port-type color feedback during drag teaches the port-compatibility system faster and rewards correct connections with a visible "yes" response.

**Pros:** Lower learning curve. More satisfying build flow. Reinforces port-type mental model (text ports only connect to text/prompt subtypes, etc).

**Cons:** Touches React Flow's edge rendering internals. Needs care to not regress existing drag-drop reliability.

**Context:** Port-compatibility logic already exists (`web/src/lib/portCompatibility.ts`). The `PORT_COLORS` map is defined. This is a rendering-layer enhancement, not a logic change.

**Depends on / blocked by:** None. Can be picked up any time after the current sprint lands.

### Phone/tablet touch interaction model for canvas

**What:** A thoughtful touch interaction model for building and running pipelines on tablet and phone form factors.

**Why:** The current sprint added phone/tablet layouts (responsive scope expansion), but the core canvas metaphor (drag nodes, drag to connect ports) was originally designed for mouse + keyboard. On touch, port-to-port drag is imprecise, node selection vs canvas pan fight for the same gesture, and screen real estate doesn't fit an inspector + canvas + sidebar.

**Pros:** Makes Clotho actually usable on iPad (real creator hardware). Opens a real creator-on-the-go use case.

**Cons:** This is a design problem, not an implementation problem. Needs a dedicated /office-hours to pick the interaction model before building. Could mean a radically different editor mode (e.g., "list mode" — node pipeline as a scrollable list on phone).

**Context:** Current sprint ships phone/tablet layouts with "works without crashing" as the bar, not "first-class experience." Flagged explicitly in the plan.

**Depends on / blocked by:** A real phone use case emerging (creator wants to do this on their commute? show a WIP to a friend in a coffee shop?). Don't invest until there's demand.

### Theme-able node styling

**What:** Allow user to swap the per-preset visual metaphor (e.g., "manuscript" vs "terminal" for Script Writer).

**Why:** Creator taste varies. Different storytellers want different vibes.

**Pros:** Higher personalization.

**Cons:** Surface-area increase. Not asked for. YAGNI until someone complains.

**Context:** Flagged during eng review as future extension point. Current sprint ships fixed metaphors.

**Depends on / blocked by:** None, but low priority.

### User-defined agent personalities (major refactor)

**What:** Rip out the hardcoded 7 built-in agent presets. Replace with a
"Create Agent" flow where users define their own persona, prompt, model,
and optional visual treatment. The paper-rule / LCD / matte materials
become user-selectable themes rather than preset-driven styling.

**Why:** Founder flagged during live testing: "I don't like the personalities
... it's like they just catered to me. What if someone else wants their own
personalities or agent, in that case they should be totally different."
The current preset system is opinionated ABOUT taste rather than enabling taste.

**Pros:** Users define meaningful agents for their workflow. Product stops
imposing a specific creator aesthetic.

**Cons:** ~2-3 CC days of work. Touches: domain/preset.go, migrations, frontend
NodePalette, AgentInspector preset-category dispatch, DESIGN.md, onboarding.

**Context:** Flagged 2026-04-15 during live testing on design/warm-amber-v2.
Log it now, plan properly in a dedicated /office-hours session.
