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

        .safety-alert {
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background: rgba(0,0,0,0.7);
            z-index: 2000;
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
        }
        .safety-alert.hidden {
            display: none;
        }
        .safety-alert-content {
            background: white;
            border-radius: 8px;
            padding: 24px;
            max-width: 600px;
            width: 90%;
            max-height: 85vh;
            overflow-y: auto;
            box-shadow: 0 8px 16px rgba(0,0,0,0.3);
        }
        .safety-alert-header {
            font-size: 18px;
            font-weight: 700;
            margin-bottom: 12px;
            color: #c62828;
        }
        .safety-alert-message {
            font-size: 14px;
            line-height: 1.6;
            margin-bottom: 16px;
            color: #333;
        }
        .safety-alert-section {
            margin-bottom: 16px;
        }
        .safety-alert-section-title {
            font-weight: 600;
            font-size: 13px;
            color: #333;
            margin-bottom: 8px;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        .safety-alert-resource {
            background: #f5f5f5;
            border-left: 4px solid #2196F3;
            padding: 12px;
            margin-bottom: 8px;
            border-radius: 3px;
        }
        .safety-alert-resource-name {
            font-weight: 600;
            font-size: 13px;
            color: #333;
        }
        .safety-alert-resource-desc {
            font-size: 12px;
            color: #666;
            margin-top: 2px;
        }
        .safety-alert-resource-contact {
            font-size: 12px;
            font-weight: 600;
            color: #1976D2;
            margin-top: 4px;
        }
        .safety-alert-button {
            background: #2196F3;
            color: white;
            border: none;
            padding: 12px 24px;
            border-radius: 4px;
            font-size: 14px;
            font-weight: 600;
            cursor: pointer;
            width: 100%;
            margin-top: 16px;
        }
        .safety-alert-button:hover {
            background: #1976D2;
        }
        .safety-alert-list {
            margin: 0;
            padding-left: 20px;
        }
        .safety-alert-list li {
            margin-bottom: 8px;
            font-size: 13px;
            line-height: 1.5;
            color: #333;
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

        .contact-selector {
            display: flex;
            gap: 8px;
            margin-bottom: 12px;
            flex-wrap: wrap;
            align-items: center;
        }
        .contact-selector label {
            font-size: 12px;
            font-weight: 500;
            color: #666;
            min-width: 60px;
        }
        .contact-selector select {
            flex: 1;
            min-width: 150px;
            padding: 6px 8px;
            border: 1px solid #ddd;
            border-radius: 4px;
            font-size: 13px;
        }
        .new-contact-btn, .contact-menu-btn, .draft-msg-btn {
            padding: 6px 12px;
            background: #2196F3;
            color: white;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 12px;
        }
        .new-contact-btn:hover, .contact-menu-btn:hover, .draft-msg-btn:hover {
            background: #1976D2;
        }
        .contact-menu-btn {
            background: #FF9800;
        }
        .contact-menu-btn:hover {
            background: #F57C00;
        }
        .draft-msg-btn {
            background: #4CAF50;
        }
        .draft-msg-btn:hover {
            background: #45a049;
        }
        .mode-analysis-btn {
            background: #9C27B0;
        }
        .mode-analysis-btn:hover {
            background: #7B1FA2;
        }

        .draft-modal, .mode-modal {
            display: none;
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background: rgba(0,0,0,0.5);
            z-index: 1000;
            flex-direction: column;
            align-items: center;
            justify-content: center;
        }
        .draft-modal.active, .mode-modal.active {
            display: flex;
        }
        .draft-modal-content, .mode-modal-content {
            background: white;
            border-radius: 8px;
            padding: 20px;
            max-width: 700px;
            width: 90%;
            max-height: 80vh;
            overflow-y: auto;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
        }
        .draft-modal-header, .mode-modal-header {
            font-size: 16px;
            font-weight: 600;
            margin-bottom: 12px;
            color: #333;
        }
        .draft-form-group {
            margin-bottom: 12px;
        }
        .draft-form-group label {
            display: block;
            font-size: 12px;
            font-weight: 600;
            margin-bottom: 4px;
            color: #333;
        }
        .draft-form-group input, .draft-form-group textarea {
            width: 100%;
            padding: 8px;
            border: 1px solid #ccc;
            border-radius: 4px;
            font-family: inherit;
            font-size: 13px;
            box-sizing: border-box;
        }
        .draft-form-group textarea {
            resize: vertical;
            min-height: 80px;
        }
        .draft-modal-buttons {
            display: flex;
            gap: 8px;
            margin-top: 16px;
        }
        .draft-modal-buttons button {
            flex: 1;
            padding: 8px;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 13px;
        }
        .draft-modal-buttons button.submit {
            background: #4CAF50;
            color: white;
        }
        .draft-modal-buttons button.cancel {
            background: #999;
            color: white;
        }

        .draft-suggestion {
            background: #f0f7ff;
            border: 1px solid #b3d9ff;
            border-radius: 4px;
            padding: 12px;
            margin-top: 12px;
            font-size: 13px;
            line-height: 1.5;
            color: #333;
        }

        .new-contact-form {
            display: none;
            background: #f5f5f5;
            border: 1px solid #ddd;
            border-radius: 4px;
            padding: 12px;
            margin-bottom: 12px;
            gap: 8px;
            flex-direction: column;
        }
        .new-contact-form.active {
            display: flex;
        }
        .form-group {
            display: flex;
            flex-direction: column;
            gap: 4px;
        }
        .form-group label {
            font-size: 12px;
            font-weight: 600;
            color: #333;
        }
        .form-group input {
            padding: 6px 8px;
            border: 1px solid #ccc;
            border-radius: 3px;
            font-size: 12px;
            font-family: inherit;
        }
        .form-buttons {
            display: flex;
            gap: 8px;
        }
        .form-buttons button {
            flex: 1;
            padding: 6px;
            font-size: 12px;
            border: none;
            border-radius: 3px;
            cursor: pointer;
        }
        .form-buttons button.create {
            background: #4CAF50;
            color: white;
        }
        .form-buttons button.cancel {
            background: #999;
            color: white;
        }

        .message.questions {
            background: #fff3e0;
            color: #333;
            border-left: 3px solid #ffb74d;
            white-space: normal;
        }
        .message.questions strong {
            color: #e65100;
            display: block;
            margin-bottom: 6px;
        }
        .questions-list {
            margin: 4px 0;
            padding-left: 12px;
        }
        .questions-list li {
            margin-bottom: 4px;
            font-size: 13px;
        }
        .understanding-level {
            font-size: 11px;
            color: #999;
            margin-top: 6px;
            padding-top: 6px;
            border-top: 1px solid #ffe0b2;
        }

        .contact-info-section {
            background: #f5f5f5;
            border-radius: 4px;
            padding: 8px 12px;
            margin-bottom: 12px;
            font-size: 12px;
            display: none;
        }
        .contact-info-section.active {
            display: block;
        }
        .info-label {
            font-weight: 600;
            color: #333;
            margin-bottom: 2px;
        }
        .info-value {
            color: #666;
            font-size: 12px;
            margin-bottom: 6px;
            padding-left: 4px;
        }
        .info-divider {
            height: 1px;
            background: #ddd;
            margin: 8px 0;
        }
        .recent-topics {
            margin-top: 6px;
        }
        .topic-tag {
            display: inline-block;
            background: #e0e0e0;
            color: #333;
            padding: 2px 6px;
            border-radius: 3px;
            font-size: 11px;
            margin-right: 4px;
            margin-bottom: 4px;
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
            <div class="contact-selector">
                <label>Contact:</label>
                <select id="contactSelect" onchange="switchContact()">
                    <option value="">Select a contact...</option>
                </select>
                <button class="new-contact-btn" onclick="toggleNewContactForm()">+ New</button>
                <button class="draft-msg-btn" id="draftMsgBtn" onclick="openDraftModal()" style="display:none;">✎ Draft</button>
                <button class="mode-analysis-btn" id="modeAnalysisBtn" onclick="openModeModal()" style="display:none;">Analyze Mode</button>
                <button class="contact-menu-btn" id="contactMenuBtn" onclick="toggleContactMenu()" style="display:none;">⋮</button>
            </div>

            <div id="contactMenu" style="display:none; background:#fff; border:1px solid #ddd; border-radius:4px; margin-bottom:12px; overflow:hidden;">
                <div style="padding:8px; cursor:pointer; hover:background:#f5f5f5; font-size:12px;" onclick="deleteContact()">Delete Contact</div>
                <div style="padding:8px; cursor:pointer; font-size:12px;" onclick="toggleContactMenu()">Close</div>
            </div>

            <div class="new-contact-form" id="newContactForm">
                <div class="form-group">
                    <label>Name</label>
                    <input type="text" id="newContactName" placeholder="Contact name">
                </div>
                <div class="form-group">
                    <label>Platform</label>
                    <input type="text" id="newContactPlatform" placeholder="e.g., whatsapp, email, slack">
                </div>
                <div class="form-group">
                    <label>Relationship</label>
                    <input type="text" id="newContactRelationship" placeholder="e.g., friend, colleague, family">
                </div>
                <div class="form-buttons">
                    <button class="create" onclick="submitNewContact()">Create</button>
                    <button class="cancel" onclick="toggleNewContactForm()">Cancel</button>
                </div>
            </div>

            <div class="contact-info-section" id="contactInfo">
                <div class="info-label">About</div>
                <div class="info-value" id="contactNotes"></div>
                <div class="info-divider"></div>
                <div class="info-label">Last Interaction</div>
                <div class="info-value" id="lastInteraction"></div>
                <div class="info-label">Recent Topics</div>
                <div id="recentTopics" class="recent-topics"></div>
            </div>

            <div class="messages" id="messages">
                <div class="message system">Select a contact to start...</div>
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
                <div id="apiKeySection" style="display: none; margin-top: 12px; padding-top: 12px; border-top: 1px solid #ddd;">
                    <div style="font-size: 13px; margin-bottom: 8px;">API Key</div>
                    <input type="password" id="apiKeyInput" placeholder="Enter your API key..." style="width: 100%; padding: 8px; border: 1px solid #ddd; border-radius: 4px; margin-bottom: 8px; font-size: 13px;">
                    <button onclick="saveAPIKey()" style="width: 100%; padding: 8px; background: #4CAF50; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 13px;">Save Key</button>
                    <div style="font-size: 11px; color: #999; margin-top: 4px;">Keys are encrypted and stored locally</div>
                </div>
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

        <div class="draft-modal" id="draftModal">
            <div class="draft-modal-content">
                <div class="draft-modal-header">Draft Message to <span id="draftContactName"></span></div>

                <div class="draft-form-group">
                    <label>What do you want to say?</label>
                    <input type="text" id="draftIntention" placeholder="e.g., Invite to hiking trip, Apologize about missed call, etc.">
                </div>

                <div class="draft-form-group">
                    <label>Your draft message</label>
                    <textarea id="draftText" placeholder="Type your draft message here..."></textarea>
                </div>

                <div class="draft-modal-buttons">
                    <button class="submit" onclick="submitDraftMessage()">Get Suggestions</button>
                    <button class="cancel" onclick="closeDraftModal()">Cancel</button>
                </div>

                <div id="draftSuggestion"></div>
            </div>
        </div>

        <div class="mode-modal" id="modeModal">
            <div class="mode-modal-content">
                <div class="mode-modal-header">Analyze Relationship Mode Transition</div>

                <div class="draft-form-group">
                    <label>Current Mode</label>
                    <select id="currentModeSelect">
                        <option value="">Select current mode...</option>
                        <option value="professional">Professional</option>
                        <option value="friendly">Friendly</option>
                        <option value="casual_flirting">Casual Flirting</option>
                        <option value="romantic">Romantic</option>
                        <option value="intimate_sexual">Intimate/Sexual</option>
                        <option value="power_exchange">Power Exchange (D/s)</option>
                    </select>
                </div>

                <div class="draft-form-group">
                    <label>Desired Mode</label>
                    <select id="desiredModeSelect">
                        <option value="">Select desired mode...</option>
                        <option value="professional">Professional</option>
                        <option value="friendly">Friendly</option>
                        <option value="casual_flirting">Casual Flirting</option>
                        <option value="romantic">Romantic</option>
                        <option value="intimate_sexual">Intimate/Sexual</option>
                        <option value="power_exchange">Power Exchange (D/s)</option>
                    </select>
                </div>

                <div class="draft-form-group">
                    <label>Context (Optional)</label>
                    <textarea id="modeContext" placeholder="Any additional context about the relationship or transition..."></textarea>
                </div>

                <div class="draft-modal-buttons">
                    <button class="submit" onclick="submitModeAnalysis()">Analyze Transition</button>
                    <button class="cancel" onclick="closeModeModal()">Cancel</button>
                </div>

                <div id="modeAnalysisResult"></div>
            </div>
        </div>

        <div class="safety-alert hidden" id="safetyAlert">
            <div class="safety-alert-content">
                <div class="safety-alert-header" id="safetyAlertTitle">Crisis Support</div>
                <div class="safety-alert-message" id="safetyAlertMessage"></div>

                <div class="safety-alert-section" id="indicatorsSection" style="display:none;">
                    <div class="safety-alert-section-title">What We Detected</div>
                    <ul class="safety-alert-list" id="safetyIndicators"></ul>
                </div>

                <div class="safety-alert-section" id="resourcesSection" style="display:none;">
                    <div class="safety-alert-section-title">Get Help Now</div>
                    <div id="safetyResources"></div>
                </div>

                <div class="safety-alert-section" id="recommendationsSection" style="display:none;">
                    <div class="safety-alert-section-title">What You Can Do</div>
                    <ul class="safety-alert-list" id="safetyRecommendations"></ul>
                </div>

                <button class="safety-alert-button" onclick="closeSafetyAlert()">I Understand - Continue</button>
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
            const isCloud = currentProvider === 'claude' || currentProvider === 'openai';

            document.getElementById('ollamaSection').style.display = isLocal ? 'block' : 'none';
            document.getElementById('modelManagementSection').style.display = isLocal ? 'block' : 'none';
            document.getElementById('apiKeySection').style.display = isCloud ? 'block' : 'none';

            if (isCloud) {
                document.getElementById('apiKeyInput').placeholder = 'Enter your ' + (currentProvider === 'claude' ? 'Anthropic' : 'OpenAI') + ' API key...';
                document.getElementById('apiKeyInput').value = '';
            }
        }

        function saveAPIKey() {
            const apiKey = document.getElementById('apiKeyInput').value.trim();
            if (!apiKey) {
                alert('Please enter an API key');
                return;
            }

            fetch('/api/api-key', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    provider: currentProvider,
                    api_key: apiKey
                })
            })
            .then(function(r) { return r.json(); })
            .then(function(data) {
                if (data.success) {
                    alert('API key saved successfully');
                    document.getElementById('apiKeyInput').value = '';
                    loadAvailableProviders();
                } else {
                    alert('Failed to save API key: ' + (data.error || 'Unknown error'));
                }
            })
            .catch(function(e) { alert('Error saving API key: ' + e.message); });
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
            const btn = event.target;
            btn.disabled = true;
            btn.textContent = 'Starting...';

            fetch('/api/ollama/start', { method: 'POST' })
                .then(function(r) { return r.json(); })
                .then(function(data) {
                    if (data.success) {
                        btn.textContent = 'Start';
                        btn.disabled = false;
                        updateOllamaStatus();
                        setTimeout(loadModels, 2000);
                    } else {
                        btn.textContent = 'Start';
                        btn.disabled = false;
                        alert('Failed to start Ollama: ' + (data.error || 'Unknown error'));
                    }
                })
                .catch(function(e) {
                    btn.textContent = 'Start';
                    btn.disabled = false;
                    alert('Error starting Ollama: ' + e.message);
                });
        }

        function stopOllama() {
            const btn = event.target;
            btn.disabled = true;
            btn.textContent = 'Stopping...';

            fetch('/api/ollama/stop', { method: 'POST' })
                .then(function(r) { return r.json(); })
                .then(function(data) {
                    if (data.success) {
                        btn.textContent = 'Stop';
                        btn.disabled = false;
                        updateOllamaStatus();
                    } else {
                        btn.textContent = 'Stop';
                        btn.disabled = false;
                        alert('Failed to stop Ollama: ' + (data.error || 'Unknown error'));
                    }
                })
                .catch(function(e) {
                    btn.textContent = 'Stop';
                    btn.disabled = false;
                    alert('Error stopping Ollama: ' + e.message);
                });
        }

        function clearMessages() {
            document.getElementById('messages').innerHTML = '<div class="message system">Start a conversation...</div>';
        }

        let selectedContactId = null;
        let selectedContactPlatform = null;

        function loadContacts() {
            fetch('/api/contacts')
                .then(function(r) { return r.json(); })
                .then(function(data) {
                    const select = document.getElementById('contactSelect');
                    select.innerHTML = '<option value="">Select a contact...</option>';
                    data.contacts.forEach(function(contact) {
                        const option = document.createElement('option');
                        option.value = contact.id;
                        option.textContent = contact.name + ' (' + contact.platform + ')';
                        select.appendChild(option);
                    });
                })
                .catch(function(e) { console.error('Failed to load contacts:', e); });
        }

        function switchContact() {
            const select = document.getElementById('contactSelect');
            selectedContactId = parseInt(select.value);

            if (!selectedContactId) {
                document.getElementById('contactInfo').classList.remove('active');
                document.getElementById('draftMsgBtn').style.display = 'none';
                document.getElementById('modeAnalysisBtn').style.display = 'none';
                document.getElementById('contactMenuBtn').style.display = 'none';
                document.getElementById('messages').innerHTML = '<div class="message system">Select a contact to start...</div>';
                return;
            }

            document.getElementById('draftMsgBtn').style.display = 'inline-block';
            document.getElementById('modeAnalysisBtn').style.display = 'inline-block';
            document.getElementById('contactMenuBtn').style.display = 'inline-block';
            loadContactInfo(selectedContactId);
            analyzeContextForContact();
        }

        function toggleContactMenu() {
            const menu = document.getElementById('contactMenu');
            menu.style.display = menu.style.display === 'none' ? 'block' : 'none';
        }

        function deleteContact() {
            const select = document.getElementById('contactSelect');
            const contactName = select.options[select.selectedIndex].text;

            if (!confirm('Delete "' + contactName + '" and all their interaction history?')) {
                return;
            }

            fetch('/api/contacts/delete', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ contact_id: selectedContactId })
            })
            .then(function(r) { return r.json(); })
            .then(function(data) {
                if (data.success) {
                    toggleContactMenu();
                    loadContacts();
                    selectedContactId = null;
                    document.getElementById('contactSelect').value = '';
                    document.getElementById('contactInfo').classList.remove('active');
                    document.getElementById('messages').innerHTML = '<div class="message system">Contact deleted. Select another contact to start...</div>';
                    document.getElementById('draftMsgBtn').style.display = 'none';
                    document.getElementById('modeAnalysisBtn').style.display = 'none';
                    document.getElementById('contactMenuBtn').style.display = 'none';
                } else {
                    alert('Error deleting contact: ' + data.error);
                }
            })
            .catch(function(e) { alert('Error deleting contact: ' + e.message); });
        }

        function openDraftModal() {
            if (!selectedContactId) return;

            const select = document.getElementById('contactSelect');
            const contactName = select.options[select.selectedIndex].text.split('(')[0].trim();
            document.getElementById('draftContactName').textContent = contactName;
            document.getElementById('draftIntention').value = '';
            document.getElementById('draftText').value = '';
            document.getElementById('draftSuggestion').innerHTML = '';
            document.getElementById('draftModal').classList.add('active');
        }

        function closeDraftModal() {
            document.getElementById('draftModal').classList.remove('active');
        }

        function openModeModal() {
            if (!selectedContactId) return;

            const select = document.getElementById('contactSelect');
            const contactName = select.options[select.selectedIndex].text.split('(')[0].trim();
            document.getElementById('currentModeSelect').value = '';
            document.getElementById('desiredModeSelect').value = '';
            document.getElementById('modeContext').value = '';
            document.getElementById('modeAnalysisResult').innerHTML = '';
            document.getElementById('modeModal').classList.add('active');
        }

        function closeModeModal() {
            document.getElementById('modeModal').classList.remove('active');
        }

        function submitModeAnalysis() {
            const currentMode = document.getElementById('currentModeSelect').value;
            const desiredMode = document.getElementById('desiredModeSelect').value;
            const context = document.getElementById('modeContext').value.trim();

            if (!currentMode) {
                alert('Please select the current mode');
                return;
            }
            if (!desiredMode) {
                alert('Please select the desired mode');
                return;
            }

            const resultDiv = document.getElementById('modeAnalysisResult');
            resultDiv.innerHTML = '<div style="text-align:center; color:#999; padding:12px;">Analyzing transition...</div>';

            fetch('/api/analyze-mode-shift', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    contact_id: selectedContactId,
                    current_mode: currentMode,
                    desired_mode: desiredMode,
                    context: context
                })
            })
            .then(function(r) { return r.json(); })
            .then(function(data) {
                if (data.success && data.analysis) {
                    displayModeAnalysis(data.analysis);
                } else {
                    resultDiv.innerHTML = '<div style="color:red;">Error: ' + (data.error || 'Unknown error') + '</div>';
                }
            })
            .catch(function(e) {
                resultDiv.innerHTML = '<div style="color:red;">Error: ' + e.message + '</div>';
            });
        }

        function displayModeAnalysis(analysis) {
            const resultDiv = document.getElementById('modeAnalysisResult');

            if (!analysis.mode_shift_detected) {
                resultDiv.innerHTML = '<div style="padding:12px; background:#e3f2fd; border-radius:4px; margin-top:12px;">No mode shift detected. Current and desired modes are the same.</div>';
                return;
            }

            let html = '<div style="margin-top:12px; padding:12px; background:#f5f5f5; border-radius:4px;">';

            html += '<div style="margin-bottom:12px;"><strong>Mode Transition: ' + analysis.current_mode + ' → ' + analysis.desired_mode + '</strong></div>';
            html += '<div style="margin-bottom:12px;"><span style="background:' + (analysis.risk_level === 'extreme' ? '#ffcccc' : analysis.risk_level === 'high' ? '#ffe6cc' : analysis.risk_level === 'medium' ? '#fffacc' : '#e6f5ff') + '; padding:4px 8px; border-radius:3px; font-weight:bold;">Risk Level: ' + analysis.risk_level.toUpperCase() + ' (' + analysis.overall_risk_score + '/100)</span></div>';

            if (analysis.implications && analysis.implications.length > 0) {
                html += '<div style="margin-bottom:12px;"><strong>Key Implications:</strong><ul>';
                analysis.implications.forEach(function(impl) {
                    html += '<li>' + impl + '</li>';
                });
                html += '</ul></div>';
            }

            if (analysis.phases && analysis.phases.length > 0) {
                html += '<div style="margin-bottom:12px;"><strong>Recommended Phases:</strong><div style="margin-top:8px;">';
                analysis.phases.forEach(function(phase) {
                    html += '<div style="margin-bottom:12px; padding:8px; background:white; border-radius:3px; border-left:3px solid #2196F3;">';
                    html += '<strong>Phase ' + phase.phase + ': ' + phase.name + '</strong><br>';
                    html += '<em style="font-size:12px; color:#666;">' + phase.description + ' (' + phase.duration + ')</em><br>';
                    if (phase.tactics && phase.tactics.length > 0) {
                        html += '<div style="margin-top:6px; font-size:12px;"><strong>Tactics:</strong><ul style="margin-top:4px; margin-bottom:0;">';
                        phase.tactics.forEach(function(tactic) {
                            html += '<li>' + tactic + '</li>';
                        });
                        html += '</ul></div>';
                    }
                    if (phase.red_flags && phase.red_flags.length > 0) {
                        html += '<div style="margin-top:6px; font-size:12px; color:#c62828;"><strong>Red Flags:</strong><ul style="margin-top:4px; margin-bottom:0;">';
                        phase.red_flags.forEach(function(flag) {
                            html += '<li>' + flag + '</li>';
                        });
                        html += '</ul></div>';
                    }
                    html += '</div>';
                });
                html += '</div></div>';
            }

            if (analysis.critical_questions && analysis.critical_questions.length > 0) {
                html += '<div style="margin-bottom:12px;"><strong>Critical Questions to Ask Yourself:</strong><ul>';
                analysis.critical_questions.forEach(function(q) {
                    html += '<li>' + q + '</li>';
                });
                html += '</ul></div>';
            }

            if (analysis.recommendations && analysis.recommendations.length > 0) {
                html += '<div style="margin-bottom:12px;"><strong>Key Recommendations:</strong><ul>';
                analysis.recommendations.forEach(function(rec) {
                    html += '<li>' + rec + '</li>';
                });
                html += '</ul></div>';
            }

            html += '</div>';
            resultDiv.innerHTML = html;
        }

        function submitDraftMessage() {
            const intention = document.getElementById('draftIntention').value.trim();
            const draft = document.getElementById('draftText').value.trim();

            if (!intention) {
                alert('Please describe your intention');
                return;
            }
            if (!draft) {
                alert('Please write your draft message');
                return;
            }

            const suggestionDiv = document.getElementById('draftSuggestion');
            suggestionDiv.innerHTML = '<div class="draft-suggestion" style="text-align:center; color:#999;">Getting suggestions from Moly...</div>';

            fetch('/api/draft-message', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    contact_id: selectedContactId,
                    intention: intention,
                    draft: draft
                })
            })
            .then(function(r) { return r.json(); })
            .then(function(data) {
                if (data.success) {
                    suggestionDiv.innerHTML = '<div class="draft-suggestion"><strong>Moly\'s Suggestions for ' + data.contact + ':</strong><br><br>' + data.suggestion.replace(/\n/g, '<br>') + '</div>';
                } else {
                    suggestionDiv.innerHTML = '<div class="draft-suggestion" style="color:red;">Error: ' + data.error + '</div>';
                }
            })
            .catch(function(e) {
                suggestionDiv.innerHTML = '<div class="draft-suggestion" style="color:red;">Error: ' + e.message + '</div>';
            });
        }

        function toggleNewContactForm() {
            const form = document.getElementById('newContactForm');
            form.classList.toggle('active');
            if (form.classList.contains('active')) {
                document.getElementById('newContactName').focus();
            } else {
                clearNewContactForm();
            }
        }

        function clearNewContactForm() {
            document.getElementById('newContactName').value = '';
            document.getElementById('newContactPlatform').value = '';
            document.getElementById('newContactRelationship').value = '';
        }

        function submitNewContact() {
            const name = document.getElementById('newContactName').value.trim();
            const platform = document.getElementById('newContactPlatform').value.trim() || 'email';
            const relationship = document.getElementById('newContactRelationship').value.trim() || 'contact';

            if (!name) {
                alert('Please enter a contact name');
                return;
            }

            fetch('/api/contacts', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name: name, platform: platform, relationship: relationship })
            })
            .then(function(r) { return r.json(); })
            .then(function(data) {
                if (data.success) {
                    clearNewContactForm();
                    toggleNewContactForm();
                    loadContacts();
                    const select = document.getElementById('contactSelect');
                    select.value = data.contact.id;
                    switchContact();
                } else {
                    alert('Error creating contact: ' + data.error);
                }
            })
            .catch(function(e) { alert('Error creating contact: ' + e.message); });
        }

        function loadContactInfo(contactId) {
            fetch('/api/contacts')
                .then(function(r) { return r.json(); })
                .then(function(data) {
                    const contact = data.contacts.find(function(c) { return c.id === contactId; });
                    if (!contact) return;

                    selectedContactPlatform = contact.platform;

                    document.getElementById('contactNotes').textContent = contact.notes || 'No notes yet';

                    let lastIntText = 'Never';
                    if (contact.last_interaction) {
                        const date = new Date(contact.last_interaction);
                        lastIntText = date.toLocaleDateString() + ' ' + date.toLocaleTimeString();
                    }
                    document.getElementById('lastInteraction').textContent = lastIntText;

                    fetch('/api/interactions?contact_id=' + contactId + '&limit=5')
                        .then(function(r) { return r.json(); })
                        .then(function(data) {
                            const topicsMap = {};
                            data.interactions.forEach(function(inter) {
                                if (inter.topic) {
                                    topicsMap[inter.topic] = true;
                                }
                            });
                            const topicsList = Object.keys(topicsMap);
                            const topicsHtml = topicsList.length > 0 ?
                                topicsList.map(function(t) { return '<div class="topic-tag">' + t + '</div>'; }).join('') :
                                '<div style="color: #999; font-size: 11px;">No topics yet</div>';
                            document.getElementById('recentTopics').innerHTML = topicsHtml;
                        });

                    document.getElementById('contactInfo').classList.add('active');
                    document.getElementById('messages').innerHTML = '<div class="message system">Ready to chat with ' + contact.name + '...</div>';
                })
                .catch(function(e) { console.error('Failed to load contact info:', e); });
        }

        function analyzeContextForContact() {
            if (!selectedContactId) return;

            const messageStart = (document.getElementById('messageInput').value || 'Hi').substring(0, 50);

            fetch('/api/analyze-context', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    contact_id: selectedContactId,
                    platform: selectedContactPlatform,
                    message_start: messageStart
                })
            })
            .then(function(r) { return r.json(); })
            .then(function(data) {
                if (data.success && data.analysis) {
                    const analysis = data.analysis;
                    const messages = document.getElementById('messages');

                    if (analysis.questions && analysis.questions.length > 0) {
                        const questionsDiv = document.createElement('div');
                        questionsDiv.className = 'message questions';

                        let html = '<strong>Clarifying Questions (Understanding: ' + analysis.understanding_level + '%):</strong>';
                        html += '<ul class="questions-list">';
                        analysis.questions.forEach(function(q) {
                            html += '<li>' + q + '</li>';
                        });
                        html += '</ul>';
                        html += '<div class="understanding-level">' + analysis.reasoning + '</div>';

                        questionsDiv.innerHTML = html;
                        messages.appendChild(questionsDiv);
                        messages.scrollTop = messages.scrollHeight;
                    }
                }
            })
            .catch(function(e) { console.error('Failed to analyze context:', e); });
        }


        function sendMessage() {
            const input = document.getElementById('messageInput');
            const message = input.value.trim();

            if (!message) return;
            if (!selectedContactId) {
                alert('Please select a contact first');
                return;
            }

            // Check for safety concerns
            fetch('/api/check-safety', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ text: message })
            })
            .then(function(r) { return r.json(); })
            .then(function(data) {
                if (data.success && data.alert) {
                    displaySafetyAlert(data.alert);
                    return;
                }
                proceedWithMessage(message, input);
            })
            .catch(function(e) {
                proceedWithMessage(message, input);
            });
        }

        function proceedWithMessage(message, input) {
            const messages = document.getElementById('messages');
            const userMsg = document.createElement('div');
            userMsg.className = 'message user';
            userMsg.textContent = message;
            messages.appendChild(userMsg);
            messages.scrollTop = messages.scrollHeight;

            input.value = '';

            const loading = document.createElement('div');
            loading.className = 'message system';
            loading.textContent = 'Moly is thinking...';
            messages.appendChild(loading);
            messages.scrollTop = messages.scrollHeight;

            const config = { provider: currentProvider, model: null, mode: 'direct' };
            fetch('/api/settings')
                .then(function(r) { return r.json(); })
                .then(function(cfg) {
                    config.model = cfg.model;
                    config.mode = cfg.mode;

                    return fetch('/api/chat', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({
                            message: message,
                            provider: config.provider,
                            model: config.model,
                            mode: config.mode
                        })
                    });
                })
                .then(function(r) { return r.json(); })
                .then(function(data) {
                    loading.remove();

                    const response = document.createElement('div');
                    response.className = 'message';
                    response.textContent = data.response || 'No response';
                    messages.appendChild(response);
                    messages.scrollTop = messages.scrollHeight;

                    if (selectedContactId) {
                        fetch('/api/extract-insights', {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify({
                                contact_id: selectedContactId,
                                user_message: message,
                                moly_response: data.response || ''
                            })
                        })
                        .then(function(r) { return r.json(); })
                        .then(function(data) {
                            if (data.success && data.insights) {
                                fetch('/api/interactions', {
                                    method: 'POST',
                                    headers: { 'Content-Type': 'application/json' },
                                    body: JSON.stringify({
                                        contact_id: selectedContactId,
                                        platform: selectedContactPlatform,
                                        topic: (data.insights.topics || []).join(', ') || 'general',
                                        sentiment: data.insights.tone_detected,
                                        ai_summary: 'Message about: ' + (data.insights.topics || ['conversation']).join(', '),
                                        user_notes: ''
                                    })
                                })
                                .catch(function(e) { console.error('Failed to record interaction:', e); });
                            }
                        })
                        .catch(function(e) { console.error('Failed to extract insights:', e); });

                        analyzeContextForContact();
                    }
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

        function displaySafetyAlert(alert) {
            if (!alert) return;

            const alertElement = document.getElementById('safetyAlert');
            document.getElementById('safetyAlertTitle').textContent = alert.title || 'Safety Alert';
            document.getElementById('safetyAlertMessage').textContent = alert.message || '';

            // Display indicators
            const indicatorsSection = document.getElementById('indicatorsSection');
            if (alert.indicators && alert.indicators.length > 0) {
                indicatorsSection.style.display = 'block';
                const indicatorsList = document.getElementById('safetyIndicators');
                indicatorsList.innerHTML = alert.indicators.map(function(ind) {
                    return '<li>' + ind + '</li>';
                }).join('');
            } else {
                indicatorsSection.style.display = 'none';
            }

            // Display resources
            const resourcesSection = document.getElementById('resourcesSection');
            if (alert.resources && alert.resources.length > 0) {
                resourcesSection.style.display = 'block';
                const resourcesDiv = document.getElementById('safetyResources');
                resourcesDiv.innerHTML = alert.resources.map(function(res) {
                    let html = '<div class="safety-alert-resource">' +
                        '<div class="safety-alert-resource-name">' + res.name + '</div>' +
                        '<div class="safety-alert-resource-desc">' + res.description + '</div>';
                    if (res.number) {
                        html += '<div class="safety-alert-resource-contact">' + res.number + '</div>';
                    }
                    if (res.url) {
                        html += '<div style="margin-top:6px;"><a href="' + res.url + '" target="_blank" style="color:#1976D2; font-size:12px; text-decoration:none;">Learn more</a></div>';
                    }
                    html += '</div>';
                    return html;
                }).join('');
            } else {
                resourcesSection.style.display = 'none';
            }

            // Display recommendations
            const recommendationsSection = document.getElementById('recommendationsSection');
            if (alert.recommendations && alert.recommendations.length > 0) {
                recommendationsSection.style.display = 'block';
                const recommendationsList = document.getElementById('safetyRecommendations');
                recommendationsList.innerHTML = alert.recommendations.map(function(rec) {
                    return '<li>' + rec + '</li>';
                }).join('');
            } else {
                recommendationsSection.style.display = 'none';
            }

            alertElement.classList.remove('hidden');
        }

        function closeSafetyAlert() {
            document.getElementById('safetyAlert').classList.add('hidden');
        }

        function initializeAppPhase3() {
            loadContacts();
            setInterval(loadContacts, 30000);
        }

        initializeApp();
        initializeAppPhase3();
    </script>
</body>
</html>`
