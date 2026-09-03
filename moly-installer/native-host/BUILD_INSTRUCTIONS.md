# Moly Native Host - Build Instructions

The Moly Native Host is a Python application compiled to standalone binaries using PyInstaller. It provides:
- Native messaging bridge for Chrome extension
- Built-in CORS proxy for Ollama integration
- Service management (start/stop Ollama)
- Auto-start configuration

## Binary Sizes

- **Linux x64**: ~9 MB
- **macOS ARM64**: ~12 MB
- **macOS x64**: ~12 MB
- **Windows x64**: ~15 MB

## Prerequisites

All platforms require:
- **Python 3.8+** (3.10+ recommended)
- **PyInstaller**: Installed in virtual environment

## Building

### Linux (x86_64)

```bash
cd moly-installer/native-host

# Create virtual environment
python3 -m venv build-env
source build-env/bin/activate

# Install PyInstaller
pip install pyinstaller

# Build
bash build-linux.sh

# Binary location: releases/moly-native-host-linux-x64
# Tarball: releases/moly-native-host-linux-x64.tar.gz
```

### macOS (ARM64 & x86_64)

```bash
cd moly-installer/native-host

# Create virtual environment
python3 -m venv build-env
source build-env/bin/activate

# Install PyInstaller
pip install pyinstaller

# Build (automatically detects architecture)
bash build-macos.sh

# Binary location: releases/moly-native-host-macos-arm64 (on Apple Silicon)
#              or releases/moly-native-host-macos-x64 (on Intel)
# Tarball: releases/moly-native-host-macos-*.tar.gz
```

### Windows (x86_64)

```batch
cd moly-installer\native-host

REM Create virtual environment
python -m venv build-env
call build-env\Scripts\activate.bat

REM Install PyInstaller
pip install pyinstaller

REM Build
build-windows.bat

REM Binary location: releases\moly-native-host-windows-x64.exe
REM Archive: releases\moly-native-host-windows-x64.zip
```

## Testing Binaries

### Test Proxy Mode (Standalone)

```bash
# Linux/macOS
./releases/moly-native-host-linux-x64 --proxy-mode

# Windows
releases\moly-native-host-windows-x64.exe --proxy-mode
```

The proxy runs on `127.0.0.1:11435` and forwards to Ollama on `127.0.0.1:11434`.

### Test Native Messaging Mode

The native messaging mode runs when invoked without arguments. It expects:
1. Chrome extension native messaging protocol (stdin/stdout in binary mode)
2. Automatically starts CORS proxy in background thread

## File Structure

After building, the `releases/` directory contains:

```
releases/
├── README-macos.txt                    # macOS-specific notes
├── moly-native-host-linux-x64          # Linux binary (executable)
├── moly-native-host-linux-x64.tar.gz   # Linux tarball for distribution
├── moly-native-host-macos-arm64        # macOS ARM64 binary
├── moly-native-host-macos-arm64.tar.gz # macOS ARM64 tarball
├── moly-native-host-macos-x64          # macOS Intel binary
├── moly-native-host-macos-x64.tar.gz   # macOS Intel tarball
├── moly-native-host-windows-x64.exe    # Windows binary (executable)
└── moly-native-host-windows-x64.zip    # Windows archive for distribution
```

## GitHub Release Setup

1. **Tag a release**: `git tag v1.0.0`
2. **Create release** on GitHub
3. **Upload binaries**:
   - Linux: `moly-native-host-linux-x64.tar.gz`
   - macOS ARM64: `moly-native-host-macos-arm64.tar.gz`
   - macOS x64: `moly-native-host-macos-x64.tar.gz`
   - Windows: `moly-native-host-windows-x64.zip`

## Installer Scripts

The installer scripts (install-linux.sh, install-macos.sh, install-windows.bat) download the appropriate binary from:

```
https://github.com/Nireus79/Moly/releases/download/v1.0.0/moly-native-host-{PLATFORM}.tar.gz
```

Update the version in installer scripts as needed.

## Build Troubleshooting

### PyInstaller Virtual Environment Issues

```bash
# Clean and rebuild
rm -rf build-env build dist-linux dist-macos *.spec
python3 -m venv build-env
source build-env/bin/activate
pip install --upgrade pip
pip install pyinstaller
```

### macOS ARM64 vs x86_64

The build script automatically detects your architecture:
- Apple Silicon (M1/M2/M3): Builds ARM64 binary
- Intel Mac: Builds x86_64 binary

To build for the other architecture, use Apple's Rosetta or a cross-compilation tool.

### Windows Build Fails

Ensure:
1. Python is installed and in PATH: `python --version`
2. You have write permissions in the current directory
3. Virtual environment is activated: `build-env\Scripts\activate.bat`
4. PyInstaller is installed: `pip list | grep PyInstaller`

## CI/CD Integration

For automated builds, create GitHub Actions workflow:

```yaml
name: Build Native Host
on:
  push:
    tags:
      - 'v*'

jobs:
  build:
    strategy:
      matrix:
        include:
          - os: ubuntu-latest
            script: bash build-linux.sh
          - os: macos-latest
            script: bash build-macos.sh
          - os: windows-latest
            script: .\build-windows.bat
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-python@v4
      - run: ${{ matrix.script }}
      - uses: actions/upload-artifact@v3
        with:
          path: releases/
```

## Version Management

Update version in:
1. `moly-host.py` - No explicit version (uses git tag)
2. `install-linux.sh` - `MOLY_VERSION="v1.0.0"`
3. `install-macos.sh` - `MOLY_VERSION="v1.0.0"`
4. `install-windows.bat` - In download URL

## Notes

- PyInstaller compiles Python to bytecode and bundles with Python runtime
- Binaries are platform-specific (must rebuild for each OS)
- Binary size includes full Python runtime (~8-10 MB)
- CORS proxy is built-in (no npm or Node.js required)
- Requires zero external dependencies at runtime
