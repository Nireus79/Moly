import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { classifyHostError, detectOS, installInfo, needsUserAction, problemText, protocolMismatch, uninstallInfo, HOST_PROTOCOL } from '../hostErrors';
import { UNINSTALL_URL } from '../../config/links';

describe('uninstall', () => {
  it('every system has steps, and the data stays unless the person deletes it', () => {
    for (const os of ['windows', 'macos', 'linux', 'unknown'] as const) {
      const info = uninstallInfo(os);
      expect(info.steps.length).toBeGreaterThanOrEqual(2);
      expect(info.steps.join(' ')).toContain('.moly');
    }
    expect(uninstallInfo('linux').steps.join(' ')).toContain('apt remove moly');
    expect(uninstallInfo('windows').steps.join(' ')).toContain('Installed apps');
  });
  it('the background worker (which cannot import) opens the same page after the extension is removed', () => {
    const src = readFileSync(resolve(__dirname, '../../background.ts'), 'utf8');
    expect(src).toContain(`const UNINSTALL_URL = '${UNINSTALL_URL}';`);
    expect(src).toContain('setUninstallURL(UNINSTALL_URL)');
  });
});

describe('classifyHostError', () => {
  it('recognises the browser messages', () => {
    expect(classifyHostError('Specified native messaging host not found.')).toBe('not-installed');
    expect(classifyHostError('Native messaging host com.moly.backend_host is not registered.')).toBe('not-installed');
    expect(classifyHostError('Access to the specified native messaging host is forbidden.')).toBe('not-allowed');
    expect(classifyHostError('Native host has exited.')).toBe('host-crashed');
    expect(classifyHostError('The Moly app did not answer "start" in time.')).toBe('start-timeout');
    expect(classifyHostError('the backend did not become ready in time (see ~/.moly/moly.log)')).toBe('start-timeout');
    expect(classifyHostError('Something odd')).toBe('unknown');
    expect(classifyHostError(undefined)).toBe('none');
    expect(classifyHostError('')).toBe('none');
  });

  it('only a missing or unrecognised app needs the user to act', () => {
    expect(needsUserAction('not-installed')).toBe(true);
    expect(needsUserAction('not-allowed')).toBe(true);
    for (const p of ['host-crashed', 'start-timeout', 'unknown', 'none'] as const) expect(needsUserAction(p)).toBe(false);
  });
});

describe('detectOS and installInfo', () => {
  it('detects the system from the user agent', () => {
    expect(detectOS('Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/130')).toBe('windows');
    expect(detectOS('Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Chrome/130')).toBe('macos');
    expect(detectOS('Mozilla/5.0 (X11; Linux x86_64) Chrome/130')).toBe('linux');
    expect(detectOS('something else')).toBe('unknown');
  });

  it('every system has steps that end by checking again, and macOS is marked beta', () => {
    for (const os of ['windows', 'macos', 'linux', 'unknown'] as const) {
      const info = installInfo(os);
      expect(info.steps.length).toBeGreaterThanOrEqual(3);
      expect(info.steps[info.steps.length - 1]).toContain('Check again');
    }
    expect(installInfo('macos').heading).toContain('beta');
    expect(installInfo('windows').steps.join(' ')).toContain('Run anyway');
  });

  it('every problem has a title and a text', () => {
    for (const p of ['not-allowed', 'host-crashed', 'start-timeout', 'unknown'] as const) {
      const t = problemText(p);
      expect(t.title.length).toBeGreaterThan(5);
      expect(t.body).toContain('Check again');
    }
  });
});

describe('protocol between the extension and the app', () => {
  it('a mismatch is its own problem that needs the user to act, with its own text', () => {
    const m = protocolMismatch('2');
    expect(classifyHostError(m)).toBe('app-mismatch');
    expect(needsUserAction('app-mismatch')).toBe(true);
    expect(problemText('app-mismatch').title).toMatch(/do not match/);
  });
  it('the background worker (which cannot import) carries the same protocol number', () => {
    const src = readFileSync(resolve(__dirname, '../../background.ts'), 'utf8');
    expect(src).toContain(`const HOST_PROTOCOL = '${HOST_PROTOCOL}';`);
  });
});

describe('the side panel', () => {
  it('is never closed by emptying its page path (that left an error page until the extension was reloaded)', () => {
    const src = readFileSync(resolve(__dirname, '../../background.ts'), 'utf8');
    expect(src).not.toMatch(/path:\s*''/);
    expect(src).toContain("path: PANEL_PATH, enabled: true");
  });
});
