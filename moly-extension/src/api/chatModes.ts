/**
 * Chat Mode Strategies
 * Socratic vs Direct message generation approaches
 */

import type { ChatMode, CommunicationContext } from '@/types';

export interface ChatModeConfig {
  mode: ChatMode;
  description: string;
  icon: string;
}

export const CHAT_MODES: Record<ChatMode, ChatModeConfig> = {
  socratic: {
    mode: 'socratic',
    description: 'Guided questions to help you think through your message',
    icon: '💭',
  },
  direct: {
    mode: 'direct',
    description: 'Get direct message suggestions immediately',
    icon: '⚡',
  },
};

/**
 * Build prompt for Socratic mode - asks guiding questions
 */
export function buildSocraticPrompt(
  context: string,
  communicationContext: CommunicationContext,
): string {
  const contextGuides: Record<CommunicationContext, string> = {
    formal: 'Professional and respectful tone',
    friendly: 'Warm and genuine approach',
    dating: 'Authentic and interested tone',
  };

  return `You are a thoughtful messaging coach helping someone craft a meaningful response.

CONTEXT: ${context}
TONE: ${contextGuides[communicationContext]}

Instead of writing the message for them, ask 3-5 guiding questions that help them think through what they want to say. These questions should:
1. Help clarify their feelings and intentions
2. Consider the recipient's perspective
3. Encourage authenticity
4. Guide tone and approach

Format your response as valid JSON (no markdown):
{
  "questions": [
    {
      "question": "The actual guiding question",
      "purpose": "Why this question helps (e.g., 'Clarifies your intent')"
    },
    ...
  ],
  "conversationContext": "Brief guidance on this conversation type",
  "toneReminder": "A reminder about the tone you're aiming for"
}

Focus on helping them become better communicators, not just writing messages for them.`;
}

/**
 * Build prompt for Direct mode - generates suggestions
 */
export function buildDirectPrompt(
  userMessage: string,
  context: string,
  communicationContext: CommunicationContext,
): string {
  const contextGuides: Record<CommunicationContext, string> = {
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

/**
 * Parse Socratic response
 */
export interface SocraticQuestion {
  question: string;
  purpose: string;
}

export interface SocraticResponse {
  questions: SocraticQuestion[];
  conversationContext: string;
  toneReminder: string;
}

export function parseSocraticResponse(response: string): SocraticResponse {
  try {
    const jsonMatch = response.match(/\{[\s\S]*\}/);
    if (!jsonMatch) {
      throw new Error('No JSON found in response');
    }

    const parsed = JSON.parse(jsonMatch[0]);
    return {
      questions: parsed.questions || [],
      conversationContext: parsed.conversationContext || 'Consider what you want to achieve',
      toneReminder: parsed.toneReminder || 'Be authentic and genuine',
    };
  } catch (error) {
    console.error('Error parsing Socratic response:', error);
    throw new Error(`Failed to parse Socratic response: ${error instanceof Error ? error.message : 'Unknown error'}`);
  }
}

/**
 * Get mode description with emoji
 */
export function getModeDescription(mode: ChatMode): string {
  const config = CHAT_MODES[mode];
  return `${config.icon} ${config.description}`;
}
