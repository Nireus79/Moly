/**
 * Background Service Worker for Moly Extension (Manifest V3 Compatible)
 * Minimalist design: service workers cannot use top-level ES6 imports, so this file is self-contained.
 * It owns the native messaging port to the Moly app (starts and keeps the backend alive); the UI handles the rest.
 */

// --- Native host connection -------------------------------------------------------------------------------------------
// The service worker owns the one native messaging port. The host (the Moly binary) starts the backend, returns its URL
// and the session token, and keeps the backend alive while this port is open. An open native port also keeps this
// service worker alive. Pages ask for the connection with MOLY_ENSURE_BACKEND and read it from chrome.storage.session.

const HOST_NAME = 'com.moly.backend_host';
const CONNECTION_KEY = 'molyConnection';
const START_TIMEOUT_MS = 100_000; // the host waits up to 90 s for the backend
// The protocol of the messages to the app; the same value as HOST_PROTOCOL in api/hostErrors.ts (this file cannot import; a test keeps them equal).
const HOST_PROTOCOL = '1';

interface HostConnection {
  url: string;
  token: string;
  version: string;
}

interface PendingCall {
  resolve: (reply: any) => void;
  reject: (error: Error) => void;
  timer: ReturnType<typeof setTimeout>;
}

let hostPort: chrome.runtime.Port | null = null;
let connection: HostConnection | null = null;
let startInFlight: Promise<HostConnection> | null = null;
let nextCallId = 1;
const pendingCalls = new Map<number, PendingCall>();

function failPending(error: Error): void {
  pendingCalls.forEach((call) => {
    clearTimeout(call.timer);
    call.reject(error);
  });
  pendingCalls.clear();
}

function forgetConnection(): void {
  connection = null;
  chrome.storage.session.remove(CONNECTION_KEY).catch(() => undefined);
}

function openHostPort(): chrome.runtime.Port {
  const port = chrome.runtime.connectNative(HOST_NAME);
  port.onMessage.addListener((message) => {
    if (message?.event === 'backend-down') {
      // The backend went away (crash, or it was stopped): the next page request starts it again.
      forgetConnection();
      return;
    }
    const call = pendingCalls.get(message?.id);
    if (call) {
      pendingCalls.delete(message.id);
      clearTimeout(call.timer);
      call.resolve(message);
    }
  });
  port.onDisconnect.addListener(() => {
    const reason = chrome.runtime.lastError?.message || 'The Moly app is not installed or not registered with this browser.';
    hostPort = null;
    forgetConnection();
    failPending(new Error(reason));
  });
  return port;
}

function callHost(action: string, timeoutMs: number): Promise<any> {
  if (!hostPort) hostPort = openHostPort();
  const port = hostPort;
  return new Promise((resolve, reject) => {
    const id = nextCallId++;
    const timer = setTimeout(() => {
      pendingCalls.delete(id);
      reject(new Error(`The Moly app did not answer "${action}" in time.`));
    }, timeoutMs);
    pendingCalls.set(id, { resolve, reject, timer });
    try {
      port.postMessage({ action, id, protocol: HOST_PROTOCOL });
    } catch (error) {
      pendingCalls.delete(id);
      clearTimeout(timer);
      reject(error instanceof Error ? error : new Error(String(error)));
    }
  });
}

/** Starts the backend through the host (or finds it running) and publishes URL and token for the pages. */
function ensureBackend(): Promise<HostConnection> {
  if (connection) return Promise.resolve(connection);
  if (!startInFlight) {
    startInFlight = callHost('start', START_TIMEOUT_MS)
      .then(async (reply) => {
        if (!reply?.success) throw new Error(reply?.error || 'The Moly app could not start the backend.');
        // An app from before this check sends no protocol: it is protocol 1.
        if (reply.hostVersion && String(reply.hostVersion) !== HOST_PROTOCOL) throw new Error(`The Moly app and this extension speak different versions of the protocol (app ${reply.hostVersion}, extension ${HOST_PROTOCOL}).`);
        connection = { url: reply.url, token: reply.token, version: reply.version || '' };
        await chrome.storage.session.set({ [CONNECTION_KEY]: connection });
        return connection;
      })
      .finally(() => {
        startInFlight = null;
      });
  }
  return startInFlight;
}

// Firefox has a sidebar (sidebarAction) instead of Chrome's side panel; the Firefox build only has the first.
const firefoxSidebar: { toggle(): Promise<void>; close(): Promise<void> } | undefined = (globalThis as any).browser?.sidebarAction;

// Track if side panel is open per tab (Chrome)
const sidePanelOpen: { [tabId: number]: boolean } = {};

// Handle extension icon click - toggle the side panel / sidebar
chrome.action.onClicked.addListener((tab) => {
  if (firefoxSidebar) {
    // Must run inside the click: Firefox only opens the sidebar for a user action.
    firefoxSidebar.toggle().catch((error) => console.warn('[Background] Failed to toggle sidebar:', error));
    return;
  }
  if (!tab.id) return;

  const tabId = tab.id;
  const isOpen = sidePanelOpen[tabId];

  if (isOpen) {
    chrome.sidePanel.setOptions({ tabId, path: '' }).catch((error) => {
      console.warn('[Background] Failed to close sidePanel:', error);
    });
    sidePanelOpen[tabId] = false;
  } else {
    (chrome.sidePanel as any).open({ tabId }).catch((error: unknown) => {
      console.error('[Background] Failed to open sidePanel:', error);
    });
    sidePanelOpen[tabId] = true;
  }
});

// Handle messages from sidebar
chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {

  // Handle close sidePanel request
  if (request.type === 'CLOSE_SIDEPANEL') {
    if (firefoxSidebar) {
      firefoxSidebar.close().then(
        () => sendResponse({ success: true }),
        (error: Error) => sendResponse({ success: false, error: error.message })
      );
      return true; // Respond asynchronously
    }
    const tabId = sender.tab?.id;
    if (tabId) {
      chrome.sidePanel.setOptions({ tabId, path: '' }).then(() => {
        sidePanelOpen[tabId] = false;
        sendResponse({ success: true });
      }).catch((error) => {
        sendResponse({ success: false, error: error.message });
      });
      return true; // Respond asynchronously
    }
  }

  // Pages ask for the backend connection (start it if needed)
  if (request.type === 'MOLY_ENSURE_BACKEND') {
    ensureBackend().then(
      (conn) => sendResponse({ ok: true, url: conn.url, version: conn.version }),
      (error: Error) => sendResponse({ ok: false, error: error.message })
    );
    return true; // Respond asynchronously
  }

  // Acknowledge other messages
  sendResponse({ received: true });
  return false;
});

// Register for lifecycle events
chrome.runtime.onInstalled.addListener(() => {
});

chrome.runtime.onStartup.addListener(() => {
});

