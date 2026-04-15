import { InspectorGroup } from './InspectorGroup';
import type { NodeDescription } from '../../lib/nodeDescriptions';

interface AboutNodeSectionProps {
  description: NodeDescription;
}

/**
 * Renders the "About this node" block at the top of every inspector.
 *
 * Shows the full human-readable description plus a tiny Input / Output
 * table so users can see what flows in and out without dragging connections
 * to find out.
 */
export function AboutNodeSection({ description }: AboutNodeSectionProps) {
  return (
    <InspectorGroup title="About this node" defaultOpen>
      <p
        style={{
          fontSize: 12,
          lineHeight: 1.5,
          color: 'var(--text-secondary)',
          margin: 0,
          marginBottom: 10,
        }}
      >
        {description.full}
      </p>
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'auto 1fr',
          rowGap: 4,
          columnGap: 10,
          fontSize: 11,
          fontFamily: 'var(--font-mono)',
          padding: '8px 10px',
          background: 'var(--surface-overlay)',
          border: '1px solid var(--surface-border)',
          borderRadius: 6,
        }}
      >
        <span style={{ color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: 0.5 }}>
          Input
        </span>
        <span style={{ color: 'var(--text-primary)' }}>{description.input}</span>
        <span style={{ color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: 0.5 }}>
          Output
        </span>
        <span style={{ color: 'var(--text-primary)' }}>{description.output}</span>
      </div>
    </InspectorGroup>
  );
}
