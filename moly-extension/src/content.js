// Inject Moly sidebar from desktop app
(function() {
  // Skip injection on restricted URLs
  const url = window.location.href;
  if (url.startsWith('chrome://') || url.startsWith('chrome-extension://') || url.startsWith('http://localhost') || url.startsWith('http://127.0.0.1')) {
    return;
  }

  // Listen for toggle message from background script
  chrome.runtime.onMessage.addListener((request) => {
    if (request.action === 'toggle-sidebar') {
      toggleSidebar();
    }
  });

  function autoInject() {
    // Check if sidebar already injected
    if (document.getElementById('moly-sidebar-container')) return;

    // Try to connect to desktop app
    fetch('http://127.0.0.1:11436/api/status', { timeout: 2000 })
      .then(r => r.json())
      .then(data => {
        if (data.status === 'running') {
          injectSidebar();
        }
      })
      .catch(() => {
        // Desktop app not running, ask background script to launch it
        chrome.runtime.sendMessage({ action: 'launch-app' }, () => {
          // Retry after a delay
          setTimeout(() => {
            fetch('http://127.0.0.1:11436/api/status', { timeout: 2000 })
              .then(r => r.json())
              .then(data => {
                if (data.status === 'running') {
                  injectSidebar();
                }
              })
              .catch(() => {
                // Still not running, give up
              });
          }, 2000);
        });
      });
  }

  function toggleSidebar() {
    const existing = document.getElementById('moly-sidebar-container');
    if (existing) {
      existing.style.display = existing.style.display === 'none' ? 'block' : 'none';
      return;
    }
    // App might not be running, launch it
    fetch('http://127.0.0.1:11436/api/status', { timeout: 2000 })
      .then(r => r.json())
      .then(data => {
        if (data.status === 'running') {
          injectSidebar();
        }
      })
      .catch(() => {
        // App not running, launch it
        chrome.runtime.sendMessage({ action: 'launch-app' }, () => {
          setTimeout(() => {
            injectSidebar();
          }, 3000);
        });
      });
  }

  function injectSidebar() {
    // Create container
    const container = document.createElement('div');
    container.id = 'moly-sidebar-container';
    container.style.cssText = `
      position: fixed;
      right: 0;
      top: 0;
      width: 400px;
      height: 100vh;
      z-index: 999999;
      border-left: 1px solid #ddd;
      box-shadow: -2px 0 8px rgba(0,0,0,0.1);
    `;

    // Create iframe
    const iframe = document.createElement('iframe');
    iframe.src = 'http://127.0.0.1:11436/sidebar.html';
    iframe.style.cssText = `
      width: 100%;
      height: 100%;
      border: none;
      background: white;
    `;

    container.appendChild(iframe);
    document.body.appendChild(container);

    console.log('[Moly] Sidebar injected');
  }
})();
