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
            padding: 16px;
        }
        .header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 16px;
            padding-bottom: 12px;
            border-bottom: 1px solid #e0e0e0;
        }
        .title {
            font-size: 18px;
            font-weight: 600;
        }
        .status {
            padding: 8px 12px;
            border-radius: 4px;
            font-size: 12px;
            background: #f5f5f5;
        }
        .status.running {
            background: #e8f5e9;
            color: #2e7d32;
        }
        .status.offline {
            background: #fff3e0;
            color: #f57c00;
        }
        .messages {
            flex: 1;
            overflow-y: auto;
            margin-bottom: 16px;
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
        .models-section {
            background: #f5f5f5;
            padding: 12px;
            border-radius: 4px;
            margin-top: 12px;
            font-size: 13px;
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
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="title">Moly</div>
            <div class="status running" id="status">Running</div>
        </div>

        <div class="settings-panel">
            <div class="setting">
                <label>Model</label>
                <select id="modelSelect" onchange="updateSettings()">
                    <option value="claude-3-sonnet">Claude 3 Sonnet</option>
                    <option value="gpt-4">GPT-4</option>
                    <option value="mistral">Mistral</option>
                </select>
            </div>
            <div class="setting">
                <label>Tone</label>
                <select id="toneSelect" onchange="updateSettings()">
                    <option value="friendly">Friendly</option>
                    <option value="formal">Formal</option>
                    <option value="playful">Playful</option>
                </select>
            </div>
        </div>

        <div class="messages" id="messages">
            <div class="message system">Start a conversation...</div>
        </div>

        <div class="input-area">
            <textarea id="messageInput" placeholder="Type a message..."></textarea>
            <div class="button-group">
                <button onclick="sendMessage()">Send</button>
                <button class="secondary" onclick="clearMessages()">Clear</button>
            </div>
        </div>

        <div class="models-section">
            <div style="font-weight: 500; margin-bottom: 8px;">Available Models</div>
            <div id="modelsList">
                <div style="color: #999; text-align: center; padding: 8px;">Loading models...</div>
            </div>
        </div>
    </div>

    <script>
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

            // Show loading indicator
            const loading = document.createElement('div');
            loading.id = 'loading-indicator';
            loading.className = 'message';
            loading.textContent = 'Thinking...';
            messages.appendChild(loading);
            messages.scrollTop = messages.scrollHeight;

            // Send to API
            const model = document.getElementById('modelSelect').value;
            const tone = document.getElementById('toneSelect').value;

            fetch('/api/chat', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ message, model, tone, mode: 'direct' })
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

        function updateSettings() {
            const model = document.getElementById('modelSelect').value;
            const tone = document.getElementById('toneSelect').value;

            fetch('/api/settings', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ model, tone })
            }).catch(e => console.error('Settings update failed:', e));
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
                        return '<div class="model-item"><span>' + name + '</span><button onclick="removeModel(\'' + name + '\')" style="padding: 2px 6px; font-size: 11px;">Remove</button></div>';
                    }).join('');
                })
                .catch(e => {
                    document.getElementById('modelsList').innerHTML = '<div style="color: #999;">Failed to load models</div>';
                });
        }

        function removeModel(name) {
            if (confirm('Remove ' + name + '?')) {
                fetch('/api/models/remove?name=' + name, { method: 'POST' })
                    .then(() => loadModels())
                    .catch(e => alert('Failed to remove model'));
            }
        }

        // Load models on startup
        loadModels();
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
