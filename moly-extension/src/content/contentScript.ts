/**
 * Content Script - Inject sidebar on all webpages
 * Works on ALL browsers (no sidePanel API required)
 */

console.log('[Moly] Content script loaded');

let container: HTMLElement | null = null;

// Inject sidebar immediately
function injectSidebar() {
  // Skip extension pages
  if (window.location.protocol === 'chrome-extension:' ||
      window.location.protocol === 'moz-extension:' ||
      window.location.protocol === 'chrome:' ||
      window.location.protocol === 'about:') {
    return;
  }

  if (container) return;

  try {
    // Create container div
    container = document.createElement('div');
    container.id = 'moly-sidebar';
    container.style.cssText = `
      position: fixed !important;
      right: 0 !important;
      top: 0 !important;
      width: 400px !important;
      height: 100vh !important;
      background: white !important;
      border-left: 1px solid #e5e7eb !important;
      box-shadow: -2px 0 8px rgba(0,0,0,0.1) !important;
      z-index: 2147483647 !important;
      display: none !important;
      margin: 0 !important;
      padding: 0 !important;
      box-sizing: border-box !important;
    `;

    // Create iframe
    const iframe = document.createElement('iframe');
    iframe.src = chrome.runtime.getURL('sidebar/sidebar.html');
    iframe.style.cssText = `
      width: 100% !important;
      height: 100% !important;
      border: none !important;
      margin: 0 !important;
      padding: 0 !important;
    `;

    container.appendChild(iframe);
    document.documentElement.appendChild(container);

    console.log('[Moly] Sidebar injected');
  } catch (error) {
    console.error('[Moly] Failed to inject sidebar:', error);
  }
}

// Inject on load
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', injectSidebar);
} else {
  injectSidebar();
}

// Also try on window load
window.addEventListener('load', injectSidebar);

// Listen for messages from background
chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
  if (request.type === 'TOGGLE_MOLY_SIDEBAR') {
    if (!container) {
      injectSidebar();
    }

    if (container) {
      const isHidden = container.style.display === 'none';
      container.style.display = isHidden ? 'block' : 'none';
      console.log('[Moly] Sidebar toggled:', container.style.display);
      sendResponse({ success: true });
    }
  } else if (request.type === 'SHOW_RESTRICTED_MESSAGE') {
    const messageDiv = document.createElement('div');
    messageDiv.id = 'moly-restricted-message';
    messageDiv.style.cssText = `
      position: fixed !important;
      top: 20px !important;
      right: 20px !important;
      padding: 16px !important;
      background: #fef3c7 !important;
      border: 1px solid #f59e0b !important;
      border-radius: 8px !important;
      color: #92400e !important;
      font-family: system-ui, -apple-system, sans-serif !important;
      font-size: 14px !important;
      font-weight: 500 !important;
      z-index: 2147483647 !important;
      max-width: 300px !important;
      box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1) !important;
    `;
    messageDiv.textContent = 'Moly works on real websites only. This page is restricted.';
    document.documentElement.appendChild(messageDiv);

    // Auto-hide after 5 seconds
    setTimeout(() => {
      messageDiv.remove();
    }, 5000);

    sendResponse({ success: true });
  }
});

console.log('[Moly] Content script ready');
