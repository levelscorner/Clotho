# Design References

Canonical visual references for Clotho's UI. These HTML files are static mockups rendered with real DESIGN.md tokens. Open them in a browser during implementation and match the output.

**Do not deviate from these without founder approval.**

## Files

### `node-personality.html`

Locked 2026-04-13 during /office-hours. Variant B (material metaphor) + Variant C (information density) hybrid.

Shows all four node kinds:

- **Script** — paper-rule background, Georgia serif preview, italic quote, mono token count
- **Image** — matte-board frame with dark inset border + inset shadow, mono model/step/ETA readout
- **Video** — perforated reel edges (dashed-strip pseudo-elements), frame-strip thumbnails, mono duration/FPS
- **Audio** — oscilloscope SVG sine in `--port-audio` stroke, mono voice/sample-rate readout

Only Variant B (row 2) and Variant C (row 3) are canonical. Row 1 (Iconographic) was rejected.

### `canvas-states.html`

Locked 2026-04-14 during /design-shotgun. Three canvas states × approved variants:

- **Empty canvas** — Variant 1B remix: ghosts at 28% opacity with readable names, central `LOAD SAMPLE PIPELINE` amber pill, corner hint `or press ⌘K for templates`
- **Error state** — Variant 2B: solid red header strip on failed node (`FAILED · {cause}`), downstream nodes dim to 55% with `· blocked` suffix, inline error + RETRY STEP button
- **Completion moment** — Variant 3B remix: amber pulse cascade (200ms × 4 nodes, 1s total) + 220px stats toast (time · cost · tokens grid + block VIEW OUTPUT CTA), auto-dismiss at 4s

## Why these exist in the repo

1. Preservation — founder explicitly said "do not lose the look and feel"
2. Portability — survives machine switches, git clones, new contributors
3. Implementation reference — engineer opens the HTML side-by-side with the React/CSS code
4. Design review — future /design-review runs can screenshot-diff against these as the baseline

## Design system

All tokens match `DESIGN.md` (warm amber, dark surfaces, port-type colors, Sora/Inter/JBMono typography). No inline magic numbers.
