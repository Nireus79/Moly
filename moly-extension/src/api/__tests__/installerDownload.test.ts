import { describe, it, expect } from 'vitest';
import { installAction, INSTALL_COMMAND } from '../installerDownload';

const releases = [
  { tag_name: 'mobile-v0.1.0-beta.5', draft: false, assets: [{ name: 'moly-0.1.0-beta.5.apk', browser_download_url: 'https://x/apk' }] },
  { tag_name: 'v0.1.0-beta.3', draft: false, assets: [{ name: 'moly-setup-0.1.0-beta.3.exe', browser_download_url: 'https://x/setup.exe' }] },
];
const reply = (body: unknown, ok = true) => (async () => ({ ok, json: async () => body })) as unknown as typeof fetch;

describe('installAction', () => {
  it('Windows gets the newest desktop installer, not the Android release', async () => {
    expect(await installAction('windows', reply(releases))).toEqual({ kind: 'download', url: 'https://x/setup.exe', fileName: 'moly-setup-0.1.0-beta.3.exe', version: '0.1.0-beta.3' });
  });
  it('Linux and Mac get the command without any lookup', async () => {
    const never = (async () => { throw new Error('no network expected'); }) as unknown as typeof fetch;
    expect(await installAction('linux', never)).toEqual({ kind: 'command', command: INSTALL_COMMAND });
    expect(await installAction('macos', never)).toEqual({ kind: 'command', command: INSTALL_COMMAND });
  });
  it('a failed lookup or an unknown system falls back to the download page', async () => {
    expect(await installAction('windows', reply([], false))).toEqual({ kind: 'page' });
    expect(await installAction('windows', reply([{ tag_name: 'v1', draft: false, assets: [] }]))).toEqual({ kind: 'page' });
    expect(await installAction('windows', (async () => { throw new Error('offline'); }) as unknown as typeof fetch)).toEqual({ kind: 'page' });
    expect(await installAction('unknown', reply(releases))).toEqual({ kind: 'page' });
  });
});
