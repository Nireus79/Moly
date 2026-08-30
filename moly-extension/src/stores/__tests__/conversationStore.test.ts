/**
 * Tests for Conversation Store
 */

import { describe, it, expect, beforeEach } from 'vitest';
import { useConversationStore } from '@/stores/conversationStore';
import type { ChatMessage } from '@/types';

describe('Conversation Store', () => {
  beforeEach(() => {
    useConversationStore.getState().setLoading(false);
    useConversationStore.getState().setError(null);
  });

  it('initializes with empty state', () => {
    const store = useConversationStore.getState();
    expect(store.conversations).toEqual({});
    expect(store.currentContactId).toBeNull();
    expect(store.isLoading).toBe(false);
    expect(store.error).toBeNull();
  });

  it('sets current contact', () => {
    const store = useConversationStore.getState();
    store.setCurrentContact('contact_123');
    expect(store.currentContactId).toBe('contact_123');
  });

  it('saves single message', async () => {
    const store = useConversationStore.getState();
    const message: ChatMessage = {
      id: 'msg_1',
      role: 'user',
      content: 'Hello',
      timestamp: Date.now(),
    };

    await store.saveMessage('contact_123', message);
    const conversation = store.getConversation('contact_123');
    expect(conversation).toHaveLength(1);
    expect(conversation[0].content).toBe('Hello');
  });

  it('saves multiple messages', async () => {
    const store = useConversationStore.getState();
    const messages: ChatMessage[] = [
      {
        id: 'msg_1',
        role: 'user',
        content: 'Hello',
        timestamp: Date.now(),
      },
      {
        id: 'msg_2',
        role: 'assistant',
        content: 'Hi there!',
        timestamp: Date.now() + 1000,
      },
    ];

    await store.saveMessages('contact_456', messages);
    const conversation = store.getConversation('contact_456');
    expect(conversation).toHaveLength(2);
    expect(conversation[0].role).toBe('user');
    expect(conversation[1].role).toBe('assistant');
  });

  it('loads conversation from storage', async () => {
    const store = useConversationStore.getState();
    const contactId = 'contact_789';

    const messages: ChatMessage[] = [
      {
        id: 'msg_1',
        role: 'user',
        content: 'Test message',
        timestamp: Date.now(),
      },
    ];

    await store.saveMessages(contactId, messages);
    await store.loadConversation(contactId);

    const conversation = store.getConversation(contactId);
    expect(conversation).toHaveLength(1);
    expect(conversation[0].content).toBe('Test message');
  });

  it('clears conversation', async () => {
    const store = useConversationStore.getState();
    const contactId = 'contact_clear';

    const message: ChatMessage = {
      id: 'msg_1',
      role: 'user',
      content: 'To be deleted',
      timestamp: Date.now(),
    };

    await store.saveMessage(contactId, message);
    expect(store.getConversation(contactId)).toHaveLength(1);

    await store.clearConversation(contactId);
    expect(store.getConversation(contactId)).toHaveLength(0);
  });

  it('returns empty array for non-existent conversation', () => {
    const store = useConversationStore.getState();
    const conversation = store.getConversation('non_existent');
    expect(conversation).toEqual([]);
  });

  it('handles multiple conversations independently', async () => {
    const store = useConversationStore.getState();

    const msg1: ChatMessage = {
      id: 'msg_1',
      role: 'user',
      content: 'Contact 1 message',
      timestamp: Date.now(),
    };

    const msg2: ChatMessage = {
      id: 'msg_2',
      role: 'user',
      content: 'Contact 2 message',
      timestamp: Date.now(),
    };

    await store.saveMessage('contact_a', msg1);
    await store.saveMessage('contact_b', msg2);

    expect(store.getConversation('contact_a')).toHaveLength(1);
    expect(store.getConversation('contact_b')).toHaveLength(1);
    expect(store.getConversation('contact_a')[0].content).toBe('Contact 1 message');
    expect(store.getConversation('contact_b')[0].content).toBe('Contact 2 message');
  });

  it('sets loading state', () => {
    const store = useConversationStore.getState();
    store.setLoading(true);
    expect(store.isLoading).toBe(true);
    store.setLoading(false);
    expect(store.isLoading).toBe(false);
  });

  it('sets error state', () => {
    const store = useConversationStore.getState();
    store.setError('Test error');
    expect(store.error).toBe('Test error');
    store.setError(null);
    expect(store.error).toBeNull();
  });

  it('appends messages to existing conversation', async () => {
    const store = useConversationStore.getState();
    const contactId = 'contact_append';

    const msg1: ChatMessage = {
      id: 'msg_1',
      role: 'user',
      content: 'First message',
      timestamp: Date.now(),
    };

    await store.saveMessage(contactId, msg1);
    expect(store.getConversation(contactId)).toHaveLength(1);

    const msg2: ChatMessage = {
      id: 'msg_2',
      role: 'assistant',
      content: 'Second message',
      timestamp: Date.now() + 1000,
    };

    await store.saveMessage(contactId, msg2);
    const conversation = store.getConversation(contactId);
    expect(conversation).toHaveLength(2);
    expect(conversation[1].content).toBe('Second message');
  });

  it('handles message with optional fields', async () => {
    const store = useConversationStore.getState();
    const message: ChatMessage = {
      id: 'msg_optional',
      role: 'user',
      content: 'Message with extras',
      timestamp: Date.now(),
      tone: 'friendly',
      confidence: 0.95,
    };

    await store.saveMessage('contact_optional', message);
    const saved = store.getConversation('contact_optional')[0];
    expect(saved.tone).toBe('friendly');
    expect(saved.confidence).toBe(0.95);
  });
});
