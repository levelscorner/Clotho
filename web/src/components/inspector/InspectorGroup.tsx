import { useEffect, useRef } from 'react';
import type { ReactNode } from 'react';
import './InspectorGroup.css';

export interface InspectorGroupProps {
  title: string;
  defaultOpen?: boolean;
  /** When true, forces the group open regardless of user toggle state
   *  (used to auto-expand a section containing an error). */
  forceOpen?: boolean;
  children: ReactNode;
}

/**
 * Collapsible inspector section. Built on native <details>/<summary>
 * for free keyboard + screen-reader accessibility.
 */
export function InspectorGroup({
  title,
  defaultOpen = false,
  forceOpen = false,
  children,
}: InspectorGroupProps) {
  const detailsRef = useRef<HTMLDetailsElement>(null);

  // When forceOpen flips to true, open the group. We don't force it closed.
  useEffect(() => {
    if (forceOpen && detailsRef.current) {
      detailsRef.current.open = true;
    }
  }, [forceOpen]);

  return (
    <details
      ref={detailsRef}
      className="clotho-inspector-group"
      open={defaultOpen || forceOpen}
    >
      <summary className="clotho-inspector-group__summary">
        <span className="clotho-inspector-group__chevron" aria-hidden="true">
          {'\u203A'}
        </span>
        <span className="clotho-inspector-group__title">{title}</span>
      </summary>
      <div className="clotho-inspector-group__content">
        <div className="clotho-inspector-group__content-inner">{children}</div>
      </div>
    </details>
  );
}
