/**
 * Claude LLM Provider
 * Uses official @anthropic-ai/sdk - no manual API version management
 */

import Anthropic from '@anthropic-ai/sdk';
import { BaseLLMProvider, type MessageSuggestion } from '@/api/providers';

export class ClaudeProvider extends BaseLLMProvider {
  type: 'claude' = 'claude';
  name = 'Claude (Anthropic)';
  models: string[] = [];

  private apiKey: string;
  private model: string;
  private client: Anthropic | null = null;

  constructor(apiKey: string, model: string = '') {
    super();
    this.apiKey = apiKey;
    this.model = model || 'claude-3-5-sonnet-20241022';
  }

  private getClient(): Anthropic {
    if (!this.client) {
      this.client = new Anthropic({
        apiKey: this.apiKey.trim(),
        dangerouslyAllowBrowser: true,
      });
    }
    return this.client;
  }

  /**
   * Discover available Claude models dynamically
   * Try model discovery separately from validation to isolate issues
   */
  async discoverModels(): Promise<string[]> {
    try {
      const trimmedKey = this.apiKey.trim();
      console.log('[Claude] Starting model discovery...');
      console.log('[Claude] API Key format:', trimmedKey.substring(0, 10) + '...' + trimmedKey.slice(-4));
      console.log('[Claude] API Key length:', trimmedKey.length, 'chars');

      if (!trimmedKey.startsWith('sk-ant-')) {
        console.warn('[Claude] Warning: API key does not start with "sk-ant-", may be invalid');
      }

      const client = this.getClient();

      try {
        console.log('[Claude] Attempting to list models via SDK...');
        const response = await client.models.list();
        console.log('[Claude] Models response received');

        const claudeModels = response.data
          .filter((m) => m.id.startsWith('claude'))
          .map((m) => m.id)
          .sort((a, b) => b.localeCompare(a)); // Newest first

        console.log(`[Claude] Discovered ${claudeModels.length} models:`, claudeModels);
        this.models = claudeModels;
        return claudeModels;
      } catch (modelError) {
        const errorMsg = modelError instanceof Error ? modelError.message : String(modelError);
        const errorStatus = (modelError as any)?.status;
        console.error('[Claude] SDK models.list() failed');
        console.error('  Message:', errorMsg);
        console.error('  Status:', errorStatus);
        console.error('  Type:', modelError?.constructor?.name);

        // Try direct fetch as fallback to diagnose the issue
        console.log('[Claude] Attempting direct fetch to diagnose...');
        try {
          const fetchResponse = await fetch('https://api.anthropic.com/v1/models', {
            method: 'GET',
            headers: {
              'Authorization': `Bearer ${trimmedKey}`,
              'anthropic-version': '2023-06-01',
            },
          });
          console.log('[Claude] Direct fetch status:', fetchResponse.status);
          const responseText = await fetchResponse.text();
          console.log('[Claude] Direct fetch response:', responseText.substring(0, 200));
        } catch (fetchError) {
          console.error('[Claude] Direct fetch also failed:', fetchError);
        }

        throw modelError;
      }
    } catch (error) {
      console.error('[Claude] Model discovery failed:', error instanceof Error ? error.message : error);
      throw error;
    }
  }

  get isConfigured(): boolean {
    return !!this.apiKey;
  }

  async generateSuggestions(
    userMessage: string,
    context: string,
    communicationContext: 'formal' | 'friendly' | 'dating',
  ): Promise<MessageSuggestion[]> {
    if (!this.isConfigured) {
      throw new Error('Claude API key not configured');
    }

    const prompt = this.buildPrompt(userMessage, context, communicationContext);

    try {
      const response = await this.getClient().messages.create({
        model: this.model,
        max_tokens: 1024,
        messages: [{ role: 'user', content: prompt }],
      });

      return this.parseMessageSuggestions(
        response.content[0].type === 'text' ? response.content[0].text : '',
      );
    } catch (error) {
      console.error('[Claude] Error generating suggestions:', error);
      throw error;
    }
  }

  async validate(): Promise<boolean> {
    try {
      if (!this.isConfigured) return false;

      await this.getClient().messages.create({
        model: this.model,
        max_tokens: 10,
        messages: [{ role: 'user', content: 'ok' }],
      });

      console.log('[Claude] API validation successful');
      return true;
    } catch (error) {
      console.error('[Claude] API validation failed:', error);
      return false;
    }
  }
}
