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
      this.client = new OpenAI({ apiKey: this.apiKey });
    }
    return this.client;
  }

  /**
   * Discover available OpenAI models dynamically
   */
  async discoverModels(): Promise<string[]> {
    try {
      console.log('[OpenAI] Starting model discovery...');
      const client = this.getClient();

      const response = await client.models.list();
      const gptModels = response.data
        .filter((m) => m.id.includes('gpt') && !m.id.startsWith('text-'))
        .map((m) => m.id)
        .sort((a, b) => b.localeCompare(a)); // Newest first

      console.log(`[OpenAI] Discovered ${gptModels.length} models:`, gptModels);
      this.models = gptModels;
      return gptModels;
    } catch (error) {
      console.warn('[OpenAI] Model discovery failed, using defaults:', error);
      this.models = ['gpt-4-turbo', 'gpt-4', 'gpt-3.5-turbo'];
      return this.models;
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
