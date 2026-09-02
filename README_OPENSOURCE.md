# Moly - Open Source AI Coaching Extension

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Chrome Web Store](https://img.shields.io/chrome-web-store/v/EXTENSION_ID?color=4285F4)](https://chrome.google.com/webstore)
[![GitHub Stars](https://img.shields.io/github/stars/Nireus79/Moly?style=social)](https://github.com/Nireus79/Moly)

**Moly** is a privacy-first AI coaching chatbot for Chrome that helps you craft better messages through intelligent dialogue.

---

## Features

### Free (Always)
- 🧠 AI-powered coaching conversations
- 💬 Socratic and direct modes
- 🎯 Context-aware suggestions
- 📱 Works offline with local models
- 🔒 Fully private (local storage, no tracking)
- 🆓 Open source (MIT licensed)

### Premium (Coming Soon)
- ☁️ Cloud sync across devices
- 👥 Team collaboration
- 🚀 Advanced AI models
- 🔑 API access
- 📊 Usage analytics

---

## Quick Start

### Install

1. **Download** from [Chrome Web Store](#) (coming soon)
2. **Open** the extension
3. **Choose your setup**:
   - Use local models (Ollama/LM Studio) - free
   - Connect Claude/OpenAI - requires API key
   - Or both for intelligent fallback

### Local Models (Recommended)

Moly auto-detects local models running on your system:

```bash
# Install Ollama (https://ollama.ai)
ollama pull mistral

# Or use LM Studio (https://lmstudio.ai)
# Download and run from the app

# Moly will automatically find them!
```

### Cloud Models

1. Get API key: [Claude](https://console.anthropic.com) or [OpenAI](https://platform.openai.com)
2. Open Moly Settings
3. Paste your API key
4. Done!

---

## Privacy & Security

- ✅ **Open Source**: Code is public, auditable by anyone
- ✅ **No Cloud Storage**: Conversations stored locally by default
- ✅ **No Tracking**: Zero analytics, zero telemetry
- ✅ **Encrypted**: Local data encrypted with AES-256
- ✅ **Your Control**: You decide what gets synced (premium)

See [Privacy Policy](./PRIVACY_POLICY.md) for details.

---

## How It Works

### The Coaching Process

1. **You explain context**: Who are you talking to? What's the situation?
2. **Moly asks questions**: Helps you think through your message
3. **You share the message**: Paste incoming message (optional)
4. **Get suggestions**: AI-generated responses based on context
5. **Copy and send**: One-click copy to clipboard

### Intelligent Fallback

- **Primary**: Your local model (fast, private)
- **Fallback 1**: Other local models (if available)
- **Fallback 2**: Claude/OpenAI (if configured)
- **Result**: Always works, always fast

---

## Architecture

```
Browser Extension (2.5 MB)
├── UI Layer (React + TypeScript)
├── State Management (Zustand)
└── Native Messaging Bridge

Native Host (~9-12 MB per platform)
├── Service Control (Ollama, CORS Proxy)
├── Model Detection
└── Encryption/Decryption

Local Storage (~.local/share/moly/)
├── Encrypted conversations
├── Settings
└── Cache

User's Models (~.ollama/models/)
├── Mistral, Llama2, etc.
└── (NOT managed by Moly)
```

See [INSTALLATION_ARCHITECTURE.md](./INSTALLATION_ARCHITECTURE.md) for details.

---

## For Developers

### Setup

```bash
git clone https://github.com/Nireus79/Moly.git
cd Moly/moly-extension
npm install
npm run build
```

### Load in Chrome

1. `chrome://extensions/`
2. Enable "Developer mode"
3. "Load unpacked" → select `dist/` folder

### Development

```bash
npm run dev        # Watch mode
npm run lint       # Type and style check
npm run build      # Production build
```

### Architecture

- **moly-extension/**: Chrome extension (React + TypeScript)
- **moly-installer/native-host/**: System service (Python)
- **Releases**: Native host binaries in [Moly releases](https://github.com/Nireus79/Moly/releases)

---

## Contributing

We welcome contributions! See [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines.

### Help Needed

- 🖥️ **Platform Support**: macOS and Windows native hosts
- 🌍 **Localization**: Translate UI to other languages
- 🧪 **Testing**: E2E testing on different systems
- 📚 **Documentation**: Improve guides and troubleshooting
- 💡 **Features**: Suggest and implement improvements

---

## Roadmap

### v1.0 (Now) ✓
- Open source browser extension
- Local model support (Ollama/LM Studio)
- Cloud provider integration (Claude/OpenAI)
- Encryption and secure storage

### v1.1 (Next)
- Better UI/UX based on feedback
- Performance optimization
- Bug fixes and stability

### v2.0 (Q4 2026)
- Cloud sync (premium feature)
- Team collaboration
- Advanced AI coaching algorithms
- API access (enterprise)

### v3.0 (2027)
- Firefox/Safari support
- Mobile apps
- White-label licensing
- Advanced analytics (premium)

---

## FAQ

### Is it really free?

Yes! The core product (coaching extension + local models) is free forever and open source.

Premium features (cloud sync, team collaboration, advanced models) will have an optional paid tier to support development.

### Is my data private?

Yes. Conversations stored locally are encrypted. We don't see them, analyze them, or share them.

If you choose to use cloud models (Claude/OpenAI), data goes to those providers per their privacy policies.

### How do I uninstall?

Click "Remove" on the extension. Your conversations and models stay on your system.

### Can I use it without local models?

Yes! Configure an API key for Claude or OpenAI and Moly works immediately.

### Can I contribute?

Absolutely! See [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines.

---

## Support

- 📖 [Quick Start Guide](./QUICKSTART.md)
- 🔧 [Troubleshooting](./TROUBLESHOOTING.md)
- 📋 [Installation Guide](./INSTALLATION_ARCHITECTURE.md)
- 💬 [GitHub Issues](https://github.com/Nireus79/Moly/issues)
- 💡 [Discussions](https://github.com/Nireus79/Moly/discussions)

---

## License

Moly is open source under the **MIT License**. See [LICENSE](./LICENSE) for details.

You're free to:
- ✓ Use it commercially
- ✓ Modify and extend it
- ✓ Distribute it
- ✓ Use it privately

You just need to:
- Include the license in copies
- State significant changes

---

## Acknowledgments

Built with:
- [React 18](https://react.dev)
- [TypeScript](https://www.typescriptlang.org)
- [Zustand](https://github.com/pmndrs/zustand)
- [Anthropic Claude API](https://anthropic.com)
- [OpenAI API](https://openai.com)
- [Ollama](https://ollama.ai)

---

## Contact

- **Author**: Efthimios Angelopoulos
- **Email**: efthimiosangelopoulos@gmail.com
- **GitHub**: [@Nireus79](https://github.com/Nireus79)

---

Made with ❤️ for better conversations

**[⬆ Back to top](#)**
