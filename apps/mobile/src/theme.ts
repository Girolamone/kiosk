/**
 * The web app's palette, restated for React Native.
 *
 * These are duplicated rather than imported because the web values live in
 * CSS custom properties, which mean nothing here. The shared package holds
 * what both platforms can genuinely share — the schema, the operations, the
 * client — and stops there. Pretending a stylesheet crosses that boundary
 * would be the kind of shared code that costs more than it saves.
 */
export const colors = {
  paper: '#faf8f5',
  raised: '#ffffff',
  ink: '#1a1a18',
  muted: '#6f6a63',
  line: '#e4dfd7',
  accent: '#1c5c47',
  accentSoft: '#e8f0ec',
  danger: '#b3261e',
} as const

export const spacing = {
  xs: 4,
  sm: 8,
  md: 16,
  lg: 24,
  xl: 32,
} as const

export const radius = {
  sm: 6,
  md: 10,
} as const
