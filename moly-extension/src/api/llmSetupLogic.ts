/** Rules of the AI provider form, kept apart from the screen so they can be tested. */

import type { ProviderId, SaveBody } from './llmSettingsAPI';

export const DEFAULT_OLLAMA_ENDPOINT = 'http://127.0.0.1:11434';
export const MIN_PASSPHRASE = 8;

export const PROVIDER_NAMES: Record<ProviderId, string> = { ollama: 'Ollama (on this computer)', claude: 'Claude (Anthropic)', openai: 'OpenAI' };
export const PROVIDER_COMPANY: Record<'claude' | 'openai', string> = { claude: 'Anthropic', openai: 'OpenAI' };

/** Where a person creates an API key at each provider, and the steps to follow there. */
export const KEY_HELP: Record<'claude' | 'openai', { url: string; button: string; steps: string[] }> = {
  claude: {
    url: 'https://console.anthropic.com/settings/keys',
    button: 'Get a Claude key',
    steps: [
      'Sign in to the Anthropic Console (or create an account).',
      'Add a little credit under Billing; a key does not work without it.',
      'Open API keys and press Create key. Name it Moly.',
      'Copy the key at once: the page shows it only one time.',
    ],
  },
  openai: {
    url: 'https://platform.openai.com/api-keys',
    button: 'Get an OpenAI key',
    steps: [
      'Sign in to the OpenAI Platform (or create an account).',
      'Add a little credit under Billing; a key does not work without it.',
      'Open API keys and press Create new secret key. Name it Moly.',
      'Copy the key at once: the page shows it only one time.',
    ],
  },
};

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
  const body: SaveBody = { provider: f.provider };
  if (isCloud(f.provider)) {
    // no model: Moly chooses the models of a Claude or OpenAI user
    const newKey = f.apiKey.trim();
    if (newKey) body.apiKey = newKey;
    if (f.usePassphrase && (newKey || (f.hasStoredKey && !f.storedKeySealed))) body.passphrase = f.passphrase;
  } else {
    body.model = f.model.trim();
    body.ollamaEndpoint = f.endpoint.trim();
  }
  return body;
}

/** The body of the connection test: never contains the passphrase. */
export function buildTestBody(f: SetupForm): Omit<SaveBody, 'passphrase'> {
  const { passphrase: _unused, ...rest } = buildSaveBody(f);
  return rest;
}
