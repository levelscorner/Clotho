import type { CSSProperties } from 'react';

// Shared inspector-field primitives. Kept as plain objects (not CSS
// modules) so they compose cleanly with the inline style pattern the
// rest of the inspector already uses.

export const fieldGroup: CSSProperties = {
  marginBottom: 12,
};

export const labelStyle: CSSProperties = {
  display: 'block',
  fontSize: 11,
  fontWeight: 600,
  color: 'var(--text-muted)',
  marginBottom: 4,
  textTransform: 'uppercase',
  letterSpacing: '0.04em',
};

export const inputStyle: CSSProperties = {
  width: '100%',
  padding: '6px 8px',
  borderRadius: 'var(--radius-sm)',
  border: '1px solid var(--surface-border)',
  background: 'var(--surface-base)',
  color: 'var(--text-primary)',
  fontSize: 13,
};

export const textareaStyle: CSSProperties = {
  ...inputStyle,
  minHeight: 140,
  resize: 'vertical',
};

export const helperTextStyle: CSSProperties = {
  marginTop: 4,
  fontSize: 11,
  color: 'var(--text-muted)',
  lineHeight: 1.4,
};
