const fs = require('fs');
const path = require('path');
const os = require('os');
const { execSync } = require('child_process');
const https = require('https');

const PLATFORM = process.platform;
const HOME = os.homedir();
const INSTALL_DIR = path.join(HOME, '.local', 'bin');
const DATA_DIR = path.join(HOME, '.local', 'share', 'moly');

class MolyInstaller {
  constructor() {
    this.version = '1.3.0';
    this.repoUrl = 'https://github.com/Nireus79/Moly';
    this.releaseUrl = `${this.repoUrl}/releases/download/v${this.version}`;
  }

  async setup() {
    console.log('[Moly Installer] Starting setup...');

    // Create directories
    this.ensureDirectories();

    // Download native host
    await this.downloadNativeHost();

    // Install to system location
    this.installBinary();

    // Setup system services
    this.setupServices();

    console.log('[Moly Installer] Setup complete');
    return true;
  }

  ensureDirectories() {
    [INSTALL_DIR, DATA_DIR].forEach(dir => {
      if (!fs.existsSync(dir)) {
        try {
          fs.mkdirSync(dir, { recursive: true, mode: 0o755 });
        } catch (e) {
          console.log(`[Moly Installer] Could not create directory ${dir}: ${e.message}`);
        }
      } else {
        // Fix permissions if directory already exists
        try {
          fs.chmodSync(dir, 0o755);
          // Recursively fix permissions on contents if possible
          try {
            fs.readdirSync(dir).forEach(file => {
              try {
                fs.chmodSync(path.join(dir, file), 0o755);
              } catch (e) {
                // Silently ignore individual file permission errors
              }
            });
          } catch (e) {
            // Silently ignore if we can't read directory
          }
        } catch (e) {
          console.log(`[Moly Installer] Could not fix permissions on ${dir}: ${e.message}`);
        }
      }
    });
  }

  async downloadNativeHost() {
    console.log('[Moly Installer] Locating native host binary...');

    // Check multiple possible locations for bundled binary
    const possiblePaths = [
      // Development: in public/resources
      path.join(process.cwd(), 'public', 'resources', 'moly-native-host'),
      // Build output: in build/resources
      path.join(process.cwd(), 'build', 'resources', 'moly-native-host'),
      // Production: inside asar archive
      path.join(__dirname, '..', 'resources', 'moly-native-host'),
      // Electron resource path
      path.join(process.resourcesPath, 'moly-native-host'),
    ];

    let bundledPath = null;
    for (const checkPath of possiblePaths) {
      if (fs.existsSync(checkPath)) {
        bundledPath = checkPath;
        console.log(`[Moly Installer] Found bundled binary at ${checkPath}`);
        break;
      }
    }

    if (bundledPath) {
      console.log('[Moly Installer] Installing bundled binary to ~/.local/bin...');
      const installPath = path.join(INSTALL_DIR, 'moly-native-host');

      try {
        // Ensure install directory exists
        if (!fs.existsSync(INSTALL_DIR)) {
          fs.mkdirSync(INSTALL_DIR, { recursive: true, mode: 0o755 });
        }

        // Remove old binary if it exists
        if (fs.existsSync(installPath)) {
          fs.unlinkSync(installPath);
        }

        // Copy binary directly to install location
        fs.copyFileSync(bundledPath, installPath);
        fs.chmodSync(installPath, 0o755);
        console.log('[Moly Installer] Binary installed successfully');
        return installPath;
      } catch (err) {
        console.log(`[Moly Installer] Installation failed: ${err.message}`);
        throw err;
      }
    }

    // Fall back to GitHub release download
    console.log('[Moly Installer] Bundled binary not found, attempting GitHub download...');
    const url = `${this.releaseUrl}/moly-native-host-linux-x64`;
    const installPath = path.join(INSTALL_DIR, 'moly-native-host');

    return new Promise((resolve, reject) => {
      https.get(url, (response) => {
        if (response.statusCode !== 200) {
          reject(new Error(`GitHub release not available (${response.statusCode}). Please use Claude/OpenAI APIs instead.`));
          return;
        }

        const file = fs.createWriteStream(installPath);
        response.pipe(file);

        file.on('finish', () => {
          file.close();
          fs.chmodSync(installPath, 0o755);
          console.log('[Moly Installer] Download complete');
          resolve(installPath);
        });

        file.on('error', reject);
      }).on('error', reject);
    });
  }

  installBinary() {
    // Binary is now installed directly during downloadNativeHost()
    // This method is kept for compatibility but does nothing
  }

  setupServices() {
    console.log('[Moly Installer] Setting up system services...');

    if (PLATFORM === 'linux') {
      this.setupLinuxService();
    } else if (PLATFORM === 'darwin') {
      this.setupMacService();
    } else if (PLATFORM === 'win32') {
      this.setupWindowsService();
    }

    console.log('[Moly Installer] Services configured');
  }

  setupLinuxService() {
    const systemdDir = path.join(HOME, '.config', 'systemd', 'user');
    const servicePath = path.join(systemdDir, 'moly-native-host.service');

    if (!fs.existsSync(systemdDir)) {
      fs.mkdirSync(systemdDir, { recursive: true });
    }

    const serviceContent = `[Unit]
Description=Moly Native Host CORS Proxy
After=network.target
PartOf=graphical-session.target

[Service]
Type=simple
ExecStart=${path.join(INSTALL_DIR, 'moly-native-host')} --proxy-mode
Restart=on-failure
RestartSec=5

[Install]
WantedBy=graphical-session.target
`;

    fs.writeFileSync(servicePath, serviceContent);

    try {
      execSync('systemctl --user daemon-reload', { stdio: 'ignore' });
      execSync('systemctl --user enable moly-native-host.service', { stdio: 'ignore' });
      execSync('systemctl --user start moly-native-host.service', { stdio: 'ignore' });
    } catch (e) {
      console.warn('[Moly Installer] Systemd setup failed, will start manually');
    }
  }

  setupMacService() {
    const launchAgentDir = path.join(HOME, 'Library/LaunchAgents');
    const plistPath = path.join(launchAgentDir, 'com.moly.native-host.plist');

    if (!fs.existsSync(launchAgentDir)) {
      fs.mkdirSync(launchAgentDir, { recursive: true });
    }

    const plistContent = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.moly.native-host</string>
    <key>ProgramArguments</key>
    <array>
        <string>${path.join(INSTALL_DIR, 'moly-native-host')}</string>
        <string>--proxy-mode</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
`;

    fs.writeFileSync(plistPath, plistContent);

    try {
      execSync(`launchctl load "${plistPath}"`, { stdio: 'ignore' });
    } catch (e) {
      console.warn('[Moly Installer] LaunchAgent setup failed');
    }
  }

  setupWindowsService() {
    // Windows: Use Task Scheduler via PowerShell
    const taskName = 'MolyNativeHost';
    const binaryPath = path.join(INSTALL_DIR, 'moly-native-host.exe');

    const psCommand = `
$action = New-ScheduledTaskAction -Execute "${binaryPath}" -Argument "--proxy-mode"
$trigger = New-ScheduledTaskTrigger -AtLogOn
$principal = New-ScheduledTaskPrincipal -UserId "$env:USERNAME" -LogonType Interactive
$task = New-ScheduledTask -Action $action -Trigger $trigger -Principal $principal
Register-ScheduledTask -TaskName "${taskName}" -InputObject $task -Force
Start-ScheduledTask -TaskName "${taskName}"
`;

    try {
      execSync(`powershell -Command "${psCommand}"`, { stdio: 'ignore' });
    } catch (e) {
      console.warn('[Moly Installer] Windows Task Scheduler setup failed');
    }
  }

  getBinaryName() {
    if (PLATFORM === 'linux') {
      return 'moly-native-host-linux-x64.tar.gz';
    } else if (PLATFORM === 'darwin') {
      const arch = os.arch() === 'arm64' ? 'arm64' : 'x64';
      return `moly-native-host-macos-${arch}.tar.gz`;
    } else if (PLATFORM === 'win32') {
      return 'moly-native-host-windows-x64.zip';
    }
    throw new Error('Unsupported platform');
  }
}

module.exports = MolyInstaller;
