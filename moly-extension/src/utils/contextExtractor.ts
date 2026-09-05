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

  // Generic phrases to filter out
  const genericPhrases = [
    'they said',
    'i think',
    'they mentioned',
    'i said',
    'they said they',
    'you know',
    'like',
    'i guess',
    'kind of',
    'sort of',
    'i mean',
  ];

  // Process sentences in reverse (prioritize recent ones)
  for (let i = sentences.length - 1; i >= 0; i--) {
    const sentence = sentences[i];
    const trimmed = sentence.trim();
    const lower = trimmed.toLowerCase();

    // Skip if too short (less than 8 words usually means low quality)
    const wordCount = trimmed.split(/\s+/).length;
    if (wordCount < 4) continue;

    // Skip if starts with generic phrase
    const startsWithGeneric = genericPhrases.some(phrase =>
      lower.startsWith(phrase)
    );
    if (startsWithGeneric) continue;

    // Skip if contains multiple generic phrases (noise)
    const genericCount = genericPhrases.filter(phrase =>
      lower.includes(phrase)
    ).length;
    if (genericCount >= 2) continue;

    // Check if sentence mentions contact and contains keyword
    const mentionsContact =
      lower.includes(contactName.toLowerCase()) ||
      lower.includes('they') ||
      lower.includes('she') ||
      lower.includes('he') ||
      lower.includes('their');

    const hasKeyword = keywords.some(kw =>
      lower.includes(kw.toLowerCase())
    );

    if (mentionsContact && hasKeyword) {
      const cleaned = trimmed
        .replace(/^[\s.!?]+|[\s.!?]+$/g, '')
        .replace(/^(i think |you know |like |i mean |they said |they mentioned )/i, '');

      // Avoid duplicates
      if (!extracted.includes(cleaned)) {
        extracted.push(cleaned);
      }
    }

    if (extracted.length >= 5) break; // Stop after finding 5
  }

  return extracted;
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
