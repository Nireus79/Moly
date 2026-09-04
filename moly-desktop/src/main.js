const { app, BrowserWindow, Menu, ipcMain, dialog } = require('electron');
const path = require('path');
const { spawn } = require('child_process');
const os = require('os');
const fs = require('fs');
const http = require('http');
const MolyInstaller = require('./services/installer');

let mainWindow;
let sidebarWindow;
let nativeHostProcess;
const installer = new MolyInstaller();

// Paths
const HOME = os.homedir();
const PLATFORM = process.platform;
const INSTALL_DIR = path.join(HOME, '.local', 'bin');
const NATIVE_HOST_PATH = path.join(INSTALL_DIR, 'moly-native-host');

function createWindow() {
  const iconPath = path.join(__dirname, '../assets/icon.png');
  const isDev = !app.isPackaged;

  const windowConfig = {
    width: 1200,
    height: 800,
    show: false,
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      nodeIntegration: false,
      contextIsolation: true,
      sandbox: !isDev,
    }
  };

  if (fs.existsSync(iconPath)) {
    windowConfig.icon = iconPath;
  }

  mainWindow = new BrowserWindow(windowConfig);

  const startUrl = `file://${path.join(__dirname, '../build/index.html')}`;
  mainWindow.loadURL(startUrl);

  mainWindow.on('closed', () => {
    mainWindow = null;
  });

  mainWindow.on('ready-to-show', () => {
    mainWindow.hide();
  });
}

function createSidebar() {
  if (sidebarWindow) {
    sidebarWindow.focus();
    return;
  }

  const iconPath = path.join(__dirname, '../assets/icon.png');
  sidebarWindow = new BrowserWindow({
    width: 420,
    height: 600,
    x: -420,
    y: 0,
    alwaysOnTop: true,
    frame: true,
    skipTaskbar: false,
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      nodeIntegration: false,
      contextIsolation: true,
    }
  });

  if (fs.existsSync(iconPath)) {
    sidebarWindow.setIcon(iconPath);
  }

  const startUrl = `file://${path.join(__dirname, '../build/index.html')}?mode=sidebar`;
  sidebarWindow.loadURL(startUrl);

  sidebarWindow.on('closed', () => {
    sidebarWindow = null;
  });
}

function startNativeHost() {
  // No longer needed - Electron app calls Ollama directly
  // Leaving this function empty for backwards compatibility
}

function stopNativeHost() {
  if (nativeHostProcess) {
    nativeHostProcess.kill();
    nativeHostProcess = null;
  }
}

// IPC Handlers
ipcMain.handle('check-native-host', () => {
  return fs.existsSync(NATIVE_HOST_PATH);
});

ipcMain.handle('get-native-host-path', () => {
  return NATIVE_HOST_PATH;
});

ipcMain.handle('get-proxy-status', async () => {
  try {
    const response = await fetch('http://127.0.0.1:11435/api/tags');
    return response.status === 200;
  } catch {
    return false;
  }
});

ipcMain.handle('install-native-host', async () => {
  // No longer needed - app calls Ollama directly
  return { success: true };
});

ipcMain.handle('get-system-info', () => {
  return {
    platform: PLATFORM,
    arch: os.arch(),
    version: app.getVersion(),
  };
});

ipcMain.handle('start-setup', async () => {
  return { setupComplete: fs.existsSync(NATIVE_HOST_PATH) };
});

ipcMain.handle('open-sidebar', () => {
  createSidebar();
  return { success: true };
});

// Start HTTP server for browser extension sidebar
function startSidebarServer() {
  const server = http.createServer((req, res) => {
    res.setHeader('Access-Control-Allow-Origin', '*');
    res.setHeader('Access-Control-Allow-Methods', 'GET, POST, OPTIONS');
    res.setHeader('Access-Control-Allow-Headers', 'Content-Type');

    if (req.method === 'OPTIONS') {
      res.writeHead(200);
      res.end();
      return;
    }

    // Sidebar HTML
    if (req.url === '/sidebar.html') {
      res.writeHead(200, {'Content-Type': 'text/html'});
      const html = `
<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    html, body { height: 100%; }
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #fafafa; }

    .sidebar { width: 400px; height: 100vh; display: flex; flex-direction: column; background: white; }

    .header {
      padding: 12px 16px;
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      color: white;
      display: flex;
      align-items: center;
      justify-content: space-between;
      flex-shrink: 0;
    }

    .header-title { font-weight: 600; font-size: 16px; }

    .header-buttons {
      display: flex;
      gap: 8px;
    }

    .icon-btn {
      width: 32px;
      height: 32px;
      border: none;
      background: rgba(255,255,255,0.2);
      color: white;
      border-radius: 6px;
      cursor: pointer;
      font-size: 16px;
      display: flex;
      align-items: center;
      justify-content: center;
      transition: background 0.2s;
    }

    .icon-btn:hover { background: rgba(255,255,255,0.3); }

    .messages-area {
      flex: 1;
      overflow-y: auto;
      padding: 16px;
      display: flex;
      flex-direction: column;
      gap: 12px;
    }

    .message {
      padding: 10px 14px;
      border-radius: 8px;
      font-size: 13px;
      line-height: 1.4;
      word-wrap: break-word;
      max-width: 90%;
    }

    .message-user {
      align-self: flex-end;
      background: #667eea;
      color: white;
      border-radius: 8px 2px 8px 8px;
    }

    .message-assistant {
      align-self: flex-start;
      background: #e8e8e8;
      color: #333;
      border-radius: 2px 8px 8px 8px;
    }

    .message-error {
      align-self: flex-start;
      background: #ffebee;
      color: #c62828;
      border-radius: 2px 8px 8px 8px;
    }

    .settings-panel {
      border-top: 1px solid #e0e0e0;
      padding: 12px;
      background: #f5f5f5;
      display: none;
      flex-direction: column;
      gap: 10px;
      max-height: 150px;
      overflow-y: auto;
      flex-shrink: 0;
    }

    .settings-panel.show { display: flex; }

    .setting-group {
      display: flex;
      flex-direction: column;
      gap: 4px;
    }

    .setting-label {
      font-size: 12px;
      font-weight: 500;
      color: #666;
      text-transform: uppercase;
    }

    select {
      padding: 6px 8px;
      border: 1px solid #ddd;
      border-radius: 4px;
      font-size: 12px;
      background: white;
      cursor: pointer;
    }

    select:focus { outline: none; border-color: #667eea; }

    .input-area {
      padding: 12px;
      border-top: 1px solid #e0e0e0;
      background: white;
      flex-shrink: 0;
      display: flex;
      flex-direction: column;
      gap: 8px;
    }

    textarea {
      width: 100%;
      height: 60px;
      border: 1px solid #ddd;
      border-radius: 6px;
      padding: 8px;
      font-family: inherit;
      font-size: 13px;
      resize: none;
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, monospace;
    }

    textarea:focus { outline: none; border-color: #667eea; }

    .button-row {
      display: flex;
      gap: 8px;
    }

    .send-btn, .clear-btn {
      flex: 1;
      padding: 8px 12px;
      border: none;
      border-radius: 6px;
      font-size: 13px;
      font-weight: 500;
      cursor: pointer;
      transition: background 0.2s;
    }

    .send-btn {
      background: #667eea;
      color: white;
    }

    .send-btn:hover { background: #5568d3; }

    .clear-btn {
      background: #e0e0e0;
      color: #333;
    }

    .clear-btn:hover { background: #d0d0d0; }

    .loading { color: #999; font-style: italic; }
  </style>
</head>
<body>
  <div class="sidebar">
    <div class="header">
      <span class="header-title">Moly</span>
      <div class="header-buttons">
        <button class="icon-btn" onclick="toggleSettings()" title="Settings">⚙️</button>
        <button class="icon-btn" onclick="expandSidebar()" title="Expand">↗️</button>
      </div>
    </div>

    <div class="messages-area" id="messages"></div>

    <div class="settings-panel" id="settings">
      <div class="setting-group">
        <label class="setting-label">Model</label>
        <select id="model" onchange="saveSettings()">
          <option value="mistral">Mistral</option>
          <option value="llama2">Llama2</option>
          <option value="neural-chat">Neural Chat</option>
        </select>
      </div>

      <div class="setting-group">
        <label class="setting-label">Tone</label>
        <select id="tone" onchange="saveSettings()">
          <option value="friendly">Friendly</option>
          <option value="formal">Formal</option>
          <option value="playful">Playful</option>
        </select>
      </div>

      <div class="setting-group">
        <label class="setting-label">Mode</label>
        <select id="mode" onchange="saveSettings()">
          <option value="direct">Direct</option>
          <option value="socratic">Socratic</option>
        </select>
      </div>
    </div>

    <div class="input-area">
      <textarea id="message" placeholder="Type your message here..."></textarea>
      <div class="button-row">
        <button class="send-btn" onclick="sendMessage()">Send</button>
        <button class="clear-btn" onclick="clearChat()">Clear</button>
      </div>
    </div>
  </div>

  <script>
    const messages = [];
    let settings = {
      model: 'mistral',
      tone: 'friendly',
      mode: 'direct'
    };

    function loadSettings() {
      const saved = localStorage.getItem('moly-settings');
      if (saved) {
        settings = JSON.parse(saved);
        document.getElementById('model').value = settings.model;
        document.getElementById('tone').value = settings.tone;
        document.getElementById('mode').value = settings.mode;
      }
    }

    function saveSettings() {
      settings.model = document.getElementById('model').value;
      settings.tone = document.getElementById('tone').value;
      settings.mode = document.getElementById('mode').value;
      localStorage.setItem('moly-settings', JSON.stringify(settings));
    }

    function toggleSettings() {
      document.getElementById('settings').classList.toggle('show');
    }

    function expandSidebar() {
      const sidebar = document.querySelector('.sidebar');
      if (sidebar.style.width === '100vw') {
        sidebar.style.width = '400px';
      } else {
        sidebar.style.width = '100vw';
      }
    }

    function buildSystemPrompt() {
      const toneMap = {
        friendly: 'warm and friendly tone',
        formal: 'professional and formal tone',
        playful: 'playful and engaging tone'
      };
      const modeMap = {
        direct: 'Provide direct, ready-to-use responses.',
        socratic: 'Ask thoughtful questions to help the user develop their own response.'
      };
      return \`You are Moly, an AI coach helping users craft better messages. Use a \${toneMap[settings.tone]}. \${modeMap[settings.mode]}\`;
    }

    async function sendMessage() {
      const text = document.getElementById('message').value.trim();
      if (!text) return;

      document.getElementById('message').value = '';
      messages.push({type: 'user', text});
      renderMessages();

      try {
        const response = await fetch('http://127.0.0.1:11434/api/generate', {
          method: 'POST',
          headers: {'Content-Type': 'application/json'},
          body: JSON.stringify({
            model: settings.model,
            prompt: buildSystemPrompt() + '\\n\\nUser message: ' + text,
            stream: false
          })
        });

        if (!response.ok) throw new Error('API error: ' + response.status);

        const data = await response.json();
        if (data.response) {
          messages.push({type: 'assistant', text: data.response});
          renderMessages();
        }
      } catch (e) {
        messages.push({type: 'error', text: '⚠️ ' + e.message});
        renderMessages();
      }
    }

    function clearChat() {
      messages.length = 0;
      renderMessages();
    }

    function renderMessages() {
      const area = document.getElementById('messages');
      if (messages.length === 0) {
        area.innerHTML = '<div class="message loading">Start a conversation...</div>';
      } else {
        area.innerHTML = messages.map(m => \`
          <div class="message message-\${m.type}">\${escapeHtml(m.text)}</div>
        \`).join('');
      }
      area.scrollTop = area.scrollHeight;
    }

    function escapeHtml(text) {
      const div = document.createElement('div');
      div.textContent = text;
      return div.innerHTML;
    }

    document.getElementById('message').addEventListener('keypress', (e) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        sendMessage();
      }
    });

    loadSettings();
    renderMessages();
  </script>
</body>
</html>
      `;
      res.end(html);
      return;
    }

    // API endpoints
    if (req.url === '/api/status') {
      res.writeHead(200, {'Content-Type': 'application/json'});
      res.end(JSON.stringify({status: 'running'}));
      return;
    }

    res.writeHead(404);
    res.end('Not found');
  });

  server.listen(11436, '127.0.0.1', () => {
    console.log('[Moly] Sidebar server listening on 127.0.0.1:11436');
  });
}

// App lifecycle
app.on('ready', () => {
  startNativeHost();
  startSidebarServer();
  createWindow();
});

app.on('window-all-closed', () => {
  stopNativeHost();
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

app.on('activate', () => {
  if (mainWindow === null) {
    createWindow();
  }
});

console.log('[Moly] Desktop app initialized');
