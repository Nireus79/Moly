package main

const sidebarHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Moly</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: #fff;
            color: #333;
        }
        .container {
            display: flex;
            flex-direction: column;
            height: 100vh;
            padding: 12px;
        }
        .header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding-bottom: 8px;
            border-bottom: 1px solid #e0e0e0;
            margin-bottom: 12px;
        }
        .title {
            font-size: 18px;
            font-weight: 600;
        }
        .settings-btn {
            background: none;
            border: none;
            cursor: pointer;
            font-size: 20px;
            padding: 4px 8px;
        }
        .settings-btn:hover {
            background: #f0f0f0;
            border-radius: 4px;
        }

        .chat-view {
            display: flex;
            flex-direction: column;
            gap: 12px;
            flex: 1;
        }
        .chat-view.hidden {
            display: none;
        }

        .messages {
            flex: 1;
            overflow-y: auto;
            margin-bottom: 8px;
            padding: 8px;
            background: #fafafa;
            border-radius: 4px;
        }
        .message {
            margin-bottom: 8px;
            padding: 8px 12px;
            border-radius: 4px;
            font-size: 14px;
            line-height: 1.4;
        }
        .message.user {
            background: #e3f2fd;
            color: #1565c0;
            text-align: right;
        }
        .message.system {
            background: #f5f5f5;
            color: #666;
            text-align: center;
            font-style: italic;
        }
        .message.error {
            background: #ffebee;
            color: #c62828;
        }

        .input-area {
            display: flex;
            gap: 8px;
        }
        textarea {
            flex: 1;
            padding: 8px 12px;
            border: 1px solid #ddd;
            border-radius: 4px;
            font-family: inherit;
            font-size: 14px;
            resize: vertical;
            min-height: 60px;
        }
        .input-buttons {
            display: flex;
            flex-direction: column;
            gap: 8px;
        }
        button {
            padding: 8px 16px;
            background: #2196F3;
            color: white;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 14px;
            transition: background 0.2s;
        }
        button:hover {
            background: #1976D2;
        }
        button.secondary {
            background: #757575;
        }
        button.secondary:hover {
            background: #616161;
        }

        .settings-view {
            display: none;
            flex-direction: column;
            gap: 12px;
            overflow-y: auto;
        }
        .settings-view.active {
            display: flex;
        }

        .settings-section {
            background: #f5f5f5;
            padding: 12px;
            border-radius: 4px;
        }
        .settings-title {
            font-weight: 600;
            margin-bottom: 8px;
            font-size: 14px;
        }
        .setting {
            margin-bottom: 12px;
        }
        .setting:last-child {
            margin-bottom: 0;
        }
        .setting label {
            display: block;
            font-weight: 500;
            margin-bottom: 4px;
            color: #666;
            font-size: 13px;
        }
        select {
            width: 100%;
            padding: 6px;
            border: 1px solid #ddd;
            border-radius: 3px;
            font-size: 13px;
        }

        .status-indicator {
            display: inline-flex;
            align-items: center;
            gap: 4px;
            font-size: 12px;
            padding: 2px 6px;
            border-radius: 3px;
        }
        .status-indicator.available {
            background: #e8f5e9;
            color: #2e7d32;
        }
        .status-indicator.unavailable {
            background: #fff3e0;
            color: #f57c00;
        }

        .provider-option {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 8px;
            margin-bottom: 8px;
            border: 1px solid #ddd;
            border-radius: 4px;
            cursor: pointer;
            transition: all 0.2s;
        }
        .provider-option:hover {
            background: #f5f5f5;
        }
        .provider-option.selected {
            border-color: #2196F3;
            background: #e3f2fd;
        }
        .provider-info {
            display: flex;
            flex-direction: column;
            gap: 2px;
        }
        .provider-name {
            font-weight: 500;
            font-size: 13px;
        }
        .provider-desc {
            font-size: 11px;
            color: #999;
        }

        .model-list {
            max-height: 200px;
            overflow-y: auto;
        }
        .model-item {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 8px;
            margin-bottom: 6px;
            background: white;
            border-radius: 3px;
            border-left: 3px solid #2196F3;
            font-size: 13px;
        }
        .model-item button {
            padding: 4px 8px;
            font-size: 12px;
        }

        .button-group {
            display: flex;
            gap: 8px;
        }
        .button-group button {
            flex: 1;
        }

        .install-model-item {
            padding: 6px;
            margin-bottom: 4px;
            background: white;
            border-radius: 3px;
            border-left: 3px solid #4CAF50;
            display: flex;
            justify-content: space-between;
            align-items: center;
            font-size: 12px;
        }
        .install-model-item button {
            padding: 4px 8px;
            font-size: 11px;
            background: #4CAF50;
        }
        .install-model-item button:hover {
            background: #45a049;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="title">Moly</div>
            <button class="settings-btn" onclick="toggleSettings()">⚙️</button>
        </div>

        <div class="chat-view" id="chatView">
            <div class="messages" id="messages">
                <div class="message system">Start a conversation...</div>
            </div>
            <div class="input-area">
                <textarea id="messageInput" placeholder="Type your message..."></textarea>
                <div class="input-buttons">
                    <button onclick="sendMessage()">Send</button>
                    <button class="secondary" onclick="clearMessages()">Clear</button>
                </div>
            </div>
        </div>

        <div class="settings-view" id="settingsView">
            <div class="settings-section">
                <div class="settings-title">AI Provider</div>
                <div id="providerList"></div>
            </div>

            <div class="settings-section">
                <div class="settings-title">Preferences</div>
                <div class="setting">
                    <label>Model</label>
                    <select id="modelSelect" onchange="updateSettings()">
                        <option>Loading models...</option>
                    </select>
                </div>
                <div class="setting">
                    <label>Mode</label>
                    <select id="modeSelect" onchange="updateSettings()">
                        <option value="direct">Direct (Ready-to-use responses)</option>
                        <option value="socratic">Socratic (Guiding questions)</option>
                    </select>
                </div>
            </div>

            <div class="settings-section" id="modelManagementSection" style="display: none;">
                <div class="settings-title">Installed Models</div>
                <div class="model-list" id="modelsList">
                    <div style="color: #999; text-align: center; padding: 8px; font-size: 12px;">No models installed</div>
                </div>
                <div style="margin-top: 12px; border-top: 1px solid #ddd; padding-top: 12px;">
                    <div style="font-size: 13px; margin-bottom: 8px;">Install Popular Models</div>
                    <div id="installList"></div>
                </div>
            </div>

            <div class="settings-section" id="ollamaSection" style="display: none;">
                <div class="settings-title">Ollama Service</div>
                <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px;">
                    <span>Status</span>
                    <div class="status-indicator" id="ollamaStatus">Checking...</div>
                </div>
                <div class="button-group">
                    <button onclick="startOllama()">Start</button>
                    <button class="secondary" onclick="stopOllama()">Stop</button>
                </div>
            </div>
        </div>
    </div>

    <script>
        let currentProvider = null;
        const popularModels = ['mistral', 'llama2', 'neural-chat', 'orca-mini', 'dolphin-mixtral'];

        function toggleSettings() {
            document.getElementById('chatView').classList.toggle('hidden');
            document.getElementById('settingsView').classList.toggle('active');
        }

        function initializeApp() {
            loadAvailableProviders();
            loadSettings();
            updateOllamaStatus();
            setInterval(updateOllamaStatus, 3000);
            setInterval(loadModels, 5000);
            setupEventListeners();
        }

        function setupEventListeners() {
            document.getElementById('messageInput').addEventListener('keypress', function(e) {
                if (e.key === 'Enter' && !e.shiftKey) {
                    e.preventDefault();
                    sendMessage();
                }
            });
        }

        function loadAvailableProviders() {
            fetch('/api/providers')
                .then(function(r) { return r.json(); })
                .then(function(data) {
                    const list = document.getElementById('providerList');
                    list.innerHTML = data.providers.map(function(provider) {
                        return '<div class="provider-option' + (provider.active ? ' selected' : '') + '" onclick="switchProvider(\'' + provider.id + '\')">' +
                               '<div class="provider-info">' +
                               '<div class="provider-name">' + provider.name + '</div>' +
                               '<div class="provider-desc">' + provider.description + '</div>' +
                               '</div>' +
                               '<div class="status-indicator' + (provider.available ? ' available' : ' unavailable') + '">' +
                               (provider.available ? '✓ Available' : '✗ Unavailable') +
                               '</div></div>';
                    }).join('');

                    currentProvider = data.active;
                    updateProviderUI();
                })
                .catch(function(e) { console.error('Failed to load providers:', e); });
        }

        function switchProvider(providerId) {
            fetch('/api/settings', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ provider: providerId })
            })
            .then(function() {
                currentProvider = providerId;
                loadAvailableProviders();
                loadSettings();
            })
            .catch(function(e) { alert('Failed to switch provider: ' + e.message); });
        }

        function updateProviderUI() {
            const isLocal = currentProvider === 'local';
            document.getElementById('ollamaSection').style.display = isLocal ? 'block' : 'none';
            document.getElementById('modelManagementSection').style.display = isLocal ? 'block' : 'none';
        }

        function loadSettings() {
            fetch('/api/settings')
                .then(function(r) { return r.json(); })
                .then(function(config) {
                    document.getElementById('modelSelect').value = config.model || 'mistral';
                    document.getElementById('modeSelect').value = config.mode || 'direct';
                    loadModels();
                })
                .catch(function(e) { console.error('Failed to load settings:', e); });
        }

        function updateSettings() {
            const model = document.getElementById('modelSelect').value;
            const mode = document.getElementById('modeSelect').value;

            fetch('/api/settings', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ model: model, mode: mode })
            }).catch(function(e) { console.error('Settings update failed:', e); });
        }

        function loadModels() {
            if (currentProvider !== 'local') return;

            fetch('/api/models/list')
                .then(function(r) { return r.json(); })
                .then(function(data) {
                    const list = document.getElementById('modelsList');
                    const models = data.models || [];

                    if (models.length === 0) {
                        list.innerHTML = '<div style="color: #999; text-align: center; padding: 8px; font-size: 12px;">No models installed</div>';
                        const select = document.getElementById('modelSelect');
                        select.innerHTML = '<option>No models installed</option>';
                        return;
                    }

                    list.innerHTML = models.map(function(m) {
                        const name = m.name || m;
                        return '<div class="model-item"><span>' + name + '</span><button onclick="removeModel(\'' + name + '\')" class="secondary">Remove</button></div>';
                    }).join('');

                    const select = document.getElementById('modelSelect');
                    select.innerHTML = models.map(function(m) {
                        const name = m.name || m;
                        return '<option value="' + name + '">' + name + '</option>';
                    }).join('');

                    if (models.length > 0) {
                        select.value = models[0].name || models[0];
                    }

                    renderInstallModels();
                })
                .catch(function(e) {
                    document.getElementById('modelsList').innerHTML = '<div style="color: #999;">Error loading models</div>';
                });
        }

        function renderInstallModels() {
            fetch('/api/models/list')
                .then(function(r) { return r.json(); })
                .then(function(data) {
                    const installed = (data.models || []).map(function(m) { return m.name || m; });
                    const toInstall = popularModels.filter(function(m) { return installed.indexOf(m) === -1; });

                    const html = toInstall.map(function(model) {
                        return '<div class="install-model-item">' +
                               '<span>' + model + '</span>' +
                               '<button onclick="installModel(\'' + model + '\')">Install</button>' +
                               '</div>';
                    }).join('');

                    document.getElementById('installList').innerHTML = html || '<div style="color: #999; font-size: 12px;">All popular models installed</div>';
                })
                .catch(function(e) { console.error('Failed to load install list:', e); });
        }

        function removeModel(name) {
            if (confirm('Remove ' + name + '?')) {
                fetch('/api/models/remove?name=' + name, { method: 'POST' })
                    .then(function() { loadModels(); })
                    .catch(function(e) { alert('Error: ' + e.message); });
            }
        }

        function installModel(name) {
            const btn = event.target;
            btn.disabled = true;
            btn.textContent = 'Installing...';

            fetch('/api/models/pull?name=' + name, { method: 'POST' })
                .then(function(r) { return r.json(); })
                .then(function(data) {
                    if (data.success) {
                        loadModels();
                        btn.textContent = 'Install';
                        btn.disabled = false;
                    } else {
                        alert('Installation failed: ' + data.error);
                        btn.textContent = 'Install';
                        btn.disabled = false;
                    }
                })
                .catch(function(e) {
                    alert('Error: ' + e.message);
                    btn.textContent = 'Install';
                    btn.disabled = false;
                });
        }

        function updateOllamaStatus() {
            fetch('/api/first-run-check')
                .then(function(r) { return r.json(); })
                .then(function(data) {
                    const status = document.getElementById('ollamaStatus');
                    if (data.ollama_running) {
                        status.className = 'status-indicator available';
                        status.textContent = '✓ Running';
                    } else if (data.ollama_installed) {
                        status.className = 'status-indicator unavailable';
                        status.textContent = '✗ Stopped';
                    } else {
                        status.className = 'status-indicator unavailable';
                        status.textContent = '✗ Not Installed';
                    }
                });
        }

        function startOllama() {
            fetch('/api/ollama/start', { method: 'POST' })
                .then(function(r) { return r.json(); })
                .then(function(data) {
                    if (data.success) {
                        updateOllamaStatus();
                        loadModels();
                    } else {
                        alert('Failed to start Ollama: ' + data.error);
                    }
                })
                .catch(function(e) { alert('Error: ' + e.message); });
        }

        function stopOllama() {
            fetch('/api/ollama/stop', { method: 'POST' })
                .then(function(r) { return r.json(); })
                .then(function(data) {
                    if (data.success) {
                        updateOllamaStatus();
                    } else {
                        alert('Failed to stop Ollama: ' + data.error);
                    }
                })
                .catch(function(e) { alert('Error: ' + e.message); });
        }

        function sendMessage() {
            const input = document.getElementById('messageInput');
            const message = input.value.trim();
            if (!message) return;

            const messages = document.getElementById('messages');
            const div = document.createElement('div');
            div.className = 'message user';
            div.textContent = message;
            messages.appendChild(div);

            input.value = '';
            messages.scrollTop = messages.scrollHeight;

            const loading = document.createElement('div');
            loading.id = 'loading-indicator';
            loading.className = 'message';
            loading.textContent = 'Thinking...';
            messages.appendChild(loading);
            messages.scrollTop = messages.scrollHeight;

            const model = document.getElementById('modelSelect').value;
            const mode = document.getElementById('modeSelect').value;

            fetch('/api/chat', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ message: message, model: model, mode: mode })
            })
            .then(function(r) { return r.json(); })
            .then(function(data) {
                loading.remove();
                if (data.success) {
                    const response = document.createElement('div');
                    response.className = 'message';
                    response.textContent = data.response;
                    messages.appendChild(response);
                } else {
                    const error = document.createElement('div');
                    error.className = 'message error';
                    error.textContent = 'Error: ' + data.error;
                    messages.appendChild(error);
                }
                messages.scrollTop = messages.scrollHeight;
            })
            .catch(function(err) {
                loading.remove();
                const error = document.createElement('div');
                error.className = 'message error';
                error.textContent = 'Error: ' + err.message;
                messages.appendChild(error);
                messages.scrollTop = messages.scrollHeight;
            });
        }

        function clearMessages() {
            document.getElementById('messages').innerHTML = '<div class="message system">Start a conversation...</div>';
        }

        initializeApp();
    </script>
</body>
</html>`
