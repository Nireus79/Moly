# Moly Installer

One-click installer for Moly with local AI models (Ollama or LM Studio) and automatic service setup.

## Features

- System requirements checker (OS, RAM, disk space)
- Interactive wizard for easy configuration
- Provider choice: Ollama (CLI) or LM Studio (GUI) or Cloud-only
- Model selection and auto-download
- Platform-specific auto-start service setup
- Cross-platform support (Windows, macOS, Linux)

## Installation

```bash
npm install -g moly-installer
```

## Usage

```bash
moly-installer
```

This will:
1. Check your system meets requirements
2. Launch an interactive wizard
3. Download and install selected components
4. Pull your chosen AI model
5. Configure auto-start services
6. Display next steps

## System Requirements

### Minimum
- OS: Windows 10+, macOS 10.15+, Linux
- RAM: 4 GB
- Disk: 10 GB free
- CPU: 2+ cores

### Recommended
- RAM: 8+ GB
- SSD with 15+ GB free
- Modern multi-core CPU

## Supported Providers

### Ollama (Recommended for developers)
- CLI-based, lightweight
- Excellent for coding tasks
- Supports many models
- Requires Node.js for CORS proxy

### LM Studio (Recommended for non-programmers)
- GUI application
- Easier interface
- Good for beginners
- Downloads models through app

### Cloud-Only (No local model)
- Uses Claude or OpenAI
- No local installation needed
- Requires API key
- Less privacy

## Supported Models

### Ollama
- Mistral 7B (4 GB, recommended)
- Llama 2 7B (4 GB)
- Neural Chat 7B (4 GB)

### LM Studio
- Mistral 7B (GGUF format)
- Llama 2 7B Chat (GGUF format)

## What Gets Installed

### Ollama Path
- Ollama binary
- Moly CORS proxy (npm global)
- Model (downloaded)
- systemd/LaunchAgent service

### LM Studio Path
- LM Studio application
- Model (downloaded via app)
- Launcher shortcut

### Both
- Moly browser extension (manual install)

## Platform-Specific Notes

### Windows
- Run as Administrator for service setup
- Uses Task Scheduler for auto-start
- Downloads to `%APPDATA%\Local\Moly`

### macOS
- Creates LaunchAgent for auto-start
- Ollama: `/Applications/Ollama.app`
- Downloads to `~/Library/Application Support/Moly`

### Linux
- Creates systemd service for auto-start
- Requires sudo for service setup
- Downloads to `~/.local/share/moly`

## Troubleshooting

### Port Already in Use
```bash
# Check what's using port 11435
netstat -tulpn | grep 11435

# Use different port
moly-proxy --port 11436
```

### Download Failed
- Check internet connection
- Try running installer again
- Download components manually:
  - Ollama: https://ollama.ai
  - LM Studio: https://lmstudio.ai

### Model Not Downloading
For Ollama:
```bash
ollama pull mistral
```

For LM Studio:
- Open app and use Search tab
- Download model manually

### Service Not Starting
Check logs:
- Linux: `journalctl -u moly-proxy -n 20`
- macOS: `log stream --predicate 'process == "node"'`
- Windows: Task Scheduler > Moly Proxy > History

## Development

```bash
git clone https://github.com/Nireus79/Moly
cd moly-installer
npm install
npm run dev
```

## Testing

```bash
npm test
```

## Architecture

```
Installer Flow:
1. System Requirements Check
   ├── OS compatibility
   ├── RAM available
   ├── Disk space
   └── CPU cores

2. Configuration Wizard
   ├── Choose provider
   ├── Select model
   └── Configure options

3. Download Components
   ├── Ollama/LM Studio binary
   ├── CORS proxy (npm)
   └── Model files

4. Setup Services
   ├── systemd (Linux)
   ├── LaunchAgent (macOS)
   └── Task Scheduler (Windows)

5. Verification & Next Steps
```

## File Structure

```
moly-installer/
├── src/
│   ├── cli.js                 # Main entry point
│   ├── systemRequirements.js  # System check
│   ├── wizard.js              # Interactive wizard
│   ├── downloadManager.js     # Component download
│   ├── modelDownloader.js     # Model fetching
│   └── serviceManager.js      # Auto-start setup
├── package.json
└── README.md
```

## Support

- Issues: https://github.com/Nireus79/Moly/issues
- Docs: https://github.com/Nireus79/Moly

## License

MIT
