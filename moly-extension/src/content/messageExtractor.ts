/**
 * Message Extraction
 * Extracts message text, sender, and metadata from DOM elements
 */

export interface ExtractedMessage {
  sender: string;
  text: string;
  timestamp: number;
  isIncoming: boolean;
}

/**
 * Extract text content from a DOM node, filtering out UI elements
 */
export function extractText(element: Element): string {
  // Clone element to avoid modifying original
  const clone = element.cloneNode(true) as Element;

  // Remove script tags
  clone.querySelectorAll('script').forEach((el) => el.remove());

  // Get text content
  let text = clone.textContent || '';

  // Clean up whitespace
  text = text
    .replace(/\s+/g, ' ') // Multiple spaces to single space
    .replace(/[\r\n]+/g, ' ') // Newlines to spaces
    .trim();

  // Remove common status indicators
  text = text
    .replace(/Delivered|Seen|Read|Sending/gi, '')
    .replace(/\d{1,2}:\d{2}\s*(AM|PM|am|pm)?/g, '') // Remove timestamps
    .trim();

  return text;
}

/**
 * Extract sender name from message element
 */
export function extractSenderName(element: Element, platformSelectors?: string[]): string {
  // Try platform-specific selectors first if provided
  if (platformSelectors && platformSelectors.length > 0) {
    for (const selector of platformSelectors) {
      try {
        const el = element.querySelector(selector);
        if (el?.textContent) {
          const text = el.textContent.trim();
          if (text && text.length > 0) {
            return text;
          }
        }
      } catch {
        // Invalid selector, skip
        continue;
      }
    }
  }

  // Try common sender name selectors
  const selectors = [
    '[data-sender]',
    '[data-from]',
    '.sender-name',
    '.message-sender',
    '.participant-name',
  ];

  for (const selector of selectors) {
    try {
      const el = element.querySelector(selector);
      if (el?.textContent) {
        return el.textContent.trim();
      }
    } catch {
      continue;
    }
  }

  // Fallback: try to extract from title or aria-label
  const title = element.getAttribute('title');
  if (title) return title;

  const ariaLabel = element.getAttribute('aria-label');
  if (ariaLabel) {
    const match = ariaLabel.match(/from (.+)/i);
    if (match) return match[1];
  }

  return 'Unknown';
}

/**
 * Determine if message is incoming (from other user)
 */
export function isIncomingMessage(element: Element, currentUserId?: string): boolean {
  // Check for common outgoing message classes
  const outgoingClasses = [
    'outgoing',
    'sent',
    'own-message',
    'my-message',
    'user-message',
  ];

  for (const cls of outgoingClasses) {
    if (element.classList.contains(cls)) {
      return false;
    }
  }

  // Check for incoming message classes
  const incomingClasses = ['incoming', 'received', 'other-message', 'them-message'];

  for (const cls of incomingClasses) {
    if (element.classList.contains(cls)) {
      return true;
    }
  }

  // Check data attributes
  const senderId = element.getAttribute('data-sender-id') || element.getAttribute('data-user-id');
  if (senderId && currentUserId && senderId !== currentUserId) {
    return true;
  }

  // Default heuristic: if sender name is not the current user
  const sender = extractSenderName(element);
  if (currentUserId && sender !== currentUserId && sender !== 'You') {
    return true;
  }

  return false;
}

/**
 * Extract timestamp from message element
 */
export function extractTimestamp(element: Element): number {
  // Try to find datetime attribute
  const timeElement = element.querySelector('[datetime], time, [data-time]');
  if (timeElement) {
    const dateTime = timeElement.getAttribute('datetime') || timeElement.textContent;
    if (dateTime) {
      const timestamp = new Date(dateTime).getTime();
      if (!isNaN(timestamp)) return timestamp;
    }
  }

  // Check for data-timestamp
  const timestamp = element.getAttribute('data-timestamp');
  if (timestamp && !isNaN(Number(timestamp))) {
    return Number(timestamp);
  }

  // Default to current time
  return Date.now();
}

/**
 * Check if element is a valid message container
 */
export function isValidMessageElement(element: Element): boolean {
  // Must have text content
  const text = element.textContent?.trim();
  if (!text || text.length < 3) return false;

  // Must not be too large (avoid extracting entire conversations)
  if (text.length > 10000) return false;

  // Check if it's not a UI element
  const classList = element.className.toLowerCase();
  const invalidPatterns = ['button', 'link', 'header', 'nav', 'menu', 'spinner', 'loader'];
  if (invalidPatterns.some((p) => classList.includes(p))) {
    return false;
  }

  return true;
}

/**
 * Check if message is new (within last 5 seconds)
 */
export function isNewMessage(timestamp: number): boolean {
  const ageMs = Date.now() - timestamp;
  return ageMs < 5000; // 5 seconds
}
