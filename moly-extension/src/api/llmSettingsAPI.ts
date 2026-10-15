/**
 * Client for the backend's AI provider settings (/api/settings/llm). The settings belong to the signed-in user: every request
 * carries that user's session, and the backend answers only for that user. The backend keeps the provider, the model and the
 * API key in its encrypted database; the key is only ever sent TO the backend and is never returned.
 */

import { getBackendManager } from './backendManager';
import { ensureBackendConnection, reportSessionRejected } from './backendAuth';
import { getSessionToken } from '@/stores/authStore';

export type ProviderId = 'ollama' | 'claude' | 'openai';
export type LLMState = 'ready' | 'locked' | 'unconfigured';

export interface KeyInfo {
  stored: boolean;
  sealed: boolean;
}

/** The models Moly chose for a Claude or OpenAI user: one for the quick checks, one for the replies. */
export interface RoleModels {
  fast: string;
  reply: string;
  /** Moly was tested with both. False: the provider's own suggestion stands in for what is not approved yet. */
  tested: boolean;
}

export interface LLMStatus {
  provider: string;
  model: string;
  /** The models in use, when Moly chose them (Claude and OpenAI). */
  models?: RoleModels;
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

/** The message when a request fails without an answer. The reason is kept in the text: "not reachable" alone says nothing. */
export const unreachable = (reason: string) => `The Moly app is not reachable (${reason}).`;

export const NOT_SIGNED_IN = 'You are not signed in to Moly. Sign in in the side panel, then try again.';

async function send(method: string, path: string, body?: unknown): Promise<Response> {
  const headers: Record<string, string> = {};
  if (body !== undefined) headers['Content-Type'] = 'application/json';
  const session = getSessionToken();
  if (session) headers['Authorization'] = `Bearer ${session}`; // whose settings these are
  return fetch(`${getBackendManager().getBackendUrl()}${path}`, {
    method,
    headers: Object.keys(headers).length ? headers : undefined,
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
    let why = ''; // what the browser said, so that "not reachable" says why
    try {
      response = await send(method, path, body);
    } catch (error) {
      response = undefined;
      why = error instanceof Error ? `${error.name}: ${error.message}` : String(error);
    }
    // The app answered, and says the USER is not signed in (no session, or it ended): that is not a connection problem, and
    // starting the app again would not help.
    if (response && response.status === 401) {
      const body = await response.clone().json().catch(() => ({}));
      if (body?.error === 'Invalid token') {
        if (getSessionToken()) reportSessionRejected(); // the app does not know this session (the profile was deleted): forget it
        return { ok: false, error: NOT_SIGNED_IN };
      }
    }
    if (!response || response.status === 401) {
      const ensured = await ensureBackendConnection();
      if (!ensured.ok) return { ok: false, error: unreachable(why || (response ? `the app answered ${response.status}` : '') || ensured.error || 'no answer') };
      response = await send(method, path, body);
    }
    const data = await response.json().catch(() => ({}));
    if (!response.ok) return { ok: false, error: data?.error || `The Moly app answered ${response.status}.` };
    return { ok: true, data: data as T };
  } catch (error) {
    return { ok: false, error: unreachable(error instanceof Error ? `${error.name}: ${error.message}` : String(error)) };
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

export interface ModelInfo {
  id: string;
  label: string;
  line?: string;
}

export interface ModelList {
  models: ModelInfo[];
  suggested: string;
  /** The models Moly would use for this key. */
  roles?: RoleModels;
  error?: string;
}

/**
 * The models a cloud provider offers to the user's key, asked of the provider by the Moly app. A key typed in the form is
 * used for this one question (and not saved); without one the saved key is used. Nothing is assumed in the extension.
 */
export async function listProviderModels(provider: 'claude' | 'openai', apiKey?: string): Promise<ModelList> {
  const r = await call<ModelList>('POST', '/api/settings/llm/models', { provider, apiKey: apiKey || undefined });
  if (!r.ok || !r.data) return { models: [], suggested: '', error: r.error || 'The models could not be listed.' };
  return { models: r.data.models || [], suggested: r.data.suggested || '', roles: r.data.roles, error: r.data.error };
}

export async function listOllamaModels(endpoint?: string): Promise<{ models: string[]; error?: string }> {
  const query = endpoint ? `?endpoint=${encodeURIComponent(endpoint)}` : '';
  const r = await call<{ models: string[]; error?: string }>('GET', `/api/settings/llm/ollama-models${query}`);
  if (!r.ok || !r.data) return { models: [], error: r.error };
  return { models: r.data.models || [], error: r.data.error };
}
