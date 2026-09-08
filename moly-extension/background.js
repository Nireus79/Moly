// Moly Background Service Worker
// Handles communication with backend, storage, and message passing

const BACKEND_URL = 'http://localhost:8080';
const STORAGE_KEYS = {
  userId: 'moly_user_id',
  backendUrl: 'moly_backend_url',
  aboutMe: 'moly_about_me',
  contacts: 'moly_contacts'
};

// Initialize extension on install
chrome.runtime.onInstalled.addListener(() => {
  console.log('[Moly] Extension installed, initializing...');
  
  // Generate unique user ID if not exists
  chrome.storage.local.get([STORAGE_KEYS.userId], (result) => {
    if (!result[STORAGE_KEYS.userId]) {
      const userId = 'moly_' + generateId();
      chrome.storage.local.set({[STORAGE_KEYS.userId]: userId});
      console.log('[Moly] Generated user ID:', userId);
    }
  });
  
  // Set default backend URL
  chrome.storage.local.get([STORAGE_KEYS.backendUrl], (result) => {
    if (!result[STORAGE_KEYS.backendUrl]) {
      chrome.storage.local.set({[STORAGE_KEYS.backendUrl]: BACKEND_URL});
      console.log('[Moly] Set backend URL:', BACKEND_URL);
    }
  });
});

// Handle messages from content scripts and popup
chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
  console.log('[Moly BG] Received message:', request.type);
  
  switch (request.type) {
    case 'GET_USER_ID':
      handleGetUserId(sendResponse);
      break;
      
    case 'GET_BACKEND_URL':
      handleGetBackendUrl(sendResponse);
      break;
      
    case 'GENERATE_SUGGESTION':
      handleGenerateSuggestion(request.payload, sendResponse);
      break;
      
    case 'RECORD_FEEDBACK':
      handleRecordFeedback(request.payload, sendResponse);
      break;
      
    case 'GET_CONTEXT':
      handleGetContext(sendResponse);
      break;
      
    case 'SAVE_ABOUT_ME':
      handleSaveAboutMe(request.payload, sendResponse);
      break;
      
    case 'SAVE_CONTACT':
      handleSaveContact(request.payload, sendResponse);
      break;
      
    case 'CHECK_HEALTH':
      handleHealthCheck(sendResponse);
      break;
      
    default:
      sendResponse({error: 'Unknown message type: ' + request.type});
  }
  
  return true; // Keep channel open for async response
});

// Get user ID
function handleGetUserId(sendResponse) {
  chrome.storage.local.get([STORAGE_KEYS.userId], (result) => {
    sendResponse({userId: result[STORAGE_KEYS.userId]});
  });
}

// Get backend URL
function handleGetBackendUrl(sendResponse) {
  chrome.storage.local.get([STORAGE_KEYS.backendUrl], (result) => {
    sendResponse({url: result[STORAGE_KEYS.backendUrl] || BACKEND_URL});
  });
}

// Generate suggestion from backend
function handleGenerateSuggestion(payload, sendResponse) {
  chrome.storage.local.get([STORAGE_KEYS.userId, STORAGE_KEYS.backendUrl], async (result) => {
    const userId = result[STORAGE_KEYS.userId];
    const backendUrl = result[STORAGE_KEYS.backendUrl] || BACKEND_URL;
    
    try {
      const response = await fetch(`${backendUrl}/api/v2/conversation/generate`, {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({
          userId: userId,
          conversationId: payload.conversationId || generateId(),
          userMessage: payload.message
        })
      });
      
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }
      
      const data = await response.json();
      console.log('[Moly BG] Got suggestion:', data);
      sendResponse({success: true, data: data});
      
    } catch (error) {
      console.error('[Moly BG] Generate suggestion error:', error);
      sendResponse({
        success: false, 
        error: error.message,
        fallback: generateFallbackSuggestion(payload.message)
      });
    }
  });
}

// Record user feedback
function handleRecordFeedback(payload, sendResponse) {
  chrome.storage.local.get([STORAGE_KEYS.userId, STORAGE_KEYS.backendUrl], async (result) => {
    const userId = result[STORAGE_KEYS.userId];
    const backendUrl = result[STORAGE_KEYS.backendUrl] || BACKEND_URL;
    
    try {
      const response = await fetch(`${backendUrl}/api/v2/conversation/feedback`, {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({
          userId: userId,
          conversationId: payload.conversationId,
          suggestionChosen: payload.suggestionIndex,
          modificationRequest: payload.modifiedText,
          userModified: payload.modified || false,
          reflectionApproved: payload.approved || false
        })
      });
      
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }
      
      const data = await response.json();
      console.log('[Moly BG] Feedback recorded:', data);
      sendResponse({success: true, data: data});
      
    } catch (error) {
      console.error('[Moly BG] Record feedback error:', error);
      sendResponse({success: false, error: error.message});
    }
  });
}

// Get context from backend
function handleGetContext(sendResponse) {
  chrome.storage.local.get([STORAGE_KEYS.userId, STORAGE_KEYS.backendUrl], async (result) => {
    const userId = result[STORAGE_KEYS.userId];
    const backendUrl = result[STORAGE_KEYS.backendUrl] || BACKEND_URL;
    
    try {
      const response = await fetch(`${backendUrl}/api/v2/context?userId=${userId}&conversationId=web`, {
        method: 'GET'
      });
      
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }
      
      const data = await response.json();
      sendResponse({success: true, data: data});
      
    } catch (error) {
      console.error('[Moly BG] Get context error:', error);
      sendResponse({success: false, error: error.message});
    }
  });
}

// Save AboutMe to backend
function handleSaveAboutMe(payload, sendResponse) {
  chrome.storage.local.get([STORAGE_KEYS.userId, STORAGE_KEYS.backendUrl], async (result) => {
    const userId = result[STORAGE_KEYS.userId];
    const backendUrl = result[STORAGE_KEYS.backendUrl] || BACKEND_URL;
    
    try {
      const response = await fetch(`${backendUrl}/api/v2/about-me`, {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({
          userId: userId,
          ...payload
        })
      });
      
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }
      
      // Also store locally
      chrome.storage.local.set({[STORAGE_KEYS.aboutMe]: payload});
      
      const data = await response.json();
      sendResponse({success: true, data: data});
      
    } catch (error) {
      console.error('[Moly BG] Save AboutMe error:', error);
      sendResponse({success: false, error: error.message});
    }
  });
}

// Save contact to backend
function handleSaveContact(payload, sendResponse) {
  chrome.storage.local.get([STORAGE_KEYS.userId, STORAGE_KEYS.backendUrl], async (result) => {
    const userId = result[STORAGE_KEYS.userId];
    const backendUrl = result[STORAGE_KEYS.backendUrl] || BACKEND_URL;
    
    try {
      const response = await fetch(`${backendUrl}/api/v2/contacts`, {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({
          userId: userId,
          ...payload
        })
      });
      
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }
      
      const data = await response.json();
      sendResponse({success: true, data: data});
      
    } catch (error) {
      console.error('[Moly BG] Save contact error:', error);
      sendResponse({success: false, error: error.message});
    }
  });
}

// Health check
function handleHealthCheck(sendResponse) {
  chrome.storage.local.get([STORAGE_KEYS.backendUrl], async (result) => {
    const backendUrl = result[STORAGE_KEYS.backendUrl] || BACKEND_URL;
    
    try {
      const response = await fetch(`${backendUrl}/api/v2/health`, {method: 'GET'});
      const data = await response.json();
      sendResponse({healthy: response.ok, data: data});
    } catch (error) {
      console.error('[Moly BG] Health check error:', error);
      sendResponse({healthy: false, error: error.message});
    }
  });
}

// Generate fallback suggestion
function generateFallbackSuggestion(message) {
  const suggestions = [
    "Consider being more direct about what you need.",
    "Start by explaining the context or situation first.",
    "Ask for the other person's perspective to understand their view.",
    "Express your feelings clearly and calmly.",
    "Focus on the behavior or situation, not the person."
  ];
  return suggestions[Math.floor(Math.random() * suggestions.length)];
}

// Generate random ID
function generateId() {
  return Math.random().toString(36).substr(2, 9);
}
