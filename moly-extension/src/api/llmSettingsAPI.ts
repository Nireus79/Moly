/**
 * Client for the backend's AI provider settings (/api/settings/llm). The backend keeps the provider, the model and the
 * API key in its encrypted database; the key is only ever sent TO the backend and is never returned.
 */

import { getBackendManager } from './backendManager';
import { ensureBackendConnection } from './backendAuth';

export type ProviderId = 'ollama' | 'claude' | 'openai';
export type LLMState = 'ready' | 'locked' | 'unconfigured';

export interface KeyInfo {
  stored: boolean;
  sealed: boolean;
}

export interface LLMStatus {
  provider: string;
  model: string;
  ollamaEndpoint: string;
  state: LLMState;
  error?: string;
  active: string;
  fixedByEnvironment: boolean;
  keys: Record<string, KeyInfo>;
}

export interface SaveBody {
  provider: ProviderId;
  model?: string;
  ollamaEndpoint?: string;
  apiKey?: string;
  passphrase?: string;
}

export interface CallResult<T = LLMStatus> {
  ok: boolean;
  data?: T;
  error?: string;
}

async function send(method: string, path: string, body?: unknown): Promise<Response> {
  return fetch(`${getBackendManager().getBackendUrl()}${path}`, {
    method,
    headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
}

/**
 * One request. A page that has no connection yet (the settings page opened before the sidebar) gets a network error or a
 * 401; it then asks the extension to start the Moly app and tries once more.
 */
async function call<T>(method: string, path: string, body?: unknown): Promise<CallResult<T>> {
  try {
    let response: Response | undefined;
    try {
      response = await send(method, path, body);
    } catch {
      response = undefined;
    }
    if (!response || response.status === 401) {
      const ensured = await ensureBackendConnection();
      if (!ensured.ok) return { ok: false, error: 'The Moly app is not reachable.' };
      response = await send(method, path, body);
    }
    const data = await response.json().catch(() => ({}));
    if (!response.ok) return { ok: false, error: data?.error || `The Moly app answered ${response.status}.` };
    return { ok: true, data: data as T };
  } catch {
    return { ok: false, error: 'The Moly app is not reachable.' };
  }
}

/** The shared request helper (token, reconnect on 401), for the other clients of the Moly app. */
export { call as callBackend };

export const getLLMStatus = () => call<LLMStatus>('GET', '/api/settings/llm');
export const saveLLMSettings = (body: SaveBody) => call<LLMStatus>('PUT', '/api/settings/llm', body);
export const unlockLLM = (passphrase: string) => call<LLMStatus>('POST', '/api/settings/llm/unlock', { passphrase });
export const lockLLM = () => call<LLMStatus>('POST', '/api/settings/llm/lock');
export const deleteLLMKey = (provider: 'claude' | 'openai') => call<LLMStatus>('DELETE', `/api/settings/llm/key?provider=${provider}`);

export async function testLLMSettings(body: Omit<SaveBody, 'passphrase'>): Promise<{ ok: boolean; error?: string }> {
  const r = await call<{ ok: boolean; error?: string }>('POST', '/api/settings/llm/test', body);
  if (!r.ok) return { ok: false, error: r.error };
  return { ok: !!r.data?.ok, error: r.data?.error };
}

export async function listOllamaModels(endpoint?: string): Promise<{ models: string[]; error?: string }> {
  const query = endpoint ? `?endpoint=${encodeURIComponent(endpoint)}` : '';
  const r = await call<{ models: string[]; error?: string }>('GET', `/api/settings/llm/ollama-models${query}`);
  if (!r.ok || !r.data) return { models: [], error: r.error };
  return { models: r.data.models || [], error: r.data.error };
}
