/**
 * Export and import of the person's data (Settings, "Your data"). The file is made and read by the Moly app; the extension only
 * moves it between the app and a file on disk.
 */

import { BACKEND_CONFIG } from './backendConfig';
import { getSessionToken } from '@/stores/authStore';

export interface TransferResult {
  ok: boolean;
  message: string;
}

function auth(): Record<string, string> | null {
  const token = getSessionToken();
  return token ? { Authorization: `Bearer ${token}` } : null;
}

/** Fetches the export as text, or tells why not. */
export async function fetchExport(): Promise<{ ok: true; text: string } | { ok: false; message: string }> {
  const headers = auth();
  if (!headers) return { ok: false, message: 'You are not signed in.' };
  try {
    const response = await fetch(`${BACKEND_CONFIG.baseUrl}/api/user/export`, { headers });
    if (!response.ok) return { ok: false, message: `The export failed (status ${response.status}).` };
    return { ok: true, text: await response.text() };
  } catch {
    return { ok: false, message: 'The Moly app is not reachable.' };
  }
}

/** Sends the text of an export file to the app. */
export async function importExport(fileText: string): Promise<TransferResult> {
  let parsed: unknown;
  try {
    parsed = JSON.parse(fileText);
  } catch {
    return { ok: false, message: 'This file is not a Moly export (it is not readable as JSON).' };
  }
  if (!parsed || typeof parsed !== 'object' || (parsed as { format?: string }).format !== 'moly-export-1') {
    return { ok: false, message: 'This file is not a Moly export.' };
  }
  const headers = auth();
  if (!headers) return { ok: false, message: 'You are not signed in.' };
  try {
    const response = await fetch(`${BACKEND_CONFIG.baseUrl}/api/user/import`, {
      method: 'POST',
      headers: { ...headers, 'Content-Type': 'application/json' },
      body: fileText,
    });
    const data = await response.json().catch(() => ({}));
    if (response.status === 409) {
      return { ok: false, message: 'This profile already has data. An import restores into an empty profile: delete the profile first (below), then import.' };
    }
    if (!response.ok) return { ok: false, message: data?.error || `The import failed (status ${response.status}). Nothing was changed.` };
    return { ok: true, message: `Restored ${data.rows ?? 0} items. Close and reopen the Moly panel to see them.` };
  } catch {
    return { ok: false, message: 'The Moly app is not reachable.' };
  }
}
