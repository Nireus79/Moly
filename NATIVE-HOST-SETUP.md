# Moly Native Host Setup

This document describes how to set up the native messaging host for Moly on different platforms. This enables the "Launch Installer" button in Moly Settings to directly launch the installer without manual download.

## Overview

The Moly extension uses Chrome's Native Messaging API to communicate with a native host application that can:
- Launch the Moly installer
- Check if the installer is installed
- Verify system requirements

Without this setup, users can still download and run the installer manually via the "Download Installer" button.

## Platform-Specific Setup

### macOS Setup

1. **Create Native Host Directory:**
   ```bash
   mkdir -p ~/Library/Application\ Support/Google/Chrome/NativeMessagingHosts
   ```

2. **Create Host Configuration File:**
   Create: `~/Library/Application Support/Google/Chrome/NativeMessagingHosts/com.moly.installer.json`
   
   Content:
   ```json
   {
     "name": "com.moly.installer",
     "description": "Moly Installer Launcher",
     "path": "/usr/local/bin/moly-native-host",
     "type": "stdio",
     "allowed_origins": [
       "chrome-extension://[EXTENSION_ID]/",
       "chrome-extension://[EXTENSION_ID]/*"
     ]
   }
   ```

3. **Install Native Host Binary:**
   ```bash
   # Copy the binary to system path
   sudo cp moly-native-host-macos /usr/local/bin/moly-native-host
   sudo chmod +x /usr/local/bin/moly-native-host
   ```

4. **Verify Installation:**
   ```bash
   # Test the host
   echo '{"action":"ping"}' | /usr/local/bin/moly-native-host
   ```

### Linux Setup

1. **Create Native Host Directory:**
   ```bash
   mkdir -p ~/.config/google-chrome/NativeMessagingHosts
   ```

2. **Create Host Configuration File:**
   Create: `~/.config/google-chrome/NativeMessagingHosts/com.moly.installer.json`
   
   Content:
   ```json
   {
     "name": "com.moly.installer",
     "description": "Moly Installer Launcher",
     "path": "/usr/local/bin/moly-native-host",
     "type": "stdio",
     "allowed_origins": [
       "chrome-extension://[EXTENSION_ID]/",
       "chrome-extension://[EXTENSION_ID]/*"
     ]
   }
   ```

3. **Install Native Host Binary:**
   ```bash
   # Copy the binary to system path
   sudo cp moly-native-host-linux /usr/local/bin/moly-native-host
   sudo chmod +x /usr/local/bin/moly-native-host
   ```

4. **Verify Installation:**
   ```bash
   # Test the host
   echo '{"action":"ping"}' | /usr/local/bin/moly-native-host
   ```

### Windows Setup

1. **Create Registry Entry:**
   Run as Administrator:
   ```powershell
   # Open Registry Editor (regedit)
   # Navigate to: HKEY_LOCAL_MACHINE\SOFTWARE\Google\Chrome\NativeMessagingHosts
   # Create new Key: com.moly.installer
   # Set Default value to the path of the host config file
   ```

2. **Create Host Configuration File:**
   Create: `C:\Program Files\Moly\com.moly.installer.json`
   
   Content:
   ```json
   {
     "name": "com.moly.installer",
     "description": "Moly Installer Launcher",
     "path": "C:\\Program Files\\Moly\\moly-native-host.exe",
     "type": "stdio",
     "allowed_origins": [
       "chrome-extension://[EXTENSION_ID]/",
       "chrome-extension://[EXTENSION_ID]/*"
     ]
   }
   ```

3. **Registry Entry Details:**
   ```
   Key: HKEY_LOCAL_MACHINE\SOFTWARE\Google\Chrome\NativeMessagingHosts\com.moly.installer
   Value: (Default) = "C:\Program Files\Moly\com.moly.installer.json"
   ```

4. **Verify Installation:**
   ```powershell
   # Test the host
   $process = New-Object System.Diagnostics.Process
   $process.StartInfo.FileName = "C:\Program Files\Moly\moly-native-host.exe"
   $process.StartInfo.UseShellExecute = $false
   $process.StartInfo.RedirectStandardInput = $true
   $process.Start()
   $process.StandardInput.WriteLine('{"action":"ping"}')
   ```

## Finding Your Extension ID

After installing the Moly extension:

1. Open `chrome://extensions/`
2. Find "Moly - Messaging Coach"
3. Copy the Extension ID (example: `abcdefghijklmnopqrstuvwxyz123456`)
4. Replace `[EXTENSION_ID]` in the configuration files above

## Native Host Implementation

The native host is a simple executable that:

```python
#!/usr/bin/env python3
import sys
import json

def handle_message(request):
    action = request.get("action")
    
    if action == "ping":
        return {"pong": True}
    
    elif action == "launch":
        # Launch the installer
        try:
            import subprocess
            import platform
            
            os_name = platform.system()
            if os_name == "Darwin":  # macOS
                subprocess.Popen(["open", "-a", "Moly Installer"])
            elif os_name == "Linux":
                subprocess.Popen(["moly-installer"])
            elif os_name == "Windows":
                subprocess.Popen(["C:\\Program Files\\Moly\\moly-installer.exe"])
            
            return {"success": True}
        except Exception as e:
            return {"success": False, "error": str(e)}
    
    else:
        return {"success": False, "error": "Unknown action"}

def main():
    while True:
        try:
            # Read message length
            message_length = sys.stdin.buffer.read(4)
            if len(message_length) == 0:
                break
            
            length = int.from_bytes(message_length, "little")
            
            # Read message
            message_data = sys.stdin.buffer.read(length).decode("utf-8")
            request = json.loads(message_data)
            
            # Handle request
            response = handle_message(request)
            
            # Send response
            response_json = json.dumps(response)
            response_bytes = response_json.encode("utf-8")
            
            sys.stdout.buffer.write(len(response_bytes).to_bytes(4, "little"))
            sys.stdout.buffer.write(response_bytes)
            sys.stdout.buffer.flush()
            
        except Exception:
            break

if __name__ == "__main__":
    main()
```

## Fallback Without Native Host

If the native host is not installed, Moly gracefully falls back to:

1. **Download Installer** button downloads the platform-specific binary
2. **Release Page** button opens GitHub releases page
3. Platform-specific terminal instructions are displayed

This ensures all users can install Moly, even without native host setup.

## Testing

To test the native host installation:

1. Open Moly Settings
2. Navigate to "Local Models Status" section
3. Click "Start Setup" if no local model is detected
4. The "Launch Installer" button should work if native host is installed

If the button doesn't work:
- Check the console for error messages (F12 → Console)
- Verify the host configuration file exists
- Verify the extension ID matches in the configuration
- Check that the native host binary is executable

## Troubleshooting

### "Native installer not found" message

This is normal if the native host is not installed. Users can still:
- Click "Download Installer" to download manually
- Click "Release Page" for full download options
- Use the terminal instructions provided

### Host configuration not found

1. Verify the directory path is correct for your OS
2. Check that the file extension is `.json`
3. Verify the `allowed_origins` includes your extension ID
4. Restart Chrome after making changes

### Permission Denied on Binary

Linux/macOS:
```bash
sudo chmod +x /usr/local/bin/moly-native-host
```

Windows:
- Run PowerShell as Administrator
- Verify the file is in `Program Files`
- Check that your user account has read/execute permissions

## Building the Native Host

See `moly-installer/native-host/` for build scripts for each platform.

### macOS
```bash
cd moly-installer/native-host
./build-macos.sh
```

### Linux
```bash
cd moly-installer/native-host
./build-linux.sh
```

### Windows
```bash
cd moly-installer/native-host
build-windows.bat
```

## Security Notes

- The native host only responds to messages from the Moly extension
- The host ID (`com.moly.installer`) is restricted to Moly's extension ID
- All commands are verified before execution
- The host runs in stdio mode (no network listening)
- User confirmation is shown before launching the installer

## Future Work

- Automatic native host installer with Moly installer
- Native host auto-update mechanism
- Support for Chromium and Brave browsers
- Firefox native host support (via WebExtensions)
