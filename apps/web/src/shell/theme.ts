// The theme toggle's choice: follow the OS, or force a scheme on this device.
export type ThemeChoice = 'system' | 'light' | 'dark';

type Scheme = Exclude<ThemeChoice, 'system'>;

// public/theme.js reads the same key before the first paint.
const storageKey = 'theme';

// Each scheme's theme-color, the top bar's --ink, as index.html sets it.
export const themeColors: Record<Scheme, string> = { light: '#111111', dark: '#ededea' };

const next: Record<ThemeChoice, ThemeChoice> = { system: 'light', light: 'dark', dark: 'system' };

export function nextChoice(choice: ThemeChoice): ThemeChoice {
  return next[choice];
}

export function loadChoice(): ThemeChoice {
  try {
    const stored = localStorage.getItem(storageKey);
    return stored === 'light' || stored === 'dark' ? stored : 'system';
  } catch {
    return 'system';
  }
}

export function saveChoice(choice: ThemeChoice): void {
  try {
    if (choice === 'system') {
      localStorage.removeItem(storageKey);
    } else {
      localStorage.setItem(storageKey, choice);
    }
  } catch {
    // Storage is off or full, so the choice lasts only as long as the page.
  }
}

// Sets the scheme that tokens.css follows, and the browser's own bar to match.
export function applyChoice(choice: ThemeChoice): void {
  const root = document.documentElement;
  if (choice === 'system') {
    delete root.dataset.theme;
  } else {
    root.dataset.theme = choice;
  }
  for (const scheme of ['light', 'dark'] as const) {
    const meta = document.querySelector<HTMLMetaElement>(
      `meta[name="theme-color"][media="(prefers-color-scheme: ${scheme})"]`
    );
    if (meta) {
      meta.content = themeColors[choice === 'system' ? scheme : choice];
    }
  }
}
