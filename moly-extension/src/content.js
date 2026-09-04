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

  function toggleSidebar() {
    const existing = document.getElementById('moly-sidebar-container');
    if (existing) {
      existing.style.display = existing.style.display === 'none' ? 'block' : 'none';
      return;
    }
    injectSidebarWithLoader();
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
      <div style="color: #666; font-size: 14px;">Loading Moly...</div>
      <div style="color: #999; font-size: 12px;">Just a moment</div>
      <style>
        @keyframes spin {
          from { transform: rotate(0deg); }
          to { transform: rotate(360deg); }
        }
      </style>
    `;

    container.appendChild(loader);
    document.body.appendChild(container);

    console.log('[Moly] Sidebar loader injected');

    // Poll for app to be ready
    const pollInterval = setInterval(async () => {
      try {
        const response = await fetch('http://127.0.0.1:11436/api/status', { timeout: 2000 });
        if (response.ok) {
          const data = await response.json();
          if (data.status === 'running') {
            clearInterval(pollInterval);
            console.log('[Moly] App connected');
            replaceLoaderWithSidebar(container);
          }
        }
      } catch (e) {
        // Connecting...
      }
    }, 500);

    // Timeout after 30 seconds
    setTimeout(() => {
      clearInterval(pollInterval);
      const loader = document.getElementById('moly-loader');
      if (loader) {
        loader.innerHTML = `
          <div style="padding: 20px; text-align: center; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; color: #d32f2f;">
            <div style="font-size: 14px; margin-bottom: 8px; font-weight: 500;">Moly Not Running</div>
            <div style="font-size: 12px; color: #666; line-height: 1.6; margin-bottom: 16px;">
              Desktop app is not running.
            </div>
            <div style="font-size: 12px; color: #666;">
              Run in terminal:<br>
              <code style="background: #f5f5f5; padding: 8px; border-radius: 4px; display: block; margin-top: 8px; font-family: monospace; font-size: 11px;">
                cd ~/vs_projects/Moly/Moly/moly-go && bash install.sh
              </code>
            </div>
          </div>
        `;
      }
    }, 30000);
  }

  function replaceLoaderWithSidebar(container) {
    const loader = document.getElementById('moly-loader');
    if (!loader) return;

    const iframe = document.createElement('iframe');
    iframe.src = 'http://127.0.0.1:11436/sidebar.html';
    iframe.id = 'moly-sidebar-frame';
    iframe.style.cssText = `
      width: 100%;
      height: 100%;
      border: none;
      background: white;
    `;

    container.innerHTML = '';
    container.appendChild(iframe);
    console.log('[Moly] Sidebar connected');
  }

  // Inject when extension icon is clicked
  toggleSidebar();
})();
