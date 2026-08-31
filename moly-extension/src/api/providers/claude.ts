/**
 * Claude LLM Provider
 * Integration with Anthropic Claude API with dynamic model discovery
 */

import { apiClient } from '@/api/client';
import { BaseLLMProvider, type MessageSuggestion } from '@/api/providers';

const CLAUDE_API_URL = 'https://api.anthropic.com/v1/messages';
const CLAUDE_MODELS_URL = 'https://api.anthropic.com/v1/models';
const DEFAULT_CLAUDE_MODELS = ['claude-3-5-sonnet-20241022', 'claude-3-opus-20250219', 'claude-3-haiku-20250307'];
const MAX_RETRIES = 3;
const INITIAL_BACKOFF_MS = 1000;

interface ClaudeMessage {
  role: 'user' | 'assistant';
  content: string;
}

interface ClaudeRequestBody {
  model: string;
  max_tokens: number;
  messages: ClaudeMessage[];
}

interface ClaudeResponse {
  id: string;
  type: string;
  role: string;
  content: Array<{
    type: string;
    text: string;
  }>;
  model: string;
  stop_reason: string;
  usage: {
    input_tokens: number;
    output_tokens: number;
  };
}

interface ClaudeModel {
  id: string;
  type: string;
  display_name: string;
  created_at: string;
}

interface ClaudeModelsResponse {
  data: ClaudeModel[];
}

export class ClaudeProvider extends BaseLLMProvider {
  type: 'claude' = 'claude';
  name = 'Claude (Anthropic)';
  models: string[] = [];

  private apiKey: string;
  private model: string;
  private discoveredAt: number = 0;
  private discoveryCache: string[] = [];

  constructor(apiKey: string, model: string = '') {
    super();
    this.apiKey = apiKey;
    this.model = model || DEFAULT_CLAUDE_MODELS[0];
    this.models = DEFAULT_CLAUDE_MODELS; // Start with defaults
  }

  get isConfigured(): boolean {
    return !!this.apiKey;
  }

  /**
   * Discover available Claude models from Anthropic API
   */
  async discoverModels(): Promise<string[]> {
    if (!this.apiKey) {
      this.models = DEFAULT_CLAUDE_MODELS;
      return this.models;
    }

    // Return cache if fresh (within 1 hour)
    if (this.discoveryCache.length > 0 && Date.now() - this.discoveredAt < 3600000) {
      this.models = this.discoveryCache;
      return this.models;
    }

    try {
      const response = await apiClient.get<ClaudeModelsResponse>(CLAUDE_MODELS_URL, {
        headers: {
          'x-api-key': this.apiKey,
          'anthropic-version': '2024-06-01',
        },
        timeout: 10000,
      });

      if (response.data && Array.isArray(response.data)) {
        const modelIds = response.data
          .filter((m) => m.type === 'model' && m.id.startsWith('claude'))
          .map((m) => m.id)
          .sort();

        if (modelIds.length > 0) {
          this.discoveryCache = modelIds;
          this.discoveredAt = Date.now();
          this.models = modelIds;
          console.log(`Discovered ${modelIds.length} Claude models:`, modelIds);
          return modelIds;
        }
      }
    } catch (error) {
      console.warn('Failed to discover Claude models, using defaults:', error);
    }

    // Fallback to defaults
    this.models = DEFAULT_CLAUDE_MODELS;
    return this.models;
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
      const response = await this.callClaudeAPIWithRetry(prompt);
      return this.parseMessageSuggestions(response);
    } catch (error) {
      console.error('Error generating suggestions:', error);
      throw error;
    }
  }

  async validate(): Promise<boolean> {
    try {
      if (!this.isConfigured) return false;

      const testPrompt = 'Respond with just "ok"';
      await this.callClaudeAPI(testPrompt);
      return true;
    } catch (error) {
      console.error('Claude API validation failed:', error);
      return false;
    }
  }

  private async callClaudeAPIWithRetry(userPrompt: string, attempt: number = 0): Promise<string> {
    try {
      const response = await this.callClaudeAPI(userPrompt);
      return response;
    } catch (error) {
      if (attempt < MAX_RETRIES - 1) {
        const backoffMs = INITIAL_BACKOFF_MS * Math.pow(2, attempt);
        console.log(`Retry attempt ${attempt + 1}, waiting ${backoffMs}ms...`);
        await new Promise((resolve) => setTimeout(resolve, backoffMs));
        return this.callClaudeAPIWithRetry(userPrompt, attempt + 1);
      }
      throw error;
    }
  }

  private async callClaudeAPI(userPrompt: string): Promise<string> {
    const body: ClaudeRequestBody = {
      model: this.model,
      max_tokens: 1024,
      messages: [
        {
          role: 'user',
          content: userPrompt,
        },
      ],
    };

    const response = await apiClient.post<ClaudeResponse>(CLAUDE_API_URL, body, {
      headers: {
        'x-api-key': this.apiKey,
        'anthropic-version': '2024-06-01',
      },
      timeout: 30000,
    });

    if (!response.content || response.content.length === 0) {
      throw new Error('Empty response from Claude API');
    }

    return response.content[0].text;
  }
}
