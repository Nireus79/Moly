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

  constructor(baseUrl: string = 'http://localhost:11435', model: string = '') {
    super();
    this.baseUrl = baseUrl;
    this.model = model || 'mistral';
    this.models = [];
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

    // Cache discovery results for 30 seconds to avoid flooding Ollama with requests
    const now = Date.now();
    if (this.discoveredAt && now - this.discoveredAt < 30000 && this.discoveryCache.length > 0) {
      console.log('[Ollama] Using cached model list');
      return this.discoveryCache;
    }

    try {
      console.log('[Ollama] Attempting to fetch models from /api/tags...');
      let response = await fetch(`${this.baseUrl}/api/tags`, {
        signal: AbortSignal.timeout(5000),
      }).catch(async (error) => {
        // If proxy fails and baseUrl is proxy, try direct Ollama
        if (this.baseUrl.includes('11435')) {
          console.log('[Ollama] Proxy failed, trying direct Ollama at 11434...');
          return fetch('http://localhost:11434/api/tags', {
            signal: AbortSignal.timeout(5000),
          });
        }
        throw error;
      });

      console.log('[Ollama] Response status:', response.status, response.statusText);

      if (!response.ok) {
        throw new Error(`API error: ${response.status} ${response.statusText}`);
      }

      const data = (await response.json()) as OllamaTagsResponse;
      console.log('[Ollama] Response data received, models count:', data.models?.length || 0);

      if (data.models && Array.isArray(data.models)) {
        // Keep full model names with tags (e.g., "stable-code:3b")
        // Group by base name and keep the latest tag for each
        const modelMap = new Map<string, string>();
        for (const model of data.models) {
          const baseName = model.name.split(':')[0];
          // Keep the full name (with tag) - this is what Ollama API expects
          if (!modelMap.has(baseName)) {
            modelMap.set(baseName, model.name);
          }
        }

        const modelNames = Array.from(modelMap.values()).sort();

        if (modelNames.length > 0) {
          this.models = modelNames;
          this.discoveryCache = modelNames;
          this.discoveredAt = Date.now();
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
      console.log('[Ollama] Calling API generate endpoint...');
      console.log('[Ollama] Model:', this.model);
      console.log('[Ollama] Prompt length:', prompt.length);
      console.log('[Ollama] Request body:', JSON.stringify(body).substring(0, 200));

      let response = await fetch(`${this.baseUrl}/api/generate`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(body),
      }).catch(async (error) => {
        // If proxy fails and baseUrl is proxy, try direct Ollama
        if (this.baseUrl.includes('11435')) {
          console.log('[Ollama] Proxy failed, trying direct Ollama at 11434...');
          return fetch('http://localhost:11434/api/generate', {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
            },
            body: JSON.stringify(body),
          });
        }
        throw error;
      });

      console.log('[Ollama] Generate response status:', response.status, response.statusText);
      console.log('[Ollama] Response headers:', response.headers.get('content-type'));

      if (!response.ok) {
        const errorText = await response.text();
        console.error('[Ollama] Error response body:', errorText);
        throw new Error(`Ollama API error: ${response.status} ${response.statusText}`);
      }

      const data = (await response.json()) as OllamaResponse;
      return data.response;
    } catch (error) {
      const msg = error instanceof Error ? error.message : 'Unknown error';
      console.error('[Ollama] callOllama error:', msg);
      throw new Error(`Failed to call Ollama API: ${msg}`);
    }
  }

  async isAvailable(): Promise<boolean> {
    try {
      // Try proxy first
      const response = await fetch(`${this.baseUrl}/api/tags`, {
        method: 'HEAD',
        signal: AbortSignal.timeout(3000),
      });
      return response.ok;
    } catch {
      try {
        // If proxy fails, try direct Ollama
        if (this.baseUrl.includes('11435')) {
          const directResponse = await fetch('http://localhost:11434/api/tags', {
            method: 'HEAD',
            signal: AbortSignal.timeout(3000),
          });
          return directResponse.ok;
        }
      } catch {
        // Both failed
      }
      return false;
    }
  }
}
