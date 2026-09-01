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
        apiKey: this.apiKey,
        dangerouslyAllowBrowser: true,
      });
    }
    return this.client;
  }

  /**
   * Discover available Claude models dynamically
   */
  async discoverModels(): Promise<string[]> {
    try {
      console.log('[Claude] Starting model discovery with official SDK...');
      const client = this.getClient();
      console.log('[Claude] Client created, calling models.list()...');

      const response = await client.models.list();
      console.log('[Claude] Models response received:', response);

      const claudeModels = response.data
        .filter((m) => m.id.startsWith('claude'))
        .map((m) => m.id)
        .sort((a, b) => b.localeCompare(a)); // Newest first

      console.log(`[Claude] Discovered ${claudeModels.length} models:`, claudeModels);
      this.models = claudeModels;
      return claudeModels;
    } catch (error) {
      console.error('[Claude] Model discovery failed with error:');
      console.error('  Error type:', error?.constructor?.name);
      console.error('  Error message:', error instanceof Error ? error.message : String(error));
      console.error('  Full error:', error);
      throw error; // Don't hide the error - let it propagate
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
