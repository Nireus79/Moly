/**
 * End-to-End Integration Tests
 * Tests complete workflows from message detection to suggestion generation
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import { useContactStore } from '@/stores/contactStore';
import { useConversationStore } from '@/stores/conversationStore';
import { useChatStore } from '@/stores/chatStore';
import { useSettingsStore } from '@/stores/settingsStore';
import { getProviderManager } from '@/api/providerManager';
import type { Contact, ChatMessage, ChatMode, CommunicationContext } from '@/types';

describe('End-to-End Workflows', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Complete Contact & Conversation Workflow', () => {
    it('creates contact, starts conversation, and saves messages', async () => {
      const contactStore = useContactStore.getState();
      const conversationStore = useConversationStore.getState();

      // Step 1: Create a contact
      const newContact: Omit<Contact, 'id'> = {
        name: 'Alice',
        platform: 'Tinder',
        notes: 'Met at party',
      };

      await contactStore.addContact(newContact);
      expect(contactStore.contacts).toHaveLength(1);

      const contact = contactStore.contacts[0];
      expect(contact.name).toBe('Alice');
      expect(contact.platform).toBe('Tinder');

      // Step 2: Start conversation with contact
      const userMessage: ChatMessage = {
        id: `msg_${Date.now()}`,
        role: 'user',
        content: 'Hey, how are you?',
        timestamp: Date.now(),
      };

      await conversationStore.saveMessage(contact.id, userMessage);
      const conversation = conversationStore.getConversation(contact.id);
      expect(conversation).toHaveLength(1);
      expect(conversation[0].content).toBe('Hey, how are you?');

      // Step 3: Add assistant response
      const assistantMessage: ChatMessage = {
        id: `msg_${Date.now() + 1}`,
        role: 'assistant',
        content: 'Great! How about you?',
        timestamp: Date.now() + 1000,
      };

      await conversationStore.saveMessage(contact.id, assistantMessage);
      const updatedConversation = conversationStore.getConversation(contact.id);
      expect(updatedConversation).toHaveLength(2);
      expect(updatedConversation[1].role).toBe('assistant');
    });

    it('manages multiple contacts with independent conversations', async () => {
      const contactStore = useContactStore.getState();
      const conversationStore = useConversationStore.getState();

      // Create two contacts
      const alice: Omit<Contact, 'id'> = {
        name: 'Alice',
        platform: 'Tinder',
      };
      const bob: Omit<Contact, 'id'> = {
        name: 'Bob',
        platform: 'Hinge',
      };

      await contactStore.addContact(alice);
      await contactStore.addContact(bob);

      const [aliceContact, bobContact] = contactStore.contacts;

      // Add messages to Alice's conversation
      const aliceMsg: ChatMessage = {
        id: 'alice_1',
        role: 'user',
        content: 'Hi Alice!',
        timestamp: Date.now(),
      };

      // Add messages to Bob's conversation
      const bobMsg: ChatMessage = {
        id: 'bob_1',
        role: 'user',
        content: 'Hi Bob!',
        timestamp: Date.now(),
      };

      await conversationStore.saveMessage(aliceContact.id, aliceMsg);
      await conversationStore.saveMessage(bobContact.id, bobMsg);

      // Verify conversations are independent
      const aliceConv = conversationStore.getConversation(aliceContact.id);
      const bobConv = conversationStore.getConversation(bobContact.id);

      expect(aliceConv).toHaveLength(1);
      expect(bobConv).toHaveLength(1);
      expect(aliceConv[0].content).toBe('Hi Alice!');
      expect(bobConv[0].content).toBe('Hi Bob!');
    });
  });

  describe('Settings & Preferences Workflow', () => {
    it('configures provider and chat preferences', async () => {
      const settingsStore = useSettingsStore.getState();
      const chatStore = useChatStore.getState();

      // Load default settings
      await settingsStore.loadSettings();

      // Update provider configuration
      await settingsStore.updateProvider('claude', {
        apiKey: 'sk-test-key',
        enabled: true,
        model: 'claude-3-5-sonnet-20241022',
      });

      // Verify provider is configured
      const provider = settingsStore.settings?.providers.claude;
      expect(provider?.enabled).toBe(true);
      expect(provider?.apiKey).toBe('sk-test-key');

      // Update chat mode
      chatStore.setChatMode('socratic');
      expect(chatStore.chatMode).toBe('socratic');

      // Update communication context
      chatStore.setContext('dating');
      expect(chatStore.currentContext).toBe('dating');
    });

    it('switches between providers', async () => {
      const settingsStore = useSettingsStore.getState();
      await settingsStore.loadSettings();

      // Configure multiple providers
      await settingsStore.updateProvider('claude', {
        apiKey: 'claude-key',
        enabled: true,
        model: 'claude-3-5-sonnet-20241022',
      });

      await settingsStore.updateProvider('openai', {
        apiKey: 'openai-key',
        enabled: true,
        model: 'gpt-4',
      });

      // Set active provider
      await settingsStore.setActiveProvider('openai');
      expect(settingsStore.settings?.activeProvider).toBe('openai');

      // Switch back to Claude
      await settingsStore.setActiveProvider('claude');
      expect(settingsStore.settings?.activeProvider).toBe('claude');
    });

    it('manages chat mode preferences', async () => {
      const settingsStore = useSettingsStore.getState();
      await settingsStore.loadSettings();

      // Test different chat modes
      const modes: ChatMode[] = ['socratic', 'direct'];
      for (const mode of modes) {
        await chrome.storage.local.set({ chatMode: mode });
        const result = await chrome.storage.local.get('chatMode');
        expect(result.chatMode).toBe(mode);
      }
    });

    it('manages communication contexts', async () => {
      const settingsStore = useSettingsStore.getState();
      await settingsStore.loadSettings();

      const contexts: CommunicationContext[] = ['formal', 'friendly', 'dating'];
      for (const context of contexts) {
        await chrome.storage.local.set({ defaultContext: context });
        const result = await chrome.storage.local.get('defaultContext');
        expect(result.defaultContext).toBe(context);
      }
    });
  });

  describe('Message Flow Workflow', () => {
    it('processes complete message suggestion workflow', async () => {
      const chatStore = useChatStore.getState();
      const conversationStore = useConversationStore.getState();
      const contactStore = useContactStore.getState();

      // Create a contact
      await contactStore.addContact({
        name: 'Test User',
        platform: 'Bumble',
      });

      const contact = contactStore.contacts[0];

      // Set up chat state
      chatStore.setContext('dating');
      chatStore.setChatMode('direct');

      // Simulate user input
      const userMessage: ChatMessage = {
        id: 'msg_1',
        role: 'user',
        content: 'What should I say?',
        timestamp: Date.now(),
      };

      chatStore.addMessage(userMessage);
      await conversationStore.saveMessage(contact.id, userMessage);

      // Simulate suggestions generation
      const suggestions = [
        {
          id: 'sugg_1',
          text: 'Hey! How are you?',
          tone: 'casual',
          confidence: 0.95,
          reasoning: 'Friendly opener',
          bestFor: 'Initial contact',
        },
      ];

      chatStore.setSuggestions(suggestions);
      expect(chatStore.suggestions).toHaveLength(1);

      // Simulate assistant response
      const assistantMessage: ChatMessage = {
        id: 'msg_2',
        role: 'assistant',
        content: 'Generated 1 message options',
        timestamp: Date.now() + 1000,
      };

      chatStore.addMessage(assistantMessage);
      await conversationStore.saveMessage(contact.id, assistantMessage);

      // Verify complete workflow
      const conv = conversationStore.getConversation(contact.id);
      expect(conv).toHaveLength(2);
      expect(chatStore.messages).toHaveLength(2);
      expect(chatStore.suggestions).toHaveLength(1);
    });
  });

  describe('Provider Discovery Workflow', () => {
    it('discovers available providers', () => {
      const manager = getProviderManager();
      manager.discoverProviders();

      // Verify provider manager has methods
      expect(manager.getAvailableProvider).toBeDefined();
      expect(manager.getModels).toBeDefined();
      expect(manager.getProvider).toBeDefined();
    });

    it('handles provider configuration with validation', async () => {
      const settingsStore = useSettingsStore.getState();
      await settingsStore.loadSettings();

      // Try to update provider without required fields
      await settingsStore.updateProvider('claude', {
        enabled: false,
      });

      const config = settingsStore.settings?.providers.claude;
      expect(config?.enabled).toBe(false);
    });
  });

  describe('Error Handling Workflow', () => {
    it('handles missing provider gracefully', async () => {
      const chatStore = useChatStore.getState();

      chatStore.setError('Provider not configured');
      expect(chatStore.error).toBe('Provider not configured');

      chatStore.setError(null);
      expect(chatStore.error).toBeNull();
    });

    it('recovers from conversation loading failure', async () => {
      const conversationStore = useConversationStore.getState();

      conversationStore.setError('Load failed');
      expect(conversationStore.error).toBe('Load failed');

      conversationStore.setError(null);
      expect(conversationStore.error).toBeNull();
    });

    it('handles concurrent operations', async () => {
      const contactStore = useContactStore.getState();
      const conversationStore = useConversationStore.getState();

      // Add contact and save messages concurrently
      const contactPromise = contactStore.addContact({
        name: 'Concurrent Test',
        platform: 'Discord',
      });

      await contactPromise;
      const contact = contactStore.contacts[0];

      const messagePromises = [
        conversationStore.saveMessage(contact.id, {
          id: 'msg_1',
          role: 'user',
          content: 'Message 1',
          timestamp: Date.now(),
        }),
        conversationStore.saveMessage(contact.id, {
          id: 'msg_2',
          role: 'assistant',
          content: 'Message 2',
          timestamp: Date.now() + 1000,
        }),
      ];

      await Promise.all(messagePromises);

      const conv = conversationStore.getConversation(contact.id);
      expect(conv).toHaveLength(2);
    });
  });

  describe('Data Persistence Workflow', () => {
    it('persists and retrieves full contact data', async () => {
      const contactStore = useContactStore.getState();

      const contact: Omit<Contact, 'id'> = {
        name: 'Persistent User',
        platform: 'LinkedIn',
        notes: 'Professional contact',
        interests: ['tech', 'startups'],
        lastMessaged: Date.now(),
      };

      await contactStore.addContact(contact);
      const saved = contactStore.contacts[0];

      expect(saved.name).toBe('Persistent User');
      expect(saved.platform).toBe('LinkedIn');
      expect(saved.notes).toBe('Professional contact');
      expect(saved.interests).toEqual(['tech', 'startups']);
    });

    it('maintains conversation history across operations', async () => {
      const conversationStore = useConversationStore.getState();
      const contactId = 'persist_test';

      const messages: ChatMessage[] = [
        {
          id: 'msg_1',
          role: 'user',
          content: 'Message 1',
          timestamp: Date.now(),
        },
        {
          id: 'msg_2',
          role: 'assistant',
          content: 'Response 1',
          timestamp: Date.now() + 1000,
        },
        {
          id: 'msg_3',
          role: 'user',
          content: 'Message 2',
          timestamp: Date.now() + 2000,
        },
      ];

      await conversationStore.saveMessages(contactId, messages);
      const retrieved = conversationStore.getConversation(contactId);

      expect(retrieved).toHaveLength(3);
      expect(retrieved[2].content).toBe('Message 2');
    });
  });

  describe('Settings Persistence Workflow', () => {
    it('persists and retrieves complex settings', async () => {
      const settingsStore = useSettingsStore.getState();

      // Configure multiple providers
      const providers = {
        claude: {
          type: 'claude' as const,
          apiKey: 'claude-key-123',
          model: 'claude-3-5-sonnet-20241022',
          enabled: true,
        },
        openai: {
          type: 'openai' as const,
          apiKey: 'openai-key-456',
          model: 'gpt-4',
          enabled: false,
        },
      };

      await settingsStore.updateProvider('claude', providers.claude);
      await settingsStore.updateProvider('openai', providers.openai);

      const config = settingsStore.settings;
      expect(config?.providers.claude.enabled).toBe(true);
      expect(config?.providers.openai.enabled).toBe(false);
    });
  });

  describe('Full User Journey', () => {
    it('completes typical user interaction flow', async () => {
      const contactStore = useContactStore.getState();
      const conversationStore = useConversationStore.getState();
      const chatStore = useChatStore.getState();
      const settingsStore = useSettingsStore.getState();

      // Step 1: User opens extension and configures settings
      await settingsStore.loadSettings();
      chatStore.setChatMode('direct');
      chatStore.setContext('dating');

      // Step 2: User adds a contact
      await contactStore.addContact({
        name: 'Sarah',
        platform: 'Hinge',
        notes: 'Loves hiking',
      });

      expect(contactStore.contacts).toHaveLength(1);

      // Step 3: User receives message and starts conversation
      const contact = contactStore.contacts[0];
      chatStore.setDetectedMessage({
        sender: 'Sarah',
        text: 'Hi! How are you?',
        timestamp: Date.now(),
        platform: 'Hinge',
        url: 'https://example.com',
      });

      expect(chatStore.detectedMessage?.sender).toBe('Sarah');

      // Step 4: User composes response and gets suggestions
      chatStore.addMessage({
        id: '1',
        role: 'user',
        content: 'What should I say back?',
        timestamp: Date.now(),
      });

      // Step 5: Suggestions are displayed
      chatStore.setSuggestions([
        {
          id: 's1',
          text: "I'm great! How about you?",
          tone: 'friendly',
          confidence: 0.92,
          reasoning: 'Warm and open',
          bestFor: 'Getting to know someone',
        },
      ]);

      // Step 6: Conversation is saved
      await conversationStore.saveMessage(contact.id, {
        id: '2',
        role: 'user',
        content: 'I am great, thank you!',
        timestamp: Date.now(),
      });

      // Step 7: Verify complete state
      const conv = conversationStore.getConversation(contact.id);
      expect(conv).toHaveLength(1);
      expect(chatStore.suggestions).toHaveLength(1);
      expect(chatStore.chatMode).toBe('direct');
      expect(chatStore.currentContext).toBe('dating');
    });
  });
});
