import { useCallback, useState } from 'react';
import { useVersionStore } from '../../stores/versionStore';
import type { PipelineVersion } from '../../lib/types';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function timeAgo(date: string): string {
  const seconds = Math.floor((Date.now() - new Date(date).getTime()) / 1000);
  if (seconds < 60) return 'just now';
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
  return `${Math.floor(seconds / 86400)}d ago`;
}

// ---------------------------------------------------------------------------
// Styles
// ---------------------------------------------------------------------------

const panelStyle: React.CSSProperties = {
  position: 'absolute',
  top: 0,
  right: 0,
  width: 280,
  height: '100%',
  background: '#12131f',
  borderLeft: '1px solid #1e2030',
  display: 'flex',
  flexDirection: 'column',
  // zIndex migrated — use .clotho-z-overlay on the rendered element
};

const headerStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'space-between',
  padding: '12px 14px',
  borderBottom: '1px solid #1e2030',
  flexShrink: 0,
};

const titleStyle: React.CSSProperties = {
  fontSize: 13,
  fontWeight: 700,
  color: '#e2e8f0',
};

const closeButtonStyle: React.CSSProperties = {
  background: 'none',
  border: 'none',
  color: '#64748b',
  fontSize: 16,
  cursor: 'pointer',
  padding: '2px 6px',
  borderRadius: 4,
};

const listStyle: React.CSSProperties = {
  flex: 1,
  overflowY: 'auto',
  padding: '8px 0',
};

const itemStyle: React.CSSProperties = {
  padding: '10px 14px',
  cursor: 'pointer',
  borderBottom: '1px solid #1e2030',
  transition: 'background 0.15s',
};

const itemHoverStyle: React.CSSProperties = {
  ...itemStyle,
  background: '#1a1c2e',
};

const versionLabelStyle: React.CSSProperties = {
  fontSize: 13,
  fontWeight: 600,
  color: '#e2e8f0',
};

const timestampStyle: React.CSSProperties = {
  fontSize: 11,
  color: '#64748b',
  marginTop: 2,
};

const badgeStyle: React.CSSProperties = {
  display: 'inline-block',
  marginLeft: 8,
  padding: '1px 6px',
  borderRadius: 3,
  background: '#1e3a5f',
  color: '#60a5fa',
  fontSize: 10,
  fontWeight: 600,
};

const emptyStyle: React.CSSProperties = {
  padding: '24px 14px',
  color: '#475569',
  fontSize: 13,
  textAlign: 'center',
};

const loadingStyle: React.CSSProperties = {
  padding: '24px 14px',
  color: '#475569',
  fontSize: 13,
  textAlign: 'center',
};

const confirmOverlayStyle: React.CSSProperties = {
  position: 'fixed',
  inset: 0,
  background: 'rgba(0, 0, 0, 0.6)',
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  // zIndex migrated — use .clotho-z-modal on the rendered element
};

const confirmBoxStyle: React.CSSProperties = {
  background: '#1a1c2e',
  border: '1px solid #334155',
  borderRadius: 8,
  padding: 20,
  maxWidth: 320,
  color: '#e2e8f0',
  fontSize: 13,
};

const confirmButtonsStyle: React.CSSProperties = {
  display: 'flex',
  gap: 8,
  marginTop: 14,
  justifyContent: 'flex-end',
};

const confirmBtnStyle: React.CSSProperties = {
  padding: '6px 14px',
  borderRadius: 4,
  border: '1px solid #334155',
  background: '#3b82f6',
  color: '#fff',
  fontSize: 12,
  cursor: 'pointer',
  fontWeight: 600,
};

const cancelBtnStyle: React.CSSProperties = {
  padding: '6px 14px',
  borderRadius: 4,
  border: '1px solid #334155',
  background: 'transparent',
  color: '#94a3b8',
  fontSize: 12,
  cursor: 'pointer',
};

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function VersionPanel() {
  const versions = useVersionStore((s) => s.versions);
  const isLoading = useVersionStore((s) => s.isLoading);
  const isOpen = useVersionStore((s) => s.isOpen);
  const close = useVersionStore((s) => s.close);
  const restoreVersion = useVersionStore((s) => s.restoreVersion);

  const [hoveredId, setHoveredId] = useState<string | null>(null);
  const [confirmVersion, setConfirmVersion] = useState<PipelineVersion | null>(
    null,
  );

  const handleRestore = useCallback(() => {
    if (confirmVersion) {
      restoreVersion(confirmVersion);
      setConfirmVersion(null);
    }
  }, [confirmVersion, restoreVersion]);

  if (!isOpen) return null;

  const latestVersion =
    versions.length > 0
      ? versions.reduce((a, b) => (a.version > b.version ? a : b))
      : null;

  const sorted = [...versions].sort((a, b) => b.version - a.version);

  return (
    <>
      <div className="clotho-z-overlay" style={panelStyle}>
        <div style={headerStyle}>
          <span style={titleStyle}>Version History</span>
          <button
            style={closeButtonStyle}
            onClick={close}
            aria-label="Close version panel"
          >
            x
          </button>
        </div>

        <div style={listStyle}>
          {isLoading && <div style={loadingStyle}>Loading...</div>}
          {!isLoading && sorted.length === 0 && (
            <div style={emptyStyle}>No versions saved yet.</div>
          )}
          {!isLoading &&
            sorted.map((v) => {
              const isLatest = latestVersion?.id === v.id;
              return (
                <div
                  key={v.id}
                  style={hoveredId === v.id ? itemHoverStyle : itemStyle}
                  onMouseEnter={() => setHoveredId(v.id)}
                  onMouseLeave={() => setHoveredId(null)}
                  onClick={() => setConfirmVersion(v)}
                >
                  <div>
                    <span style={versionLabelStyle}>v{v.version}</span>
                    {isLatest && <span style={badgeStyle}>Current</span>}
                  </div>
                  <div style={timestampStyle}>{timeAgo(v.created_at)}</div>
                </div>
              );
            })}
        </div>
      </div>

      {confirmVersion && (
        <div className="clotho-z-modal" style={confirmOverlayStyle} onClick={() => setConfirmVersion(null)}>
          <div style={confirmBoxStyle} onClick={(e) => e.stopPropagation()}>
            <div style={{ fontWeight: 600, marginBottom: 8 }}>
              Restore Version
            </div>
            <div>
              Restore to <strong>v{confirmVersion.version}</strong>? Your
              current unsaved changes will be replaced.
            </div>
            <div style={confirmButtonsStyle}>
              <button
                style={cancelBtnStyle}
                onClick={() => setConfirmVersion(null)}
              >
                Cancel
              </button>
              <button style={confirmBtnStyle} onClick={handleRestore}>
                Restore
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
