/**
 * Ollama LLM Provider
 * Integration with local Ollama instance (requires local Ollama server running)
 */

import { BaseLLMProvider, type MessageSuggestion } from '@/api/providers';
const DEFAULT_OLLAMA_MODELS = ['mistral', 'llama2', 'neural-chat'];

interface OllamaRequest {
  model: string;
  prompt: string;
  stream: boolean;
}

interface OllamaResponse {
  model: string;
  created_at: string;
  response: string;
  done: boolean;
}

interface OllamaTagsResponse {
  models: Array<{
    name: string;
    modified_at: string;
    size: number;
  }>;
}

export class OllamaProvider extends BaseLLMProvider {
  type: 'ollama' = 'ollama';
  name = 'Ollama (Local Models)';
  models: string[] = [];

  private baseUrl: string;
  private model: string;
  private discoveredAt: number = 0;
  private discoveryCache: string[] = [];

  constructor(baseUrl: string = 'http://localhost:11434', model: string = '') {
    super();
    this.baseUrl = baseUrl;
    this.model = model || 'mistral';
    this.models = DEFAULT_OLLAMA_MODELS;
  }

  get isConfigured(): boolean {
    return true; // Ollama is always "configured" if localhost is running
  }

  /**
   * Discover available Ollama models from local server
   */
  async discoverModels(): Promise<string[]> {
    console.log('[Ollama] Starting model discovery...');
    console.log('[Ollama] Endpoint:', this.baseUrl);

    // Always rediscover for local servers (they change frequently)
    try {
      console.log('[Ollama] Attempting to fetch models from /api/tags...');
      const response = await fetch(`${this.baseUrl}/api/tags`, {
        signal: AbortSignal.timeout(5000),
      });

      console.log('[Ollama] Response status:', response.status, response.statusText);

      if (!response.ok) {
        throw new Error(`API error: ${response.status} ${response.statusText}`);
      }

      const data = (await response.json()) as OllamaTagsResponse;
      console.log('[Ollama] Response data received, models count:', data.models?.length || 0);

      if (data.models && Array.isArray(data.models)) {
        // Extract model names, removing tags (e.g., "mistral:latest" -> "mistral")
        const modelNames = data.models
          .map((m) => m.name.split(':')[0])
          .filter((name, idx, arr) => arr.indexOf(name) === idx) // Deduplicate
          .sort();

        if (modelNames.length > 0) {
          this.models = modelNames;
          console.log(`[Ollama] Discovered ${modelNames.length} models:`, modelNames);
          return modelNames;
        }

        console.log('[Ollama] No unique model names after processing');
      } else {
        console.log('[Ollama] No models array in response');
      }

      console.log('[Ollama] No models found in response');
      return [];
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : String(error);
      const errorStatus = (error as any)?.status;
      console.error('[Ollama] Discovery failed');
      console.error('  Message:', errorMsg);
      console.error('  Type:', error?.constructor?.name);
      console.error('  Endpoint:', this.baseUrl);
      if (errorStatus) console.error('  Status:', errorStatus);
      throw error;
    }
  }

  async generateSuggestions(
    userMessage: string,
    context: string,
    communicationContext: 'formal' | 'friendly' | 'dating',
  ): Promise<MessageSuggestion[]> {
    const prompt = this.buildPrompt(userMessage, context, communicationContext);

    try {
      const response = await this.callOllama(prompt);
      return this.parseMessageSuggestions(response);
    } catch (error) {
      console.error('[Ollama] Error generating suggestions:', error);
      throw error;
    }
  }

  async validate(): Promise<boolean> {
    try {
      const models = await this.discoverModels();
      // If we found models, validation succeeds (skip test call to avoid CORS issues)
      if (models.length > 0) {
        console.log('[Ollama] Validation successful - found models');
        return true;
      }
      return false;
    } catch (error) {
      console.error('[Ollama] Validation failed:', error);
      return false;
    }
  }

  private async callOllama(prompt: string): Promise<string> {
    const body: OllamaRequest = {
      model: this.model,
      prompt,
      stream: false,
    };

    try {
      const response = await fetch(`${this.baseUrl}/api/generate`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(body),
      });

      if (!response.ok) {
        throw new Error(`Ollama API error: ${response.statusText}`);
      }

      const data = (await response.json()) as OllamaResponse;
      return data.response;
    } catch (error) {
      throw new Error(`Failed to call Ollama API: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }

  async isAvailable(): Promise<boolean> {
    try {
      const response = await fetch(`${this.baseUrl}/api/tags`, { method: 'HEAD' });
      return response.ok;
    } catch {
      return false;
    }
  }
}
