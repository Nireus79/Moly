/**
 * Profiles with a password, next to the local profile of an installed Moly. The local profile stays the default and needs nothing; a person
 * who shares the computer can make another profile in Settings (a name and a password). A forgotten password is reset with the recovery
 * code shown once when the profile was made (the app has no server to mail anything).
 */

import { getBackendManager } from './backendManager';
import { getSessionToken, type Session } from '@/stores/authStore';

const LAST_KEY = 'moly_last_profile';

/** 'local' for the local profile, an account name, or '' when the person asked to choose again. */
export function lastProfile(): string {
  try {
    return localStorage.getItem(LAST_KEY) ?? '';
  } catch {
    return '';
  }
}

export function rememberProfile(who: string): void {
  try {
    localStorage.setItem(LAST_KEY, who);
  } catch {
    /* storage unavailable: the chooser shows next time */
  }
}

/** A profile made or reset: its session and the recovery code, which is shown once. */
export interface AccountResult {
  session: Session;
  recoveryCode: string;
}

function base(): string {
  return getBackendManager().getBackendUrl();
}

async function post(path: string, body: unknown): Promise<any> {
  const response = await fetch(`${base()}${path}`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
  const data = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(typeof data.error === 'string' && data.error ? data.error : `The request failed (${response.status}).`);
  if (!data.token || !data.userId) throw new Error('The app did not return a session.');
  return data;
}

function sessionOf(data: any): Session {
  return { sessionId: data.token, userId: data.userId, expiresAt: Date.now() + (data.expiresIn ?? 86400) * 1000 };
}

/** True when the app is older than 0.1.2: it has no list of profiles and cannot make a profile without an email address or a recovery code. */
export async function appLacksProfiles(): Promise<boolean> {
  try {
    return (await fetch(`${base()}/api/auth/profiles`)).status === 404;
  } catch {
    return false;
  }
}

/** The names of the profiles that have a password (the local profile is not listed). An empty list when the app cannot say. */
export async function listProfiles(): Promise<string[]> {
  try {
    const response = await fetch(`${base()}/api/auth/profiles`);
    if (!response.ok) return [];
    const data = await response.json();
    return Array.isArray(data.profiles) ? data.profiles.filter((n: unknown): n is string => typeof n === 'string') : [];
  } catch {
    return [];
  }
}

export async function createProfile(name: string, password: string): Promise<AccountResult> {
  const data = await post('/api/auth/register', { name: name.trim(), password });
  return { session: sessionOf(data), recoveryCode: String(data.recoveryCode ?? '') };
}

export async function signIn(name: string, password: string): Promise<Session> {
  return sessionOf(await post('/api/auth/login', { emailOrUsername: name.trim(), password }));
}

export async function resetPassword(name: string, recoveryCode: string, newPassword: string): Promise<AccountResult> {
  const data = await post('/api/auth/reset', { emailOrUsername: name.trim(), recoveryCode, newPassword });
  return { session: sessionOf(data), recoveryCode: String(data.recoveryCode ?? '') };
}

/** Replaces the password of the signed-in profile. The current password must be given; the other sessions of the profile end. */
export async function changePassword(currentPassword: string, newPassword: string): Promise<void> {
  const response = await fetch(`${base()}/api/auth/password`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${getSessionToken() ?? ''}` },
    body: JSON.stringify({ currentPassword, newPassword }),
  });
  const data = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(typeof data.error === 'string' && data.error ? data.error : `The password could not be changed (${response.status}).`);
}
