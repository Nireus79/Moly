/**
 * Mode-Aware LLM Client
 * Handles both Socratic and Direct modes
 */

import { getProviderManager } from '@/api/providerManager';
import { parseSocraticResponse } from '@/api/chatModes';
import type { ChatMode, CommunicationContext, MessageSuggestion } from '@/types';

export interface SocraticQuestion {
  question: string;
  purpose: string;
}

export interface SocraticResult {
  questions: SocraticQuestion[];
  conversationContext: string;
  toneReminder: string;
}

/**
 * Generate content based on chat mode
 */
export async function generateModeAwareContent(
  userMessage: string,
  context: string,
  communicationContext: CommunicationContext,
  chatMode: ChatMode,
): Promise<MessageSuggestion[] | SocraticResult> {
  const manager = getProviderManager();
  const provider = manager.getActiveProvider();

  if (!provider) {
    throw new Error('No active LLM provider configured');
  }

  if (chatMode === 'socratic') {
    return generateSocraticQuestions(context, communicationContext, provider);
  } else {
    return generateDirectSuggestions(
      userMessage,
      context,
      communicationContext,
      provider,
    );
  }
}

/**
 * Generate Socratic guiding questions
 */
async function generateSocraticQuestions(
  context: string,
  communicationContext: CommunicationContext,
  provider: any,
): Promise<SocraticResult> {
  try {
    const response = await provider.generateSuggestions('', context, communicationContext);
    // For Socratic mode, we call the provider with a special prompt structure
    // The provider should use buildSocraticPrompt internally, but here we parse the response
    const result = parseSocraticResponse(
      Array.isArray(response) && response.length > 0
        ? response[0].text
        : JSON.stringify(response),
    );
    return result;
  } catch (error) {
    console.error('Error generating Socratic questions:', error);
    // Fallback Socratic questions
    return {
      questions: [
        {
          question: 'What is your main goal with this message?',
          purpose: 'Clarifies your intent',
        },
        {
          question: 'How do you want the other person to feel after reading it?',
          purpose: 'Guides emotional tone',
        },
        {
          question: 'What is most authentic about how you feel?',
          purpose: 'Ensures genuineness',
        },
      ],
      conversationContext: context,
      toneReminder: 'Be authentic and genuine',
    };
  }
}

/**
 * Generate Direct message suggestions
 */
async function generateDirectSuggestions(
  userMessage: string,
  context: string,
  communicationContext: CommunicationContext,
  provider: any,
): Promise<MessageSuggestion[]> {
  try {
    const suggestions = await provider.generateSuggestions(
      userMessage,
      context,
      communicationContext,
    );
    return suggestions;
  } catch (error) {
    console.error('Error generating direct suggestions:', error);
    throw error;
  }
}

/**
 * Check if result is Socratic (questions) vs Direct (suggestions)
 */
export function isSocraticResult(
  result: MessageSuggestion[] | SocraticResult,
): result is SocraticResult {
  return 'questions' in result && Array.isArray((result as any).questions);
}
