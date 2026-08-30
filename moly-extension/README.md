# Moly Extension

Universal messaging coach - Browser extension that automatically detects incoming messages and helps you craft better responses.

## Development Setup

### Prerequisites
- Node.js 16+ and npm 8+
- Clone the Moly repository

### Installation

```bash
cd moly-extension
npm install
```

### Development

```bash
# Start dev server with hot reload
npm run dev

# Type check
npm run type-check

# Lint code
npm run lint

# Run tests
npm run test
```

### Build

```bash
# Build extension
npm run build

# Output will be in dist/ directory
```

### Load Extension in Browser

**Chrome/Edge:**
1. Open `chrome://extensions/`
2. Enable "Developer mode"
3. Click "Load unpacked"
4. Select the `dist/` folder

**Firefox:**
1. Open `about:debugging`
2. Click "This Firefox"
3. Click "Load Temporary Add-on"
4. Select any file in the `dist/` folder

## Project Structure

```
src/
├── background/           # Service worker
├── content/             # Content scripts (message detection)
├── sidebar/             # Main UI (React)
├── popup/               # Popup UI (React)
├── api/                 # API clients
├── stores/              # Zustand stores (state management)
├── types/               # TypeScript types
└── utils/               # Utilities

manifest.json            # Extension manifest
vite.config.ts          # Build config
tsconfig.json           # TypeScript config
```

## Architecture

- **Frontend:** React 18 + TypeScript + Tailwind CSS
- **State:** Zustand (lightweight store)
- **Build:** Vite
- **Storage:** IndexedDB + chrome.storage
- **Encryption:** TweetNaCl.js (Phase 2)

## Key Files

- `manifest.json` - Extension configuration (Manifest V3)
- `src/background/serviceWorker.ts` - Background script
- `src/content/contentScript.ts` - Message detection
- `src/sidebar/Sidebar.tsx` - Main chat UI
- `src/popup/Popup.tsx` - Quick access popup

## Configuration

1. Copy `.env.example` to `.env`
2. Add your Claude API key:
   ```
   VITE_CLAUDE_API_KEY=sk-ant-...
   ```

## Bundle Size

Target: 35-40 KB (gzipped)

Current optimizations:
- Tree-shaking unused code
- Minification in production
- CSS puring with Tailwind

## Testing

Currently no tests configured (Task #10).

## License

Proprietary - Moly Project
