# Build Moly Native Host on macOS

Run these commands on a macOS machine (Intel or Apple Silicon):

```bash
# Navigate to the project
cd ~/path/to/Moly/moly-installer/native-host

# Install Homebrew if needed (only run if you don't have it)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install Python 3
brew install python3

# Create virtual environment
python3 -m venv build-env
source build-env/bin/activate

# Install PyInstaller
pip install pyinstaller

# Build the binary
python3 -m PyInstaller \
    --onefile \
    --console \
    --name "moly-native-host-$(uname -m)" \
    --distpath "./dist-macos" \
    moly-host.py

# Prepare release
mkdir -p releases
ARCH=$(uname -m)
cp "dist-macos/moly-native-host-$ARCH" "releases/moly-native-host-macos-$ARCH"
chmod +x "releases/moly-native-host-macos-$ARCH"
tar -czf "releases/moly-native-host-macos-$ARCH.tar.gz" -C releases "moly-native-host-macos-$ARCH"

# Cleanup
rm -rf dist-macos build *.spec __pycache__

# Verify
ls -lh releases/moly-native-host-macos-*.tar.gz
```

## What to expect:

- On Apple Silicon: `moly-native-host-macos-arm64.tar.gz` (~10 MB)
- On Intel Mac: `moly-native-host-macos-x64.tar.gz` (~10 MB)

## After building:

1. Copy the tarball to your Linux machine
2. Place it in: `moly-installer/native-host/releases/`
3. Tell me when done

---

**Troubleshooting:**

If `brew install python3` fails, try:
```bash
brew install python@3.11
/usr/local/opt/python@3.11/bin/python3 -m venv build-env
```

If PyInstaller fails, try updating it:
```bash
pip install --upgrade pyinstaller
```
