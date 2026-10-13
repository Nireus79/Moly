import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { createRoot } from 'react-dom/client';
import { act } from 'react-dom/test-utils';

(globalThis as any).IS_REACT_ACT_ENVIRONMENT = true;

const api = vi.hoisted(() => ({
  getKeystoreStatus: vi.fn(),
  recheckKeystore: vi.fn(),
  unlockKeystore: vi.fn(),
  setupKeystore: vi.fn(),
  backendIsReady: vi.fn(),
}));
vi.mock('@/api/keystoreAPI', () => api);

import { KeystoreGate } from '../KeystoreGate';

const ok = (data: object) => Promise.resolve({ ok: true, data });
const status = (state: string, extra: object = {}) => ({ state, keychainAvailable: false, ...extra });

let host: HTMLDivElement | null = null;
const flush = () => act(async () => { await new Promise((r) => setTimeout(r, 0)); });
async function mount() {
  host = document.createElement('div');
  document.body.appendChild(host);
  await act(async () => createRoot(host!).render(<KeystoreGate><div>THE APP</div></KeystoreGate>));
  await flush();
  await flush();
}
const text = () => host!.textContent || '';
const button = (name: string) => [...host!.querySelectorAll('button')].find((b) => b.textContent === name) as HTMLButtonElement;
const field = (labelStart: string) => {
  const label = [...host!.querySelectorAll('label')].find((l) => (l.textContent || '').startsWith(labelStart))!;
  return host!.querySelector('#' + label.getAttribute('for')) as HTMLInputElement;
};
async function type(input: HTMLInputElement, value: string) {
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')!.set!.call(input, value);
    input.dispatchEvent(new Event('input', { bubbles: true }));
  });
}
async function click(el: HTMLElement) {
  await act(async () => { el.dispatchEvent(new MouseEvent('click', { bubbles: true })); });
  await flush();
  await flush();
}

afterEach(() => { host?.remove(); host = null; });

describe('KeystoreGate', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    api.backendIsReady.mockResolvedValue(true);
  });

  it('passes through when the app has no keystore gate (development backend)', async () => {
    api.getKeystoreStatus.mockResolvedValue({ ok: false, error: 'not found' });
    await mount();
    expect(text()).toContain('THE APP');
  });

  it('passes through, once the app is open, when the key is already available', async () => {
    api.getKeystoreStatus.mockReturnValue(ok(status('ready')));
    await mount();
    expect(text()).toContain('THE APP');
    expect(api.backendIsReady).toHaveBeenCalled();
  });

  it('asks for a passphrase when there is no keychain, and accepts only a matching, long enough one', async () => {
    api.getKeystoreStatus.mockReturnValue(ok(status('needs_setup', { options: ['passphrase', 'file'] })));
    api.setupKeystore.mockReturnValue(ok(status('ready')));
    await mount();
    expect(text()).toContain('Protect your data');
    const submit = button('Protect with this passphrase');
    expect(submit.disabled).toBe(true);
    await type(field('Passphrase (at least'), 'short');
    expect(submit.disabled).toBe(true);
    await type(field('Passphrase (at least'), 'a long enough one');
    await type(field('Once more'), 'different one here');
    expect(submit.disabled).toBe(true);
    await type(field('Once more'), 'a long enough one');
    expect(submit.disabled).toBe(false);
    await click(submit);
    expect(api.setupKeystore).toHaveBeenCalledWith('passphrase', 'a long enough one');
    expect(text()).toContain('THE APP');
  });

  it('offers the key file only with a clear warning and an explicit confirmation', async () => {
    api.getKeystoreStatus.mockReturnValue(ok(status('needs_setup', { options: ['passphrase', 'file'] })));
    api.setupKeystore.mockReturnValue(ok(status('ready')));
    await mount();
    await click(button('Use a key file instead'));
    expect(text()).toContain('less secure');
    const submit = button('Use a key file');
    expect(submit.disabled).toBe(true);
    await click(host!.querySelector('input[type=checkbox]') as HTMLElement);
    expect(submit.disabled).toBe(false);
    await click(submit);
    expect(api.setupKeystore).toHaveBeenCalledWith('file', undefined, true);
  });

  it('unlocks with the passphrase and shows the error for a wrong one', async () => {
    api.getKeystoreStatus.mockReturnValue(ok(status('locked')));
    api.unlockKeystore.mockResolvedValueOnce({ ok: false, error: 'That passphrase is not right.' }).mockReturnValueOnce(ok(status('ready')));
    await mount();
    expect(text()).toContain('Unlock Moly');
    await type(field('Passphrase'), 'wrong one');
    await click(button('Unlock'));
    expect(text()).toContain('That passphrase is not right.');
    expect(text()).not.toContain('THE APP');
    await type(field('Passphrase'), 'the right one');
    await click(button('Unlock'));
    expect(text()).toContain('THE APP');
    expect(api.unlockKeystore).toHaveBeenLastCalledWith('the right one');
  });

  it('explains a lost key and lets the person check again', async () => {
    api.getKeystoreStatus.mockReturnValue(ok(status('unknown_key', { reason: 'The key is no longer in the keychain.' })));
    api.recheckKeystore.mockReturnValue(ok(status('ready')));
    await mount();
    expect(text()).toContain('Moly cannot find the key to your data');
    expect(text()).toContain('The key is no longer in the keychain.');
    await click(button('Check again'));
    expect(text()).toContain('THE APP');
  });

  it('never shows the passphrase back after a failed attempt', async () => {
    api.getKeystoreStatus.mockReturnValue(ok(status('locked')));
    api.unlockKeystore.mockResolvedValue({ ok: false, error: 'no' });
    await mount();
    await type(field('Passphrase'), 'secret passphrase');
    await click(button('Unlock'));
    expect(text()).toContain('no');
    expect(text()).not.toContain('secret passphrase');
  });
});
