import { describe, it, expect } from 'vitest';
import { installAction, INSTALL_COMMAND, compareVersions, latestVersion } from '../installerDownload';

const releases = [
  { tag_name: 'mobile-v0.1.0-beta.5', draft: false, assets: [{ name: 'moly-0.1.0-beta.5.apk', browser_download_url: 'https://x/apk' }] },
  { tag_name: 'v0.1.0-beta.3', draft: false, assets: [{ name: 'moly-setup-0.1.0-beta.3.exe', browser_download_url: 'https://github.com/Nireus79/Moly/releases/download/v0.1.0-beta.3/moly-setup-0.1.0-beta.3.exe' }] },
];
const reply = (body: unknown, ok = true) => (async () => ({ ok, json: async () => body })) as unknown as typeof fetch;

describe('installAction', () => {
  it('Windows gets the newest desktop installer, not the Android release', async () => {
    expect(await installAction('windows', reply(releases))).toEqual({ kind: 'download', url: 'https://github.com/Nireus79/Moly/releases/download/v0.1.0-beta.3/moly-setup-0.1.0-beta.3.exe', fileName: 'moly-setup-0.1.0-beta.3.exe', version: '0.1.0-beta.3' });
  });
  it('Linux and Mac get the command while the release has no .deb or .pkg, and when the lookup fails', async () => {
    const offline = (async () => { throw new Error('offline'); }) as unknown as typeof fetch;
    expect(await installAction('linux', reply(releases))).toEqual({ kind: 'command', command: INSTALL_COMMAND });
    expect(await installAction('macos', reply(releases))).toEqual({ kind: 'command', command: INSTALL_COMMAND });
    expect(await installAction('linux', offline)).toEqual({ kind: 'command', command: INSTALL_COMMAND });
  });
  it('Linux and Mac get the installer to open once the release has one', async () => {
    const base = 'https://github.com/Nireus79/Moly/releases/download/v0.1.2/';
    const rel = [{ tag_name: 'v0.1.2', draft: false, assets: [
      { name: 'moly-setup-0.1.2.exe', browser_download_url: base + 'moly-setup-0.1.2.exe' },
      { name: 'moly_0.1.2_amd64.deb', browser_download_url: base + 'moly_0.1.2_amd64.deb' },
      { name: 'moly-0.1.2-macos-universal.pkg', browser_download_url: base + 'moly-0.1.2-macos-universal.pkg' },
      { name: 'moly-0.1.2-linux-amd64.tar.gz', browser_download_url: base + 'moly-0.1.2-linux-amd64.tar.gz' },
    ] }];
    expect(await installAction('linux', reply(rel))).toMatchObject({ kind: 'download', fileName: 'moly_0.1.2_amd64.deb', version: '0.1.2' });
    expect(await installAction('macos', reply(rel))).toMatchObject({ kind: 'download', fileName: 'moly-0.1.2-macos-universal.pkg' });
  });
  it('a file that is not from the Moly releases is never offered', async () => {
    const evil = [{ tag_name: 'v9.9.9', draft: false, assets: [{ name: 'moly-setup-9.9.9.exe', browser_download_url: 'https://evil.example/moly-setup-9.9.9.exe' }] }];
    expect(await installAction('windows', reply(evil))).toEqual({ kind: 'page' });
  });
  it('a failed lookup or an unknown system falls back to the download page', async () => {
    expect(await installAction('windows', reply([], false))).toEqual({ kind: 'page' });
    expect(await installAction('windows', reply([{ tag_name: 'v1', draft: false, assets: [] }]))).toEqual({ kind: 'page' });
    expect(await installAction('windows', (async () => { throw new Error('offline'); }) as unknown as typeof fetch)).toEqual({ kind: 'page' });
    expect(await installAction('unknown', reply(releases))).toEqual({ kind: 'page' });
  });
});

describe('versions', () => {
  it('orders releases, with a pre-release older than its release', () => {
    expect(compareVersions('0.1.2', '0.1.0-beta.4')).toBeGreaterThan(0);
    expect(compareVersions('0.1.0-beta.5', '0.1.0-beta.4')).toBeGreaterThan(0);
    expect(compareVersions('0.1.0-beta.10', '0.1.0-beta.9')).toBeGreaterThan(0);
    expect(compareVersions('0.1.0', '0.1.0-beta.4')).toBeGreaterThan(0);
    expect(compareVersions('0.1.1', '0.1.2')).toBeLessThan(0);
    expect(compareVersions('v0.1.2', '0.1.2')).toBe(0);
    expect(compareVersions('0.2.0', '0.1.9')).toBeGreaterThan(0);
  });
  it('says nothing about a development build', () => {
    expect(compareVersions('0.1.2', 'dev')).toBeNull();
    expect(compareVersions('', '0.1.2')).toBeNull();
  });
  it('the newest version is the newest desktop release', async () => {
    expect(await latestVersion(reply(releases))).toBe('0.1.0-beta.3');
    expect(await latestVersion(reply([], false))).toBeNull();
  });
});
