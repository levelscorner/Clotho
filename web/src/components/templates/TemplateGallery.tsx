import { useEffect, useCallback } from 'react';
import { useTemplateStore } from '../../stores/templateStore';
import type { TemplateSummary } from '../../lib/api';

// ---------------------------------------------------------------------------
// Category badge color mapping
// ---------------------------------------------------------------------------

const categoryColors: Record<string, { bg: string; text: string }> = {
  video: { bg: 'rgba(239, 68, 68, 0.15)', text: '#f87171' },
  image: { bg: 'rgba(245, 158, 11, 0.15)', text: '#f59e0b' },
  text: { bg: 'rgba(167, 139, 250, 0.15)', text: '#a78bfa' },
  audio: { bg: 'rgba(6, 182, 212, 0.15)', text: '#06b6d4' },
};

// ---------------------------------------------------------------------------
// TemplateCard
// ---------------------------------------------------------------------------

interface TemplateCardProps {
  template: TemplateSummary;
  onSelect: (id: string) => void;
}

function TemplateCard({ template, onSelect }: TemplateCardProps) {
  const colors = categoryColors[template.category] ?? {
    bg: 'rgba(136, 136, 160, 0.15)',
    text: '#8888a0',
  };

  return (
    <button
      onClick={() => onSelect(template.id)}
      style={{
        display: 'flex',
        flexDirection: 'column',
        gap: 10,
        padding: 16,
        background: '#1a1a20',
        border: '1px solid #2e2e38',
        borderRadius: 10,
        cursor: 'pointer',
        textAlign: 'left',
        transition: 'border-color 150ms, background 150ms',
      }}
      onMouseEnter={(e) => {
        e.currentTarget.style.borderColor = 'rgba(229, 168, 75, 0.4)';
        e.currentTarget.style.background = '#222228';
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.borderColor = '#2e2e38';
        e.currentTarget.style.background = '#1a1a20';
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <span
          style={{
            fontSize: 14,
            fontWeight: 600,
            color: '#ececf0',
            flex: 1,
          }}
        >
          {template.name}
        </span>
        <span
          style={{
            fontSize: 10,
            fontWeight: 600,
            textTransform: 'uppercase',
            letterSpacing: '0.6px',
            padding: '2px 6px',
            borderRadius: 4,
            background: colors.bg,
            color: colors.text,
          }}
        >
          {template.category}
        </span>
      </div>

      <p
        style={{
          margin: 0,
          fontSize: 12,
          color: '#8888a0',
          lineHeight: 1.5,
        }}
      >
        {template.description}
      </p>

      <div
        style={{
          fontSize: 11,
          color: '#55556a',
          marginTop: 'auto',
        }}
      >
        {template.node_count} node{template.node_count !== 1 ? 's' : ''}
      </div>
    </button>
  );
}

// ---------------------------------------------------------------------------
// TemplateGallery
// ---------------------------------------------------------------------------

interface TemplateGalleryProps {
  onClose: () => void;
}

export function TemplateGallery({ onClose }: TemplateGalleryProps) {
  const templates = useTemplateStore((s) => s.templates);
  const loading = useTemplateStore((s) => s.loading);
  const error = useTemplateStore((s) => s.error);
  const fetchTemplates = useTemplateStore((s) => s.fetchTemplates);
  const applyTemplate = useTemplateStore((s) => s.applyTemplate);

  useEffect(() => {
    if (templates.length === 0) {
      void fetchTemplates();
    }
  }, [templates.length, fetchTemplates]);

  const handleSelect = useCallback(
    async (id: string) => {
      await applyTemplate(id);
      onClose();
    },
    [applyTemplate, onClose],
  );

  // Close on Escape
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose();
    }
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onClose]);

  return (
    <div
      className="clotho-z-modal"
      style={{
        position: 'fixed',
        inset: 0,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: 'rgba(0, 0, 0, 0.6)',
        backdropFilter: 'blur(4px)',
      }}
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div
        style={{
          width: '90%',
          maxWidth: 640,
          maxHeight: '80vh',
          background: '#121216',
          border: '1px solid #2e2e38',
          borderRadius: 14,
          display: 'flex',
          flexDirection: 'column',
          overflow: 'hidden',
        }}
      >
        {/* Header */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            padding: '16px 20px',
            borderBottom: '1px solid #2e2e38',
          }}
        >
          <h2
            style={{
              margin: 0,
              fontSize: 15,
              fontWeight: 600,
              color: '#ececf0',
              flex: 1,
            }}
          >
            Pipeline Templates
          </h2>
          <button
            onClick={onClose}
            style={{
              background: 'none',
              border: 'none',
              color: '#8888a0',
              fontSize: 18,
              cursor: 'pointer',
              padding: '2px 6px',
              borderRadius: 4,
              lineHeight: 1,
            }}
          >
            x
          </button>
        </div>

        {/* Body */}
        <div
          style={{
            padding: 20,
            overflowY: 'auto',
            flex: 1,
          }}
        >
          {loading && (
            <p style={{ color: '#8888a0', fontSize: 13, textAlign: 'center' }}>
              Loading templates...
            </p>
          )}

          {error && (
            <p style={{ color: '#f87171', fontSize: 13, textAlign: 'center' }}>
              {error}
            </p>
          )}

          {!loading && !error && templates.length === 0 && (
            <p style={{ color: '#55556a', fontSize: 13, textAlign: 'center' }}>
              No templates available.
            </p>
          )}

          {!loading && templates.length > 0 && (
            <div
              style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(auto-fill, minmax(260px, 1fr))',
                gap: 12,
              }}
            >
              {templates.map((t) => (
                <TemplateCard
                  key={t.id}
                  template={t}
                  onSelect={handleSelect}
                />
              ))}
            </div>
          )}
        </div>

        {/* Footer hint */}
        <div
          style={{
            padding: '10px 20px',
            borderTop: '1px solid #2e2e38',
            fontSize: 11,
            color: '#55556a',
            textAlign: 'center',
          }}
        >
          Click a template to load it onto your canvas. Your current graph will
          be replaced.
        </div>
      </div>
    </div>
  );
}
