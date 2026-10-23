// Runs in <head> before the body is parsed, so the first paint already has the scheme that the
// theme toggle saved (src/shell/theme.ts). A classic file, not inline, so a CSP without
// 'unsafe-inline' still runs it.
try {
  const theme = localStorage.getItem('theme');
  if (theme === 'light' || theme === 'dark') {
    document.documentElement.dataset.theme = theme;
  }
} catch {
  // Storage is off, so the OS's scheme applies.
}
