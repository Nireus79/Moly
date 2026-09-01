/**
 * Service Worker for Moly Extension
 * Handles background tasks and message routing
 */

import { getProviderManager } from '@/api/providerManager';
import type { ExtensionSettings } from '@/stores/settingsStore';

chrome.runtime.onInstalled.addListener(() => {
  console.log('Moly extension installed');
});

// Handle extension icon click (open sidebar as floating window)
chrome.action.onClicked.addListener(async () => {
  const sidebarUrl = chrome.runtime.getURL('sidebar/sidebar.html');

  // Open as floating window: ~400px wide, 600px tall
  await chrome.windows.create({
    url: sidebarUrl,
    type: 'popup',
    width: 400,
    height: 700,
    left: 0,
    top: 0,
  });

  console.log('Sidebar opened as floating window');
});

// Listen for messages from sidebar
chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
  console.log('Background received message:', request.type);

  if (request.type === 'GENERATE_SUGGESTIONS') {
    generateSuggestions(request.data)
      .then((suggestions) => {
        sendResponse({ success: true, suggestions });
      })
      .catch((error) => {
        sendResponse({
          success: false,
          error: error instanceof Error ? error.message : 'Unknown error',
        });
      });
    return true; // Keep channel open for async response
  }
  return false;
});

async function generateSuggestions(data: any): Promise<string[]> {
  try {
    const settings = await getSettings();
    if (!settings) {
      throw new Error('No settings found. Please configure a provider in Settings.');
    }

    const activeProviderType = settings.activeProvider;
    const providerConfig = settings.providers[activeProviderType];

    if (!providerConfig?.enabled) {
      throw new Error(`Provider ${activeProviderType} is not enabled. Please configure it in Settings.`);
    }

    // Configure the active provider
    const manager = getProviderManager();
    const configured = await manager.configureProvider({
      type: activeProviderType,
      apiKey: providerConfig.apiKey,
      baseUrl: providerConfig.baseUrl,
      model: providerConfig.model,
    });

    if (!configured) {
      throw new Error(`Failed to configure ${activeProviderType} provider.`);
    }

    const provider = manager.getActiveProvider();
    if (!provider) {
      throw new Error('No active provider available.');
    }

    // Build context from conversation history
    const conversationContext = buildConversationContext(data.conversationHistory || []);
    const mode = data.mode || 'direct';
    const communicationContext = data.communicationContext || 'friendly';
    const userMessage = data.userMessage || '';
    const recipientContext = data.context || 'Unknown person';

    console.log(`Generating ${mode} suggestions with ${activeProviderType} for:`, {
      recipient: recipientContext,
      context: communicationContext,
      mode,
      messageLength: userMessage.length,
      conversationLength: data.conversationHistory?.length || 0,
    });

    // Generate suggestions with full context
    const suggestions = await provider.generateSuggestions(
      userMessage,
      recipientContext,
      communicationContext,
    );

    console.log(`Generated ${suggestions.length} suggestions with mode: ${mode}`);
    return suggestions;
  } catch (error) {
    console.error('Error generating suggestions:', error);
    throw error;
  }
}

function buildConversationContext(messages: any[]): string {
  if (!messages || messages.length === 0) {
    return '';
  }

  return messages
    .filter((m) => m.type !== 'suggestion') // Skip suggestions
    .map((m) => {
      const sender = m.type === 'user' ? 'You' : m.type === 'incoming' ? 'Them' : 'Moly';
      return `${sender}: ${m.content}`;
    })
    .join('\n');
}

async function getSettings(): Promise<ExtensionSettings | null> {
  return new Promise((resolve) => {
    chrome.storage.local.get('settings', (result) => {
      resolve(result.settings || null);
    });
  });
}
