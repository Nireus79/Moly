/**
 * LLM Provider Manager
 * Discovers, configures, and switches between available LLM providers
 */

import { ClaudeProvider } from '@/api/providers/claude';
import { OpenAIProvider } from '@/api/providers/openai';
import { OllamaProvider } from '@/api/providers/ollama';
import type { LLMProvider, LLMProviderType, ProviderCredentials } from '@/api/providers';

export interface ProviderInfo {
  type: LLMProviderType;
  name: string;
  isConfigured: boolean;
  isAvailable: boolean;
  priority: number;
}

export class LLMProviderManager {
  private providers: Map<LLMProviderType, LLMProvider> = new Map();
  private activeProvider: LLMProvider | null = null;

  constructor() {
    this.initializeProviders();
  }

  private initializeProviders() {
    // Initialize all providers (may not be configured yet)
    this.providers.set('claude', new ClaudeProvider(''));
    this.providers.set('openai', new OpenAIProvider(''));
    this.providers.set('ollama', new OllamaProvider());
  }

  /**
   * Configure a specific provider with credentials
   */
  async configureProvider(credentials: ProviderCredentials): Promise<boolean> {
    try {
      let provider: LLMProvider;

      switch (credentials.type) {
        case 'claude':
          provider = new ClaudeProvider(credentials.apiKey || '', credentials.model || 'claude-3-5-sonnet-20241022');
          break;
        case 'openai':
          provider = new OpenAIProvider(credentials.apiKey || '', credentials.model || 'gpt-4-turbo');
          break;
        case 'ollama':
          provider = new OllamaProvider(credentials.baseUrl || 'http://localhost:11435', credentials.model || 'mistral');
          break;
        default:
          throw new Error(`Unknown provider type: ${credentials.type}`);
      }

      // Validate provider (skip for Ollama - let generate fail fast if there's an issue)
      if (credentials.type !== 'ollama') {
        const isValid = await provider.validate();
        if (!isValid) {
          throw new Error(`Provider validation failed for ${credentials.type}`);
        }
      } else {
        // For Ollama, just check it's reachable (lightweight check)
        const isAvailable = await (provider as any).isAvailable();
        if (!isAvailable) {
          throw new Error('Ollama is not reachable');
        }
      }

      this.providers.set(credentials.type, provider);
      this.activeProvider = provider;
      return true;
    } catch (error) {
      console.error('Failed to configure provider:', error);
      return false;
    }
  }

  /**
   * Discover available providers (checks which ones are configured/available + their models)
   */
  async discoverProviders(): Promise<ProviderInfo[]> {
    const infos: ProviderInfo[] = [];
    const priorityMap: Record<LLMProviderType, number> = {
      claude: 1,
      openai: 2,
      ollama: 3,
    };

    for (const [type, provider] of this.providers) {
      let isAvailable = provider.isConfigured;

      // Discover models for configured providers
      if (provider.isConfigured) {
        try {
          // Call discoverModels if available
          if ('discoverModels' in provider && typeof (provider as any).discoverModels === 'function') {
            await (provider as any).discoverModels();
          }
        } catch (error) {
          console.warn(`Failed to discover models for ${type}:`, error);
        }
      }

      // For Ollama, check if server is running
      if (type === 'ollama') {
        isAvailable = await (provider as InstanceType<typeof OllamaProvider>).isAvailable();
      }

      infos.push({
        type,
        name: provider.name,
        isConfigured: provider.isConfigured,
        isAvailable,
        priority: priorityMap[type],
      });
    }

    return infos.sort((a, b) => a.priority - b.priority);
  }

  /**
   * Get the best available provider in priority order
   */
  async getAvailableProvider(): Promise<LLMProvider | null> {
    const providers = await this.discoverProviders();

    for (const info of providers) {
      if (info.isAvailable && info.isConfigured) {
        return this.providers.get(info.type) || null;
      }
    }

    return null;
  }

  /**
   * Set the active provider
   */
  setActiveProvider(type: LLMProviderType): boolean {
    const provider = this.providers.get(type);
    if (!provider) {
      console.error(`Provider not found: ${type}`);
      return false;
    }

    this.activeProvider = provider;
    return true;
  }

  /**
   * Get the current active provider
   */
  getActiveProvider(): LLMProvider | null {
    return this.activeProvider;
  }

  /**
   * Get a specific provider by type
   */
  getProvider(type: LLMProviderType): LLMProvider | null {
    return this.providers.get(type) || null;
  }

  /**
   * Get all available model names for a provider
   */
  getModels(type: LLMProviderType): string[] {
    const provider = this.providers.get(type);
    return provider?.models || [];
  }

  /**
   * Validate the active provider
   */
  async validateActiveProvider(): Promise<boolean> {
    if (!this.activeProvider) {
      return false;
    }
    return this.activeProvider.validate();
  }
}

/**
 * Singleton instance
 */
let manager: LLMProviderManager | null = null;

export function getProviderManager(): LLMProviderManager {
  if (!manager) {
    manager = new LLMProviderManager();
  }
  return manager;
}
