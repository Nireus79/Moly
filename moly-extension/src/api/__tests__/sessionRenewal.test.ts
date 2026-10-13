import { describe, it, expect, beforeEach, vi } from 'vitest';

const startLocalSession = vi.fn();
vi.mock('../localProfile', () => ({ startLocalSession: () => startLocalSession() }));
vi.mock('../backendAuth', () => ({ setSessionRenewer: vi.fn() }));

import { useAuthStore } from '@/stores/authStore';
import { renewLocalSession, localSessionNeedsRenewal } from '../sessionRenewal';

const DAY = 24 * 60 * 60 * 1000;

describe('session renewal', () => {
  beforeEach(() => {
    startLocalSession.mockReset();
    useAuthStore.setState({ session: null });
    localStorage.clear();
  });

  it('renews only the local profile', async () => {
    useAuthStore.setState({ session: { sessionId: 'acct', userId: 'u1', expiresAt: Date.now() + DAY } });
    expect(await renewLocalSession()).toBeNull();
    expect(startLocalSession).not.toHaveBeenCalled();
  });

  it('stores the new session and returns its token', async () => {
    useAuthStore.setState({ session: { sessionId: 'old', userId: 'user_local', expiresAt: Date.now() + 1000, local: true } });
    startLocalSession.mockResolvedValue({ sessionId: 'new', userId: 'user_local', expiresAt: Date.now() + 30 * DAY, local: true });
    expect(await renewLocalSession()).toBe('new');
    expect(useAuthStore.getState().session?.sessionId).toBe('new');
    expect(JSON.parse(localStorage.getItem('moly_session')!).sessionId).toBe('new');
  });

  it('keeps the old session when the app does not answer', async () => {
    useAuthStore.setState({ session: { sessionId: 'old', userId: 'user_local', expiresAt: Date.now() + 1000, local: true } });
    startLocalSession.mockResolvedValue(null);
    expect(await renewLocalSession()).toBeNull();
    expect(useAuthStore.getState().session?.sessionId).toBe('old');
  });

  it('asks the app once when several requests fail together', async () => {
    useAuthStore.setState({ session: { sessionId: 'old', userId: 'user_local', expiresAt: Date.now() + 1000, local: true } });
    startLocalSession.mockImplementation(async () => {
      await new Promise((r) => setTimeout(r, 20));
      return { sessionId: 'new', userId: 'user_local', expiresAt: Date.now() + 30 * DAY, local: true };
    });
    const results = await Promise.all([renewLocalSession(), renewLocalSession(), renewLocalSession()]);
    expect(results).toEqual(['new', 'new', 'new']);
    expect(startLocalSession).toHaveBeenCalledTimes(1);
  });

  it('knows when a local session is close to its end', () => {
    const now = Date.now();
    useAuthStore.setState({ session: { sessionId: 's', userId: 'user_local', expiresAt: now + 2 * DAY, local: true } });
    expect(localSessionNeedsRenewal(now)).toBe(true);
    useAuthStore.setState({ session: { sessionId: 's', userId: 'user_local', expiresAt: now + 20 * DAY, local: true } });
    expect(localSessionNeedsRenewal(now)).toBe(false);
    useAuthStore.setState({ session: { sessionId: 's', userId: 'u', expiresAt: now + 1000 } });
    expect(localSessionNeedsRenewal(now)).toBe(false);
  });
});
