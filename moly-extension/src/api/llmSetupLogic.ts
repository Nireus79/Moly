/** Rules of the AI provider form, kept apart from the screen so they can be tested. */

import type { ProviderId, SaveBody } from './llmSettingsAPI';

export const CLAUDE_MODELS = [
  { id: 'claude-sonnet-5-5', label: 'Claude Sonnet 5.5 (balanced)' },
  { id: 'claude-opus-5-5', label: 'Claude Opus 5.5 (most capable)' },
  { id: 'claude-haiku-5-5', label: 'Claude Haiku 5.5 (fastest, lowest cost)' },
];

export const OPENAI_MODELS = [{ id: 'gpt-4-turbo', label: 'GPT-4 Turbo' }];

export const DEFAULT_OLLAMA_ENDPOINT = 'http://127.0.0.1:11434';
export const MIN_PASSPHRASE = 8;

export const PROVIDER_NAMES: Record<ProviderId, string> = { ollama: 'Ollama (on this computer)', claude: 'Claude (Anthropic)', openai: 'OpenAI' };
export const PROVIDER_COMPANY: Record<'claude' | 'openai', string> = { claude: 'Anthropic', openai: 'OpenAI' };

export interface SetupForm {
  provider: ProviderId;
  apiKey: string;
  model: string;
  endpoint: string;
  usePassphrase: boolean; // on by default
  passphrase: string;
  passphrase2: string;
  consent: boolean;
  hasStoredKey: boolean; // a key is already saved for this provider
  storedKeySealed: boolean;
}

export function emptyForm(provider: ProviderId = 'ollama'): SetupForm {
  return {
    provider,
    apiKey: '',
    model: '',
    endpoint: DEFAULT_OLLAMA_ENDPOINT,
    usePassphrase: true,
    passphrase: '',
    passphrase2: '',
    consent: false,
    hasStoredKey: false,
    storedKeySealed: false,
  };
}

export const isCloud = (p: ProviderId): p is 'claude' | 'openai' => p === 'claude' || p === 'openai';

export function passphraseStrength(p: string): 'none' | 'short' | 'ok' | 'good' {
  if (!p) return 'none';
  if (p.length < MIN_PASSPHRASE) return 'short';
  return p.length >= 14 ? 'good' : 'ok';
}

/** Problems that stop the form from being saved; an empty list means it can be saved. */
export function validateForm(f: SetupForm): string[] {
  const problems: string[] = [];
  if (isCloud(f.provider)) {
    const newKey = f.apiKey.trim() !== '';
    if (!newKey && !f.hasStoredKey) problems.push('Enter your API key.');
    if (/\s/.test(f.apiKey.trim())) problems.push('The API key must not contain spaces.');
    if (!f.consent) problems.push(`Confirm that your messages will be sent to ${PROVIDER_COMPANY[f.provider]}.`);
    const needsPassphrase = f.usePassphrase && (newKey || (f.hasStoredKey && !f.storedKeySealed));
    if (needsPassphrase) {
      if (f.passphrase.length < MIN_PASSPHRASE) problems.push(`The passphrase needs at least ${MIN_PASSPHRASE} characters.`);
      else if (f.passphrase !== f.passphrase2) problems.push('The two passphrases are different.');
    }
  } else {
    try {
      const u = new URL(f.endpoint.trim());
      if (u.protocol !== 'http:' && u.protocol !== 'https:') problems.push('The Ollama address must start with http:// or https://.');
    } catch {
      problems.push('Enter the Ollama address, for example http://127.0.0.1:11434.');
    }
  }
  return problems;
}

/** The request body for saving. The passphrase is sent only when a key is being saved or sealed. */
export function buildSaveBody(f: SetupForm): SaveBody {
  const body: SaveBody = { provider: f.provider, model: f.model.trim() };
  if (isCloud(f.provider)) {
    const newKey = f.apiKey.trim();
    if (newKey) body.apiKey = newKey;
    if (f.usePassphrase && (newKey || (f.hasStoredKey && !f.storedKeySealed))) body.passphrase = f.passphrase;
  } else {
    body.ollamaEndpoint = f.endpoint.trim();
  }
  return body;
}

/** The body of the connection test: never contains the passphrase. */
export function buildTestBody(f: SetupForm): Omit<SaveBody, 'passphrase'> {
  const { passphrase: _unused, ...rest } = buildSaveBody(f);
  return rest;
}
