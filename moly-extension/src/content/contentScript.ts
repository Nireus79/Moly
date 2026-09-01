/**
 * Content Script - Inject sidebar into webpages
 */

console.log('[Moly] Content script executing');

let sidebarContainer: HTMLElement | null = null;

// Create sidebar container immediately
function createSidebarContainer() {
  if (sidebarContainer) return;
  if (!document.body) return;

  try {
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

// Try to create immediately
if (document.body) {
  createSidebarContainer();
} else {
  // If no body yet, wait for it
  document.addEventListener('DOMContentLoaded', createSidebarContainer);
  window.addEventListener('load', createSidebarContainer);
}

// Listen for show command from background
chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
  console.log('[Moly] Message received:', request.type);

  if (request.type === 'SHOW_MOLY_SIDEBAR') {
    try {
      if (!sidebarContainer) {
        createSidebarContainer();
      }

      if (sidebarContainer) {
        const isHidden = sidebarContainer.style.display === 'none';
        sidebarContainer.style.display = isHidden ? 'block' : 'none';
        console.log('[Moly] Sidebar toggled:', sidebarContainer.style.display);
        sendResponse({ success: true });
      } else {
        console.error('[Moly] No sidebar container');
        sendResponse({ success: false });
      }
    } catch (error) {
      console.error('[Moly] Error:', error);
      sendResponse({ success: false });
    }
  }

  return true;
});

console.log('[Moly] Content script ready');
