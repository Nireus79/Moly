# Moly Desktop Application

Professional Electron-based desktop application for Moly AI Coaching.

## Features

- **One-click installation** - Auto-downloads and installs all system components
- **Zero terminal** - No command-line required, everything is automated
- **Cross-platform** - Windows, macOS, Linux
- **System integration** - Auto-starts CORS proxy on login
- **Built-in setup** - Guides first-time users through configuration

## Architecture

### Main Process (`src/main.js`)
- Window management
- Native host lifecycle (start/stop proxy)
- IPC handlers for setup and status

### Services (`src/services/`)
- **installer.js** - Downloads and installs native host binary, configures system services
  - Linux: systemd user service
  - macOS: LaunchAgent
  - Windows: Task Scheduler

### Preload (`src/preload.js`)
- Safe IPC bridge to renderer process

### Build
- `electron-builder.yml` - Packaging configuration for all platforms
- Outputs: .exe (Windows), .dmg + .zip (macOS), .AppImage + .deb (Linux)

## Installation

```bash
npm install
npm start         # Development
npm run build     # Production build
```

## Build for Platform

```bash
npm run build-win      # Windows
npm run build-mac      # macOS
npm run build-linux    # Linux
```

## TODO

- [ ] Create React UI component (integrate from moly-extension)
- [ ] Setup wizard UI component
- [ ] Icons and branding assets
- [ ] Code signing for releases
- [ ] Auto-update mechanism
- [ ] Crash reporting
- [ ] Integration testing

## Technical Notes

- Uses native binaries built by `moly-installer/native-host`
- CORS proxy runs automatically on localhost:11435
- System services auto-start proxy on login
- All user data stored in `~/.local/share/moly`
