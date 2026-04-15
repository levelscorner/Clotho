import { Robot, Wrench } from 'phosphor-react';
import type { Icon } from 'phosphor-react';
import { useUIStore, type PaletteSection } from '../../stores/uiStore';

// ---------------------------------------------------------------------------
// ActivityRail
//
// VS Code-style slim icon column. Always visible on desktop/tablet. Each
// icon toggles a wider flyout panel that renders the section's palette
// contents (Agent / Tools).
//
// Mobile keeps the hamburger + full drawer behavior — the rail is hidden
// at phone breakpoint via CSS.
// ---------------------------------------------------------------------------

interface RailItem {
  section: PaletteSection;
  label: string;
  icon: Icon;
}

const ITEMS: RailItem[] = [
  { section: 'agent', label: 'Agents', icon: Robot },
  { section: 'tools', label: 'Tools', icon: Wrench },
];

export function ActivityRail() {
  const active = useUIStore((s) => s.activePaletteSection);
  const toggle = useUIStore((s) => s.togglePaletteSection);

  return (
    <nav
      className="clotho-activity-rail"
      aria-label="Palette sections"
      data-testid="activity-rail"
    >
      {ITEMS.map((item) => {
        const isActive = active === item.section;
        const Icon = item.icon;
        return (
          <button
            key={item.section}
            type="button"
            className="clotho-activity-rail__btn"
            data-active={isActive ? 'true' : 'false'}
            data-testid={`rail-${item.section}`}
            aria-label={item.label}
            aria-pressed={isActive}
            title={item.label}
            onClick={() => toggle(item.section)}
          >
            <Icon size={22} weight={isActive ? 'fill' : 'regular'} aria-hidden="true" />
          </button>
        );
      })}
    </nav>
  );
}
