import { describe, it, expect } from 'vitest';
import { emptyForm, validateForm, buildSaveBody, buildTestBody, passphraseStrength, type SetupForm } from '../llmSetupLogic';

const claude = (over: Partial<SetupForm> = {}): SetupForm => ({
  ...emptyForm('claude'),
  apiKey: 'sk-ant-abc',
  model: 'a-model-the-provider-listed',
  consent: true,
  passphrase: 'correct horse battery',
  passphrase2: 'correct horse battery',
  ...over,
});

describe('llmSetupLogic', () => {
  // The model is chosen from what the provider lists; a typed name can only be wrong.
  it('a cloud form without a chosen model is not valid', () => {
    expect(validateForm(claude({ model: '' })).join(' ')).toContain('Choose a model');
  });

  it('the passphrase is on by default', () => {
    expect(emptyForm('claude').usePassphrase).toBe(true);
  });

  it('a complete cloud form is valid and sends the key and the passphrase', () => {
    const f = claude();
    expect(validateForm(f)).toEqual([]);
    const body = buildSaveBody(f);
    expect(body).toMatchObject({ provider: 'claude', apiKey: 'sk-ant-abc', passphrase: 'correct horse battery' });
  });

  it('refuses a missing key, a missing consent, a short or mismatched passphrase', () => {
    expect(validateForm(claude({ apiKey: '' }))).toContain('Enter your API key.');
    expect(validateForm(claude({ consent: false })).join(' ')).toContain('sent to Anthropic');
    expect(validateForm(claude({ passphrase: 'short', passphrase2: 'short' })).join(' ')).toContain('at least 8');
    expect(validateForm(claude({ passphrase2: 'something else entirely' }))).toContain('The two passphrases are different.');
    expect(validateForm(claude({ provider: 'openai' })).join(' ')).not.toContain('Anthropic');
  });

  it('opting out of the passphrase needs none and sends none', () => {
    const f = claude({ usePassphrase: false, passphrase: '', passphrase2: '' });
    expect(validateForm(f)).toEqual([]);
    expect(buildSaveBody(f).passphrase).toBeUndefined();
  });

  it('keeping a stored sealed key needs no new key and no passphrase', () => {
    const f = claude({ apiKey: '', hasStoredKey: true, storedKeySealed: true, passphrase: '', passphrase2: '' });
    expect(validateForm(f)).toEqual([]);
    const body = buildSaveBody(f);
    expect(body.apiKey).toBeUndefined();
    expect(body.passphrase).toBeUndefined();
  });

  it('a stored plain key can be sealed later with a passphrase', () => {
    const f = claude({ apiKey: '', hasStoredKey: true, storedKeySealed: false });
    expect(validateForm(f)).toEqual([]);
    expect(buildSaveBody(f)).toMatchObject({ passphrase: 'correct horse battery' });
    expect(buildSaveBody(f).apiKey).toBeUndefined();
  });

  it('the connection test never carries the passphrase', () => {
    expect(JSON.stringify(buildTestBody(claude()))).not.toContain('correct horse');
    expect(buildTestBody(claude()).apiKey).toBe('sk-ant-abc');
  });

  it('Ollama needs an http address and no key or consent', () => {
    expect(validateForm(emptyForm('ollama'))).toEqual([]);
    expect(validateForm({ ...emptyForm('ollama'), endpoint: 'ftp://x' }).length).toBe(1);
    expect(validateForm({ ...emptyForm('ollama'), endpoint: 'nonsense' }).length).toBe(1);
    expect(buildSaveBody(emptyForm('ollama'))).toMatchObject({ provider: 'ollama', ollamaEndpoint: 'http://127.0.0.1:11434' });
    expect(buildSaveBody(emptyForm('ollama')).apiKey).toBeUndefined();
  });

  it('rates passphrases', () => {
    expect(passphraseStrength('')).toBe('none');
    expect(passphraseStrength('abc')).toBe('short');
    expect(passphraseStrength('abcdefgh')).toBe('ok');
    expect(passphraseStrength('a long enough passphrase')).toBe('good');
  });
});
