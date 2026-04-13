import { describe, it, expect } from 'vitest';
import { renderToStaticMarkup } from 'react-dom/server';
import { InspectorGroup } from '../InspectorGroup';

describe('InspectorGroup', () => {
  it('renders the title', () => {
    const html = renderToStaticMarkup(
      <InspectorGroup title="General">content</InspectorGroup>,
    );
    expect(html).toContain('General');
    expect(html).toContain('content');
  });

  it('is collapsed by default (no open attribute)', () => {
    const html = renderToStaticMarkup(
      <InspectorGroup title="General">content</InspectorGroup>,
    );
    expect(html).not.toMatch(/<details[^>]*\sopen/);
  });

  it('renders open when defaultOpen is true', () => {
    const html = renderToStaticMarkup(
      <InspectorGroup title="General" defaultOpen>
        content
      </InspectorGroup>,
    );
    expect(html).toMatch(/<details[^>]*\sopen/);
  });

  it('forceOpen overrides closed default', () => {
    const html = renderToStaticMarkup(
      <InspectorGroup title="Errors" forceOpen>
        err
      </InspectorGroup>,
    );
    expect(html).toMatch(/<details[^>]*\sopen/);
  });

  it('uses a native <summary> element for accessibility', () => {
    const html = renderToStaticMarkup(
      <InspectorGroup title="General">content</InspectorGroup>,
    );
    expect(html).toContain('<summary');
  });
});
