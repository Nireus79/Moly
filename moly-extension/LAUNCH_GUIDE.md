# Moly Launch Guide

## Quick Start

### Installation for Development

```bash
# Clone the repository
git clone https://github.com/Nireus79/Moly.git
cd Moly/Moly/moly-extension

# Install dependencies
npm install

# Build the extension
npm run build

# Run tests
npm test
```

### Building for Chrome Web Store

```bash
# Production build (optimized)
npm run build

# Output will be in ./dist directory
# Ready to submit to Chrome Web Store
```

## Installation in Chrome

### Development Installation (Unpacked)

1. Open Chrome
2. Navigate to `chrome://extensions`
3. Enable "Developer mode" (toggle in top right)
4. Click "Load unpacked"
5. Select the `dist` directory from your Moly installation
6. Extension will now appear in your extensions menu

### Installation from Chrome Web Store (Production)

Once submitted and approved:
1. Visit the Moly extension page on Chrome Web Store
2. Click "Add to Chrome"
3. Confirm permissions
4. Extension is now installed and ready to use

## Initial Setup

### Step 1: Configure Your AI Provider

1. Click the Moly icon (🧠) in the toolbar
2. Click "⚙️ Settings"
3. Choose your preferred AI provider:
   - **Claude** (Recommended): Most capable, best for nuanced responses
   - **OpenAI**: Fast and reliable, good for general suggestions
   - **Ollama**: Local models, no API costs, full privacy

### Step 2: Add Your API Key

#### For Claude:
1. Go to https://console.anthropic.com/keys
2. Create a new API key
3. Copy and paste into Moly settings
4. Click "Save & Validate"

#### For OpenAI:
1. Go to https://platform.openai.com/keys
2. Create a new API key
3. Copy and paste into Moly settings
4. Click "Save & Validate"

#### For Ollama:
1. Download Ollama from https://ollama.ai
2. Install and run Ollama locally
3. In Moly settings, keep default URL: `http://localhost:11434`
4. Click "Save & Validate"

### Step 3: Add Your First Contact

1. Click the Moly icon (🧠)
2. Click "📋 Recent" tab
3. Start a conversation with anyone
4. Moly will automatically detect the contact
5. Or manually add contacts in the sidebar

### Step 4: Get Your First Suggestion

1. Open any messaging app (Tinder, Discord, etc.)
2. Read a message you need to respond to
3. Click the Moly icon (🧠) to open the sidebar
4. Moly will show message suggestions
5. Click "📋 Copy" to copy a suggestion
6. Paste into your message field

## Features Overview

### Chat Modes

**Socratic Mode** (💭)
- Get thoughtful questions to refine your response
- Helps you think deeper about what you want to say
- Perfect for important messages
- Encourages authentic communication

**Direct Mode** (⚡)
- Get 3 ready-to-use message suggestions
- Quick and efficient
- Perfect for casual conversations
- Suggestions include tone and confidence score

### Communication Contexts

**Formal** (💼)
- Professional and respectful tone
- Best for: LinkedIn, professional contexts
- Maintains boundaries and professionalism

**Friendly** (👋)
- Warm and approachable tone
- Best for: General social interactions
- Builds rapport and connection

**Dating** (💕)
- Flirty and romantic tone
- Best for: Dating apps, romantic contexts
- Adds personality and authentic interest

### Contact Management

**Add Contact**
- Click "+" in sidebar
- Enter name, platform, and notes
- Save to your contacts

**View Conversation History**
- Click a contact in the Recent list
- See all past messages and responses
- Review suggestions you've used

**Search Contacts**
- Use the search box in sidebar
- Find contacts by name or platform
- Quick access to frequent contacts

### Settings

**Provider Settings**
- Configure API keys
- Choose default model
- Test connectivity
- Switch between providers

**Preferences**
- Default chat mode (Socratic/Direct)
- Default communication context
- Add/remove contacts
- Clear all settings (with confirmation)

**About**
- Version information
- Supported platforms
- Feature list

## Troubleshooting

### Extension Won't Load

**Problem:** "Could not load the manifest"
**Solution:**
1. Ensure you ran `npm run build`
2. Check that `dist/manifest.json` exists
3. Try removing and re-adding the extension

### No Suggestions Generated

**Problem:** "Failed to generate suggestions"
**Solutions:**
1. Check that your API key is configured in Settings
2. Verify your API key has credits/valid billing
3. Ensure internet connection is active
4. Try selecting a different provider
5. Check browser console for errors: `F12 → Console`

### Messages Not Being Detected

**Problem:** Sidebar doesn't show detected messages
**Solutions:**
1. Check that content script is injected: `F12 → Sources`
2. Try refreshing the page
3. Ensure Moly has permission for the website
4. Check if website uses custom message formats
5. Report issue with website name and URL

### Ollama Not Connecting

**Problem:** "Could not connect to Ollama"
**Solutions:**
1. Start Ollama: `ollama serve`
2. Verify Ollama is running on port 11434
3. Pull a model: `ollama pull mistral`
4. Check firewall isn't blocking localhost
5. Try accessing http://localhost:11434 in browser

### Settings Not Saving

**Problem:** Settings reset after closing Chrome
**Solutions:**
1. Check that storage permissions are granted
2. Clear browser cache and try again
3. Ensure Chrome storage is enabled
4. Try a different provider configuration
5. Reinstall the extension

## Performance Tips

### Reduce Latency

1. **Use Claude for fastest responses** (lowest latency)
2. **Use Direct mode** instead of Socratic (fewer requests)
3. **Keep browser updated** (better performance)
4. **Close unused tabs** (reduces browser load)

### Reduce API Costs

1. **Use Ollama** (free, local models)
2. **Use Direct mode** (fewer tokens per request)
3. **Batch requests** (combine multiple suggestions)
4. **Monitor API usage** (check your provider dashboard)

### Improve Quality

1. **Use Socratic mode** for important messages
2. **Set appropriate context** (Formal/Friendly/Dating)
3. **Provide more context** in the detected message
4. **Review suggestions carefully** before sending

## Browser Compatibility

### Supported Browsers

- ✅ **Chrome 120+** (Full support)
- ✅ **Edge 120+** (Chromium-based, full support)
- ✅ **Brave 1.70+** (Chromium-based, full support)
- ✅ **Opera 106+** (Chromium-based, full support)

### Not Supported

- ❌ **Firefox** (Manifest V3 not fully supported)
- ❌ **Safari** (Different extension format)
- ❌ **Chrome <120** (Requires Manifest V3)

## Permissions Explained

| Permission | Why Needed | How Used |
|-----------|-----------|----------|
| `activeTab` | Know which tab is active | Detect messages on current page |
| `scripting` | Inject content script | Detect messages and run on websites |
| `storage` | Save data | Store contacts, conversations, API keys |
| `notifications` | Show alerts | Notify when suggestions are ready |
| `clipboardRead` | Access clipboard | Quick suggestion copying |
| `<all_urls>` | Run on any website | Support all messaging platforms |

## Privacy & Security

### What Moly Does NOT Do

- ❌ Never sends your messages to Moly servers
- ❌ Never collects personal data
- ❌ Never shares data with third parties
- ❌ Never uses tracking or analytics
- ❌ Never stores your data after you delete it

### What Moly DOES Do

- ✅ Stores your API keys securely in local browser storage
- ✅ Saves your contacts and conversations locally
- ✅ Sends your messages to YOUR chosen AI provider (Claude, OpenAI, etc.)
- ✅ Lets you delete all data with one button

### Best Practices

1. **Keep API keys private** - Never share your API keys
2. **Use strong passwords** - Protect your browser
3. **Monitor API usage** - Check your provider dashboard
4. **Review suggestions** - Don't send without reading
5. **Delete when done** - Clear history if sharing computer

## Advanced Configuration

### Custom Model Selection

1. Open Settings
2. Choose your provider (Claude, OpenAI, Ollama)
3. In the "Model" dropdown, select your preferred model
4. Click "Save & Validate"

**Recommended Models:**
- Claude: `claude-3-5-sonnet-20241022` (balanced, recommended)
- OpenAI: `gpt-4` (most capable), `gpt-3.5-turbo` (fast, cheap)
- Ollama: `mistral` (fast), `neural-chat` (conversational)

### Custom Communication Context

1. Open Settings → Preferences
2. Choose your default context
3. Can be overridden per conversation in sidebar

### Batch Operations

1. Add multiple contacts in Settings
2. Review all conversations in popup
3. Clear all settings at once (destructive!)

## Updating & Maintenance

### Checking for Updates

- Chrome automatically updates extensions from Web Store
- Updates appear when new versions are released
- No manual action required

### Reporting Bugs

1. Open Settings
2. Click the version number
3. Copy error message from console (`F12`)
4. Report on GitHub: https://github.com/Nireus79/Moly/issues

### Providing Feedback

- Request features on GitHub Discussions
- Share your experience on Chrome Web Store reviews
- Email: feedback@moly.ai

## Uninstalling

### Remove Moly from Chrome

1. Open Chrome extensions page: `chrome://extensions`
2. Find Moly extension
3. Click "Remove"
4. Confirm deletion

### Your Data

- All local data is deleted when extension is removed
- No data remains on Moly servers (we have none!)
- Download conversations before uninstalling if needed

## Support & Resources

- **GitHub Issues:** https://github.com/Nireus79/Moly/issues
- **GitHub Discussions:** https://github.com/Nireus79/Moly/discussions
- **Email Support:** support@moly.ai
- **Privacy Policy:** https://moly.ai/privacy
- **Terms of Service:** https://moly.ai/terms

## License

Moly is open source software. See LICENSE file for details.

## Contributing

Want to contribute? Great!

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request
5. Help us make Moly better!

See CONTRIBUTING.md for detailed guidelines.

---

**Version:** 0.1.0 (Phase 1 - MVP)
**Last Updated:** 2026-08-31
**Status:** Ready for Launch 🚀
