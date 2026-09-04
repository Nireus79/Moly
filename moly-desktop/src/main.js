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
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; }
    #moly-sidebar { width: 400px; height: 100vh; background: #f5f5f5; border-left: 1px solid #ddd; display: flex; flex-direction: column; }
    .sidebar-header { padding: 16px; background: #667eea; color: white; font-weight: bold; }
    .messages-area { flex: 1; overflow-y: auto; padding: 16px; }
    .message { margin: 8px 0; padding: 8px; border-radius: 4px; font-size: 14px; }
    .message-user { background: #667eea; color: white; margin-left: 20px; }
    .message-assistant { background: #e0e0e0; color: #333; margin-right: 20px; }
    .input-area { padding: 12px; border-top: 1px solid #ddd; }
    textarea { width: 100%; height: 60px; border: 1px solid #ddd; border-radius: 4px; padding: 8px; font-family: inherit; resize: none; }
    button { width: 100%; margin-top: 8px; padding: 10px; background: #667eea; color: white; border: none; border-radius: 4px; cursor: pointer; font-weight: bold; }
    button:hover { background: #5568d3; }
  </style>
</head>
<body>
  <div id="moly-sidebar">
    <div class="sidebar-header">Moly Chat</div>
    <div class="messages-area" id="messages"></div>
    <div class="input-area">
      <textarea id="message" placeholder="Type message..."></textarea>
      <button onclick="sendMessage()">Send</button>
    </div>
  </div>
  <script>
    const messages = [];
    const selectedModel = 'mistral';
    const provider = 'ollama';

    function buildSystemPrompt() {
      return \`You are Moly, an AI coach helping users craft better messages.\`;
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
            model: selectedModel,
            prompt: buildSystemPrompt() + '\\n\\n' + text,
            stream: false
          })
        });

        const data = await response.json();
        if (data.response) {
          messages.push({type: 'assistant', text: data.response});
          renderMessages();
        }
      } catch (e) {
        messages.push({type: 'error', text: 'Error: ' + e.message});
        renderMessages();
      }
    }

    function renderMessages() {
      const area = document.getElementById('messages');
      area.innerHTML = messages.map((m, i) => \`
        <div class="message message-\${m.type}">\${m.text}</div>
      \`).join('');
      area.scrollTop = area.scrollHeight;
    }

    document.getElementById('message').addEventListener('keypress', (e) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        sendMessage();
      }
    });
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
