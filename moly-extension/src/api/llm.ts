/**
 * Claude LLM API Client
 * Handles all communication with Anthropic Claude API
 */

import { apiClient } from './client';
import type { MessageSuggestion } from '@/types';

export interface ClaudeMessage {
  role: 'user' | 'assistant';
  content: string;
}

export interface ClaudeRequestBody {
  model: string;
  max_tokens: number;
  messages: ClaudeMessage[];
  system?: string;
}

export interface ClaudeResponse {
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

const CLAUDE_API_URL = 'https://api.anthropic.com/v1/messages';
const CLAUDE_MODEL = 'claude-3-5-sonnet-20241022';
const MAX_RETRIES = 3;
const INITIAL_BACKOFF_MS = 1000;

/**
 * Generate message suggestions using Claude API
 */
export async function generateMessageSuggestions(
  userMessage: string,
  context: string,
  communicationContext: 'formal' | 'friendly' | 'dating',
  apiKey: string,
): Promise<MessageSuggestion[]> {
  if (!apiKey) {
    throw new Error('Claude API key not configured');
  }

  const prompt = buildPrompt(userMessage, context, communicationContext);

  try {
    const response = await callClaudeAPIWithRetry(prompt, apiKey);
    return parseMessageSuggestions(response);
  } catch (error) {
    console.error('Error generating suggestions:', error);
    throw error;
  }
}

/**
 * Build system prompt for message generation
 */
function buildPrompt(
  userMessage: string,
  context: string,
  communicationContext: 'formal' | 'friendly' | 'dating',
): string {
  const contextGuides: Record<string, string> = {
    formal:
      'Professional, respectful, and direct. Use proper grammar and maintain formality.',
    friendly:
      'Warm, approachable, and genuine. Natural conversation style while being authentic.',
    dating:
      'Genuine, warm, and shows authentic interest. Demonstrates personality while being respectful.',
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

/**
 * Call Claude API with retry logic
 */
async function callClaudeAPIWithRetry(
  userPrompt: string,
  apiKey: string,
  attempt: number = 0,
): Promise<string> {
  try {
    const response = await callClaudeAPI(userPrompt, apiKey);
    return response;
  } catch (error) {
    if (attempt < MAX_RETRIES - 1) {
      const backoffMs = INITIAL_BACKOFF_MS * Math.pow(2, attempt);
      console.log(`Retry attempt ${attempt + 1}, waiting ${backoffMs}ms...`);
      await new Promise((resolve) => setTimeout(resolve, backoffMs));
      return callClaudeAPIWithRetry(userPrompt, apiKey, attempt + 1);
    }
    throw error;
  }
}

/**
 * Call Claude API
 */
async function callClaudeAPI(userPrompt: string, apiKey: string): Promise<string> {
  const body: ClaudeRequestBody = {
    model: CLAUDE_MODEL,
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
      'x-api-key': apiKey,
      'anthropic-version': '2023-06-01',
    },
    timeout: 30000,
  });

  if (!response.content || response.content.length === 0) {
    throw new Error('Empty response from Claude API');
  }

  return response.content[0].text;
}

/**
 * Parse Claude response into message suggestions
 */
function parseMessageSuggestions(response: string): MessageSuggestion[] {
  try {
    // Extract JSON from response (in case Claude adds extra text)
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
    throw new Error(`Failed to parse Claude response: ${error instanceof Error ? error.message : 'Unknown error'}`);
  }
}

/**
 * Validate API key by making a test call
 */
export async function validateApiKey(apiKey: string): Promise<boolean> {
  try {
    const testPrompt = 'Respond with just "ok"';
    await callClaudeAPI(testPrompt, apiKey);
    return true;
  } catch (error) {
    console.error('API key validation failed:', error);
    return false;
  }
}
