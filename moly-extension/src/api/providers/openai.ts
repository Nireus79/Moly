/**
 * OpenAI LLM Provider
 * Integration with OpenAI API with dynamic model discovery
 */

import { apiClient } from '@/api/client';
import { BaseLLMProvider, type MessageSuggestion } from '@/api/providers';

const OPENAI_API_URL = 'https://api.openai.com/v1/chat/completions';
const OPENAI_MODELS_URL = 'https://api.openai.com/v1/models';
const DEFAULT_OPENAI_MODELS = ['gpt-4-turbo', 'gpt-4', 'gpt-3.5-turbo'];
const MAX_RETRIES = 3;
const INITIAL_BACKOFF_MS = 1000;

interface OpenAIMessage {
  role: 'system' | 'user' | 'assistant';
  content: string;
}

interface OpenAIRequestBody {
  model: string;
  messages: OpenAIMessage[];
  max_tokens: number;
  temperature: number;
}

interface OpenAIResponse {
  id: string;
  object: string;
  created: number;
  model: string;
  choices: Array<{
    index: number;
    message: {
      role: string;
      content: string;
    };
    finish_reason: string;
  }>;
  usage: {
    prompt_tokens: number;
    completion_tokens: number;
    total_tokens: number;
  };
}

interface OpenAIModel {
  id: string;
  object: string;
  created: number;
  owned_by: string;
  permission?: unknown[];
  root?: string;
  parent?: string;
}

interface OpenAIModelsResponse {
  object: string;
  data: OpenAIModel[];
}

export class OpenAIProvider extends BaseLLMProvider {
  type: 'openai' = 'openai';
  name = 'OpenAI (GPT-4/3.5)';
  models: string[] = [];

  private apiKey: string;
  private model: string;
  private discoveredAt: number = 0;
  private discoveryCache: string[] = [];

  constructor(apiKey: string, model: string = '') {
    super();
    this.apiKey = apiKey;
    this.model = model || DEFAULT_OPENAI_MODELS[0];
    this.models = DEFAULT_OPENAI_MODELS; // Start with defaults
  }

  get isConfigured(): boolean {
    return !!this.apiKey;
  }

  /**
   * Discover available OpenAI models from API
   */
  async discoverModels(): Promise<string[]> {
    if (!this.apiKey) {
      this.models = DEFAULT_OPENAI_MODELS;
      return this.models;
    }

    // Return cache if fresh (within 1 hour)
    if (this.discoveryCache.length > 0 && Date.now() - this.discoveredAt < 3600000) {
      this.models = this.discoveryCache;
      return this.models;
    }

    try {
      const response = await apiClient.get<OpenAIModelsResponse>(OPENAI_MODELS_URL, {
        headers: {
          'Authorization': `Bearer ${this.apiKey}`,
        },
        timeout: 10000,
      });

      if (response.data && Array.isArray(response.data)) {
        // Filter for chat models (gpt-4, gpt-3.5, etc)
        const modelIds = response.data
          .filter((m) => m.id && (m.id.includes('gpt-4') || m.id.includes('gpt-3.5')))
          .map((m) => m.id)
          .sort()
          .reverse(); // Most recent first

        if (modelIds.length > 0) {
          this.discoveryCache = modelIds;
          this.discoveredAt = Date.now();
          this.models = modelIds;
          console.log(`Discovered ${modelIds.length} OpenAI models:`, modelIds);
          return modelIds;
        }
      }
    } catch (error) {
      console.warn('Failed to discover OpenAI models, using defaults:', error);
    }

    // Fallback to defaults
    this.models = DEFAULT_OPENAI_MODELS;
    return this.models;
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
      const response = await this.callOpenAIWithRetry(prompt);
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
      await this.callOpenAI(testPrompt);
      return true;
    } catch (error) {
      console.error('OpenAI API validation failed:', error);
      return false;
    }
  }

  private async callOpenAIWithRetry(userPrompt: string, attempt: number = 0): Promise<string> {
    try {
      const response = await this.callOpenAI(userPrompt);
      return response;
    } catch (error) {
      if (attempt < MAX_RETRIES - 1) {
        const backoffMs = INITIAL_BACKOFF_MS * Math.pow(2, attempt);
        console.log(`Retry attempt ${attempt + 1}, waiting ${backoffMs}ms...`);
        await new Promise((resolve) => setTimeout(resolve, backoffMs));
        return this.callOpenAIWithRetry(userPrompt, attempt + 1);
      }
      throw error;
    }
  }

  private async callOpenAI(userPrompt: string): Promise<string> {
    const body: OpenAIRequestBody = {
      model: this.model,
      messages: [
        {
          role: 'user',
          content: userPrompt,
        },
      ],
      max_tokens: 1024,
      temperature: 0.7,
    };

    const response = await apiClient.post<OpenAIResponse>(OPENAI_API_URL, body, {
      headers: {
        'Authorization': `Bearer ${this.apiKey}`,
        'Content-Type': 'application/json',
      },
      timeout: 30000,
    });

    if (!response.choices || response.choices.length === 0) {
      throw new Error('Empty response from OpenAI API');
    }

    return response.choices[0].message.content;
  }
}
