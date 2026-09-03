#!/usr/bin/env python3
"""
Moly Native Messaging Host
Enables Chrome extension to launch installer and control local services
Includes built-in CORS Proxy for Ollama integration
"""

import sys
import json
import subprocess
import platform
import os
from pathlib import Path
import threading
import time
import http.server
import socketserver
import urllib.request
import urllib.error


# CORS Proxy Server
class CORSProxyHandler(http.server.BaseHTTPRequestHandler):
    """HTTP handler that strips browser security headers and forwards to Ollama"""

    def do_GET(self):
        self._handle_request()

    def do_POST(self):
        self._handle_request()

    def do_PUT(self):
        self._handle_request()

    def do_DELETE(self):
        self._handle_request()

    def do_HEAD(self):
        self._handle_request()

    def do_OPTIONS(self):
        self.send_response(200)
        self.send_header('Access-Control-Allow-Origin', '*')
        self.send_header('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, OPTIONS, HEAD')
        self.send_header('Access-Control-Allow-Headers', 'Content-Type, Accept')
        self.send_header('Access-Control-Max-Age', '3600')
        self.end_headers()

    def _handle_request(self):
        """Forward request to Ollama, stripping problematic browser headers"""
        # Add CORS headers
        self.send_response(200)
        self.send_header('Access-Control-Allow-Origin', '*')
        self.send_header('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, OPTIONS, HEAD')
        self.send_header('Access-Control-Allow-Headers', 'Content-Type, Accept')

        # Build request to Ollama
        headers = dict(self.headers)

        # Strip problematic browser security headers
        headers_to_remove = [
            'host', 'origin', 'sec-fetch-site', 'sec-fetch-mode', 'sec-fetch-dest',
            'sec-fetch-storage-access', 'sec-ch-ua', 'sec-ch-ua-platform',
            'sec-ch-ua-mobile', 'sec-gpc'
        ]
        for header in headers_to_remove:
            headers.pop(header.lower(), None)

        # Prepare Ollama request
        url = f'http://127.0.0.1:11434{self.path}'

        try:
            req = urllib.request.Request(url, method=self.command, headers=headers)

            # Read body if present
            content_length = self.headers.get('Content-Length')
            if content_length:
                body = self.rfile.read(int(content_length))
                req.data = body

            # Forward to Ollama
            response = urllib.request.urlopen(req, timeout=30)

            # Send response headers
            self.send_response(response.status)
            for header, value in response.headers.items():
                if header.lower() not in ['connection', 'transfer-encoding']:
                    self.send_header(header, value)
            self.end_headers()

            # Send response body
            self.wfile.write(response.read())

        except urllib.error.HTTPError as e:
            self.send_response(e.code)
            # Forward headers from error response
            for header, value in e.headers.items():
                if header.lower() not in ['connection', 'transfer-encoding']:
                    self.send_header(header, value)
            # If no content-type, default to JSON
            if 'content-type' not in {h.lower(): v for h, v in e.headers.items()}:
                self.send_header('Content-Type', 'application/json')
            self.end_headers()
            # Forward error response body if available
            try:
                error_body = e.read()
                self.wfile.write(error_body)
            except:
                # Fallback to plain text error
                self.wfile.write(f'{{"error": "{e.reason}"}}'.encode())
        except Exception as e:
            self.send_response(502)
            self.send_header('Content-Type', 'text/plain')
            self.end_headers()
            self.wfile.write(f'Bad Gateway: {str(e)}'.encode())

    def log_message(self, format, *args):
        """Suppress logging"""
        pass


def start_cors_proxy_server():
    """Start CORS proxy server in background"""
    try:
        server = socketserver.TCPServer(('127.0.0.1', 11435), CORSProxyHandler)
        server.allow_reuse_address = True

        # Run in daemon thread
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()

        return True
    except Exception as e:
        return False


def check_cors_proxy_running():
    """Check if CORS proxy is running"""
    try:
        response = urllib.request.urlopen('http://localhost:11435/api/tags', timeout=2)
        return response.status == 200
    except:
        return False


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
    """Start Ollama service (handles snap, systemd, and manual installations)"""
    try:
        import time
        import urllib.request

        os_name = platform.system()

        if os_name == "Darwin":
            # macOS: start the app
            subprocess.Popen(
                ["/Applications/Ollama.app/Contents/MacOS/ollama", "serve"],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL
            )
            # Wait and verify
            for _ in range(10):
                time.sleep(1)
                try:
                    urllib.request.urlopen('http://localhost:11434/api/tags', timeout=1)
                    return {"success": True, "message": "Ollama started and ready"}
                except:
                    pass
            return {"success": False, "error": "Ollama started but not responding"}

        elif os_name == "Linux":
            # Try snap first
            snap_result = subprocess.run(["snap", "start", "ollama"], capture_output=True)
            if snap_result.returncode == 0:
                # Wait and verify
                for _ in range(10):
                    time.sleep(1)
                    try:
                        urllib.request.urlopen('http://localhost:11434/api/tags', timeout=1)
                        return {"success": True, "message": "Ollama (snap) started and ready"}
                    except:
                        pass
                return {"success": False, "error": "Ollama snap started but not responding"}

            # Try systemd
            systemd_result = subprocess.run(["systemctl", "start", "ollama"], capture_output=True)
            if systemd_result.returncode == 0:
                # Wait and verify
                for _ in range(10):
                    time.sleep(1)
                    try:
                        urllib.request.urlopen('http://localhost:11434/api/tags', timeout=1)
                        return {"success": True, "message": "Ollama (systemd) started and ready"}
                    except:
                        pass
                return {"success": False, "error": "Ollama systemd started but not responding"}

            # Try manual start
            try:
                subprocess.Popen(
                    ["ollama", "serve"],
                    stdout=subprocess.DEVNULL,
                    stderr=subprocess.DEVNULL
                )
                # Wait and verify
                for _ in range(10):
                    time.sleep(1)
                    try:
                        urllib.request.urlopen('http://localhost:11434/api/tags', timeout=1)
                        return {"success": True, "message": "Ollama started and ready"}
                    except:
                        pass
                return {"success": False, "error": "Ollama started but not responding"}
            except FileNotFoundError:
                return {"success": False, "error": "Ollama not found in PATH. Install from https://ollama.ai"}

        elif os_name == "Windows":
            user = os.getenv("USERNAME")
            ollama_path = f"C:\\Users\\{user}\\AppData\\Local\\Programs\\Ollama\\ollama.exe"
            subprocess.Popen(
                [ollama_path, "serve"],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL
            )
            # Wait and verify
            for _ in range(10):
                time.sleep(1)
                try:
                    urllib.request.urlopen('http://localhost:11434/api/tags', timeout=1)
                    return {"success": True, "message": "Ollama started and ready"}
                except:
                    pass
            return {"success": False, "error": "Ollama started but not responding"}

        else:
            return {"success": False, "error": "Unsupported platform"}

    except Exception as e:
        return {"success": False, "error": f"Error starting Ollama: {str(e)}"}


def stop_ollama():
    """Stop Ollama service (handles snap, systemd, and manual installations)"""
    try:
        import time
        import urllib.request

        os_name = platform.system()

        if os_name == "Darwin":
            # macOS: kill the process
            subprocess.run(["pkill", "-f", "ollama serve"], capture_output=True)
            time.sleep(1)
            # Verify it stopped
            try:
                urllib.request.urlopen('http://localhost:11434/api/tags', timeout=1)
                return {"success": False, "error": "Failed to stop Ollama"}
            except:
                return {"success": True, "message": "Ollama stopped"}

        elif os_name == "Linux":
            # Try snap first
            snap_result = subprocess.run(["snap", "stop", "ollama"], capture_output=True)
            if snap_result.returncode == 0:
                time.sleep(1)
                return {"success": True, "message": "Ollama (snap) stopped"}

            # Try systemd
            systemd_result = subprocess.run(["systemctl", "stop", "ollama"], capture_output=True)
            if systemd_result.returncode == 0:
                time.sleep(1)
                return {"success": True, "message": "Ollama (systemd) stopped"}

            # Try pkill for manual installations
            subprocess.run(["pkill", "-f", "ollama serve"], capture_output=True)
            time.sleep(1)

            # Verify actually stopped
            try:
                urllib.request.urlopen('http://localhost:11434/api/tags', timeout=1)
                return {"success": False, "error": "Ollama still running. Try: pkill -f 'ollama serve'"}
            except:
                return {"success": True, "message": "Ollama stopped"}

        elif os_name == "Windows":
            # Windows: kill the process
            subprocess.run(["taskkill", "/IM", "ollama.exe", "/F"], capture_output=True)
            time.sleep(1)
            # Verify it stopped
            try:
                urllib.request.urlopen('http://localhost:11434/api/tags', timeout=1)
                return {"success": False, "error": "Failed to stop Ollama"}
            except:
                return {"success": True, "message": "Ollama stopped"}

        else:
            return {"success": False, "error": "Unsupported platform"}

    except Exception as e:
        return {"success": False, "error": f"Error stopping Ollama: {str(e)}"}


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


def pull_model(model_name):
    """Pull a model from Ollama"""
    try:
        result = subprocess.run(
            ["ollama", "pull", model_name],
            capture_output=True,
            timeout=3600  # 1 hour timeout for model download
        )

        if result.returncode == 0:
            return {
                "success": True,
                "message": f"Model {model_name} pulled successfully"
            }
        else:
            return {
                "success": False,
                "error": result.stderr.decode("utf-8", errors="ignore") or "Failed to pull model"
            }

    except subprocess.TimeoutExpired:
        return {
            "success": False,
            "error": "Download timeout after 1 hour"
        }
    except FileNotFoundError:
        return {
            "success": False,
            "error": "Ollama not found in PATH"
        }
    except Exception as e:
        return {
            "success": False,
            "error": f"Failed to pull model: {str(e)}"
        }


def install_cors_proxy():
    """Start the built-in CORS proxy"""
    try:
        # Check if already running
        if check_cors_proxy_running():
            return {
                "success": True,
                "message": "CORS proxy already running"
            }

        # Start the proxy
        if start_cors_proxy_server():
            # Verify it started
            time.sleep(0.5)
            if check_cors_proxy_running():
                return {
                    "success": True,
                    "message": "CORS proxy started successfully"
                }
            else:
                return {
                    "success": False,
                    "error": "CORS proxy failed to start"
                }
        else:
            return {
                "success": False,
                "error": "Failed to start CORS proxy"
            }

    except Exception as e:
        return {
            "success": False,
            "error": f"Failed to start CORS proxy: {str(e)}"
        }


def setup_autostart():
    """Configure auto-start for native host (includes built-in CORS proxy)"""
    try:
        os_name = platform.system()

        if os_name == "Darwin":  # macOS
            # Create LaunchAgent for native host (which includes proxy)
            plist_path = Path.home() / "Library" / "LaunchAgents" / "com.moly.native_host.plist"
            plist_content = f"""<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.moly.native_host</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/moly-native-host</string>
        <string>--proxy-mode</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>StandardOutPath</key>
    <string>{Path.home()}/.moly/native_host.log</string>
    <key>StandardErrorPath</key>
    <string>{Path.home()}/.moly/native_host_error.log</string>
</dict>
</plist>"""
            plist_path.parent.mkdir(parents=True, exist_ok=True)
            with open(plist_path, 'w') as f:
                f.write(plist_content)
            os.chmod(plist_path, 0o644)

        elif os_name == "Linux":
            # Create systemd service for native host (which includes proxy)
            service_path = Path("/etc/systemd/user/moly-native-host.service")
            service_content = """[Unit]
Description=Moly Native Host (CORS Proxy)
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/moly-native-host --proxy-mode
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
                user_service_path = user_service_dir / "moly-native-host.service"
                with open(user_service_path, 'w') as f:
                    f.write(service_content)
                os.chmod(user_service_path, 0o644)

        elif os_name == "Windows":
            # Create scheduled task for native host (which includes proxy)
            user = os.getenv("USERNAME")
            task_xml = f"""<?xml version="1.0" encoding="UTF-16"?>
<Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo>
    <Date>2026-09-03T00:00:00</Date>
    <Author>Moly</Author>
    <Description>Moly Native Host (CORS Proxy)</Description>
  </RegistrationInfo>
  <Triggers>
    <LogonTrigger>
      <Enabled>true</Enabled>
      <UserId>{user}</UserId>
    </LogonTrigger>
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
    <StartWhenAvailable>true</StartWhenAvailable>
    <RunOnlyIfNetworkAvailable>false</RunOnlyIfNetworkAvailable>
    <AllowStartOnDemand>true</AllowStartOnDemand>
    <Enabled>true</Enabled>
    <Hidden>false</Hidden>
    <ExecutionTimeLimit>PT0S</ExecutionTimeLimit>
  </Settings>
  <Actions Context="Author">
    <Exec>
      <Command>C:\\Program Files\\Moly\\moly-native-host.exe</Command>
      <Arguments>--proxy-mode</Arguments>
    </Exec>
  </Actions>
</Task>"""
            try:
                task_path = Path(f"C:\\Windows\\Tasks\\MolyNativeHost.xml")
                with open(task_path, 'w') as f:
                    f.write(task_xml)
            except:
                pass  # Will fail without admin, that's OK

        return {"success": True, "message": "Auto-start configured for native host with proxy"}

    except Exception as e:
        return {"success": False, "error": f"Auto-start setup failed: {str(e)}"}


def get_installed_models():
    """Query Ollama for all installed models"""
    try:
        result = subprocess.run(
            ["ollama", "list"],
            capture_output=True,
            timeout=10
        )

        if result.returncode != 0:
            return {
                "success": False,
                "error": "Failed to query Ollama"
            }

        # Parse output: each line is "name\tdigests"
        models = []
        output = result.stdout.decode("utf-8", errors="ignore").strip()
        for line in output.split("\n")[1:]:  # Skip header
            if line.strip():
                parts = line.split("\t")
                if len(parts) >= 1:
                    models.append({"name": parts[0].strip()})

        return {
            "success": True,
            "models": models
        }

    except FileNotFoundError:
        return {
            "success": False,
            "error": "Ollama not found in PATH"
        }
    except Exception as e:
        return {
            "success": False,
            "error": f"Failed to get models: {str(e)}"
        }


def remove_model(model_name):
    """Remove a model from Ollama"""
    try:
        result = subprocess.run(
            ["ollama", "rm", model_name],
            capture_output=True,
            timeout=30
        )

        if result.returncode == 0:
            return {
                "success": True,
                "message": f"Model {model_name} removed"
            }
        else:
            return {
                "success": False,
                "error": result.stderr.decode("utf-8", errors="ignore") or "Failed to remove model"
            }

    except FileNotFoundError:
        return {
            "success": False,
            "error": "Ollama not found in PATH"
        }
    except Exception as e:
        return {
            "success": False,
            "error": f"Failed to remove model: {str(e)}"
        }


def cleanup_moly(keep_models=True):
    """Clean up Moly data folder on uninstall"""
    try:
        import shutil

        # Remove Moly data folder
        moly_data = Path.home() / ".local" / "share" / "moly"
        if moly_data.exists():
            shutil.rmtree(moly_data)

        # Remove native host binary
        native_host_path = Path("/usr/local/bin/moly-native-host")
        if native_host_path.exists():
            try:
                native_host_path.unlink()
            except:
                pass

        return {
            "success": True,
            "message": "Moly cleaned up",
            "models_kept": keep_models
        }

    except Exception as e:
        return {
            "success": False,
            "error": f"Cleanup failed: {str(e)}"
        }


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


def setup_all():
    """Orchestrate complete Moly setup - install all missing components"""
    try:
        steps_completed = []
        errors = []

        # Step 1: Ensure native host is installed
        extension_id = "nonheafhmdhjpbggfpdhjeoanofnkijc"  # Moly extension ID
        try:
            result = install_native_host(extension_id)
            if result.get("success"):
                steps_completed.append("native-host")
            else:
                errors.append(f"Native host: {result.get('error', 'unknown error')}")
        except Exception as e:
            errors.append(f"Native host: {str(e)}")

        # Step 2: Start Ollama if not running
        ollama_status = check_ollama()
        if not ollama_status.get("running"):
            try:
                result = start_ollama()
                if result.get("success"):
                    steps_completed.append("ollama")
                else:
                    errors.append(f"Ollama: {result.get('error', 'unknown error')}")
            except Exception as e:
                errors.append(f"Ollama: {str(e)}")
        else:
            steps_completed.append("ollama")

        # Step 3: Install CORS proxy
        try:
            result = install_cors_proxy()
            if result.get("success"):
                steps_completed.append("cors-proxy")
            else:
                # CORS proxy install may fail if npm not installed, but that's okay
                # User can install Node.js manually
                pass
        except Exception as e:
            pass

        # Step 4: Setup autostart services
        try:
            result = setup_autostart()
            if result.get("success"):
                steps_completed.append("autostart")
        except Exception as e:
            pass

        return {
            "success": len(errors) == 0,
            "completed": steps_completed,
            "errors": errors,
            "message": f"Setup complete. Installed: {', '.join(steps_completed)}" if steps_completed else "Setup failed. See errors."
        }

    except Exception as e:
        return {
            "success": False,
            "error": f"Setup orchestration failed: {str(e)}"
        }


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

        elif action == "setup-all":
            return setup_all()

        elif action == "pull-model":
            model = request.get("model")
            if not model:
                return {"success": False, "error": "model parameter required"}
            return pull_model(model)

        elif action == "install-cors-proxy":
            return install_cors_proxy()

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

        elif action == "get-models":
            return get_installed_models()

        elif action == "remove-model":
            model = request.get("model")
            if not model:
                return {"success": False, "error": "model parameter required"}
            return remove_model(model)

        elif action == "cleanup":
            keep_models = request.get("keep_models", True)
            return cleanup_moly(keep_models)

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


def proxy_mode():
    """Run ONLY the CORS proxy server (for autostart)"""
    try:
        if start_cors_proxy_server():
            # Keep running forever
            while True:
                time.sleep(1)
        else:
            sys.exit(1)
    except KeyboardInterrupt:
        pass
    except Exception as e:
        sys.exit(1)


def main():
    """Main native messaging loop"""
    # Check for proxy-only mode
    if len(sys.argv) > 1 and sys.argv[1] == "--proxy-mode":
        proxy_mode()
        return

    # Start CORS proxy in background
    start_cors_proxy_server()

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
