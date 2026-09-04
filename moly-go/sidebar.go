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
        .modal {
            display: none;
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: rgba(0,0,0,0.5);
            z-index: 10000;
            justify-content: center;
            align-items: center;
        }
        .modal.active {
            display: flex;
        }
        .modal-content {
            background: white;
            border-radius: 8px;
            padding: 32px;
            max-width: 400px;
            box-shadow: 0 4px 16px rgba(0,0,0,0.2);
        }
        .modal-title {
            font-size: 24px;
            font-weight: 600;
            margin-bottom: 8px;
        }
        .modal-subtitle {
            font-size: 14px;
            color: #666;
            margin-bottom: 24px;
        }
        .provider-option {
            padding: 16px;
            margin-bottom: 12px;
            border: 2px solid #e0e0e0;
            border-radius: 6px;
            cursor: pointer;
            transition: all 0.2s;
        }
        .provider-option:hover {
            border-color: #2196F3;
            background: #f5f9ff;
        }
        .provider-option.selected {
            border-color: #2196F3;
            background: #e3f2fd;
        }
        .provider-name {
            font-weight: 500;
            margin-bottom: 4px;
        }
        .provider-desc {
            font-size: 12px;
            color: #999;
        }
        .api-key-input {
            display: none;
            margin-top: 16px;
        }
        .api-key-input.active {
            display: block;
        }
        .api-key-input input {
            width: 100%;
            padding: 8px 12px;
            border: 1px solid #ddd;
            border-radius: 4px;
            font-size: 13px;
            margin-bottom: 8px;
        }
        .api-key-note {
            font-size: 11px;
            color: #999;
            margin-bottom: 12px;
        }
        .modal-buttons {
            display: flex;
            gap: 8px;
            margin-top: 24px;
        }
        .modal-buttons button {
            flex: 1;
            padding: 10px;
            border: none;
            border-radius: 4px;
            font-size: 14px;
            font-weight: 500;
            cursor: pointer;
            transition: background 0.2s;
        }
        .modal-buttons .continue-btn {
            background: #2196F3;
            color: white;
        }
        .modal-buttons .continue-btn:hover {
            background: #1976D2;
        }
        .modal-buttons .continue-btn:disabled {
            background: #ccc;
            cursor: not-allowed;
        }
        .container {
            display: flex;
            flex-direction: column;
            height: 100vh;
            padding: 16px;
        }
        .header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 12px;
            padding-bottom: 8px;
            border-bottom: 1px solid #e0e0e0;
        }
        .title {
            font-size: 18px;
            font-weight: 600;
        }
        .settings-panel {
            background: #f5f5f5;
            padding: 12px;
            border-radius: 4px;
            margin-bottom: 12px;
            font-size: 13px;
        }
        .setting {
            margin-bottom: 8px;
        }
        .setting label {
            display: block;
            font-weight: 500;
            margin-bottom: 4px;
            color: #666;
        }
        select {
            width: 100%;
            padding: 6px;
            border: 1px solid #ddd;
            border-radius: 3px;
            font-size: 13px;
        }
        .ollama-section {
            background: #f5f5f5;
            padding: 12px;
            border-radius: 4px;
            margin-bottom: 12px;
            font-size: 13px;
        }
        .ollama-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 8px;
        }
        .ollama-status {
            padding: 6px 10px;
            border-radius: 3px;
            font-size: 12px;
            font-weight: 500;
        }
        .ollama-status.running {
            background: #e8f5e9;
            color: #2e7d32;
        }
        .ollama-status.offline {
            background: #fff3e0;
            color: #f57c00;
        }
        .ollama-buttons {
            display: flex;
            gap: 6px;
        }
        .ollama-buttons button {
            flex: 1;
            padding: 6px 10px;
            font-size: 12px;
        }
        .messages {
            flex: 1;
            overflow-y: auto;
            margin-bottom: 12px;
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
        .input-area {
            display: flex;
            gap: 8px;
            margin-bottom: 12px;
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
        .button-group {
            display: flex;
            gap: 8px;
        }
        .button-group button {
            flex: 1;
        }
        .models-section {
            background: #f5f5f5;
            padding: 12px;
            border-radius: 4px;
            font-size: 13px;
            max-height: 200px;
            overflow-y: auto;
        }
        .models-section-title {
            font-weight: 500;
            margin-bottom: 8px;
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
        }
        .model-item button {
            padding: 4px 8px;
            font-size: 12px;
        }
        .install-models {
            background: #f5f5f5;
            padding: 12px;
            border-radius: 4px;
            font-size: 13px;
            max-height: 150px;
            overflow-y: auto;
        }
        .install-models-title {
            font-weight: 500;
            margin-bottom: 8px;
        }
        .install-model-item {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 8px;
            margin-bottom: 6px;
            background: white;
            border-radius: 3px;
            border-left: 3px solid #4CAF50;
        }
        .install-model-item span {
            font-size: 13px;
        }
        .install-model-item button {
            padding: 4px 8px;
            font-size: 12px;
            background: #4CAF50;
        }
        .install-model-item button:hover {
            background: #45a049;
        }
    </style>
</head>
<body>
    <!-- First-run setup modal -->
    <div class="modal" id="setupModal">
        <div class="modal-content">
            <div class="modal-title">Welcome to Moly</div>
            <div class="modal-subtitle">Choose your AI provider</div>

            <div class="provider-option" onclick="selectProvider('local')">
                <div class="provider-name">Local Models (Ollama)</div>
                <div class="provider-desc">Run models locally, no API key needed</div>
            </div>

            <div class="provider-option" onclick="selectProvider('claude')">
                <div class="provider-name">Claude (Anthropic)</div>
                <div class="provider-desc">Requires Claude API key</div>
            </div>

            <div class="provider-option" onclick="selectProvider('openai')">
                <div class="provider-name">ChatGPT (OpenAI)</div>
                <div class="provider-desc">Requires OpenAI API key</div>
            </div>

            <div class="api-key-input" id="apiKeyInput">
                <div class="api-key-note">Enter your API key (stored locally, never shared)</div>
                <input type="password" id="apiKeyField" placeholder="sk-...">
            </div>

            <div class="modal-buttons">
                <button class="continue-btn" id="continueBtn" onclick="completeSetup()" disabled>Continue</button>
            </div>
        </div>
    </div>

    <div class="container">
        <div class="header">
            <div class="title">Moly</div>
        </div>

        <!-- Settings Panel -->
        <div class="settings-panel">
            <div class="setting">
                <label>Model</label>
                <select id="modelSelect" onchange="updateSettings()">
                    <option value="mistral">Mistral</option>
                    <option value="llama2">Llama 2</option>
                    <option value="claude-3-sonnet">Claude 3 Sonnet</option>
                    <option value="gpt-3.5-turbo">GPT-3.5 Turbo</option>
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

        <!-- Ollama Section -->
        <div class="ollama-section">
            <div class="ollama-header">
                <span>Ollama</span>
                <div class="ollama-status" id="ollamaStatus">Checking...</div>
            </div>
            <div class="ollama-buttons">
                <button onclick="startOllama()">Start</button>
                <button class="secondary" onclick="stopOllama()">Stop</button>
            </div>
        </div>

        <!-- Chat Messages -->
        <div class="messages" id="messages">
            <div class="message system">Start a conversation...</div>
        </div>

        <!-- Chat Input -->
        <div class="input-area">
            <textarea id="messageInput" placeholder="Type your message..."></textarea>
            <div class="button-group">
                <button onclick="sendMessage()">Send</button>
                <button class="secondary" onclick="clearMessages()">Clear</button>
            </div>
        </div>

        <!-- Available Models -->
        <div class="models-section">
            <div class="models-section-title">Available Models</div>
            <div id="modelsList">
                <div style="color: #999; text-align: center; padding: 8px;">Loading...</div>
            </div>
        </div>

        <!-- Install Models -->
        <div class="install-models">
            <div class="install-models-title">Install Models</div>
            <div id="installList">
                <div class="install-model-item">
                    <span>mistral (4.1GB)</span>
                    <button onclick="installModel('mistral')">Install</button>
                </div>
                <div class="install-model-item">
                    <span>llama2 (3.8GB)</span>
                    <button onclick="installModel('llama2')">Install</button>
                </div>
                <div class="install-model-item">
                    <span>neural-chat (4.0GB)</span>
                    <button onclick="installModel('neural-chat')">Install</button>
                </div>
                <div class="install-model-item">
                    <span>orca-mini (1.3GB)</span>
                    <button onclick="installModel('orca-mini')">Install</button>
                </div>
                <div class="install-model-item">
                    <span>dolphin-mixtral (26GB)</span>
                    <button onclick="installModel('dolphin-mixtral')">Install</button>
                </div>
            </div>
        </div>
    </div>

    <script>
        let selectedProvider = null;

        function selectProvider(provider) {
            selectedProvider = provider;
            document.querySelectorAll('.provider-option').forEach(el => el.classList.remove('selected'));
            event.target.closest('.provider-option').classList.add('selected');

            const apiKeyInput = document.getElementById('apiKeyInput');
            const continueBtn = document.getElementById('continueBtn');

            if (provider === 'local') {
                apiKeyInput.classList.remove('active');
                continueBtn.disabled = false;
            } else {
                apiKeyInput.classList.add('active');
                continueBtn.disabled = true;
                document.getElementById('apiKeyField').addEventListener('input', () => {
                    continueBtn.disabled = !document.getElementById('apiKeyField').value.trim();
                });
            }
        }

        function completeSetup() {
            const apiKey = document.getElementById('apiKeyField').value.trim();
            const data = {
                provider: selectedProvider,
                first_run_complete: true
            };

            if (selectedProvider !== 'local' && apiKey) {
                data.api_keys = {};
                data.api_keys[selectedProvider] = apiKey;
            }

            fetch('/api/settings', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(data)
            })
            .then(() => {
                document.getElementById('setupModal').classList.remove('active');
                loadSettings();
            })
            .catch(err => alert('Setup failed: ' + err.message));
        }

        function checkFirstRun() {
            fetch('/api/first-run-check')
                .then(r => r.json())
                .then(data => {
                    if (!data.first_run_complete) {
                        document.getElementById('setupModal').classList.add('active');
                    } else {
                        loadSettings();
                    }
                    updateOllamaStatus();
                });
        }

        function loadSettings() {
            fetch('/api/settings')
                .then(r => r.json())
                .then(config => {
                    document.getElementById('modelSelect').value = config.model || 'mistral';
                    document.getElementById('modeSelect').value = config.mode || 'direct';
                });
        }

        function updateSettings() {
            const model = document.getElementById('modelSelect').value;
            const mode = document.getElementById('modeSelect').value;

            fetch('/api/settings', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ model, mode })
            }).catch(e => console.error('Settings update failed:', e));
        }

        function updateOllamaStatus() {
            fetch('/api/first-run-check')
                .then(r => r.json())
                .then(data => {
                    const status = document.getElementById('ollamaStatus');
                    if (data.ollama_running) {
                        status.className = 'ollama-status running';
                        status.textContent = 'Running';
                    } else if (data.ollama_installed) {
                        status.className = 'ollama-status offline';
                        status.textContent = 'Installed (Stopped)';
                    } else {
                        status.className = 'ollama-status offline';
                        status.textContent = 'Not Installed';
                    }
                });
        }

        function startOllama() {
            fetch('/api/ollama/start', { method: 'POST' })
                .then(r => r.json())
                .then(data => {
                    if (data.success) {
                        updateOllamaStatus();
                        loadModels();
                    } else {
                        alert('Failed to start Ollama: ' + data.error);
                    }
                })
                .catch(e => alert('Error: ' + e.message));
        }

        function stopOllama() {
            fetch('/api/ollama/stop', { method: 'POST' })
                .then(r => r.json())
                .then(data => {
                    if (data.success) {
                        updateOllamaStatus();
                    } else {
                        alert('Failed to stop Ollama: ' + data.error);
                    }
                })
                .catch(e => alert('Error: ' + e.message));
        }

        function loadModels() {
            fetch('/api/models/list')
                .then(r => r.json())
                .then(data => {
                    const list = document.getElementById('modelsList');
                    const models = data.models || [];

                    if (models.length === 0) {
                        list.innerHTML = '<div style="color: #999; text-align: center; padding: 8px;">No models found</div>';
                        return;
                    }

                    list.innerHTML = models.map(m => {
                        const name = m.name || m;
                        return '<div class="model-item"><span>' + name + '</span><button onclick="removeModel(\'' + name + '\')" class="secondary">Remove</button></div>';
                    }).join('');
                })
                .catch(e => {
                    document.getElementById('modelsList').innerHTML = '<div style="color: #999;">Error loading models</div>';
                });
        }

        function removeModel(name) {
            if (confirm('Remove ' + name + '?')) {
                fetch('/api/models/remove?name=' + name, { method: 'POST' })
                    .then(() => loadModels())
                    .catch(e => alert('Error: ' + e.message));
            }
        }

        function installModel(name) {
            const btn = event.target;
            btn.disabled = true;
            btn.textContent = 'Installing...';

            fetch('/api/models/pull?name=' + name, { method: 'POST' })
                .then(r => r.json())
                .then(data => {
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
                .catch(e => {
                    alert('Error: ' + e.message);
                    btn.textContent = 'Install';
                    btn.disabled = false;
                });
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
                body: JSON.stringify({ message, model, mode })
            })
            .then(r => r.json())
            .then(data => {
                loading.remove();
                if (data.success) {
                    const response = document.createElement('div');
                    response.className = 'message';
                    response.textContent = data.response;
                    messages.appendChild(response);
                } else {
                    const error = document.createElement('div');
                    error.className = 'message';
                    error.style.color = '#d32f2f';
                    error.textContent = 'Error: ' + data.error;
                    messages.appendChild(error);
                }
                messages.scrollTop = messages.scrollHeight;
            })
            .catch(err => {
                loading.remove();
                const error = document.createElement('div');
                error.className = 'message';
                error.style.color = '#d32f2f';
                error.textContent = 'Error: ' + err.message;
                messages.appendChild(error);
                messages.scrollTop = messages.scrollHeight;
            });
        }

        function clearMessages() {
            document.getElementById('messages').innerHTML = '<div class="message system">Start a conversation...</div>';
        }

        // Initialize on load
        checkFirstRun();
        loadModels();
        setInterval(updateOllamaStatus, 3000);
        setInterval(loadModels, 5000);

        // Allow Enter to send
        document.getElementById('messageInput').addEventListener('keypress', (e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                sendMessage();
            }
        });
    </script>
</body>
</html>`
