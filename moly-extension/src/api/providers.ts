/**
 * LLM Provider Abstraction Layer
 * Supports Claude, OpenAI, Ollama (local), and other providers
 */

import type { MessageSuggestion } from '@/types';

export type { MessageSuggestion };
export type LLMProviderType = 'claude' | 'openai' | 'ollama' | 'custom';

export interface LLMProviderConfig {
  type: LLMProviderType;
  apiKey?: string;
  model: string;
  baseUrl?: string;
  enabled: boolean;
}

export interface LLMProvider {
  type: LLMProviderType;
  name: string;
  models: string[];
  isConfigured: boolean;

  generateSuggestions(
    userMessage: string,
    context: string,
    communicationContext: 'formal' | 'friendly' | 'dating',
  ): Promise<MessageSuggestion[]>;

  validate(): Promise<boolean>;
}

export interface ProviderCredentials {
  type: LLMProviderType;
  apiKey?: string;
  baseUrl?: string;
  model: string;
}

/**
 * Base provider class with common functionality
 */
export abstract class BaseLLMProvider implements LLMProvider {
  abstract type: LLMProviderType;
  abstract name: string;
  abstract models: string[];

  abstract generateSuggestions(
    userMessage: string,
    context: string,
    communicationContext: 'formal' | 'friendly' | 'dating',
  ): Promise<MessageSuggestion[]>;

  abstract validate(): Promise<boolean>;

  abstract get isConfigured(): boolean;

  protected buildPrompt(
    userMessage: string,
    context: string,
    communicationContext: 'formal' | 'friendly' | 'dating',
  ): string {
    const contextGuides: Record<string, string> = {
      formal: 'Professional, respectful, and direct. Use proper grammar and maintain formality.',
      friendly: 'Warm, approachable, and genuine. Natural conversation style while being authentic.',
      dating: 'Genuine, warm, and shows authentic interest. Demonstrates personality while being respectful.',
    };

    return `You are a messaging coach helping someone craft the perfect response.

CONTEXT: ${context}

COMMUNICATION CONTEXT: ${communicationContext}
TONE GUIDE: ${contextGuides[communicationContext]}

USER'S MESSAGE: "${userMessage}"

Generate 3 different message options that vary in approach but all feel authentic to the user's voice. Each should show genuine interest and be appropriate for the context.

Format your response as valid JSON (no markdown, no code blocks, just raw JSON):
{
  "suggestions": [
    {
      "text": "The actual message suggestion here",
      "tone": "brief descriptor of tone/approach",
      "reasoning": "why this approach works",
      "confidence": 0.85
    },
    ...
  ]
}

Remember:
- Each suggestion should feel natural and authentic
- Vary the approaches (e.g., witty vs sincere, direct vs curious)
- Never be manipulative or fake
- Always respect the other person's autonomy
- Confidence scores should reflect how well this likely resonates (0.5-0.95)`;
  }

  protected parseMessageSuggestions(response: string): MessageSuggestion[] {
    try {
      const jsonMatch = response.match(/\{[\s\S]*\}/);
      if (!jsonMatch) {
        throw new Error('No JSON found in response');
      }

      const parsed = JSON.parse(jsonMatch[0]);
      const suggestions = parsed.suggestions || [];

      return suggestions.map((s: any, idx: number) => ({
        id: `suggestion-${Date.now()}-${idx}`,
        text: s.text || '',
        tone: s.tone || 'general',
        confidence: Math.max(0.5, Math.min(0.95, s.confidence || 0.75)),
        reasoning: s.reasoning || 'This approach fits your communication context.',
        bestFor: s.bestFor || 'Most situations',
      }));
    } catch (error) {
      console.error('Error parsing suggestions:', error, response);
      throw new Error(`Failed to parse response: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }
}
