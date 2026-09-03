// Inject Moly sidebar from desktop app
(function() {
  // Check if sidebar already injected
  if (document.getElementById('moly-sidebar-container')) return;

  // Try to connect to desktop app
  fetch('http://127.0.0.1:11436/api/status')
    .then(r => r.json())
    .then(data => {
      if (data.status === 'running') {
        injectSidebar();
      }
    })
    .catch(() => {
      // Desktop app not running
    });

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
