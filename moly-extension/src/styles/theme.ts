/**
 * Appearance: follow the system by default; Settings can force light or dark. The choice is a per-viewer convenience
 * kept in localStorage (it may be unavailable, then the system theme is used).
 */
export type ThemeChoice = 'system' | 'light' | 'dark';
const KEY = 'moly_theme';

export function getThemeChoice(): ThemeChoice {
  try {
    const v = localStorage.getItem(KEY);
    return v === 'light' || v === 'dark' ? v : 'system';
  } catch {
    return 'system';
  }
}

export function applyTheme(choice: ThemeChoice = getThemeChoice()): void {
  const root = document.documentElement;
  if (choice === 'system') root.removeAttribute('data-theme');
  else root.setAttribute('data-theme', choice);
}

export function setThemeChoice(choice: ThemeChoice): void {
  try {
    if (choice === 'system') localStorage.removeItem(KEY);
    else localStorage.setItem(KEY, choice);
  } catch {
    /* storage unavailable: the choice lasts until the page closes */
  }
  applyTheme(choice);
}
