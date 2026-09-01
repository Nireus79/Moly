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

// Listen for toggle command from background
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
  } else if (request.type === 'SHOW_RESTRICTED_PAGE_MESSAGE') {
    alert('Moly works on real websites. This page is restricted.');
    sendResponse({ success: true });
  }
});

console.log('[Moly] Content script ready');
