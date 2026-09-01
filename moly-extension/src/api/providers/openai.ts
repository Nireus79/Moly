/**
 * OpenAI LLM Provider
 * Uses official openai SDK
 */

import OpenAI from 'openai';
import { BaseLLMProvider, type MessageSuggestion } from '@/api/providers';

export class OpenAIProvider extends BaseLLMProvider {
  type: 'openai' = 'openai';
  name = 'OpenAI';
  models: string[] = [];

  private apiKey: string;
  private model: string;
  private client: OpenAI | null = null;

  constructor(apiKey: string, model: string = '') {
    super();
    this.apiKey = apiKey;
    this.model = model || 'gpt-4-turbo';
  }

  private getClient(): OpenAI {
    if (!this.client) {
      this.client = new OpenAI({
        apiKey: this.apiKey.trim(),
        dangerouslyAllowBrowser: true,
      });
    }
    return this.client;
  }

  /**
   * Discover available OpenAI models dynamically
   */
  async discoverModels(): Promise<string[]> {
    try {
      const trimmedKey = this.apiKey.trim();
      console.log('[OpenAI] Starting model discovery...');
      console.log('[OpenAI] API Key format:', trimmedKey.substring(0, 10) + '...' + trimmedKey.slice(-4));
      console.log('[OpenAI] API Key length:', trimmedKey.length, 'chars');

      if (!trimmedKey.startsWith('sk-')) {
        console.warn('[OpenAI] Warning: API key does not start with "sk-", may be invalid');
      }

      const client = this.getClient();

      try {
        console.log('[OpenAI] Attempting to list models via SDK...');
        const response = await client.models.list();
        console.log('[OpenAI] Models response received');

        const gptModels = response.data
          .filter((m) => m.id.includes('gpt') && !m.id.startsWith('text-'))
          .map((m) => m.id)
          .sort((a, b) => b.localeCompare(a)); // Newest first

        console.log(`[OpenAI] Discovered ${gptModels.length} models:`, gptModels);
        this.models = gptModels;
        return gptModels;
      } catch (modelError) {
        const errorMsg = modelError instanceof Error ? modelError.message : String(modelError);
        const errorStatus = (modelError as any)?.status;
        console.error('[OpenAI] SDK models.list() failed');
        console.error('  Message:', errorMsg);
        console.error('  Status:', errorStatus);
        console.error('  Type:', modelError?.constructor?.name);

        // Try direct fetch as fallback to diagnose the issue
        console.log('[OpenAI] Attempting direct fetch to diagnose...');
        try {
          const fetchResponse = await fetch('https://api.openai.com/v1/models', {
            method: 'GET',
            headers: {
              'Authorization': `Bearer ${trimmedKey}`,
            },
          });
          console.log('[OpenAI] Direct fetch status:', fetchResponse.status);
          const responseText = await fetchResponse.text();
          console.log('[OpenAI] Direct fetch response:', responseText.substring(0, 200));
        } catch (fetchError) {
          console.error('[OpenAI] Direct fetch also failed:', fetchError);
        }

        throw modelError;
      }
    } catch (error) {
      console.error('[OpenAI] Model discovery failed:', error instanceof Error ? error.message : error);
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
      throw new Error('OpenAI API key not configured');
    }

    const prompt = this.buildPrompt(userMessage, context, communicationContext);

    try {
      const response = await this.getClient().chat.completions.create({
        model: this.model,
        max_tokens: 1024,
        messages: [{ role: 'user', content: prompt }],
      });

      const text = response.choices[0]?.message?.content || '';
      return this.parseMessageSuggestions(text);
    } catch (error) {
      console.error('[OpenAI] Error generating suggestions:', error);
      throw error;
    }
  }

  async validate(): Promise<boolean> {
    try {
      if (!this.isConfigured) return false;

      await this.getClient().chat.completions.create({
        model: this.model,
        max_tokens: 10,
        messages: [{ role: 'user', content: 'ok' }],
      });

      console.log('[OpenAI] API validation successful');
      return true;
    } catch (error) {
      console.error('[OpenAI] API validation failed:', error);
      return false;
    }
  }
}
