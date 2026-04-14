import { describe, it, expect, beforeEach, vi } from 'vitest';
import { handleGlobalKeydown } from '../useGlobalKeyboardShortcuts';
import { useUIStore } from '../../stores/uiStore';

// ---------------------------------------------------------------------------
// Fake KeyboardEvent — vitest runs in the `node` env, so we construct a
// minimal event shape compatible with the handler's usage.
// ---------------------------------------------------------------------------

interface FakeEventInit {
  key: string;
  metaKey?: boolean;
  ctrlKey?: boolean;
  target?: { tagName?: string; isContentEditable?: boolean } | null;
}

function makeEvent(init: FakeEventInit) {
  const preventDefault = vi.fn();
  const target = init.target ?? null;
  // Cast via unknown so TS accepts our lightweight stand-in for a real element.
  const eventLike = {
    key: init.key,
    metaKey: init.metaKey ?? false,
    ctrlKey: init.ctrlKey ?? false,
    target,
    preventDefault,
  };
  return { event: eventLike as unknown as KeyboardEvent, preventDefault };
}

// Our handler calls `instanceof HTMLElement`. In node there is no HTMLElement,
// so we stub a global one keyed off a `__isHtmlElement` marker.
class FakeHTMLElement {
  tagName: string;
  isContentEditable: boolean;
  constructor(tagName: string, isContentEditable = false) {
    this.tagName = tagName;
    this.isContentEditable = isContentEditable;
  }
}

beforeEach(() => {
  useUIStore.setState({ templateGalleryOpen: false });
  // Make `instanceof HTMLElement` succeed for our fakes.
  (globalThis as unknown as { HTMLElement: unknown }).HTMLElement =
    FakeHTMLElement;
});

describe('handleGlobalKeydown', () => {
  it('toggles the template gallery on ⌘K (metaKey)', () => {
    const { event, preventDefault } = makeEvent({
      key: 'k',
      metaKey: true,
      target: new FakeHTMLElement('DIV'),
    });
    handleGlobalKeydown(event);
    expect(preventDefault).toHaveBeenCalledOnce();
    expect(useUIStore.getState().templateGalleryOpen).toBe(true);
  });

  it('toggles the template gallery on Ctrl+K', () => {
    const { event, preventDefault } = makeEvent({
      key: 'K', // case-insensitive
      ctrlKey: true,
      target: new FakeHTMLElement('DIV'),
    });
    handleGlobalKeydown(event);
    expect(preventDefault).toHaveBeenCalledOnce();
    expect(useUIStore.getState().templateGalleryOpen).toBe(true);
  });

  it('toggles closed when already open', () => {
    useUIStore.setState({ templateGalleryOpen: true });
    const { event } = makeEvent({
      key: 'k',
      metaKey: true,
      target: new FakeHTMLElement('DIV'),
    });
    handleGlobalKeydown(event);
    expect(useUIStore.getState().templateGalleryOpen).toBe(false);
  });

  it('does NOT fire when focused inside an <input>', () => {
    const { event, preventDefault } = makeEvent({
      key: 'k',
      metaKey: true,
      target: new FakeHTMLElement('INPUT'),
    });
    handleGlobalKeydown(event);
    expect(preventDefault).not.toHaveBeenCalled();
    expect(useUIStore.getState().templateGalleryOpen).toBe(false);
  });

  it('does NOT fire when focused inside a <textarea>', () => {
    const { event, preventDefault } = makeEvent({
      key: 'k',
      metaKey: true,
      target: new FakeHTMLElement('TEXTAREA'),
    });
    handleGlobalKeydown(event);
    expect(preventDefault).not.toHaveBeenCalled();
    expect(useUIStore.getState().templateGalleryOpen).toBe(false);
  });

  it('does NOT fire when focused inside a contenteditable element', () => {
    const { event, preventDefault } = makeEvent({
      key: 'k',
      metaKey: true,
      target: new FakeHTMLElement('DIV', true),
    });
    handleGlobalKeydown(event);
    expect(preventDefault).not.toHaveBeenCalled();
    expect(useUIStore.getState().templateGalleryOpen).toBe(false);
  });

  it('Escape closes the gallery when it is open', () => {
    useUIStore.setState({ templateGalleryOpen: true });
    const { event, preventDefault } = makeEvent({
      key: 'Escape',
      target: new FakeHTMLElement('DIV'),
    });
    handleGlobalKeydown(event);
    expect(preventDefault).toHaveBeenCalledOnce();
    expect(useUIStore.getState().templateGalleryOpen).toBe(false);
  });

  it('Escape is a no-op when the gallery is already closed', () => {
    const { event, preventDefault } = makeEvent({
      key: 'Escape',
      target: new FakeHTMLElement('DIV'),
    });
    handleGlobalKeydown(event);
    expect(preventDefault).not.toHaveBeenCalled();
    expect(useUIStore.getState().templateGalleryOpen).toBe(false);
  });

  it('ignores unrelated keys', () => {
    const { event, preventDefault } = makeEvent({
      key: 'a',
      metaKey: true,
      target: new FakeHTMLElement('DIV'),
    });
    handleGlobalKeydown(event);
    expect(preventDefault).not.toHaveBeenCalled();
    expect(useUIStore.getState().templateGalleryOpen).toBe(false);
  });
});

describe('useGlobalKeyboardShortcuts module surface', () => {
  it('exports the hook and the handler', async () => {
    const mod = await import('../useGlobalKeyboardShortcuts');
    expect(typeof mod.useGlobalKeyboardShortcuts).toBe('function');
    expect(typeof mod.handleGlobalKeydown).toBe('function');
  });
});
