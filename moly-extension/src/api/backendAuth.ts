/**
 * Backend connection for extension pages (sidebar, popup, settings).
 *
 * The service worker starts the backend through the native host and publishes {url, token} in chrome.storage.session.
 * installBackendAuth() wraps window.fetch once: every request to that backend gets the X-Moly-Token header, and a 401
 * triggers one refresh of the connection (the backend may have restarted with a new token) and one retry. Requests to
 * any other address are left alone. Without a connection (a development backend started by hand) nothing is changed.
 */

import { BACKEND_CONFIG } from './backendConfig';

export const CONNECTION_KEY = 'molyConnection';
export const TOKEN_HEADER = 'X-Moly-Token';

export interface BackendConnection {
  url: string;
  token: string;
  version: string;
}

export interface EnsureResult {
  ok: boolean;
  url?: string;
  version?: string;
  error?: string;
}

let current: BackendConnection | null = null;
let ready: Promise<void> | null = null;
let installed = false;

function sessionStorage(): chrome.storage.StorageArea | undefined {
  return typeof chrome !== 'undefined' ? chrome.storage?.session : undefined;
}

function adopt(conn: BackendConnection | null): void {
  current = conn;
  if (conn?.url) BACKEND_CONFIG.baseUrl = conn.url; // code that builds URLs from BACKEND_CONFIG follows the real port
}

async function loadConnection(): Promise<void> {
  try {
    const stored = await sessionStorage()?.get(CONNECTION_KEY);
    adopt((stored && stored[CONNECTION_KEY]) || null);
  } catch {
    adopt(null);
  }
}

const LOOPBACK_HOSTS = new Set(['127.0.0.1', 'localhost', '[::1]']);

/** True for the backend's own address; localhost and 127.0.0.1 with the same port are the same backend. */
export function isBackendUrl(candidate: string, conn: BackendConnection): boolean {
  try {
    const a = new URL(candidate);
    const b = new URL(conn.url);
    if (a.protocol !== b.protocol || a.port !== b.port) return false;
    if (a.hostname === b.hostname) return true;
    return LOOPBACK_HOSTS.has(a.hostname) && LOOPBACK_HOSTS.has(b.hostname);
  } catch {
    return false;
  }
}

export function getConnection(): BackendConnection | null {
  return current;
}

/** Asks the service worker to start the backend (if needed). Resolves with the result; never throws. */
export async function ensureBackendConnection(): Promise<EnsureResult> {
  try {
    const result: EnsureResult | undefined = await chrome.runtime.sendMessage({ type: 'MOLY_ENSURE_BACKEND' });
    await loadConnection();
    return result ?? { ok: false, error: 'No answer from the extension background.' };
  } catch (error) {
    return { ok: false, error: error instanceof Error ? error.message : String(error) };
  }
}

function urlOf(input: RequestInfo | URL): string {
  if (typeof input === 'string') return input;
  if (input instanceof URL) return input.href;
  return input.url;
}

function withToken(input: RequestInfo | URL, init: RequestInit | undefined, token: string): [RequestInfo | URL, RequestInit] {
  const headers = new Headers(init?.headers ?? (input instanceof Request ? input.headers : undefined));
  headers.set(TOKEN_HEADER, token);
  return [input, { ...init, headers }];
}

/** Wraps window.fetch (once) and loads the stored connection. Call before the first render. */
export function installBackendAuth(): Promise<void> {
  if (installed) return ready ?? Promise.resolve();
  installed = true;

  ready = loadConnection();
  sessionStorage()?.onChanged?.addListener?.((changes) => {
    if (changes[CONNECTION_KEY]) adopt(changes[CONNECTION_KEY].newValue || null);
  });

  const original = window.fetch.bind(window);
  window.fetch = async (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
    await ready;
    const conn = current;
    if (!conn || !isBackendUrl(urlOf(input), conn)) return original(input, init);

    const response = await original(...withToken(input, init, conn.token));
    if (response.status !== 401) return response;

    // 401 is also the answer to an expired user session; only a changed token is worth a retry.
    const before = conn.token;
    await ensureBackendConnection();
    const fresh = current;
    if (!fresh || fresh.token === before) return response;
    return original(...withToken(input, init, fresh.token));
  };
  return ready;
}
