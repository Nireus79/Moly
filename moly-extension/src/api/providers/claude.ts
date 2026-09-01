/**
 * Claude LLM Provider
 * Integration with Anthropic Claude API with dynamic model discovery
 */

import { apiClient } from '@/api/client';
import { BaseLLMProvider, type MessageSuggestion } from '@/api/providers';

const CLAUDE_API_URL = 'https://api.anthropic.com/v1/messages';
const CLAUDE_MODELS_URL = 'https://api.anthropic.com/v1/models';
const CLAUDE_FALLBACK_MODELS = ['claude-2.1', 'claude-2', 'claude-instant-1.2', 'claude-instant-1'];
const MAX_RETRIES = 3;
const INITIAL_BACKOFF_MS = 1000;

// Fallback API versions to try if cached version fails
const FALLBACK_API_VERSIONS = [
  '2023-06-01',
  '2024-01-15',
  '2024-06-15',
  '2025-01-15',
];

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
  private workingApiVersion: string | null = null;

  constructor(apiKey: string, model: string = '') {
    super();
    this.apiKey = apiKey;
    this.model = model || CLAUDE_FALLBACK_MODELS[0];
    this.models = CLAUDE_FALLBACK_MODELS; // Start with fallback
  }

  /**
   * Discover which API version actually works (adapts if Anthropic changes versions)
   * Stores working version in Chrome storage for persistence across sessions
   */
  private async discoverApiVersion(): Promise<string> {
    // Return in-memory cache if available
    if (this.workingApiVersion) {
      return this.workingApiVersion;
    }

    // Try to get cached version from storage
    try {
      const stored = await new Promise<any>((resolve) => {
        chrome.storage.local.get('claudeApiVersion', resolve);
      });

      if (stored.claudeApiVersion) {
        console.log(`[Claude] Using cached API version: ${stored.claudeApiVersion}`);
        this.workingApiVersion = stored.claudeApiVersion;
        return stored.claudeApiVersion;
      }
    } catch (error) {
      console.log('[Claude] Could not read from storage, will discover version');
    }

    // Try each version until one works
    for (const version of FALLBACK_API_VERSIONS) {
      try {
        const testBody = {
          model: CLAUDE_FALLBACK_MODELS[0],
          max_tokens: 10,
          messages: [{ role: 'user' as const, content: 'ok' }],
        };

        await apiClient.post(CLAUDE_API_URL, testBody, {
          headers: {
            'x-api-key': this.apiKey,
            'anthropic-version': version,
          },
          timeout: 5000,
        });

        // If we got here without error, this version works
        console.log(`[Claude] Discovered working API version: ${version}`);
        this.workingApiVersion = version;

        // Save to storage for future sessions
        try {
          await new Promise<void>((resolve) => {
            chrome.storage.local.set({ claudeApiVersion: version }, resolve);
          });
        } catch (storageError) {
          console.log('[Claude] Could not save version to storage');
        }

        return version;
      } catch (error) {
        console.log(`[Claude] Version ${version} failed, trying next...`);
        continue;
      }
    }

    // Fallback to first version if all fail
    console.warn('[Claude] Could not discover working API version, using first fallback');
    this.workingApiVersion = FALLBACK_API_VERSIONS[0];
    return this.workingApiVersion;
  }

  get isConfigured(): boolean {
    return !!this.apiKey;
  }

  /**
   * Discover available Claude models from Anthropic API (dynamically discovers real models)
   */
  async discoverModels(): Promise<string[]> {
    if (!this.apiKey) {
      this.models = CLAUDE_FALLBACK_MODELS;
      return this.models;
    }

    // Return cache if fresh (within 1 hour)
    if (this.discoveryCache.length > 0 && Date.now() - this.discoveredAt < 3600000) {
      this.models = this.discoveryCache;
      return this.models;
    }

    try {
      // Get the working API version for this request
      const apiVersion = await this.discoverApiVersion();

      const response = await apiClient.get<ClaudeModelsResponse>(CLAUDE_MODELS_URL, {
        headers: {
          'x-api-key': this.apiKey,
          'anthropic-version': apiVersion,
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
          console.log(`[Claude] Discovered ${modelIds.length} real models:`, modelIds);
          return modelIds;
        }
      }
    } catch (error) {
      console.warn('[Claude] Failed to discover models, using fallback:', error);
    }

    // Fallback to fallback models
    this.models = CLAUDE_FALLBACK_MODELS;
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

    // Discover working API version dynamically
    const apiVersion = await this.discoverApiVersion();

    console.log('[Claude] Calling API with:');
    console.log('[Claude] - API Key (first 20 chars):', this.apiKey.substring(0, 20) + '...');
    console.log('[Claude] - API Version:', apiVersion);
    console.log('[Claude] - Model:', this.model);

    const response = await apiClient.post<ClaudeResponse>(CLAUDE_API_URL, body, {
      headers: {
        'x-api-key': this.apiKey,
        'anthropic-version': apiVersion,
      },
      timeout: 30000,
    });

    if (!response.content || response.content.length === 0) {
      throw new Error('Empty response from Claude API');
    }

    return response.content[0].text;
  }
}
