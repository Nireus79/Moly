import { describe, it, expect, afterEach, vi } from 'vitest';
import { createRoot } from 'react-dom/client';
import { act } from 'react-dom/test-utils';

vi.mock('@/api/accounts', () => ({
  signIn: vi.fn(async () => ({ sessionId: 's', userId: 'u', expiresAt: Date.now() + 1000 })),
  resetPassword: vi.fn(async () => ({ session: { sessionId: 's2', userId: 'u', expiresAt: Date.now() + 1000 }, recoveryCode: 'NEW1-NEW2-NEW3-NEW4' })),
}));
import { ProfileChooser } from '../ProfileChooser';
import { RecoveryCode } from '../RecoveryCode';
import { resetPassword } from '@/api/accounts';

(globalThis as any).IS_REACT_ACT_ENVIRONMENT = true;
let host: HTMLDivElement | null = null;
async function render(node: React.ReactElement) {
  host = document.createElement('div');
  document.body.appendChild(host);
  await act(async () => createRoot(host!).render(node));
  return host;
}
afterEach(() => { host?.remove(); host = null; });
const buttons = (h: HTMLElement) => Array.from(h.querySelectorAll('button'));
const byText = (h: HTMLElement, t: string) => buttons(h).find((b) => b.textContent?.includes(t))!;
function type(el: HTMLInputElement, value: string) {
  const set = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')!.set!;
  set.call(el, value);
  el.dispatchEvent(new Event('input', { bubbles: true }));
}

describe('who is using Moly', () => {
  it('offers the local profile and every profile with a password', async () => {
    const h = await render(<ProfileChooser profiles={['Anna', 'Ben']} onLocal={() => {}} onSession={() => {}} />);
    expect(buttons(h).map((b) => b.textContent)).toEqual(['You (no password)', 'Anna', 'Ben']);
  });

  it('the local profile asks for nothing', async () => {
    const onLocal = vi.fn();
    const h = await render(<ProfileChooser profiles={['Anna']} onLocal={onLocal} onSession={() => {}} />);
    await act(async () => byText(h, 'You').click());
    expect(onLocal).toHaveBeenCalled();
  });

  it('opens straight at the password of the profile used last', async () => {
    const h = await render(<ProfileChooser profiles={['Anna', 'Ben']} preferred="Ben" onLocal={() => {}} onSession={() => {}} />);
    expect(h.textContent).toContain('Welcome back, Ben');
    expect(h.querySelector('input[type=password]')).not.toBeNull();
  });

  it('a forgotten password: the code, a new password, then the new code must be kept before going on', async () => {
    const onSession = vi.fn();
    const h = await render(<ProfileChooser profiles={['Anna']} preferred="Anna" onLocal={() => {}} onSession={onSession} />);
    await act(async () => byText(h, 'forgot').click());
    await act(async () => type(h.querySelector('#profile-code') as HTMLInputElement, 'aaaa-bbbb-cccc-dddd'));
    await act(async () => type(h.querySelector('#profile-newpass') as HTMLInputElement, 'brandnew123'));
    await act(async () => { h.querySelector('form')!.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true })); });
    expect(resetPassword).toHaveBeenCalledWith('Anna', 'aaaa-bbbb-cccc-dddd', 'brandnew123');
    expect(h.textContent).toContain('NEW1-NEW2-NEW3-NEW4');
    expect(onSession).not.toHaveBeenCalled(); // not before the code is kept
    const go = byText(h, 'Continue');
    expect(go.disabled).toBe(true);
    await act(async () => (h.querySelector('input[type=checkbox]') as HTMLInputElement).click());
    await act(async () => byText(h, 'Continue').click());
    expect(onSession).toHaveBeenCalledWith('Anna', expect.objectContaining({ sessionId: 's2' }));
  });
});

describe('recovery code', () => {
  it('cannot be passed until the person says they kept it', async () => {
    const done = vi.fn();
    const h = await render(<RecoveryCode code="AAAA-BBBB-CCCC-DDDD" onDone={done} />);
    expect(byText(h, 'Continue').disabled).toBe(true);
    await act(async () => (h.querySelector('input[type=checkbox]') as HTMLInputElement).click());
    await act(async () => byText(h, 'Continue').click());
    expect(done).toHaveBeenCalled();
  });
});
