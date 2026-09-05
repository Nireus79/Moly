import type { Message } from '@/sidebar/components/ChatHistory';

export interface ExtractedContactContext {
  characteristics?: string[];
  intentions?: string[];
  behaviors?: string[];
}

/**
 * Extract useful context about a contact from conversation messages.
 * Analyzes user messages to find facts they shared about the contact.
 */
export function extractContactContextFromConversation(
  messages: Message[],
  contactName: string
): ExtractedContactContext {
  const context: ExtractedContactContext = {
    characteristics: [],
    intentions: [],
    behaviors: [],
  };

  // Filter to user messages only
  const userMessages = messages.filter(m => m.type === 'user');

  if (userMessages.length === 0) return context;

  // Combine all user messages into one text for analysis
  const allUserText = userMessages.map(m => m.content).join(' ');

  // Extract characteristics (traits, roles, attributes)
  const characteristicKeywords = [
    'is a',
    'works as',
    'colleague',
    'friend',
    'family',
    'romantic',
    'busy',
    'creative',
    'analytical',
    'outgoing',
    'introverted',
    'perfectionist',
    'laid-back',
  ];

  const characteristicSentences = extractSentencesWithKeywords(
    allUserText,
    characteristicKeywords,
    contactName
  );
  context.characteristics = [...new Set(characteristicSentences)];

  // Extract intentions (what they want to do, their goals)
  const intentionKeywords = [
    'wants to',
    'trying to',
    'hoping to',
    'planning to',
    'asked if',
    'asked to',
    'considering',
    'thinking about',
    'interested in',
  ];

  const intentionSentences = extractSentencesWithKeywords(
    allUserText,
    intentionKeywords,
    contactName
  );
  context.intentions = [...new Set(intentionSentences)];

  // Extract behaviors (actions, habits, recent events)
  const behaviorKeywords = [
    'recently',
    'just',
    'been',
    'stressed about',
    'excited about',
    'working on',
    'mentioned',
    'told me',
    'said they',
    'complained about',
    'loves',
    'hates',
    'prefers',
  ];

  const behaviorSentences = extractSentencesWithKeywords(
    allUserText,
    behaviorKeywords,
    contactName
  );
  context.behaviors = [...new Set(behaviorSentences)];

  return context;
}

/**
 * Extract sentences that contain keywords related to the contact.
 */
function extractSentencesWithKeywords(
  text: string,
  keywords: string[],
  contactName: string
): string[] {
  const sentences = text.match(/[^.!?]+[.!?]+/g) || [];
  const extracted: string[] = [];

  for (const sentence of sentences) {
    const trimmed = sentence.trim();

    // Check if sentence mentions contact and contains keyword
    const mentionsContact =
      trimmed.toLowerCase().includes(contactName.toLowerCase()) ||
      trimmed.toLowerCase().includes('they') ||
      trimmed.toLowerCase().includes('she') ||
      trimmed.toLowerCase().includes('he') ||
      trimmed.toLowerCase().includes('their');

    const hasKeyword = keywords.some(kw =>
      trimmed.toLowerCase().includes(kw.toLowerCase())
    );

    if (mentionsContact && hasKeyword) {
      extracted.push(trimmed.replace(/^[\s.!?]+|[\s.!?]+$/g, ''));
    }
  }

  return extracted.slice(0, 5); // Limit to top 5 to avoid overwhelming
}

/**
 * Format extracted context for display.
 */
export function formatExtractedContext(context: ExtractedContactContext): string {
  const parts: string[] = [];

  if (context.characteristics?.length) {
    parts.push(
      `Characteristics: ${context.characteristics.join('; ')}`
    );
  }

  if (context.intentions?.length) {
    parts.push(
      `Intentions: ${context.intentions.join('; ')}`
    );
  }

  if (context.behaviors?.length) {
    parts.push(
      `Behaviors: ${context.behaviors.join('; ')}`
    );
  }

  return parts.join('\n\n');
}
