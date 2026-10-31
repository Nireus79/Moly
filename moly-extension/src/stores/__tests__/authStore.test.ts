import { describe, it, expect, beforeEach } from 'vitest';
import { useAuthStore, getStoredSession, getSessionToken, type Session } from '@/stores/authStore';

const account: Session = { sessionId: 'acc-token', userId: 'user_1', expiresAt: Date.now() + 3600_000 };
const local: Session = { sessionId: 'loc-token', userId: 'user_local', expiresAt: Date.now() + 3600_000, local: true };

describe('where a session is kept', () => {
  beforeEach(() => {
    localStorage.clear();
    sessionStorage.clear();
    useAuthStore.setState({ session: null });
  });

  it('keeps an account session only for the open panel: nothing in localStorage, which outlives it', () => {
    useAuthStore.getState().setSession(account);
    expect(sessionStorage.getItem('moly_session')).toContain('acc-token');
    expect(localStorage.getItem('moly_session')).toBeNull();
    expect(getSessionToken()).toBe('acc-token');
  });

  it('after the panel is closed (sessionStorage gone) an account session is not found: the login page comes first', () => {
    useAuthStore.getState().setSession(account);
    sessionStorage.clear(); // what the browser does when the panel closes
    expect(getStoredSession()).toBeNull();
    useAuthStore.setState({ session: null });
    useAuthStore.getState().initFromStorage();
    expect(useAuthStore.getState().session).toBeNull();
  });

  it('on a computer a passwordless local session left by an older version is dropped: the login comes first', () => {
    useAuthStore.getState().setSession(local);
    sessionStorage.clear();
    expect(getStoredSession()).toBeNull();
    expect(localStorage.getItem('moly_session')).toBeNull();
  });

  it('keeps the local profile of the phone app (one owner, no password) in localStorage, shared by all pages', () => {
    (window as any).molyPhone = { readClipboard: async () => '', takeSharedText: async () => '' };
    useAuthStore.getState().setSession(local);
    expect(localStorage.getItem('moly_session')).toContain('loc-token');
    expect(sessionStorage.getItem('moly_session')).toBeNull();
    sessionStorage.clear();
    expect(getStoredSession()?.userId).toBe('user_local');
    delete (window as any).molyPhone;
  });

  it('drops an account session that an older version left in localStorage', () => {
    localStorage.setItem('moly_session', JSON.stringify(account));
    expect(getStoredSession()).toBeNull();
    expect(localStorage.getItem('moly_session')).toBeNull();
  });

  it('ignores an expired session and removes it', () => {
    sessionStorage.setItem('moly_session', JSON.stringify({ ...account, expiresAt: Date.now() - 1 }));
    expect(getStoredSession()).toBeNull();
    expect(sessionStorage.getItem('moly_session')).toBeNull();
  });

  it('logging out removes the session everywhere', () => {
    useAuthStore.getState().setSession(account);
    useAuthStore.getState().setSession(null);
    expect(sessionStorage.getItem('moly_session')).toBeNull();
    expect(localStorage.getItem('moly_session')).toBeNull();
  });

  // Found live (2026-10-14): the profile was deleted in the settings page; the open panel kept its session.
  it('forgets the local session when another page removes it (a profile deleted, a sign-out)', () => {
    useAuthStore.getState().setSession(local);
    localStorage.clear(); // what the settings page does
    window.dispatchEvent(Object.assign(new Event('storage'), { key: null, storageArea: localStorage }));
    expect(useAuthStore.getState().session).toBeNull();
  });

  it('keeps the session when the change is about something else or the session is still there', () => {
    useAuthStore.getState().setSession(local);
    window.dispatchEvent(Object.assign(new Event('storage'), { key: 'other', storageArea: localStorage }));
    window.dispatchEvent(Object.assign(new Event('storage'), { key: 'moly_session', storageArea: localStorage }));
    expect(useAuthStore.getState().session?.sessionId).toBe('loc-token');
  });
});


describe('one sign-in for every page of the extension', () => {
  const held: Record<string, unknown> = {};
  const g = globalThis as any;
  const saved = g.chrome;
  beforeEach(() => {
    localStorage.clear();
    sessionStorage.clear();
    for (const k of Object.keys(held)) delete held[k];
    useAuthStore.setState({ session: null });
    g.chrome = { ...(saved ?? {}), storage: { ...(saved?.storage ?? {}), session: {
      set: async (o: Record<string, unknown>) => { Object.assign(held, o); },
      get: async (k: string) => ({ [k]: held[k] }),
      remove: async (k: string) => { delete held[k]; },
    } } };
  });

  it('a new tab (Settings) takes the session the panel holds', async () => {
    useAuthStore.getState().setSession(account);
    await Promise.resolve();
    sessionStorage.clear(); // a new tab has its own, empty sessionStorage
    const { restoreSharedSession } = await import('@/stores/authStore');
    expect((await restoreSharedSession())?.sessionId).toBe('acc-token');
    expect(getStoredSession()?.sessionId).toBe('acc-token');
  });

  it('after sign-out the other pages find nothing', async () => {
    useAuthStore.getState().setSession(account);
    useAuthStore.getState().setSession(null);
    await Promise.resolve();
    sessionStorage.clear();
    const { restoreSharedSession } = await import('@/stores/authStore');
    expect(await restoreSharedSession()).toBeNull();
  });

  it('a passwordless local session is never taken from the shared area', async () => {
    held['moly_session'] = local;
    const { restoreSharedSession } = await import('@/stores/authStore');
    expect(await restoreSharedSession()).toBeNull();
  });
});
