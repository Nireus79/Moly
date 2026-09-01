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

        elif action == "launch":
            return launch_installer()

        elif action == "check-ollama":
            return check_ollama()

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
