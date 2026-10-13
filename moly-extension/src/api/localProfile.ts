/**
 * The local profile of an installed Moly. The backend (production mode) hands out a session for one profile without email
 * or password; the API token that every request carries already keeps other programs out.
 */

import { getBackendManager } from './backendManager';
import type { Session } from '@/stores/authStore';

export async function startLocalSession(): Promise<Session | null> {
  try {
    const response = await fetch(`${getBackendManager().getBackendUrl()}/api/auth/local`, { method: 'POST' });
    if (!response.ok) return null;
    const data = await response.json();
    if (!data.token || !data.userId) return null;
    return { sessionId: data.token, userId: data.userId, expiresAt: Date.now() + (data.expiresIn ?? 86400) * 1000, local: true };
  } catch {
    return null;
  }
}
