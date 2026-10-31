import { describe, it, expect, vi } from 'vitest';
import { createRoot } from 'react-dom/client';
import { act } from 'react-dom/test-utils';

const createProfile = vi.fn(async () => ({ session: { sessionId: 's1', userId: 'u1', expiresAt: 1 }, recoveryCode: 'AAAA-BBBB-CCCC-DDDD' }));
vi.mock('@/api/accounts', () => ({ createProfile: (...a: unknown[]) => createProfile(...(a as [])) }));
import { FirstProfile } from '../FirstProfile';

(globalThis as any).IS_REACT_ACT_ENVIRONMENT = true;
async function render(onSession = vi.fn()) {
  const h = document.createElement('div');
  document.body.appendChild(h);
  await act(async () => createRoot(h).render(<FirstProfile onSession={onSession} />));
  return h;
}
function type(el: HTMLInputElement, value: string) {
  Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')!.set!.call(el, value);
  el.dispatchEvent(new Event('input', { bubbles: true }));
}
const submit = (h: HTMLElement) => act(async () => { h.querySelector('form')!.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true })); });
const fill = async (h: HTMLElement, name: string, p1: string, p2: string) => {
  await act(async () => type(h.querySelector('#first-name') as HTMLInputElement, name));
  await act(async () => type(h.querySelector('#first-password') as HTMLInputElement, p1));
  await act(async () => type(h.querySelector('#first-confirm') as HTMLInputElement, p2));
};

describe('first profile', () => {
  it('asks for a name and a password and offers no way in without them', async () => {
    const h = await render();
    expect(h.textContent).toContain('Make your profile');
    await submit(h);
    expect(h.textContent).toContain('Choose a name.');
    expect(createProfile).not.toHaveBeenCalled();
  });
  it('refuses a short password and two different passwords', async () => {
    const h = await render();
    await fill(h, 'Anna', 'short', 'short');
    await submit(h);
    expect(h.textContent).toContain('at least 8 characters');
    await fill(h, 'Anna', 'longenough1', 'longenough2');
    await submit(h);
    expect(h.textContent).toContain('not the same');
    expect(createProfile).not.toHaveBeenCalled();
  });
  it('shows the recovery code once, and enters only after it was kept', async () => {
    const onSession = vi.fn();
    const h = await render(onSession);
    await fill(h, ' Anna ', 'longenough1', 'longenough1');
    await submit(h);
    expect(createProfile).toHaveBeenCalledWith(' Anna ', 'longenough1');
    expect(h.textContent).toContain('AAAA-BBBB-CCCC-DDDD');
    expect(onSession).not.toHaveBeenCalled();
    await act(async () => (h.querySelector('input[type=checkbox]') as HTMLInputElement).click());
    await act(async () => Array.from(h.querySelectorAll('button')).find((b) => b.textContent === 'Continue')!.click());
    expect(onSession).toHaveBeenCalledWith('Anna', expect.objectContaining({ sessionId: 's1' }));
  });
});
