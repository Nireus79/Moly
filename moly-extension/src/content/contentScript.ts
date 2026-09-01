/**
 * Content Script - Inject sidebar into webpages
 * NO automatic message reading - user-controlled only
 */

console.log('Moly content script loaded');

// Inject sidebar when page loads
window.addEventListener('load', () => {
  injectSidebar();
});

function injectSidebar() {
  // Don't inject on extension pages or chrome pages
  if (window.location.protocol === 'chrome-extension:' || window.location.protocol === 'moz-extension:' || window.location.protocol === 'chrome:') {
    return;
  }

  // Create container for iframe
  const container = document.createElement('div');
  container.id = 'moly-sidebar-container';
  container.style.cssText = `
    position: fixed;
    right: 0;
    top: 0;
    width: 400px;
    height: 100vh;
    background: white;
    border-left: 1px solid #e5e7eb;
    box-shadow: -2px 0 8px rgba(0,0,0,0.1);
    z-index: 2147483647;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    display: none;
  `;

  // Create iframe
  const iframe = document.createElement('iframe');
  iframe.src = chrome.runtime.getURL('sidebar/sidebar.html');
  iframe.style.cssText = `
    width: 100%;
    height: 100%;
    border: none;
    margin: 0;
    padding: 0;
  `;

  container.appendChild(iframe);
  document.body.appendChild(container);

  // Handle messages from iframe
  window.addEventListener('message', (event) => {
    if (event.source === iframe.contentWindow) {
      if (event.data.type === 'MOLY_CLOSE') {
        container.style.display = 'none';
      }
    }
  });
}

// Listen for show command from background
chrome.runtime.onMessage.addListener((request) => {
  if (request.type === 'SHOW_MOLY_SIDEBAR') {
    const container = document.getElementById('moly-sidebar-container');
    if (container) {
      container.style.display = container.style.display === 'none' ? 'block' : 'none';
    } else {
      injectSidebar();
      setTimeout(() => {
        const newContainer = document.getElementById('moly-sidebar-container');
        if (newContainer) newContainer.style.display = 'block';
      }, 100);
    }
  }
});
