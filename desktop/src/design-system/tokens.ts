export const colors = {
  bg: {
    primary: 'var(--bg-primary)',
    secondary: 'var(--bg-secondary)',
    card: 'var(--bg-card)',
  },
  text: {
    primary: 'var(--text-primary)',
    secondary: 'var(--text-secondary)',
    muted: 'var(--text-muted)',
  },
  accent: {
    DEFAULT: 'var(--accent)',
    text: 'var(--accent-text)',
    hover: 'var(--accent-hover)',
  },
  border: {
    DEFAULT: 'var(--border)',
    subtle: 'var(--border-subtle)',
  },
  phase: {
    pending: 'var(--phase-pending)',
    active: 'var(--phase-active)',
    completed: 'var(--phase-completed)',
    error: 'var(--phase-error)',
    warning: 'var(--phase-warning)',
  },
  role: {
    primary: 'var(--role-primary)',
    engineer: 'var(--role-engineer)',
    designer: 'var(--role-designer)',
  },
} as const

export const spacing = {
  xs: 4,
  sm: 8,
  md: 12,
  base: 16,
  lg: 24,
  xl: 32,
  '2xl': 48,
  '3xl': 64,
} as const

export const radius = {
  sm: 8,
  card: 12,
  node: 16,
} as const

export const shadows = {
  sm: '0 1px 2px rgba(0, 0, 0, 0.05)',
  card: '0 1px 3px rgba(0, 0, 0, 0.08), 0 1px 2px rgba(0, 0, 0, 0.04)',
} as const

export const typography = {
  fontFamily: {
    heading: "'Inter', system-ui, sans-serif",
    body: "'Inter', system-ui, sans-serif",
    mono: "'JetBrains Mono', monospace",
  },
} as const
