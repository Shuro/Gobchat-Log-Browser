// Theme + per-theme highlight color overrides (config.colors). Overrides are
// written as inline CSS custom properties on the document root, on top of the
// theme defaults from style.css.

export type ThemeName = 'dark' | 'light' | 'dark-gobchat-ex'
export type ColorCategory = 'speech' | 'emote' | 'ooc' | 'mention-fg' | 'mention-bg'

// Category → CSS custom property carrying it.
export const COLOR_VARS: Record<ColorCategory, string> = {
  speech: '--color-speech',
  emote: '--color-emote',
  ooc: '--color-ooc',
  'mention-fg': '--color-mention-fg',
  'mention-bg': '--color-mention-bg',
}

// Theme default colors. Must match the --color-* values in style.css
// (:root and :root[data-theme='light']).
export const DEFAULT_COLORS: Record<ThemeName, Record<ColorCategory, string>> = {
  dark: {
    speech: '#ffffff',
    emote: '#f0c674',
    ooc: '#8c9bb5',
    'mention-fg': '#c8f0c0',
    'mention-bg': '#3a5a40',
  },
  light: {
    speech: '#0b1a2b',
    emote: '#8a5a00',
    ooc: '#6b7787',
    'mention-fg': '#1d4d1d',
    'mention-bg': '#cdeccd',
  },
  // GobchatEx "FFXIV Modern" dark palette: ink speech, warm gold emote/mention.
  'dark-gobchat-ex': {
    speech: '#e8eaee',
    emote: '#f0c074',
    ooc: '#a0a7b4',
    'mention-fg': '#f0c074',
    'mention-bg': 'rgba(224,164,78,.16)',
  },
}

export function normalizeTheme(theme: string | undefined): ThemeName {
  if (theme === 'light') return 'light'
  if (theme === 'dark-gobchat-ex') return 'dark-gobchat-ex'
  return 'dark'
}

// applyTheme sets the active theme on the document root; CSS variables under
// :root and :root[data-theme="light"] do the rest. Defaults to dark. Color
// overrides for the active theme are applied on top; missing entries fall
// back to the stylesheet defaults (stale inline overrides are removed so a
// theme switch or a reset takes effect).
export function applyTheme(
  theme: string | undefined,
  colors?: Record<string, Record<string, string>> | null,
): void {
  const t = normalizeTheme(theme)
  const root = document.documentElement
  root.setAttribute('data-theme', t)
  const overrides = colors?.[t] ?? {}
  for (const [cat, cssVar] of Object.entries(COLOR_VARS)) {
    const v = overrides[cat]
    if (v) root.style.setProperty(cssVar, v)
    else root.style.removeProperty(cssVar)
  }
}
