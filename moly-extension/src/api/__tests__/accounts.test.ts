import { describe, it, expect, vi, afterEach } from 'vitest';

vi.mock('../backendManager', () => ({ getBackendManager: () => ({ getBackendUrl: () => 'http://app' }) }));
import { appLacksProfiles, changePassword, createProfile, listProfiles, resetPassword, signIn, lastProfile, rememberProfile } from '../accounts';

const realFetch = globalThis.fetch;
afterEach(() => {
  globalThis.fetch = realFetch;
  localStorage.clear();
});
const reply = (body: unknown, ok = true, status = 200) => vi.fn(async () => ({ ok, status, json: async () => body })) as unknown as typeof fetch;

describe('accounts', () => {
  it('lists the names, and an empty list when the app cannot say', async () => {
    globalThis.fetch = reply({ profiles: ['Anna', 7, 'Ben'] });
    expect(await listProfiles()).toEqual(['Anna', 'Ben']);
    globalThis.fetch = reply({}, false, 500);
    expect(await listProfiles()).toEqual([]);
    globalThis.fetch = vi.fn(async () => { throw new Error('down'); }) as unknown as typeof fetch;
    expect(await listProfiles()).toEqual([]);
  });

  it('a new profile sends a name and a password and keeps the recovery code apart from the session', async () => {
    const f = reply({ userId: 'u1', token: 't1', expiresIn: 86400, recoveryCode: 'AAAA-BBBB-CCCC-DDDD' });
    globalThis.fetch = f;
    const r = await createProfile('  Anna ', 'longenough1');
    expect(JSON.parse((f as any).mock.calls[0][1].body)).toEqual({ name: 'Anna', password: 'longenough1' });
    expect(r.recoveryCode).toBe('AAAA-BBBB-CCCC-DDDD');
    expect(r.session.sessionId).toBe('t1');
    expect(r.session.local).toBeUndefined(); // an account session is not the local one
  });

  it('shows the app\'s own reason when it refuses', async () => {
    globalThis.fetch = reply({ error: 'That name is already used by another profile' }, false, 409);
    await expect(createProfile('Anna', 'longenough1')).rejects.toThrow('already used');
    globalThis.fetch = reply({ error: 'Invalid email, username, or password' }, false, 401);
    await expect(signIn('Anna', 'wrong')).rejects.toThrow('Invalid');
  });

  it('a reset sends the code and returns the new one', async () => {
    const f = reply({ userId: 'u1', token: 't2', expiresIn: 86400, recoveryCode: 'NEW1-NEW2-NEW3-NEW4' });
    globalThis.fetch = f;
    const r = await resetPassword('Anna', 'aaaa bbbb', 'brandnew123');
    expect(JSON.parse((f as any).mock.calls[0][1].body)).toEqual({ emailOrUsername: 'Anna', recoveryCode: 'aaaa bbbb', newPassword: 'brandnew123' });
    expect(r.recoveryCode).toBe('NEW1-NEW2-NEW3-NEW4');
  });

  it('knows an app older than 0.1.2 by the missing list of profiles', async () => {
    globalThis.fetch = reply({}, false, 404);
    expect(await appLacksProfiles()).toBe(true);
    globalThis.fetch = reply({ profiles: [] });
    expect(await appLacksProfiles()).toBe(false);
    globalThis.fetch = vi.fn(async () => { throw new Error('down'); }) as unknown as typeof fetch;
    expect(await appLacksProfiles()).toBe(false);
  });

  it('remembers who used Moly last', () => {
    expect(lastProfile()).toBe('');
    rememberProfile('local');
    expect(lastProfile()).toBe('local');
  });

  it('changing the password sends the session and both passwords, and says why it was refused', async () => {
    sessionStorage.setItem('moly_session', JSON.stringify({ sessionId: 'tok-9', userId: 'u1', expiresAt: Date.now() + 60000 }));
    const f = reply({ success: true });
    globalThis.fetch = f;
    await changePassword('oldoldold1', 'brandnew123');
    const [url, init] = (f as any).mock.calls[0];
    expect(url).toBe('http://app/api/auth/password');
    expect(init.headers.Authorization).toBe('Bearer tok-9');
    expect(JSON.parse(init.body)).toEqual({ currentPassword: 'oldoldold1', newPassword: 'brandnew123' });
    globalThis.fetch = reply({ error: 'The current password is not right' }, false, 401);
    await expect(changePassword('x', 'brandnew123')).rejects.toThrow('The current password is not right');
    globalThis.fetch = reply({ error: 'Too many wrong tries. Wait 5 minutes and try again.' }, false, 429);
    await expect(changePassword('x', 'brandnew123')).rejects.toThrow('Wait 5 minutes');
    sessionStorage.clear();
  });
});
