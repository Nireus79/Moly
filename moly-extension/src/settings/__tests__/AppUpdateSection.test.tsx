import { describe, it, expect, afterEach, vi } from 'vitest';
import { createRoot } from 'react-dom/client';
import { act } from 'react-dom/test-utils';

const state = { app: '0.1.0-beta.4' as string | undefined, latest: '0.1.2' as string | null };
vi.mock('@/api/backendManager', () => ({ getBackendManager: () => ({ getStatus: async () => ({ running: true, version: state.app }) }) }));
vi.mock('@/api/installerDownload', async (orig) => ({
  ...(await orig<typeof import('@/api/installerDownload')>()),
  latestVersion: async () => state.latest,
  installAction: async () => ({ kind: 'page' }),
}));
import { AppUpdateSection } from '../AppUpdateSection';

(globalThis as any).IS_REACT_ACT_ENVIRONMENT = true;
let host: HTMLDivElement | null = null;
async function render() {
  host = document.createElement('div');
  document.body.appendChild(host);
  await act(async () => createRoot(host!).render(<AppUpdateSection />));
  await act(async () => { await new Promise((r) => setTimeout(r, 10)); });
  return host;
}
afterEach(() => { host?.remove(); host = null; });

describe('Moly app section', () => {
  it('offers the update when a newer release exists', async () => {
    state.app = '0.1.0-beta.4'; state.latest = '0.1.2';
    const h = await render();
    expect(h.textContent).toContain('0.1.0-beta.4 is installed; version 0.1.2 is available');
    expect(Array.from(h.querySelectorAll('button')).some((b) => b.textContent === 'Update to 0.1.2')).toBe(true);
  });
  it('treats the interface number of apps before 0.1.2 as an older app and offers the update', async () => {
    state.app = '2.1'; state.latest = '0.1.2';
    const h = await render();
    expect(h.textContent).toContain('An older version of the Moly app is installed');
    expect(Array.from(h.querySelectorAll('button')).some((b) => b.textContent === 'Update to 0.1.2')).toBe(true);
  });
  it('says so when the app is the newest, and shows no button', async () => {
    state.app = '0.1.2'; state.latest = '0.1.2';
    const h = await render();
    expect(h.textContent).toContain('This is the newest version');
    expect(h.querySelector('button')).toBeNull();
  });
  it('does not claim anything about a development build', async () => {
    state.app = 'dev'; state.latest = '0.1.2';
    const h = await render();
    expect(h.textContent).toContain('could not tell');
    expect(h.querySelector('button')).toBeNull();
  });
});
