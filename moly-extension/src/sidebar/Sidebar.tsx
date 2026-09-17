import React, { useEffect, useState } from 'react';
import { useChatStore } from '@/stores/chatStore';
import { useSettingsStore, initializeSettings } from '@/stores/settingsStore';
import { useClarificationStore } from '@/stores/clarificationStore';
import { useAuthStore } from '@/stores/authStore';
import { useAuth } from '@/hooks/useAuth';
import { useAboutMe } from '@/hooks/useAboutMe';
import { ChatInterface } from './components';
import IncomingMessageInput from './components/IncomingMessageInput';
import { Settings } from '@/settings/Settings';
import { ClarificationAPI } from '@/api/clarificationAPI';
import { extractContactContextFromConversation, type ExtractedContactContext } from '@/utils/contextExtractor';
import { profileAPI } from '@/api/profileAPI';
import { getFirst, hasItems, getAt, getProperty } from '@/utils/safeAccess';
import type { Message } from './components';
import type { CommunicationContext, ChatMode, ConversationData } from '@/types';
import './sidebar.css';
import './components/suggestions-v2.css';

interface Contact {
  id: string;
  name: string;
  platform: string;
  relationship: string;
}

export const Sidebar: React.FC = () => {
  const [currentConversation, setCurrentConversation] = useState<ConversationData | null>(null);
  const [conversationMembers, setConversationMembers] = useState<ConversationData['members']>([]);
  const [conversationContext, setConversationContext] = useState<any>(null);
  const [showNewConversationModal, setShowNewConversationModal] = useState(false);
  const [showContactManager, setShowContactManager] = useState(false);
  const [showReflection, setShowReflection] = useState(false);
  const [conversationMessages, setConversationMessages] = useState<Message[]>([]);
  const [suggestions, setSuggestions] = useState<string[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [processingStage, setProcessingStage] = useState<string>('');
  const [processingSeconds, setProcessingSeconds] = useState(0);
  const [error, setError] = useState<string | null>(null);
  const [chatMode, setChatMode] = useState<ChatMode>('direct');
  const [context, setContext] = useState<CommunicationContext>('friendly');
  const [showSettings, setShowSettings] = useState(false);
  const [activeProvider, setActiveProvider] = useState<string>('');
  const [extractedContactContext, setExtractedContactContext] = useState<ExtractedContactContext | undefined>(undefined);
  const [isExtractingContext, setIsExtractingContext] = useState(false);
  const [showMeProfile, setShowMeProfile] = useState(false);
  const [meProfile, setMeProfile] = useState<any>(null);
  const [incomingMessage, setIncomingMessage] = useState<string | null>(null);

  // Context binding modals
  const [showAboutMeModal, setShowAboutMeModal] = useState(false);
  const [showConversationSelector, setShowConversationSelector] = useState(false);
  const [showContactSelector, setShowContactSelector] = useState(false);
  const [showConflictModal, setShowConflictModal] = useState(false);
  const [pendingConflicts, setPendingConflicts] = useState<any[]>([]);
  const [selectedContacts, setSelectedContacts] = useState<string[]>([]);
  const [availableConversations, setAvailableConversations] = useState<ConversationData[]>([]);
  const [pendingMessage, setPendingMessage] = useState<string | null>(null);

  const { settings, loadSettings } = useSettingsStore();
  const { session } = useAuth();
  const { profile: aboutMe, hasProfile, saveProfile, isLoading: profileIsLoading } = useAboutMe();
  const { isOpen: isClarificationOpen, setOpen: setClarificationOpen, clear: clearClarification } = useClarificationStore();
  const clearAuth = useAuthStore((state) => state.clearAuth);

  // Track processing time for slow systems (like Ollama on old hardware)
  useEffect(() => {
    if (!isLoading) {
      setProcessingSeconds(0);
      return;
    }

    const timer = setInterval(() => {
      setProcessingSeconds(s => s + 1);
    }, 1000);

    return () => clearInterval(timer);
  }, [isLoading]);

  useEffect(() => {
    const initializeApp = async () => {
      try {
        initializeSettings();
        await loadSettings();

        // Load context from Phase 1.2 backend
        await loadBackendContext();

        await loadConversationHistory();
        await loadMeProfile();
      } catch (err) {
        console.error('[Sidebar] Failed to initialize app:', err);
      }
    };

    initializeApp();
  }, [loadSettings]);

  // Show About Me modal on first load if user has no profile
  useEffect(() => {
    if (!hasProfile && !profileIsLoading) {
      console.info('[Sidebar] ✓ About Me profile missing - showing modal on startup');
      setShowAboutMeModal(true);
    }
  }, [hasProfile, profileIsLoading]);

  const loadBackendContext = async () => {
    try {
      console.log('[Sidebar] Loading context from Phase 1.2 backend...');

      // Load AboutMe profile from backend
      try {
        const aboutMe = await profileAPI.getAboutMe();
        if (aboutMe) {
          setMeProfile(aboutMe);
          console.log('[Sidebar] ✓ AboutMe loaded from backend');
        } else {
          console.log('[Sidebar] ℹ No AboutMe profile yet (first login or skipped)');
        }
      } catch (err) {
        console.debug('[Sidebar] AboutMe load failed (may be expected on first login):', err);
      }

      // Load conversations from backend
      try {
        const conversations = await profileAPI.getConversations();
        setAvailableConversations(conversations);
        console.log('[Sidebar] ✓ Conversations loaded:', conversations.length);
      } catch (err) {
        console.warn('[Sidebar] Failed to load conversations:', err);
      }

      // Load contacts from backend
      try {
        const contacts = await profileAPI.getContacts();
        console.log('[Sidebar] ✓ Contacts loaded:', contacts.length);
      } catch (err) {
        console.warn('[Sidebar] Failed to load contacts:', err);
      }
    } catch (err) {
      console.error('[Sidebar] loadBackendContext failed:', err);
    }
  };

  const loadMeProfile = async () => {
    try {
      // Try to load from backend first
      try {
        const profileAPI = require('@/api/profileAPI').profileAPI;
        const aboutMe = await profileAPI.getAboutMe();
        if (aboutMe) {
          setMeProfile(aboutMe);
          await chrome.storage.local.set({ meProfile: aboutMe });
          return;
        }
      } catch (backendErr) {
        console.debug('[Sidebar] Backend profile load failed, using local:', backendErr);
      }

      // Fallback to local storage
      const result = await chrome.storage.local.get('meProfile');
      if (result.meProfile) {
        setMeProfile(result.meProfile);
      } else {
        // Create default Me profile
        const defaultProfile = {
          id: 'me',
          name: 'Me',
          description: '',
          communicationStyle: '',
          values: '',
          goals: '',
          patterns: '',
          notes: '',
        };
        await chrome.storage.local.set({ meProfile: defaultProfile });
        setMeProfile(defaultProfile);
      }
    } catch (err) {
      console.error('[Sidebar] Failed to load Me profile:', err);
    }
  };

  // Fetch conversation context when conversation is selected
  // CRITICAL: Also refresh contact data to use latest reflected insights
  useEffect(() => {
    if (!currentConversation) {
      setConversationContext(null);
      setConversationMembers([]);
      return;
    }

    const loadContext = async () => {
      try {
        // Load fresh contact data from storage to get any updated notes from reflections
        console.log('[Sidebar] Refreshing contact data for conversation members...');
        const contactsResult = await chrome.storage.local.get('contacts');
        const contacts = contactsResult.contacts || [];

        // Merge fresh contact data into conversation members (if any)
        const members = currentConversation.members || [];
        const updatedMembers = members.map(member => {
          const freshContact = contacts.find(c => c.id === member.id.toString());
          if (freshContact) {
            return {
              ...member,
              notes: freshContact.notes || member.notes, // Use updated notes from reflection
              platform: freshContact.platform || member.platform,
              relationship: freshContact.relationship || member.relationship,
            };
          }
          return member;
        });

        setConversationMembers(updatedMembers);
        if (updatedMembers.length > 0) {
          console.log('[Sidebar] Updated conversation members with fresh contact data:', updatedMembers);
        }

        // Conversation context loading via ChatInterface
        // Backend sync handled by profileAPI
      } catch (err) {
        console.warn('[Sidebar] Could not load context from backend:', err);
        // Fall back to local data in currentConversation
      }
    };

    loadContext();
  }, [currentConversation]);

  const loadConversationHistory = async () => {
    try {
      const result = await chrome.storage.local.get('conversations');
      if (result.conversations && Array.isArray(result.conversations)) {
        if (result.conversations.length > 0) {
          const conversation = result.conversations[0];
          setConversationMessages(conversation.messages || []);
          setChatMode(conversation.settings?.mode || 'direct');
          setContext(conversation.settings?.context || 'friendly');
        }
      }
    } catch (err) {
      console.error('Failed to load conversation history:', err);
    }
  };

  const saveConversationHistory = async (messages: Message[]) => {
    try {
      const result = await chrome.storage.local.get('conversations');
      const conversations = result.conversations || [];

      if (conversations.length === 0) {
        conversations.push({
          id: Date.now().toString(),
          messages,
          settings: { mode: chatMode, context, llmProvider: settings?.activeProvider || 'claude' },
          createdAt: Date.now(),
          updatedAt: Date.now(),
        });
      } else {
        conversations[0] = {
          ...conversations[0],
          messages,
          settings: { mode: chatMode, context, llmProvider: settings?.activeProvider || 'claude' },
          updatedAt: Date.now(),
        };
      }

      await chrome.storage.local.set({ conversations });
    } catch (err) {
      console.error('Failed to save conversation history:', err);
    }
  };

  const handleSendMessage = async (userMessage: string) => {
    if (!userMessage.trim()) return;

    console.info('[Sidebar] ===== CONTEXT BINDING FLOW START =====');
    console.info('[Sidebar] Loading all checks in parallel...');

    // Store message for context binding flow
    setPendingMessage(userMessage);

    // Load all checks in PARALLEL instead of sequential
    const [aboutMeStatus, conversationStatus, contactStatus] = await Promise.all([
      // Check 1: About Me profile
      (async () => {
        try {
          return { hasProfile, needsModal: !hasProfile };
        } catch (err) {
          console.warn('[Sidebar] Error checking About Me:', err);
          return { hasProfile: false, needsModal: true };
        }
      })(),

      // Check 2: Conversation selected + load from backend
      (async () => {
        try {
          if (currentConversation) {
            return { selected: true, conversations: [], needsModal: false };
          }
          // Load conversations from backend in parallel
          const conversations = await profileAPI.getConversations();
          return { selected: false, conversations: conversations as ConversationData[], needsModal: true };
        } catch (err) {
          console.warn('[Sidebar] Failed to load conversations:', err);
          return { selected: !!currentConversation, conversations: [], needsModal: !currentConversation };
        }
      })(),

      // Check 3: Contacts mentioned in message
      (async () => {
        try {
          const messageWords = userMessage.toLowerCase().split(/\s+/);
          const mentionedNames = messageWords.filter(word =>
            word.length > 3 && word[0] === word[0].toUpperCase()
          );
          return { mentionedNames, needsModal: mentionedNames.length > 0 };
        } catch (err) {
          console.warn('[Sidebar] Error detecting contacts:', err);
          return { mentionedNames: [], needsModal: false };
        }
      })(),
    ]);

    console.info('[Sidebar] Parallel checks complete:', {
      aboutMeNeeded: aboutMeStatus.needsModal,
      conversationNeeded: conversationStatus.needsModal,
      contactsNeeded: contactStatus.needsModal,
      conversationsLoaded: conversationStatus.conversations.length
    });

    // Update state with loaded conversations
    if (conversationStatus.conversations.length > 0) {
      setAvailableConversations(conversationStatus.conversations);
    }

    // Show modals in sequence: About Me → Conversation → Contacts
    if (aboutMeStatus.needsModal) {
      console.info('[Sidebar] Showing About Me modal (1/3)');
      setShowAboutMeModal(true);
      return;
    }

    if (conversationStatus.needsModal) {
      console.info('[Sidebar] Showing Conversation selector (2/3)');
      console.info('[Sidebar] Available conversations:', conversationStatus.conversations.length);
      setShowConversationSelector(true);
      return;
    }

    if (contactStatus.needsModal) {
      console.info('[Sidebar] Showing Contact selector (3/3)');
      setShowContactSelector(true);
      return;
    }

    // All context gathered - process immediately
    console.info('[Sidebar] All context ready - processing message');
    await processMessageWithContext(userMessage, []);
  };

  // Helper: Continue to next modal or process message
  const continueContextBinding = async () => {
    if (!pendingMessage) return;

    // Re-run the checks to see what's needed next
    const messageWords = pendingMessage.toLowerCase().split(/\s+/);
    const mentionedNames = messageWords.filter(word =>
      word.length > 3 && word[0] === word[0].toUpperCase()
    );

    // If contacts mentioned, show contact selector
    if (mentionedNames.length > 0) {
      console.info('[Sidebar] Moving to contact selector (3/3)');
      setShowContactSelector(true);
      return;
    }

    // Otherwise process the message
    console.info('[Sidebar] All context complete - processing message');
    setPendingMessage(null);
    await processMessageWithContext(pendingMessage, []);
  };

  // Modal handlers for context binding
  const handleAboutMeSave = async (profile: any) => {
    try {
      console.log('[Sidebar] ✓ About Me saved to local storage (1/3)');
      saveProfile(profile);
      setShowAboutMeModal(false);

      // Try to save to backend (non-blocking) - Issue #19 error visibility
      profileAPI.saveAboutMe({
        communicationStyle: profile.communicationStyle,
        coreValues: profile.coreValues || [],
        tonePreference: profile.tonePreference || '',
        preferences: profile.preferences?.notes ? { notes: profile.preferences.notes } : { notes: '' },
        goals: profile.goals || [],
        patterns: profile.patterns || [],
      }).then(() => {
        console.info('[Sidebar] ✓ About Me synced to backend successfully');
      }).catch(err => {
        const errorMsg = `Failed to save About Me to backend: ${err instanceof Error ? err.message : String(err)}`;
        console.warn('[Sidebar] ⚠️ BACKEND SYNC FAILED (Issue #19): About Me save error:', {
          error: err,
          message: errorMsg,
          profile: { style: profile.communicationStyle, valuesCount: profile.coreValues?.length || 0, goalsCount: profile.goals?.length || 0 }
        });
        setError(errorMsg);
        setTimeout(() => setError(null), 5000);
      });

      // Continue to next modal
      await continueContextBinding();
    } catch (err) {
      console.error('[Sidebar] ERROR saving About Me to local storage:', err);
      const errorMsg = err instanceof Error ? err.message : 'Failed to save About Me';
      setError(errorMsg);
      setTimeout(() => setError(null), 5000);
    }
  };

  const handleConversationSelect = async (conversationId: string | number) => {
    try {
      console.log('[Sidebar] ✓ Conversation selected (2/3)');
      // Handle both string and number IDs for compatibility
      const conv = availableConversations.find(c =>
        c.id === conversationId || String(c.id) === String(conversationId)
      );
      if (conv) {
        setCurrentConversation(conv as ConversationData);
        // Load message history for this conversation
        try {
          const messages = await profileAPI.getMessages((conv as any).backendId || conv.id);
          console.log('[Sidebar] Loaded messages:', messages.length);
        } catch (err) {
          console.warn('[Sidebar] Failed to load messages:', err);
        }
      } else {
        console.warn('[Sidebar] Could not find conversation:', conversationId);
      }
      setShowConversationSelector(false);

      // Continue to next modal
      await continueContextBinding();
    } catch (err) {
      console.error('[Sidebar] Error selecting conversation:', err);
    }
  };

  const handleConversationCreate = async (name: string, type?: string, description?: string) => {
    try {
      console.log('[Sidebar] ✓ Conversation created (2/3)');

      // Create in backend
      const newConv = await profileAPI.createConversation(name, type, description);
      console.log('[Sidebar] Backend conversation response:', newConv);

      if (!newConv || !newConv.id) {
        console.error('[Sidebar] ERROR: Backend did not return conversation ID!', { newConv });
        setError(`Failed to create conversation on backend: No ID returned`);
        return;
      }

      const conversation: ConversationData = {
        id: Date.now().toString(),  // Local ID for storage
        backendId: newConv.id,       // Backend ID for API calls (format: conv_<ts>_<ns>)
        name: newConv.name,
        type: newConv.type,
        description: newConv.description,
        purpose: newConv.purpose,
        members: newConv.members,
        settings: newConv.settings,
        notes: newConv.notes,
        createdAt: newConv.createdAt,
        updatedAt: newConv.updatedAt,
      };

      console.log('[Sidebar] Created conversation with backendId:', newConv.id);

      setCurrentConversation(conversation);

      // Add to available conversations so it shows in future dropdowns
      setAvailableConversations(prev => [...prev, conversation]);

      setShowConversationSelector(false);

      // Continue to next modal
      await continueContextBinding();
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to create conversation';
      console.error('[Sidebar] Error creating conversation:', err);
      setError(`Conversation creation failed: ${errorMsg}. Please try again.`);
      setShowConversationSelector(true);
    }
  };

  const handleContactSelect = async (contactIds: string[]) => {
    try {
      console.log('[Sidebar] ✓ Contacts selected (3/3)');
      setSelectedContacts(contactIds);
      setShowContactSelector(false);

      // Process message with selected contacts
      if (pendingMessage) {
        const msg = pendingMessage;
        setPendingMessage(null);
        console.info('[Sidebar] ===== CONTEXT BINDING COMPLETE - PROCESSING MESSAGE =====');
        await processMessageWithContext(msg, contactIds);
      }
    } catch (err) {
      console.error('[Sidebar] Error selecting contacts:', err);
    }
  };

  // Load contacts when contact selector modal opens (for faster display)
  useEffect(() => {
    if (showContactSelector) {
      const loadContactsAsync = async () => {
        try {
          console.log('[Sidebar] Pre-loading contacts for modal...');
          const contacts = await profileAPI.getContacts();
          // Don't set state - just have it ready for the modal
          console.log('[Sidebar] Contacts pre-loaded:', contacts.length);
        } catch (err) {
          console.warn('[Sidebar] Failed to pre-load contacts:', err);
        }
      };
      loadContactsAsync();
    }
  }, [showContactSelector]);

  const handleContactCreate = async (name: string, relationship?: string) => {
    try {
      console.log('[Sidebar] ✓ Contact created (3/3)');

      // Create in backend
      const newContact = await profileAPI.saveContact({ name, relationship });

      // Add to selected contacts
      const contactIds = [...selectedContacts, newContact.id];
      setSelectedContacts(contactIds);
      setShowContactSelector(false);

      // Process message with new contact
      if (pendingMessage) {
        const msg = pendingMessage;
        setPendingMessage(null);
        console.info('[Sidebar] ===== CONTEXT BINDING COMPLETE - PROCESSING MESSAGE =====');
        await processMessageWithContext(msg, contactIds);
      }
    } catch (err) {
      console.error('[Sidebar] Error creating contact:', err);
    }
  };

  const handleConflictResolved = async (resolutions: Record<string, string>) => {
    try {
      console.log('[Sidebar] ✓ Conflicts resolved (Issue #16)', {
        resolutionCount: Object.keys(resolutions).length,
        resolutions: resolutions
      });

      // Send resolutions to backend endpoint
      const backendErrors: string[] = [];
      for (const [conflictId, resolution] of Object.entries(resolutions)) {
        try {
          const response = await fetch(`${process.env.REACT_APP_BACKEND_URL || 'http://localhost:3000'}/api/v2/conflicts/resolve`, {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              'Authorization': `Bearer ${localStorage.getItem('authToken')}`,
            },
            body: JSON.stringify({
              conflictId: parseInt(conflictId, 10),
              resolution: resolution
            })
          });

          if (!response.ok) {
            backendErrors.push(`Conflict ${conflictId}: ${response.status} ${response.statusText}`);
            console.warn(`[Sidebar] Failed to send resolution for conflict ${conflictId}:`, response.status);
          } else {
            console.log(`[Sidebar] ✓ Resolution persisted for conflict ${conflictId}`);
          }
        } catch (fetchErr) {
          backendErrors.push(`Conflict ${conflictId}: ${String(fetchErr)}`);
          console.error(`[Sidebar] Error sending resolution for conflict ${conflictId}:`, fetchErr);
        }
      }

      if (backendErrors.length > 0) {
        console.warn('[Sidebar] Some conflicts failed to persist:', backendErrors);
      }

      setShowConflictModal(false);
      setPendingConflicts([]);

      // Add confirmation message with resolved conflicts info
      const resolvedCount = Object.keys(resolutions).length;
      const molyMsg: Message = {
        id: (Date.now() + 1).toString(),
        type: 'moly',
        content: `Thanks for clarifying ${resolvedCount} conflict${resolvedCount !== 1 ? 's' : ''}. I've updated the information with your resolutions.`,
        timestamp: Date.now(),
        metadata: { mode: chatMode, context },
      };

      const messagesWithConfirm = [...conversationMessages, molyMsg];
      setConversationMessages(messagesWithConfirm);
      saveConversationHistory(messagesWithConfirm);

      setIsLoading(false);
    } catch (err) {
      console.error('[Sidebar] Error resolving conflicts:', err);
      setError('Failed to process conflict resolution');
      setTimeout(() => setError(null), 5000);
    }
  };

  const processMessageWithContext = async (userMessage: string, contactIds: string[]) => {
    const activeProviderConfig = settings?.providers[settings?.activeProvider];
    if (!activeProviderConfig?.enabled) {
      setError(`Please configure ${settings?.activeProvider || 'an LLM'} provider in Settings.`);
      return;
    }

    const effectiveChatMode = chatMode || settings?.chatMode || 'direct';
    const effectiveContext = context || settings?.defaultContext || 'friendly';

    const userMsg: Message = {
      id: Date.now().toString(),
      type: 'user',
      content: userMessage,
      timestamp: Date.now(),
      metadata: { mode: effectiveChatMode, context: effectiveContext },
    };

    const updatedMessages = [...conversationMessages, userMsg];
    setConversationMessages(updatedMessages);
    saveConversationHistory(updatedMessages);
    setError(null);
    setIsLoading(true);
    setProcessingStage('');
    setProcessingSeconds(0);

    // Phase 1: Analyze for safety and ethics
    let analysisResults: any = null;
    try {
      setProcessingStage('Checking safety and ethics...');
      // Build context string with full conversation info
      let contextString = 'No conversation selected';

      // Add user context first
      if (meProfile) {
        let meContext = 'About me: ';
        const parts = [];
        if (meProfile.communicationStyle) parts.push(`Communication style: ${meProfile.communicationStyle}`);
        if (meProfile.values) parts.push(`Values: ${meProfile.values}`);
        if (meProfile.goals) parts.push(`Goals: ${meProfile.goals}`);
        if (meProfile.patterns) parts.push(`Patterns: ${meProfile.patterns}`);
        if (parts.length > 0) {
          contextString = meContext + parts.join('. ') + '.';
        }
      }

      if (currentConversation) {
        // Build member list with names and notes for personalization
        // Use conversationMembers which has fresh contact data with reflected insights
        const membersList = conversationMembers && conversationMembers.length > 0
          ? conversationMembers.map(m => m.notes ? `${m.name} (${m.notes})` : m.name).join('; ')
          : 'No members';
        const purpose = currentConversation.purpose ? ` Purpose: ${currentConversation.purpose}.` : '';
        const convType = currentConversation.type || 'conversation';
        const conversationContextStr = `Conversation: "${currentConversation.name}" (${convType}). Members: ${membersList}.${purpose}`;

        contextString = contextString === 'No conversation selected'
          ? conversationContextStr
          : contextString + ' ' + conversationContextStr;

        // If we have full context from backend, include member details
        if (conversationContext?.members && conversationContext.members.length > 0) {
          const memberDetails = conversationContext.members
            .map(m => {
              let detail = `${m.name} (${m.relationship})`;
              if (m.notes) detail += `: ${m.notes}`;
              return detail;
            })
            .join('; ');
          contextString += ` Details: ${memberDetails}.`;
        }
      }

      // Analysis handled by ChatInterface component
      // analysisResults = await analyze(...)
    } catch (err) {
      console.warn('[Moly] Backend analysis not available:', err);
    }

    // Phase 2: Check if we should gate suggestions based on analysis results
    const hasCrisis = analysisResults?.safety?.alert_type === 'crisis' || analysisResults?.safety?.alert_type === 'illegal';
    const hasEthicsViolations = analysisResults?.constitution?.violations && analysisResults.constitution.violations.length > 0;

    // If crisis detected, don't generate suggestions - show resources instead
    if (hasCrisis) {
      const molyMsg: Message = {
        id: (Date.now() + 1).toString(),
        type: 'moly',
        content: 'I detected a safety concern. Your wellbeing comes first. Please reach out to one of the resources shown above.',
        timestamp: Date.now(),
        metadata: { mode: chatMode, context },
      };
      const messagesWithResponse = [...updatedMessages, molyMsg];
      setConversationMessages(messagesWithResponse);
      saveConversationHistory(messagesWithResponse);
      setIsLoading(false);
      return;
    }

    // Phase 3: If ethics violations, ask clarifying questions instead of suggesting
    if (hasEthicsViolations) {
      const clarifyingQuestions = [
        'Can you tell me more about why you want to do this?',
        'How do you think this might affect the other person?',
        'Are there other options you\'ve considered?',
        'What outcome are you hoping for?'
      ];

      const molyMsg: Message = {
        id: (Date.now() + 1).toString(),
        type: 'moly',
        content: `I want to understand this better. ${clarifyingQuestions[Math.floor(Math.random() * clarifyingQuestions.length)]}`,
        timestamp: Date.now(),
        metadata: { mode: chatMode, context },
      };

      const messagesWithResponse = [...updatedMessages, molyMsg];
      setConversationMessages(messagesWithResponse);
      saveConversationHistory(messagesWithResponse);
      setIsLoading(false);
      return;
    }

    // Conversation is optional (Issue #12 - Direct messaging without setup)
    // User can send messages without creating a conversation
    // Facts will be extracted and saved to About Me
    const workingConversation = currentConversation;

    // Phase 4: Process through Phase 5 orchestrator (checks for clarification needs)
    try {
      console.info('[Sidebar] ===== PHASE 5 ORCHESTRATOR START =====');

      // Conversation is now optional - can be null
      // If no conversation: facts will be extracted to About Me only
      // If conversation exists: facts saved to both conversation and About Me
      const backendConversationId = workingConversation
        ? ((workingConversation as any).backendId || String(workingConversation.id))
        : '';

      console.info('[Sidebar] Message Details:', {
        userId: session?.userId?.substring(0, 16),
        hasConversation: !!workingConversation,
        conversationId: workingConversation?.id,
        backendId: workingConversation ? (workingConversation as any).backendId : null,
        usingId: backendConversationId || '(no conversation - facts to About Me only)',
        messageLength: userMessage.length,
        messagePreview: userMessage.substring(0, 50)
      });

      setProcessingStage('Extracting context and facts...');

      const phase5Response = await ClarificationAPI.processMessage(
        userMessage,
        backendConversationId,
        aboutMe,
        contactIds
      );
      setProcessingStage('Analyzing and storing results...');

      console.info('[Sidebar] Phase 5 Response Summary:', {
        needsClarification: phase5Response.action_required.needsClarification,
        clarificationQsCount: phase5Response.action_required.clarificationQs?.length || 0,
        temporaryFactsCount: phase5Response.action_required.temporaryFacts?.length || 0,
        extractedFactsCount: phase5Response.phase1?.facts?.length || 0,
        savedAttributesCount: phase5Response.phase4?.saved_attributes?.length || 0,
        conflictsFound: phase5Response.action_required?.conflicts?.length || 0
      });

      // ✨ SYNC ALL LAYERS: Update About Me, Contacts, and Conversation
      // Layer sync handled via backend API calls in profileAPI
      console.info('[Sidebar] ✓ All layers will be synced via backend APIs');

      // PHASE 5-6: Show reflection modal if insights were extracted
      // Note: Insights extracted in Phase 1 but backend's ClarificationEngine
      // will generate proper Socratic questions in Phase 2 below.
      // Don't show insights here - wait for clarification flow.

      // Check if we have contact questions first - offer to create contact
      const contactQuestions = phase5Response.action_required.clarificationQs?.filter(
        (q: any) => q.type === 'contact_context'
      ) || [];

      if (contactQuestions.length > 0) {
        console.info('[Sidebar] ✓ CONTACT DETECTION - User can add contact', {
          contactQuestionsCount: contactQuestions.length
        });

        // Extract contact names from question context
        const contactsToCreate = contactQuestions.map((q: any) => ({
          name: q.contactName || q.context?.match(/\b([A-Z][a-z]+)\b/)?.[1] || 'Unknown',
          relationship: q.relationship || 'acquaintance',
          context: q.question || ''
        }));

        // Show message offering to add contact (user choice)
        const molyMsg: Message = {
          id: (Date.now() + 1).toString(),
          type: 'moly',
          content: `I noticed you mentioned ${contactsToCreate[0].name}. Would you like to add them as a contact?`,
          timestamp: Date.now(),
          metadata: { mode: chatMode, context },
        };

        const messagesWithOffer = [...updatedMessages, molyMsg];
        setConversationMessages(messagesWithOffer);
        saveConversationHistory(messagesWithOffer);

        // Show contact selector - user can choose to create or skip
        setShowContactSelector(true);

        return;
      }

      // If clarification questions are needed, show the clarification chat
      if (
        phase5Response.action_required.needsClarification &&
        phase5Response.action_required.clarificationQs &&
        phase5Response.action_required.clarificationQs.length > 0
      ) {
        console.info('[Sidebar] ✓ CLARIFICATION REQUIRED - Showing modal', {
          questionsCount: phase5Response.action_required.clarificationQs.length,
          temporaryFactsCount: phase5Response.action_required.temporaryFacts?.length || 0
        });

        // Populate the clarification store (clear old questions first)
        const { clear, setQuestions } = useClarificationStore.getState();
        clear(); // Clear any previous questions
        setQuestions(
          phase5Response.action_required.clarificationQs,
          phase5Response.action_required.temporaryFacts || []
        );

        console.info('[Sidebar] Clarification store populated, modal will display');

        // Show Moly message indicating we're gathering context
        const molyMsg: Message = {
          id: (Date.now() + 1).toString(),
          type: 'moly',
          content: 'I want to understand you better. Let me ask a few quick questions.',
          timestamp: Date.now(),
          metadata: { mode: chatMode, context },
        };

        const messagesWithResponse = [...updatedMessages, molyMsg];
        setConversationMessages(messagesWithResponse);
        saveConversationHistory(messagesWithResponse);
        setProcessingStage('Gathering context...');
        setSuggestions([]);
        setIsLoading(false);

        console.info('[Sidebar] Clarification flow initiated - waiting for user answers');
        return;
      }

      // Check for conflicts (Issue #16 - Conflict detection not shown to user)
      if (phase5Response.action_required.hasConflicts && phase5Response.action_required.conflicts && phase5Response.action_required.conflicts.length > 0) {
        console.info('[Sidebar] ✓ CONFLICT DETECTION (Issue #16): User has conflicting information', {
          conflictsCount: phase5Response.action_required.conflicts.length,
          conflicts: phase5Response.action_required.conflicts.map((c: any) => ({
            id: c.id?.substring(0, 8) + '...',
            type: c.factType,
            currentValue: c.currentValue,
            conflictingValue: c.conflictingValue
          }))
        });

        setPendingConflicts(phase5Response.action_required.conflicts);
        setShowConflictModal(true);

        // Show Moly message explaining conflicts
        const molyMsg: Message = {
          id: (Date.now() + 1).toString(),
          type: 'moly',
          content: 'I found some conflicting information. Let me clarify with you.',
          timestamp: Date.now(),
          metadata: { mode: chatMode, context },
        };

        const messagesWithConflict = [...updatedMessages, molyMsg];
        setConversationMessages(messagesWithConflict);
        saveConversationHistory(messagesWithConflict);
        setIsLoading(false);

        console.info('[Sidebar] Conflict resolution flow initiated');
        return;
      }

      // NO CLARIFICATION NEEDED - Just acknowledge in Field 1 (context building)
      // Do NOT generate suggestions in Field 1
      // Suggestions are only for Field 2 (when pasting incoming message or initiating contact)
      console.info('[Sidebar] ✓ NO CLARIFICATION NEEDED - Acknowledging context in Field 1');

      const extractedFacts = phase5Response.phase1?.facts || [];
      console.info('[Sidebar] Extracted facts for context:', {
        count: extractedFacts.length,
        facts: extractedFacts.map((f: any) => ({
          type: f.type,
          value: f.value,
          subject: f.subject
        }))
      });

      // Show Moly acknowledgment message (no suggestions here)
      const acknowledgmentMessages = [
        'Got it. That helps me understand you better.',
        'Thanks for sharing that. I\'ll remember it.',
        'I understand. That\'s useful context.',
        'Noted. That\'s good to know about you.',
      ];

      const molyAcknowledgment: Message = {
        id: (Date.now() + 1).toString(),
        type: 'moly',
        content: acknowledgmentMessages[Math.floor(Math.random() * acknowledgmentMessages.length)],
        timestamp: Date.now(),
        metadata: { mode: chatMode, context },
      };

      const messagesWithAcknowledgment = [...updatedMessages, molyAcknowledgment];
      setConversationMessages(messagesWithAcknowledgment);
      saveConversationHistory(messagesWithAcknowledgment);
      setSuggestions([]); // Clear any suggestions - Field 1 doesn't show suggestions
      setIsLoading(false);

      console.info('[Sidebar] Field 1 context stored - user can continue chatting or move to Field 2');
    } catch (err) {
      console.error('[Sidebar] Error processing message:', err);
      setError(`Error: ${err instanceof Error ? err.message : 'Unknown error'}`);
      setIsLoading(false);
    }
  };

  // Context binding modal handlers
  const handleIncomingMessage = async (message: string) => {
    // User pasted incoming message - now generate suggestions based on full context
    setIncomingMessage(message);

    // Add incoming message to conversation history
    const incomingMsg: Message = {
      id: Date.now().toString(),
      type: 'incoming',
      content: message,
      timestamp: Date.now(),
      metadata: { source: 'pasted', mode: chatMode, context },
    };

    const updatedMessages = [...conversationMessages, incomingMsg];
    setConversationMessages(updatedMessages);
    saveConversationHistory(updatedMessages);

    // Now generate suggestions with full context including incoming message
    setIsLoading(true);
    setProcessingStage('Analyzing with incoming message...');
    setSuggestions([]);
    setError(null);

    try {
      // STEP 1: Detect sender from incoming message
      // Contact detection handled via profileAPI.getContacts()
      let senderContact = null;

      // STEP 2: Build complete context with incoming message
      let contextString = 'No conversation selected';

      if (meProfile) {
        let meContext = 'About me: ';
        const parts = [];
        if (meProfile.communicationStyle) parts.push(`Communication style: ${meProfile.communicationStyle}`);
        if (meProfile.values) parts.push(`Values: ${meProfile.values}`);
        if (meProfile.goals) parts.push(`Goals: ${meProfile.goals}`);
        if (meProfile.patterns) parts.push(`Patterns: ${meProfile.patterns}`);
        if (parts.length > 0) {
          contextString = meContext + parts.join('. ') + '.';
        }
      }

      if (currentConversation) {
        const membersList = conversationMembers
          .map(m => m.notes ? `${m.name} (${m.notes})` : m.name)
          .join('; ');
        const purpose = currentConversation.purpose ? ` Purpose: ${currentConversation.purpose}.` : '';
        const conversationContextStr = `Conversation: "${currentConversation.name}" (${currentConversation.type}). Members: ${membersList}.${purpose} INCOMING MESSAGE: "${message}"`;

        contextString = contextString === 'No conversation selected'
          ? conversationContextStr
          : contextString + ' ' + conversationContextStr;
      }

      // Add sender info to context if detected
      if (senderContact) {
        contextString += ` INCOMING FROM: ${senderContact.name}`;
      }

      // STEP 3: Generate suggestions with full context
      // Analysis handled by ChatInterface component
      let generatedSuggestions: string[] = [];

      if (generatedSuggestions && generatedSuggestions.length > 0) {
        setSuggestions(generatedSuggestions);
        setProcessingStage(`Generated ${generatedSuggestions.length} response suggestions`);

        const molyMsg: Message = {
          id: (Date.now() + 1).toString(),
          type: 'moly',
          content: `Here are ${generatedSuggestions.length} response suggestions based on your conversation and communication style.`,
          timestamp: Date.now(),
          metadata: { mode: chatMode, context },
        };

        const messagesWithSuggestions = [...updatedMessages, molyMsg];
        setConversationMessages(messagesWithSuggestions);
        saveConversationHistory(messagesWithSuggestions);
      } else {
        setError('Could not generate suggestions. Please try again.');
      }
    } catch (err) {
      setError(`Error: ${err instanceof Error ? err.message : 'Unknown error'}`);
    } finally {
      setIsLoading(false);
    }
  };

  const handleCopySuggestion = (text: string) => {
    const suggestionMsg: Message = {
      id: (Date.now() + 2).toString(),
      type: 'suggestion',
      content: text,
      timestamp: Date.now(),
    };

    const updatedMessages = [...conversationMessages, suggestionMsg];
    setConversationMessages(updatedMessages);
    saveConversationHistory(updatedMessages);
  };

  const handleDeleteMessage = (id: string) => {
    const filtered = conversationMessages.filter((msg) => msg.id !== id);
    setConversationMessages(filtered);
    saveConversationHistory(filtered);
  };

  const handleExportConversation = () => {
    const dataStr = JSON.stringify(
      {
        messages: conversationMessages,
        exportedAt: new Date().toISOString(),
      },
      null,
      2
    );

    const dataBlob = new Blob([dataStr], { type: 'application/json' });
    const url = URL.createObjectURL(dataBlob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `moly-conversation-${Date.now()}.json`;
    link.click();
    URL.revokeObjectURL(url);
  };

  const handleOpenSettings = () => {
    setShowSettings(!showSettings);
  };

  const handleLogout = async () => {
    if (window.confirm('Are you sure you want to logout?')) {
      try {
        // Call backend logout to invalidate session token
        const token = localStorage.getItem('authToken');
        if (token) {
          try {
            const backendUrl = getBackendManager().getBackendUrl();
            await fetch(`${backendUrl}/api/auth/logout`, {
              method: 'POST',
              headers: {
                'Authorization': `Bearer ${token}`,
                'Content-Type': 'application/json'
              }
            });
            console.log('[Sidebar] ✓ Backend logout successful');
          } catch (backendErr) {
            console.warn('[Sidebar] Backend logout failed (non-blocking):', backendErr);
            // Continue with local logout even if backend call fails
          }
        }
      } catch (err) {
        console.error('[Sidebar] Error during logout:', err);
      } finally {
        // Clear local auth
        await clearAuth();
        localStorage.removeItem('userId');
        localStorage.removeItem('authToken');
        // Reload to show LoginScreen
        window.location.reload();
      }
    }
  };

  const handleSaveContactContext = async (context: ExtractedContactContext) => {
    if (!currentConversation) return;

    // Only proceed if conversation has members (optional in new model)
    const members = currentConversation.members || [];
    if (members.length === 0) return;

    // Get the first contact from the conversation members
    const contactId = members[0].id.toString();
    const contactName = getFirst(currentConversation.members)?.name || 'contact';

    try {
      // Build notes from extracted context
      const contextNotes: string[] = [];
      if (context.characteristics?.length) {
        contextNotes.push(`Characteristics: ${context.characteristics.join('; ')}`);
      }
      if (context.intentions?.length) {
        contextNotes.push(`Intentions: ${context.intentions.join('; ')}`);
      }
      if (context.behaviors?.length) {
        contextNotes.push(`Behaviors: ${context.behaviors.join('; ')}`);
      }

      const newContextText = contextNotes.join('\n');

      // PHASE 5-6: Save to backend via profileAPI
      try {
        await profileAPI.saveContact({
          id: contactId,
          name: contactName,
          notes: newContextText
        });
        console.log('[Sidebar] ✓ Contact context saved to backend:', contactName);
      } catch (backendErr) {
        console.warn('[Sidebar] Backend save failed, falling back to local:', backendErr);
      }

      // Also save to local storage for offline support
      const result = await chrome.storage.local.get('contacts');
      const contacts = result.contacts || [];

      const updatedContacts = contacts.map(c => {
        if (c.id === contactId) {
          const updatedNotes = c.notes
            ? `${c.notes}\n\n${newContextText}`
            : newContextText;
          return {
            ...c,
            notes: updatedNotes,
          };
        }
        return c;
      });

      await chrome.storage.local.set({ contacts: updatedContacts });
      console.log('[Sidebar] Contact context saved to local storage:', contactName);
    } catch (err) {
      console.error('[Sidebar] Failed to save contact context:', err);
    }
  };

  const handleSaveReflection = async (notes: string) => {
    if (!currentConversation) return;

    // Update conversation with notes
    const updated: ConversationData = {
      ...currentConversation,
      notes: (currentConversation.notes ? currentConversation.notes + '\n\n' : '') + notes,
      updated_at: Date.now(),
    };

    try {
      // Update local storage
      const result = await chrome.storage.local.get('conversations');
      const conversations = (result.conversations || []).map(c =>
        c.id === currentConversation.id ? updated : c
      );
      await chrome.storage.local.set({ conversations });

      // Backend sync handled by profileAPI

      setCurrentConversation(updated);
      setShowReflection(false);
      setExtractedContactContext(undefined);
    } catch (err) {
      console.error('[Sidebar] Failed to save reflection:', err);
    }
  };

  return <ChatInterface />;
};

export default Sidebar;
