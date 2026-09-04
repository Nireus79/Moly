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
    // Show loading state immediately
    injectSidebarWithLoader();
    // Launch app asynchronously in background
    launchAppAsync();
  }

  function launchAppAsync() {
    chrome.runtime.sendMessage({ action: 'launch-app' }, (response) => {
      console.log('[Moly] Launch response:', response);
    });
  }

  function injectSidebarWithLoader() {
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
      background: white;
    `;

    // Create loading content
    const loader = document.createElement('div');
    loader.id = 'moly-loader';
    loader.style.cssText = `
      width: 100%;
      height: 100%;
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      gap: 16px;
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
    `;
    loader.innerHTML = `
      <div style="font-size: 24px; animation: spin 1s linear infinite;">⚙️</div>
      <div style="color: #666; font-size: 14px;">Starting Moly...</div>
      <div style="color: #999; font-size: 12px;">This may take a moment</div>
      <style>
        @keyframes spin {
          from { transform: rotate(0deg); }
          to { transform: rotate(360deg); }
        }
      </style>
    `;

    container.appendChild(loader);
    document.body.appendChild(container);

    console.log('[Moly] Sidebar loader injected - starting app');

    // Poll for app to be ready
    const pollInterval = setInterval(async () => {
      try {
        const response = await fetch('http://127.0.0.1:11436/api/status', { timeout: 2000 });
        if (response.ok) {
          const data = await response.json();
          if (data.status === 'running') {
            clearInterval(pollInterval);
            console.log('[Moly] App ready - loading sidebar');
            replaceLoaderWithSidebar(container);
          }
        }
      } catch (e) {
        // Still connecting...
      }
    }, 500);

    // Timeout after 90 seconds
    setTimeout(() => {
      clearInterval(pollInterval);
      if (document.getElementById('moly-loader')) {
        loader.innerHTML = `
          <div style="color: #d32f2f; font-size: 14px; text-align: center; padding: 20px;">
            <div style="margin-bottom: 8px;">Failed to start Moly</div>
            <div style="font-size: 12px; color: #999;">This may take a minute or two on first startup</div>
            <div style="font-size: 12px; color: #999;">Please ensure Node.js npm are installed</div>
          </div>
        `;
      }
    }, 90000);
  }

  function replaceLoaderWithSidebar(container) {
    const loader = document.getElementById('moly-loader');
    if (!loader) return;

    // Create iframe
    const iframe = document.createElement('iframe');
    iframe.src = 'http://127.0.0.1:11436/sidebar.html';
    iframe.id = 'moly-sidebar-frame';
    iframe.style.cssText = `
      width: 100%;
      height: 100%;
      border: none;
      background: white;
    `;

    // Replace loader with iframe
    container.innerHTML = '';
    container.appendChild(iframe);
    console.log('[Moly] Sidebar loaded');
  }

  function injectSidebar() {
    // Legacy function - now uses async loader
    injectSidebarWithLoader();
  }
})();
