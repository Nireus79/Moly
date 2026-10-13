import { describe, it, expect, beforeEach } from 'vitest';
import fs from 'node:fs';
import path from 'node:path';
import { applyTheme, getThemeChoice, setThemeChoice } from '@/styles/theme';

const SRC = path.resolve(__dirname, '..');
const PICTOGRAPH = /[\u{1F300}-\u{1FAFF}\u{2600}-\u{2612}\u{2614}-\u{2712}\u{2728}\u{2705}\u{274C}\u{26A0}\u{23F0}\u{23F3}\u{2B50}]/u;

function sourceFiles(dir: string): string[] {
  return fs.readdirSync(dir, { withFileTypes: true }).flatMap((e) => {
    const p = path.join(dir, e.name);
    if (e.isDirectory()) return e.name === '__tests__' ? [] : sourceFiles(p);
    return /\.(tsx|css)$/.test(e.name) ? [p] : [];
  });
}

describe('interface style', () => {
  it('uses no emoji in the visible interface (icons are drawn by components/Icon.tsx)', () => {
    const offenders: string[] = [];
    for (const file of sourceFiles(SRC)) {
      fs.readFileSync(file, 'utf8').split('\n').forEach((line, i) => {
        const code = line.trim();
        if (code.startsWith('//') || code.startsWith('*') || code.startsWith('/*') || line.includes('console.')) return;
        if (PICTOGRAPH.test(line)) offenders.push(`${path.relative(SRC, file)}:${i + 1}`);
      });
    }
    expect(offenders).toEqual([]);
  });

  it('keeps every colour of the screens in the theme variables (no hex colours in screen stylesheets)', () => {
    const allowed = new Set(['styles/theme.css', 'styles/brand.css']);
    const offenders: string[] = [];
    for (const file of sourceFiles(SRC).filter((f) => f.endsWith('.css'))) {
      const rel = path.relative(SRC, file);
      if (allowed.has(rel)) continue;
      if (/#[0-9a-fA-F]{3,8}\b/.test(fs.readFileSync(file, 'utf8'))) offenders.push(rel);
    }
    // the login screen is only for a hand-started development backend
    expect(offenders.filter((f) => !f.includes('login-screen'))).toEqual([]);
  });
});

describe('theme choice', () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.removeAttribute('data-theme');
  });

  it('follows the system by default and can be forced to dark or light', () => {
    expect(getThemeChoice()).toBe('system');
    setThemeChoice('dark');
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark');
    expect(getThemeChoice()).toBe('dark');
    setThemeChoice('light');
    expect(document.documentElement.getAttribute('data-theme')).toBe('light');
    setThemeChoice('system');
    expect(document.documentElement.hasAttribute('data-theme')).toBe(false);
    expect(getThemeChoice()).toBe('system');
  });

  it('applies the stored choice at start', () => {
    localStorage.setItem('moly_theme', 'dark');
    applyTheme();
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark');
  });
});
