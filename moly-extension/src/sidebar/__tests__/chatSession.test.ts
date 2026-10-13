import { describe, it, expect, beforeEach } from 'vitest';
import { chatStarted, markChatStarted } from '../chatSession';

describe('chat session marker', () => {
  beforeEach(() => sessionStorage.clear());

  it('is false before a message is sent and true after, for this panel only', () => {
    expect(chatStarted()).toBe(false);
    markChatStarted();
    expect(chatStarted()).toBe(true);
    expect(localStorage.getItem('moly_chat_started')).toBeNull(); // nothing that outlives the panel
    sessionStorage.clear(); // the panel is closed
    expect(chatStarted()).toBe(false);
  });
});
