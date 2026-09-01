/**
 * Content Script - Inject sidebar into webpages
 * NO automatic message reading - user-controlled only
 */

console.log('[Moly] Content script loaded');

let sidebarContainer: HTMLElement | null = null;

// Inject sidebar when page loads
window.addEventListener('DOMContentLoaded', () => {
  try {
    createSidebarContainer();
  } catch (error) {
    console.error('[Moly] Error creating sidebar:', error);
  }
});

function createSidebarContainer() {
  // Don't inject on extension pages
  if (window.location.protocol === 'chrome-extension:' ||
      window.location.protocol === 'moz-extension:' ||
      window.location.protocol === 'chrome:') {
    console.log('[Moly] Skipping injection on extension page');
    return;
  }

  if (sidebarContainer) return;

  try {
    // Create container
    sidebarContainer = document.createElement('div');
    sidebarContainer.id = 'moly-sidebar-container';
    sidebarContainer.style.cssText = `
      position: fixed;
      right: 0;
      top: 0;
      width: 400px;
      height: 100vh;
      background: white;
      border-left: 1px solid #e5e7eb;
      box-shadow: -2px 0 8px rgba(0,0,0,0.1);
      z-index: 2147483647;
      display: none;
      margin: 0;
      padding: 0;
    `;

    // Create iframe with src to sidebar
    const iframe = document.createElement('iframe');
    iframe.src = chrome.runtime.getURL('sidebar/sidebar.html');
    iframe.style.cssText = `
      width: 100%;
      height: 100%;
      border: none;
      margin: 0;
      padding: 0;
    `;

    sidebarContainer.appendChild(iframe);
    document.body.appendChild(sidebarContainer);

    console.log('[Moly] Sidebar container created');
  } catch (error) {
    console.error('[Moly] Failed to create sidebar:', error);
  }
}

// Listen for show command
chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
  try {
    if (request.type === 'SHOW_MOLY_SIDEBAR') {
      console.log('[Moly] Received show command');

      if (!sidebarContainer) {
        createSidebarContainer();
      }

      if (sidebarContainer) {
        const isHidden = sidebarContainer.style.display === 'none';
        sidebarContainer.style.display = isHidden ? 'block' : 'none';
        console.log('[Moly] Sidebar toggled to:', sidebarContainer.style.display);
        sendResponse({ success: true });
      }
    }
  } catch (error) {
    console.error('[Moly] Error handling message:', error);
    sendResponse({ success: false, error: error instanceof Error ? error.message : 'Unknown error' });
  }

  return true;
});
