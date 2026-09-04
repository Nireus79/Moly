const { app, BrowserWindow, Menu, ipcMain, dialog } = require('electron');
const path = require('path');
const { spawn } = require('child_process');
const os = require('os');
const fs = require('fs');
const http = require('http');
const url = require('url');
const MolyInstaller = require('./services/installer');
const ModelManager = require('./services/modelManager');

let mainWindow;
let sidebarWindow;
let nativeHostProcess;
const installer = new MolyInstaller();
const modelManager = new ModelManager();

// Paths
const HOME = os.homedir();
const PLATFORM = process.platform;
const INSTALL_DIR = path.join(HOME, '.local', 'bin');
const NATIVE_HOST_PATH = path.join(INSTALL_DIR, 'moly-native-host');

function createWindow() {
  const iconPath = path.join(__dirname, '../assets/icon.png');
  const isDev = !app.isPackaged;

  const windowConfig = {
    width: 1400,
    height: 900,
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

  const startUrl = `file://${path.join(__dirname, '../build/integrated-layout.html')}`;
  mainWindow.loadURL(startUrl);

  mainWindow.on('closed', () => {
    mainWindow = null;
  });

  mainWindow.on('ready-to-show', () => {
    mainWindow.show();
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
  if (mainWindow) {
    mainWindow.focus();
  }
  return { success: true };
});

// Start HTTP server for browser extension sidebar
function startSidebarServer() {
  const server = http.createServer(async (req, res) => {
    res.setHeader('Access-Control-Allow-Origin', '*');
    res.setHeader('Access-Control-Allow-Methods', 'GET, POST, OPTIONS');
    res.setHeader('Access-Control-Allow-Headers', 'Content-Type');

    if (req.method === 'OPTIONS') {
      res.writeHead(200);
      res.end();
      return;
    }

    const parsedUrl = url.parse(req.url, true);
    const pathname = parsedUrl.pathname;
    const query = parsedUrl.query;

    // Helper to read request body
    const readBody = (req) => {
      return new Promise((resolve, reject) => {
        let data = '';
        req.on('data', chunk => {
          data += chunk;
        });
        req.on('end', () => {
          try {
            resolve(data ? JSON.parse(data) : {});
          } catch (e) {
            resolve({});
          }
        });
        req.on('error', reject);
      });
    };

    // API Routes
    if (pathname !== '/sidebar.html' && !pathname.includes('.html')) {
      console.log('[Moly] HTTP Request -', req.method, pathname);
    }

    if (pathname === '/api/status') {
      res.writeHead(200, {'Content-Type': 'application/json'});
      res.end(JSON.stringify({status: 'running'}));
      return;
    }

    if (pathname === '/api/first-run-check') {
      res.writeHead(200, {'Content-Type': 'application/json'});
      res.end(JSON.stringify({
        ollama_installed: false,
        ollama_running: false,
        first_run_complete: false
      }));
      return;
    }

    if (pathname === '/api/models/list') {
      try {
        const result = await modelManager.getInstalledModels();
        res.writeHead(200, {'Content-Type': 'application/json'});
        res.end(JSON.stringify(result));
      } catch (e) {
        console.error('[Moly API Error] models/list:', e.message);
        res.writeHead(200, {'Content-Type': 'application/json'});
        res.end(JSON.stringify({ models: [], error: e.message }));
      }
      return;
    }

    if (pathname === '/api/models/pull') {
      if (req.method !== 'POST') {
        res.writeHead(405);
        res.end('Method not allowed');
        return;
      }
      try {
        const body = await readBody(req);
        const modelName = body.name || query.name;
        if (!modelName) {
          res.writeHead(400, {'Content-Type': 'application/json'});
          res.end(JSON.stringify({error: 'Model name required'}));
          return;
        }
        const result = await modelManager.pullModel(modelName);
        res.writeHead(200, {'Content-Type': 'application/json'});
        res.end(JSON.stringify(result));
      } catch (e) {
        console.error('[Moly API Error] models/pull:', e.message);
        res.writeHead(200, {'Content-Type': 'application/json'});
        res.end(JSON.stringify({success: false, error: e.message}));
      }
      return;
    }

    if (pathname === '/api/models/remove') {
      if (req.method !== 'POST') {
        res.writeHead(405);
        res.end('Method not allowed');
        return;
      }
      try {
        const body = await readBody(req);
        const modelName = body.name || query.name;
        if (!modelName) {
          res.writeHead(400, {'Content-Type': 'application/json'});
          res.end(JSON.stringify({error: 'Model name required'}));
          return;
        }
        const result = await modelManager.removeModel(modelName);
        res.writeHead(200, {'Content-Type': 'application/json'});
        res.end(JSON.stringify(result));
      } catch (e) {
        console.error('[Moly API Error] models/remove:', e.message);
        res.writeHead(200, {'Content-Type': 'application/json'});
        res.end(JSON.stringify({success: false, error: e.message}));
      }
      return;
    }

    if (pathname === '/api/ollama/start') {
      if (req.method !== 'POST') {
        res.writeHead(405);
        res.end('Method not allowed');
        return;
      }
      try {
        const result = await modelManager.startOllama();
        res.writeHead(200, {'Content-Type': 'application/json'});
        res.end(JSON.stringify(result));
      } catch (e) {
        console.error('[Moly API Error] ollama/start:', e.message);
        res.writeHead(200, {'Content-Type': 'application/json'});
        res.end(JSON.stringify({success: false, error: e.message}));
      }
      return;
    }

    if (pathname === '/api/ollama/stop') {
      if (req.method !== 'POST') {
        res.writeHead(405);
        res.end('Method not allowed');
        return;
      }
      try {
        const result = await modelManager.stopOllama();
        res.writeHead(200, {'Content-Type': 'application/json'});
        res.end(JSON.stringify(result));
      } catch (e) {
        console.error('[Moly API Error] ollama/stop:', e.message);
        res.writeHead(200, {'Content-Type': 'application/json'});
        res.end(JSON.stringify({success: false, error: e.message}));
      }
      return;
    }

    if (pathname === '/api/settings') {
      try {
        if (req.method === 'GET') {
          const config = modelManager.loadConfig();
          res.writeHead(200, {'Content-Type': 'application/json'});
          res.end(JSON.stringify(config));
          return;
        }
        if (req.method === 'POST') {
          const body = await readBody(req);
          const config = modelManager.loadConfig();
          const updated = { ...config, ...body, updated_at: new Date().toISOString() };
          modelManager.saveConfig(updated);
          res.writeHead(200, {'Content-Type': 'application/json'});
          res.end(JSON.stringify({success: true, config: updated}));
          return;
        }
      } catch (e) {
        console.error('[Moly API Error] settings:', e.message);
        res.writeHead(200, {'Content-Type': 'application/json'});
        res.end(JSON.stringify({success: false, error: e.message}));
        return;
      }
      res.writeHead(405);
      res.end('Method not allowed');
      return;
    }

    // Sidebar HTML
    if (pathname === '/sidebar.html') {
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

    .modal-overlay {
      display: none;
      position: fixed;
      top: 0;
      left: 0;
      right: 0;
      bottom: 0;
      background: rgba(0, 0, 0, 0.5);
      z-index: 1000000;
      align-items: center;
      justify-content: center;
    }

    .modal-overlay.show {
      display: flex;
    }

    .modal {
      background: white;
      border-radius: 12px;
      padding: 32px;
      max-width: 400px;
      width: 90%;
      box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
      max-height: 90vh;
      overflow-y: auto;
    }

    .modal-title {
      font-size: 24px;
      font-weight: 600;
      color: #333;
      margin-bottom: 16px;
    }

    .modal-subtitle {
      font-size: 14px;
      color: #666;
      margin-bottom: 24px;
      line-height: 1.5;
    }

    .modal-option {
      padding: 16px;
      border: 2px solid #ddd;
      border-radius: 8px;
      margin-bottom: 12px;
      cursor: pointer;
      transition: all 0.2s;
    }

    .modal-option:hover {
      border-color: #667eea;
      background: #f9f9ff;
    }

    .modal-option.selected {
      border-color: #667eea;
      background: #f0f2ff;
    }

    .modal-option-title {
      font-weight: 600;
      color: #333;
      margin-bottom: 4px;
    }

    .modal-option-desc {
      font-size: 12px;
      color: #999;
    }

    .modal-buttons {
      display: flex;
      gap: 12px;
      margin-top: 24px;
    }

    .modal-btn {
      flex: 1;
      padding: 12px 16px;
      border: none;
      border-radius: 6px;
      font-size: 14px;
      font-weight: 600;
      cursor: pointer;
      transition: background 0.2s;
    }

    .modal-btn-primary {
      background: #667eea;
      color: white;
    }

    .modal-btn-primary:hover {
      background: #5568d3;
    }

    .modal-btn-secondary {
      background: #e8e8e8;
      color: #333;
    }

    .modal-btn-secondary:hover {
      background: #d8d8d8;
    }
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
        <div style="display: flex; gap: 8px;">
          <select id="model" onchange="saveSettings()" style="flex: 1;">
            <option value="">Loading models...</option>
          </select>
          <button class="icon-btn" onclick="refreshModels()" title="Refresh" style="width: 32px; height: 32px; margin: 0;">🔄</button>
        </div>
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

      <div style="border-top: 1px solid #ddd; margin-top: 10px; padding-top: 10px;">
        <label class="setting-label">Ollama</label>
        <div id="ollama-status" style="font-size: 11px; color: #999; margin-bottom: 8px;">Checking...</div>
        <div id="model-controls" style="display: flex; gap: 4px; flex-wrap: wrap;">
          <button class="send-btn" onclick="startOllama()" title="Start Ollama" style="flex: 1; padding: 6px; font-size: 12px;">Start</button>
          <button class="clear-btn" onclick="stopOllama()" title="Stop Ollama" style="flex: 1; padding: 6px; font-size: 12px;">Stop</button>
        </div>
      </div>

      <div style="border-top: 1px solid #ddd; margin-top: 10px; padding-top: 10px;">
        <label class="setting-label">Install Models</label>
        <div id="model-install-options" style="display: none;">
          <div id="available-models" style="max-height: 120px; overflow-y: auto; margin-bottom: 8px;"></div>
          <div id="install-progress" style="display: none; margin-bottom: 8px;">
            <div id="install-progress-text" style="font-size: 11px; color: #667eea; margin-bottom: 4px;">Installing...</div>
            <div style="background: #e8e8e8; height: 6px; border-radius: 3px; overflow: hidden;">
              <div id="install-progress-bar" style="background: #667eea; height: 100%; width: 0%; transition: width 0.3s;"></div>
            </div>
          </div>
        </div>
        <div id="cloud-notice" style="font-size: 11px; color: #999; display: none;">Model installation only available with Ollama.</div>
      </div>
    </div>

    <div class="input-area">
      <textarea id="message" placeholder="Type your message here..."></textarea>
      <div class="button-row">
        <button id="sendBtn" class="send-btn">Send</button>
        <button id="clearBtn" class="clear-btn">Clear</button>
      </div>
    </div>

    <div id="firstRunModal" class="modal-overlay">
      <div class="modal">
        <div class="modal-title">Welcome to Moly</div>
        <div class="modal-subtitle">Let's set up your AI coaching assistant. How would you like to use Moly?</div>

        <div id="providerOptions">
          <div class="modal-option" onclick="selectProvider('local')">
            <div class="modal-option-title">Local Models (Ollama)</div>
            <div class="modal-option-desc">Fast, private, no API key needed. Requires Ollama installed.</div>
          </div>
          <div class="modal-option" onclick="selectProvider('claude')">
            <div class="modal-option-title">Claude (Anthropic)</div>
            <div class="modal-option-desc">High quality, requires API key. Pay per use.</div>
          </div>
          <div class="modal-option" onclick="selectProvider('openai')">
            <div class="modal-option-title">ChatGPT (OpenAI)</div>
            <div class="modal-option-desc">Popular, requires API key. Pay per use.</div>
          </div>
        </div>

        <div id="apiKeyInput" style="display: none; margin-top: 20px;">
          <label style="display: block; margin-bottom: 8px; font-size: 14px; font-weight: 600; color: #333;">API Key</label>
          <input type="password" id="apiKeyField" placeholder="Enter your API key" style="width: 100%; padding: 8px; border: 1px solid #ddd; border-radius: 4px; font-size: 14px; box-sizing: border-box;">
          <div style="font-size: 12px; color: #999; margin-top: 8px;">Your API key is stored locally and never sent to external servers.</div>
        </div>

        <div class="modal-buttons">
          <button class="modal-btn modal-btn-primary" onclick="completeFirstRun()">Continue</button>
        </div>
      </div>
    </div>
  </div>

  <script>
    const messages = [];
    let settings = {
      model: 'mistral',
      tone: 'friendly',
      mode: 'direct',
      provider: null
    };
    let selectedProvider = null;

    async function loadInstalledModels() {
      try {
        const response = await fetch('http://127.0.0.1:11436/api/models/list');
        const data = await response.json();
        const modelSelect = document.getElementById('model');
        modelSelect.innerHTML = '';

        if (data.models && data.models.length > 0) {
          data.models.forEach(model => {
            const name = typeof model === 'string' ? model : model.name;
            const option = document.createElement('option');
            option.value = name;
            option.textContent = name;
            modelSelect.appendChild(option);
          });
          if (settings.model && data.models.includes(settings.model)) {
            modelSelect.value = settings.model;
          }
        } else {
          const option = document.createElement('option');
          option.value = '';
          option.textContent = 'No local models found';
          modelSelect.appendChild(option);
          modelSelect.disabled = true;
        }
      } catch (e) {
        console.error('Error loading models:', e);
        const modelSelect = document.getElementById('model');
        modelSelect.innerHTML = '<option value="">Error loading models</option>';
      }
    }

    async function loadSettings() {
      try {
        const response = await fetch('http://127.0.0.1:11436/api/settings');
        const serverSettings = await response.json();
        settings = {
          model: serverSettings.model || 'mistral',
          tone: serverSettings.tone || 'friendly',
          mode: serverSettings.mode || 'direct',
          provider: serverSettings.provider || 'local',
          api_keys: serverSettings.api_keys || {}
        };
      } catch (e) {
        const saved = localStorage.getItem('moly-settings');
        if (saved) {
          settings = JSON.parse(saved);
        }
      }

      document.getElementById('tone').value = settings.tone;
      document.getElementById('mode').value = settings.mode;

      if (settings.provider === 'local') {
        await loadInstalledModels();
      } else {
        const modelSelect = document.getElementById('model');
        modelSelect.innerHTML = '';
        const providers = {
          claude: 'claude-3-sonnet',
          openai: 'gpt-4-turbo'
        };
        if (settings.provider && providers[settings.provider]) {
          const option = document.createElement('option');
          option.value = providers[settings.provider];
          option.textContent = providers[settings.provider];
          modelSelect.appendChild(option);
          modelSelect.value = providers[settings.provider];
          modelSelect.disabled = true;
        }
      }

      updateModelInstallUI();
    }

    async function refreshModels() {
      await loadInstalledModels();
    }

    async function checkFirstRun() {
      try {
        const response = await fetch('http://127.0.0.1:11436/api/first-run-check');
        const data = await response.json();

        if (!data.first_run_complete) {
          showFirstRunModal(data.ollama_installed);
          return true;
        }
        return false;
      } catch (e) {
        console.log('First-run check error:', e);
        return false;
      }
    }

    function showFirstRunModal(ollamaInstalled) {
      const modal = document.getElementById('firstRunModal');
      modal.classList.add('show');

      if (!ollamaInstalled) {
        const localOption = document.querySelector('[onclick="selectProvider(\'local\')"]');
        if (localOption) {
          localOption.style.opacity = '0.5';
          localOption.style.pointerEvents = 'none';
          const desc = localOption.querySelector('.modal-option-desc');
          desc.textContent = 'Ollama not installed on this system.';
        }
      }
    }

    function selectProvider(provider) {
      selectedProvider = provider;

      document.querySelectorAll('.modal-option').forEach(el => {
        el.classList.remove('selected');
      });

      event.target.closest('.modal-option').classList.add('selected');

      if (provider === 'local') {
        document.getElementById('apiKeyInput').style.display = 'none';
      } else {
        document.getElementById('apiKeyInput').style.display = 'block';
      }
    }

    async function completeFirstRun() {
      if (!selectedProvider) {
        alert('Please select a provider');
        return;
      }

      let newSettings = { ...settings };
      newSettings.provider = selectedProvider;

      if (selectedProvider === 'local') {
        newSettings.model = 'mistral';
        await checkOllamaStatus();
      } else {
        const apiKey = document.getElementById('apiKeyField').value.trim();
        if (!apiKey) {
          alert('Please enter your API key');
          return;
        }
        newSettings.api_keys = newSettings.api_keys || {};
        newSettings.api_keys[selectedProvider] = apiKey;
        newSettings.model = selectedProvider === 'claude' ? 'claude-3-sonnet' : 'gpt-4-turbo';
      }

      newSettings.first_run_complete = true;

      try {
        const response = await fetch('http://127.0.0.1:11436/api/settings', {
          method: 'POST',
          headers: {'Content-Type': 'application/json'},
          body: JSON.stringify(newSettings)
        });

        if (response.ok) {
          settings = newSettings;
          document.getElementById('firstRunModal').classList.remove('show');
          await loadSettings();
          updateModelInstallUI();
        }
      } catch (e) {
        alert('Error saving settings: ' + e.message);
      }
    }

    async function checkOllamaStatus() {
      try {
        const response = await fetch('http://127.0.0.1:11436/api/first-run-check');
        const data = await response.json();
        const statusEl = document.getElementById('ollama-status');

        if (data.ollama_installed) {
          if (data.ollama_running) {
            statusEl.textContent = 'Ollama: Running';
            statusEl.style.color = '#2ecc71';
          } else {
            statusEl.textContent = 'Ollama: Installed but not running';
            statusEl.style.color = '#f39c12';
          }
        } else {
          statusEl.textContent = 'Ollama: Not installed';
          statusEl.style.color = '#e74c3c';
        }
      } catch (e) {
        document.getElementById('ollama-status').textContent = 'Error checking Ollama';
      }
    }

    async function startOllama() {
      try {
        const response = await fetch('http://127.0.0.1:11436/api/ollama/start', {method: 'POST'});
        const data = await response.json();
        if (data.success) {
          document.getElementById('ollama-status').textContent = 'Ollama: Starting...';
          setTimeout(checkOllamaStatus, 2000);
        }
      } catch (e) {
        alert('Error starting Ollama: ' + e.message);
      }
    }

    async function stopOllama() {
      try {
        const response = await fetch('http://127.0.0.1:11436/api/ollama/stop', {method: 'POST'});
        const data = await response.json();
        if (data.success) {
          checkOllamaStatus();
        }
      } catch (e) {
        alert('Error stopping Ollama: ' + e.message);
      }
    }

    const AVAILABLE_MODELS = [
      { name: 'mistral', size: '4.1GB', desc: 'Fast, good quality' },
      { name: 'llama2', size: '3.8GB', desc: 'Versatile, well-known' },
      { name: 'neural-chat', size: '4.0GB', desc: 'Great for chat' },
      { name: 'orca-mini', size: '1.3GB', desc: 'Tiny, fast' },
      { name: 'dolphin-mixtral', size: '26.0GB', desc: 'Powerful' }
    ];

    function updateModelInstallUI() {
      const installOptions = document.getElementById('model-install-options');
      const cloudNotice = document.getElementById('cloud-notice');

      if (settings.provider === 'local') {
        installOptions.style.display = 'block';
        cloudNotice.style.display = 'none';
        renderAvailableModels();
      } else {
        installOptions.style.display = 'none';
        cloudNotice.style.display = 'block';
      }
    }

    function renderAvailableModels() {
      const container = document.getElementById('available-models');
      container.innerHTML = '';

      AVAILABLE_MODELS.forEach(model => {
        const div = document.createElement('div');
        div.style.cssText = 'padding: 8px; background: #f9f9f9; border-radius: 4px; margin-bottom: 6px; display: flex; justify-content: space-between; align-items: center; font-size: 12px;';

        const info = document.createElement('div');
        info.innerHTML = '<strong>' + model.name + '</strong> (' + model.size + ')<br><span style="color: #999; font-size: 11px;">' + model.desc + '</span>';

        const button = document.createElement('button');
        button.textContent = 'Install';
        button.style.cssText = 'padding: 4px 8px; background: #667eea; color: white; border: none; border-radius: 3px; cursor: pointer; font-size: 11px; white-space: nowrap;';
        button.onclick = () => installModel(model.name);

        div.appendChild(info);
        div.appendChild(button);
        container.appendChild(div);
      });
    }

    async function installModel(modelName) {
      const progressDiv = document.getElementById('install-progress');
      const progressText = document.getElementById('install-progress-text');
      const progressBar = document.getElementById('install-progress-bar');

      progressDiv.style.display = 'block';
      progressText.textContent = 'Installing ' + modelName + '...';
      progressBar.style.width = '10%';

      try {
        const response = await fetch('http://127.0.0.1:11436/api/models/pull', {
          method: 'POST',
          headers: {'Content-Type': 'application/json'},
          body: JSON.stringify({name: modelName})
        });

        const data = await response.json();

        if (data.success) {
          progressBar.style.width = '100%';
          progressText.textContent = 'Successfully installed ' + modelName + '!';
          setTimeout(() => {
            progressDiv.style.display = 'none';
            progressBar.style.width = '0%';
            loadInstalledModels();
          }, 1500);
        } else {
          progressText.textContent = 'Installation failed: ' + (data.error || 'Unknown error');
          progressBar.style.width = '0%';
          setTimeout(() => {
            progressDiv.style.display = 'none';
          }, 3000);
        }
      } catch (e) {
        progressText.textContent = 'Error: ' + e.message;
        progressBar.style.width = '0%';
        setTimeout(() => {
          progressDiv.style.display = 'none';
        }, 3000);
      }
    }

    async function saveSettings() {
      settings.model = document.getElementById('model').value;
      settings.tone = document.getElementById('tone').value;
      settings.mode = document.getElementById('mode').value;

      try {
        await fetch('http://127.0.0.1:11436/api/settings', {
          method: 'POST',
          headers: {'Content-Type': 'application/json'},
          body: JSON.stringify(settings)
        });
      } catch (e) {
        console.error('Error saving settings:', e);
      }

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

    (async () => {
      const isFirstRun = await checkFirstRun();
      if (!isFirstRun) {
        await loadSettings();
        await checkOllamaStatus();
      }
      renderMessages();

      document.getElementById('sendBtn').addEventListener('click', sendMessage);
      document.getElementById('clearBtn').addEventListener('click', clearChat);
      document.getElementById('message').addEventListener('keypress', (e) => {
        if (e.key === 'Enter' && !e.shiftKey) {
          e.preventDefault();
          sendMessage();
        }
      });
    })();
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

    res.writeHead(404, {'Content-Type': 'application/json'});
    res.end(JSON.stringify({error: 'Not found'}));
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
