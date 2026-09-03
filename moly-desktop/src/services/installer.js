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
        fs.mkdirSync(dir, { recursive: true });
      }
    });
  }

  async downloadNativeHost() {
    console.log('[Moly Installer] Downloading native host...');

    const binaryName = this.getBinaryName();
    const url = `${this.releaseUrl}/${binaryName}`;
    const tempPath = path.join(DATA_DIR, binaryName);

    return new Promise((resolve, reject) => {
      https.get(url, (response) => {
        if (response.statusCode !== 200) {
          reject(new Error(`Failed to download: ${response.statusCode}`));
          return;
        }

        const file = fs.createWriteStream(tempPath);
        response.pipe(file);

        file.on('finish', () => {
          file.close();
          console.log('[Moly Installer] Download complete');
          resolve(tempPath);
        });

        file.on('error', reject);
      }).on('error', reject);
    });
  }

  installBinary() {
    console.log('[Moly Installer] Installing binary...');

    const binaryName = this.getBinaryName();
    const tempPath = path.join(DATA_DIR, binaryName);
    const installPath = path.join(INSTALL_DIR, 'moly-native-host');

    if (!fs.existsSync(tempPath)) {
      throw new Error('Binary not found after download');
    }

    // Make executable
    fs.chmodSync(tempPath, 0o755);

    // Copy to install directory
    fs.copyFileSync(tempPath, installPath);
    fs.chmodSync(installPath, 0o755);

    console.log(`[Moly Installer] Installed to ${installPath}`);
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
