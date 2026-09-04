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
    console.log('[Moly] Attempting to launch app via native host');
    chrome.runtime.sendMessage({ action: 'launch-app' }, (response) => {
      console.log('[Moly] Launch response:', response);
      if (chrome.runtime.lastError) {
        console.error('[Moly] Native host error:', chrome.runtime.lastError);
      }
    });
  }

  function showManualLaunchPrompt() {
    const loader = document.getElementById('moly-loader');
    if (!loader) return;

    loader.innerHTML = `
      <div style="padding: 20px; text-align: center; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;">
        <div style="font-size: 24px; margin-bottom: 16px;">⚙️</div>
        <div style="color: #333; font-size: 14px; margin-bottom: 8px; font-weight: 500;">Start Moly Desktop App</div>
        <div style="color: #666; font-size: 12px; margin-bottom: 16px; line-height: 1.5;">
          Open a terminal and run:<br>
          <code style="background: #f5f5f5; padding: 8px 12px; border-radius: 4px; display: inline-block; margin-top: 8px; font-family: monospace; font-size: 11px;">
            cd ~/vs_projects/Moly/Moly/moly-desktop && npm start
          </code>
        </div>
        <div style="color: #999; font-size: 11px;">Once running, refresh this page</div>
      </div>
    `;
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

    // Show manual launch prompt after 10 seconds (native host likely didn't work)
    setTimeout(() => {
      const loader = document.getElementById('moly-loader');
      if (loader) {
        // Check if still loading (hasn't connected yet)
        const stillLoading = loader.textContent && loader.textContent.includes('Starting Moly');
        if (stillLoading) {
          showManualLaunchPrompt();
        }
      }
    }, 10000);

    // Final timeout after 120 seconds
    setTimeout(() => {
      clearInterval(pollInterval);
      if (document.getElementById('moly-loader')) {
        const loader = document.getElementById('moly-loader');
        loader.innerHTML = `
          <div style="padding: 20px; text-align: center; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;">
            <div style="font-size: 14px; color: #d32f2f; margin-bottom: 8px;">Still waiting...</div>
            <div style="font-size: 12px; color: #666; line-height: 1.6;">
              If you started the app but it's still loading, please wait or refresh the page.
            </div>
          </div>
        `;
      }
    }, 120000);
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
