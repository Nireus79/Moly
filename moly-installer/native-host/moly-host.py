#!/usr/bin/env python3
"""
Moly Native Messaging Host
Enables Chrome extension to launch installer and control local services
"""

import sys
import json
import subprocess
import platform
import os
from pathlib import Path


def get_installer_path():
    """Get path to moly-installer executable for current platform"""
    os_name = platform.system()

    if os_name == "Darwin":  # macOS
        # Check common installation paths
        paths = [
            "/Applications/Moly Installer.app/Contents/MacOS/Moly Installer",
            Path.home() / "Applications" / "Moly Installer.app" / "Contents" / "MacOS" / "Moly Installer",
            "/usr/local/bin/moly-installer",
        ]
    elif os_name == "Linux":
        paths = [
            "/usr/local/bin/moly-installer",
            "/usr/bin/moly-installer",
            Path.home() / ".local" / "bin" / "moly-installer",
        ]
    elif os_name == "Windows":
        paths = [
            "C:\\Program Files\\Moly\\moly-installer.exe",
            "C:\\Program Files (x86)\\Moly\\moly-installer.exe",
            Path.home() / "AppData" / "Local" / "Moly" / "moly-installer.exe",
        ]
    else:
        return None

    for path in paths:
        if isinstance(path, str):
            path = Path(path)
        if path.exists():
            return str(path)

    return None


def launch_installer():
    """Launch moly-installer application"""
    try:
        installer_path = get_installer_path()
        if not installer_path:
            return {
                "success": False,
                "error": "Installer not found. Please download from releases."
            }

        os_name = platform.system()

        if os_name == "Darwin":  # macOS
            subprocess.Popen(["open", "-a", installer_path])
        elif os_name == "Linux":
            subprocess.Popen([installer_path])
        elif os_name == "Windows":
            subprocess.Popen([installer_path])

        return {"success": True, "message": "Installer launched"}

    except Exception as e:
        return {"success": False, "error": f"Failed to launch: {str(e)}"}


def check_ollama():
    """Check if Ollama is installed and running"""
    try:
        # Check if running
        import urllib.request
        try:
            urllib.request.urlopen('http://localhost:11434/api/tags', timeout=2)
            return {"installed": True, "running": True}
        except:
            pass

        # Check if installed but not running
        os_name = platform.system()
        if os_name == "Darwin":
            if Path("/Applications/Ollama.app").exists():
                return {"installed": True, "running": False}
        elif os_name == "Linux":
            result = subprocess.run(["which", "ollama"], capture_output=True)
            if result.returncode == 0:
                return {"installed": True, "running": False}
        elif os_name == "Windows":
            if Path("C:\\Users") / os.getenv("USERNAME") / "AppData" / "Local" / "Programs" / "Ollama" / "ollama.exe":
                return {"installed": True, "running": False}

        return {"installed": False, "running": False}

    except Exception as e:
        return {"error": str(e)}


def start_ollama():
    """Start Ollama service"""
    try:
        os_name = platform.system()

        if os_name == "Darwin":
            subprocess.Popen(
                ["/Applications/Ollama.app/Contents/MacOS/ollama", "serve"],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL
            )
            return {"success": True, "message": "Ollama started"}

        elif os_name == "Linux":
            result = subprocess.run(["systemctl", "start", "ollama"], capture_output=True)
            if result.returncode == 0:
                return {"success": True, "message": "Ollama service started"}
            else:
                return {"success": False, "error": "Failed to start service"}

        elif os_name == "Windows":
            user = os.getenv("USERNAME")
            ollama_path = f"C:\\Users\\{user}\\AppData\\Local\\Programs\\Ollama\\ollama.exe"
            subprocess.Popen(
                [ollama_path, "serve"],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL
            )
            return {"success": True, "message": "Ollama started"}

        else:
            return {"success": False, "error": "Unsupported platform"}

    except Exception as e:
        return {"success": False, "error": str(e)}


def stop_ollama():
    """Stop Ollama service"""
    try:
        os_name = platform.system()

        if os_name == "Darwin":
            subprocess.run(["pkill", "-f", "ollama serve"], capture_output=True)
            return {"success": True, "message": "Ollama stopped"}

        elif os_name == "Linux":
            result = subprocess.run(["systemctl", "stop", "ollama"], capture_output=True)
            if result.returncode == 0:
                return {"success": True, "message": "Ollama service stopped"}
            else:
                return {"success": False, "error": "Failed to stop service"}

        elif os_name == "Windows":
            subprocess.run(["taskkill", "/IM", "ollama.exe", "/F"], capture_output=True)
            return {"success": True, "message": "Ollama stopped"}

        else:
            return {"success": False, "error": "Unsupported platform"}

    except Exception as e:
        return {"success": False, "error": str(e)}


def install_native_host(extension_id):
    """Self-install native host to system location"""
    try:
        import shutil
        os_name = platform.system()
        current_binary = Path(sys.executable) if hasattr(sys, 'frozen') else Path(__file__).parent / "dist" / "moly-native-host"

        if os_name == "Darwin":  # macOS
            install_path = Path("/usr/local/bin/moly-native-host")
            manifest_dir = Path.home() / "Library" / "Application Support" / "Google" / "Chrome" / "NativeMessagingHosts"

        elif os_name == "Linux":
            install_path = Path("/usr/local/bin/moly-native-host")
            manifest_dir = Path.home() / ".config" / "google-chrome" / "NativeMessagingHosts"

        elif os_name == "Windows":
            install_path = Path("C:\\Program Files\\Moly\\moly-native-host.exe")
            manifest_dir = Path.home() / "AppData" / "Local" / "Google" / "Chrome" / "User Data" / "NativeMessagingHosts"
        else:
            return {"success": False, "error": "Unsupported platform"}

        # Create directories
        manifest_dir.mkdir(parents=True, exist_ok=True)
        if os_name != "Windows":
            install_path.parent.mkdir(parents=True, exist_ok=True)
        else:
            install_path.parent.mkdir(parents=True, exist_ok=True)

        # Copy binary
        if os_name == "Windows":
            current_binary_path = sys.executable
        else:
            current_binary_path = sys.argv[0] if sys.argv[0] != "-c" else Path(__file__)

        try:
            shutil.copy2(current_binary_path, install_path)
            if os_name != "Windows":
                os.chmod(install_path, 0o755)
        except Exception as e:
            return {"success": False, "error": f"Failed to copy binary: {str(e)}"}

        # Create native messaging manifest
        manifest = {
            "name": "com.moly.native_host",
            "description": "Moly Native Messaging Host",
            "path": str(install_path),
            "type": "stdio",
            "allowed_origins": [
                f"chrome-extension://{extension_id}/",
            ]
        }

        manifest_file = manifest_dir / "com.moly.native_host.json"
        try:
            with open(manifest_file, 'w') as f:
                json.dump(manifest, f, indent=2)
            if os_name != "Windows":
                os.chmod(manifest_file, 0o644)
        except Exception as e:
            return {"success": False, "error": f"Failed to write manifest: {str(e)}"}

        return {
            "success": True,
            "message": "Native host installed successfully",
            "install_path": str(install_path),
            "manifest_path": str(manifest_file)
        }

    except Exception as e:
        return {"success": False, "error": f"Installation failed: {str(e)}"}


def setup_autostart():
    """Configure auto-start for services"""
    try:
        os_name = platform.system()

        if os_name == "Darwin":  # macOS
            # Create LaunchAgent for CORS proxy
            plist_path = Path.home() / "Library" / "LaunchAgents" / "com.moly.proxy.plist"
            plist_content = f"""<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.moly.proxy</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/moly-proxy</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>StandardOutPath</key>
    <string>{Path.home()}/.moly/proxy.log</string>
    <key>StandardErrorPath</key>
    <string>{Path.home()}/.moly/proxy-error.log</string>
</dict>
</plist>"""
            plist_path.parent.mkdir(parents=True, exist_ok=True)
            with open(plist_path, 'w') as f:
                f.write(plist_content)
            os.chmod(plist_path, 0o644)

        elif os_name == "Linux":
            # Create systemd service for CORS proxy
            service_path = Path("/etc/systemd/user/moly-proxy.service")
            service_content = """[Unit]
Description=Moly CORS Proxy
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/moly-proxy
Restart=on-failure
RestartSec=5s
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=default.target
"""
            try:
                # Try to write to system location (requires sudo)
                with open(service_path, 'w') as f:
                    f.write(service_content)
                os.chmod(service_path, 0o644)
            except:
                # Fall back to user location
                user_service_dir = Path.home() / ".config" / "systemd" / "user"
                user_service_dir.mkdir(parents=True, exist_ok=True)
                user_service_path = user_service_dir / "moly-proxy.service"
                with open(user_service_path, 'w') as f:
                    f.write(service_content)
                os.chmod(user_service_path, 0o644)

        elif os_name == "Windows":
            # Create scheduled task for CORS proxy
            task_xml = """<?xml version="1.0" encoding="UTF-16"?>
<Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo>
    <Date>2026-09-02T00:00:00</Date>
    <Author>Moly</Author>
    <Description>Moly CORS Proxy</Description>
  </RegistrationInfo>
  <Triggers>
    <BootTrigger>
      <Enabled>true</Enabled>
    </BootTrigger>
  </Triggers>
  <Principals>
    <Principal id="Author">
      <UserId>S-1-5-21-0-0-0-1001</UserId>
      <LogonType>InteractiveToken</LogonType>
      <RunLevel>LeastPrivilege</RunLevel>
    </Principal>
  </Principals>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>
    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>
    <AllowHardTerminate>true</AllowHardTerminate>
    <StartWhenAvailable>false</StartWhenAvailable>
    <RunOnlyIfNetworkAvailable>false</RunOnlyIfNetworkAvailable>
    <IdleSettings>
      <Duration>PT10M</Duration>
      <WaitTimeout>PT1H</WaitTimeout>
      <StopOnIdleEnd>false</StopOnIdleEnd>
      <RestartOnIdle>false</RestartOnIdle>
    </IdleSettings>
    <AllowStartOnDemand>true</AllowStartOnDemand>
    <Enabled>true</Enabled>
    <Hidden>false</Hidden>
    <RunOnlyIfIdle>false</RunOnlyIfIdle>
    <DisallowStartOnRemoteAppSession>false</DisallowStartOnRemoteAppSession>
    <UseUnifiedSchedulingEngine>true</UseUnifiedSchedulingEngine>
    <WakeToRun>false</WakeToRun>
    <ExecutionTimeLimit>PT0S</ExecutionTimeLimit>
    <Priority>7</Priority>
  </Settings>
  <Actions Context="Author">
    <Exec>
      <Command>C:\\Program Files\\Moly\\moly-proxy.exe</Command>
    </Exec>
  </Actions>
</Task>"""
            try:
                task_path = Path("C:\\Windows\\Tasks\\MolyProxy.xml")
                with open(task_path, 'w') as f:
                    f.write(task_xml)
            except:
                pass  # Will fail without admin, that's OK

        return {"success": True, "message": "Auto-start configured"}

    except Exception as e:
        return {"success": False, "error": f"Auto-start setup failed: {str(e)}"}


def get_system_info():
    """Get system information"""
    try:
        return {
            "platform": platform.system(),
            "platform_version": platform.release(),
            "machine": platform.machine(),
            "processor": platform.processor(),
            "python_version": platform.python_version(),
        }
    except Exception as e:
        return {"error": str(e)}


def handle_message(request):
    """Handle incoming message from Chrome extension"""
    try:
        action = request.get("action")

        if action == "ping":
            return {"pong": True}

        elif action == "install":
            extension_id = request.get("extension_id")
            if not extension_id:
                return {"success": False, "error": "extension_id required"}
            return install_native_host(extension_id)

        elif action == "setup-autostart":
            return setup_autostart()

        elif action == "launch":
            return launch_installer()

        elif action == "check-ollama":
            return check_ollama()

        elif action == "start-ollama":
            return start_ollama()

        elif action == "stop-ollama":
            return stop_ollama()

        elif action == "system-info":
            return get_system_info()

        elif action == "get-installer-path":
            path = get_installer_path()
            return {
                "path": path,
                "exists": path is not None
            }

        else:
            return {
                "success": False,
                "error": f"Unknown action: {action}"
            }

    except Exception as e:
        return {
            "success": False,
            "error": f"Error handling request: {str(e)}"
        }


def main():
    """Main native messaging loop"""
    # Set binary mode for stdin/stdout
    if sys.platform == "win32":
        import msvcrt
        import os
        msvcrt.setmode(sys.stdin.fileno(), os.O_BINARY)
        msvcrt.setmode(sys.stdout.fileno(), os.O_BINARY)

    while True:
        try:
            # Read message length (4 bytes, little-endian)
            message_length_bytes = sys.stdin.buffer.read(4)
            if len(message_length_bytes) == 0:
                break

            message_length = int.from_bytes(message_length_bytes, "little")

            # Read message data
            message_data = sys.stdin.buffer.read(message_length).decode("utf-8")
            request = json.loads(message_data)

            # Handle the request
            response = handle_message(request)

            # Send response
            response_json = json.dumps(response)
            response_bytes = response_json.encode("utf-8")

            # Send length + response
            sys.stdout.buffer.write(len(response_bytes).to_bytes(4, "little"))
            sys.stdout.buffer.write(response_bytes)
            sys.stdout.buffer.flush()

        except EOFError:
            break
        except Exception as e:
            # Send error response
            error_response = json.dumps({"error": str(e)})
            error_bytes = error_response.encode("utf-8")
            try:
                sys.stdout.buffer.write(len(error_bytes).to_bytes(4, "little"))
                sys.stdout.buffer.write(error_bytes)
                sys.stdout.buffer.flush()
            except:
                break


if __name__ == "__main__":
    main()
