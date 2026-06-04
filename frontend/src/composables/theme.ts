// applyTheme sets the active theme on the document root; CSS variables under
// :root and :root[data-theme="light"] do the rest. Defaults to dark.
export function applyTheme(theme: string | undefined): void {
  const t = theme === 'light' ? 'light' : 'dark'
  document.documentElement.setAttribute('data-theme', t)
}
