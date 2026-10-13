import { describe, it, expect } from 'vitest';
import { withoutApiKeys } from '../settingsStore';

const base: any = {
  activeProvider: 'claude',
  providers: {
    claude: { type: 'claude', model: 'm', enabled: true, apiKey: 'sk-ant-secret' },
    openai: { type: 'openai', model: 'g', enabled: false },
    ollama: { type: 'ollama', model: 'mistral', baseUrl: 'http://127.0.0.1:11434', enabled: true },
  },
  defaultContext: 'dating',
  chatMode: 'direct',
  isConfigured: true,
};

describe('withoutApiKeys', () => {
  it('removes a key saved by an older version and keeps everything else', () => {
    const r = withoutApiKeys(base);
    expect(r.removed).toBe(true);
    expect(JSON.stringify(r.settings)).not.toContain('sk-ant-secret');
    expect(r.settings.providers.claude.model).toBe('m');
    expect(r.settings.providers.ollama.baseUrl).toBe('http://127.0.0.1:11434');
    expect(r.settings.chatMode).toBe('direct');
  });

  it('reports nothing removed when there was no key, and does not touch the input', () => {
    const clean = withoutApiKeys(base).settings;
    expect(withoutApiKeys(clean).removed).toBe(false);
    expect(base.providers.claude.apiKey).toBe('sk-ant-secret');
  });
});
