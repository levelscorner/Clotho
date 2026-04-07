# DESIGN.md — Clotho Design System

Updated: 2026-04-07
Status: v1 (established during plan-design-review)

## Direction

Dark, premium, creative. Think Figma meets a professional video editing suite.
The canvas is a creative production environment, not a developer tool.
Warm palette (amber, not purple). Minimal chrome. Every pixel earns its place.

## Color System

### Surfaces (dark scale)
```css
--surface-base:    #121216;   /* Page background, canvas */
--surface-raised:  #1a1a20;   /* Cards, nodes, panels */
--surface-overlay: #222228;   /* Overlays, modals, top bar */
--surface-hover:   #2a2a32;   /* Hover states */
--surface-border:  #2e2e38;   /* Borders, dividers */
```

### Text
```css
--text-primary:    #ececf0;   /* Headings, labels */
--text-secondary:  #8888a0;   /* Body text, descriptions */
--text-muted:      #55556a;   /* Placeholders, disabled */
```

### Accent (warm amber, NOT blue/purple)
```css
--accent:          #e5a84b;   /* Primary accent */
--accent-soft:     rgba(229, 168, 75, 0.12);  /* Backgrounds */
--accent-glow:     rgba(229, 168, 75, 0.25);  /* Glow effects */
```

### Status
```css
--status-running:  #e5a84b;   /* Amber — execution in progress */
--status-complete: #4ade80;   /* Green — step done */
--status-failed:   #f87171;   /* Red — step failed */
--status-queued:   #55556a;   /* Gray — waiting */
```

### Port Types
```css
--port-text:       #a78bfa;   /* Text connections */
--port-image:      #f59e0b;   /* Image connections */
--port-video:      #ef4444;   /* Video connections */
--port-audio:      #06b6d4;   /* Audio connections */
```

## Typography

| Use | Font | Weight | Size | Notes |
|-----|------|--------|------|-------|
| Logo, display | Sora | 700 | 28px | -0.5px letter-spacing |
| Section headers | Inter | 600 | 14px | |
| Body, labels | Inter | 400 | 13px | |
| Small labels | Inter | 600 | 10-11px | Uppercase, 0.6-0.8px tracking |
| Streaming output | JetBrains Mono | 400 | 11px | Monospace for LLM output |
| Cost values | JetBrains Mono | 400 | 11px | |

Load: Inter (400, 600), Sora (700), JetBrains Mono (400). Three families, four weights total. `font-display: swap`.

## Spacing Scale

```css
--space-xs:  4px;
--space-sm:  8px;
--space-md:  12px;
--space-lg:  16px;
--space-xl:  24px;
--space-2xl: 32px;
```

## Border Radius

```css
--radius-sm: 6px;    /* Buttons, inputs, port dots */
--radius-md: 10px;   /* Nodes, cards */
--radius-lg: 14px;   /* Panels, modals */
```

## Motion

```css
--ease-out: cubic-bezier(0.16, 1, 0.3, 1);
--duration-fast:   120ms;    /* Hover, focus */
--duration-normal: 200ms;    /* Transitions */
```

Respect `prefers-reduced-motion`: disable pulse animations, cursor blink, glow transitions.

## Node Design

### Agent Nodes
- Header: subtle warm gradient `linear-gradient(135deg, #1a1a20, #231e18)` with amber text
- Icon: 28x28 rounded square, amber soft background
- Body: preview of streaming output (monospace, amber cursor)
- Footer: status dot + timing + cost
- NOT blue-to-purple gradient

### Media Nodes (Phase 2)
- Same structure as agent nodes
- Icon in gold/warm tones
- Body: progress bar for generation, thumbnail grid for completed images

### Tool Nodes
- Same structure, gray tones
- More minimal than agent nodes

### Node States
- **Idle**: default surface, gray status dot
- **Running**: amber border, amber glow (`box-shadow: 0 0 16px var(--accent-glow)`), pulse animation
- **Complete**: green border, subtle green glow
- **Failed**: red border, red glow, error message + 3 action buttons (Retry, Edit Prompt, Switch Provider)
- **Selected**: amber outline, 2px offset

## Canvas Layout

```
┌──────────────────────────────────────────────────────────────────┐
│ TOP BAR: [Logo] [Project > Pipeline]          [Cost] [Run btn]  │
├──────────┬──────────────────────────────┬────────────────────────┤
│ PALETTE  │         CANVAS               │      INSPECTOR         │
│ 180px    │     (flex: 1)                │      260px             │
│          │                              │                        │
│ Icon grid│  Nodes + connections         │ Selected: config       │
│ 3 cols   │  Dot grid background         │ Nothing: pipeline stats│
│          │  Minimap bottom-right        │                        │
└──────────┴──────────────────────────────┴────────────────────────┘
```

### Sidebar Palette
- **Icon grid** (Figma-style, 3 columns)
- Groups: Agent (generic + personalities), Media, Tools
- Personalities appear as saveable icon tiles
- Drag to canvas creates a pre-configured node
- "+ New" tile opens personality creation

## Interaction States

Every feature must define: loading, empty, error, success, partial states.
Empty states have: warmth (not "No items found"), a primary action, context.

## Accessibility

- Keyboard: Tab through palette, arrow keys between nodes, Enter to select
- ARIA labels on nodes: "{name} agent, status: {status}, cost: {cost}"
- Focus: amber outline via `focus-visible`
- Reduced motion: honor `prefers-reduced-motion`
- Touch targets: 44px minimum (override React Flow handle defaults)

## Anti-Patterns (DO NOT USE)

- Blue-to-purple gradients (AI slop)
- System font stack as the only typography
- Uniform card grids with no hierarchy
- Generic "No items found" empty states
- Decorative blobs or floating circles
- Centered-everything layouts
- Default shadcn/Tailwind look
