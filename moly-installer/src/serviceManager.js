import chalk from 'chalk';
import { exec } from 'child_process';
import { promisify } from 'util';
import { promises as fs } from 'fs';
import os from 'os';
import path from 'path';

const execAsync = promisify(exec);

export class ServiceManager {
  constructor(provider) {
    this.provider = provider;
    this.platform = process.platform;
    this.username = os.userInfo().username;
    this.homeDir = os.homedir();
  }

  async setupAutoStart() {
    console.log(chalk.dim(`Setting up auto-start for ${this.provider} on ${this.platform}...\n`));

    try {
      if (this.platform === 'linux') {
        await this.setupLinuxAutoStart();
      } else if (this.platform === 'darwin') {
        await this.setupMacOSAutoStart();
      } else if (this.platform === 'win32') {
        await this.setupWindowsAutoStart();
      }

      console.log(chalk.green(`✓ Auto-start configured\n`));
    } catch (error) {
      console.error(chalk.red(`✗ Auto-start setup failed: ${error.message}`));
      console.log(chalk.yellow('ℹ️  You can start services manually later'));
    }
  }

  async setupLinuxAutoStart() {
    if (this.provider === 'ollama') {
      // Ollama typically installs its own systemd service
      console.log(chalk.dim('Note: Ollama will auto-start if installed via official installer'));

      // Try to enable ollama service if it exists
      try {
        await execAsync('systemctl is-active --quiet ollama');
        console.log(chalk.green('✓ Ollama service found'));
        await execAsync('sudo systemctl enable ollama');
      } catch {
        console.log(chalk.yellow('ℹ️  Ollama service not found (may start manually)'));
      }

      // Set up CORS proxy auto-start
      console.log(chalk.dim('Setting up CORS proxy auto-start...'));
      const serviceContent = `[Unit]
Description=Moly CORS Proxy for Ollama
After=network.target ollama.service

[Service]
Type=simple
User=${this.username}
ExecStart=/usr/bin/moly-proxy
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
`;

      const servicePath = '/etc/systemd/system/moly-proxy.service';
      try {
        await fs.writeFile(servicePath, serviceContent);
        await execAsync('sudo systemctl daemon-reload');
        await execAsync('sudo systemctl enable moly-proxy');
        console.log(chalk.green('✓ CORS proxy auto-start enabled'));
      } catch (error) {
        console.log(chalk.yellow(`✗ Could not set up CORS proxy service: ${error.message}`));
      }
    }
  }

  async setupMacOSAutoStart() {
    if (this.provider === 'ollama') {
      // Check if Ollama LaunchAgent exists
      const ollamaLaunchAgent = `${this.homeDir}/Library/LaunchAgents/ai.ollama.launch.plist`;
      try {
        await fs.access(ollamaLaunchAgent);
        console.log(chalk.green('✓ Ollama LaunchAgent found'));
      } catch {
        console.log(chalk.yellow('ℹ️  Ollama LaunchAgent not found'));
      }

      // Set up CORS proxy LaunchAgent
      console.log(chalk.dim('Setting up CORS proxy auto-start...'));
      const plistContent = `<?xml version="1.0" encoding="UTF-8"?>
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
    <key>KeepAlive</key>
    <dict>
        <key>SuccessfulExit</key>
        <false/>
    </dict>
    <key>StandardOutPath</key>
    <string>/var/log/moly-proxy.log</string>
    <key>StandardErrorPath</key>
    <string>/var/log/moly-proxy-error.log</string>
</dict>
</plist>
`;

      const launchAgentDir = `${this.homeDir}/Library/LaunchAgents`;
      const plistPath = `${launchAgentDir}/com.moly.proxy.plist`;

      try {
        await fs.mkdir(launchAgentDir, { recursive: true });
        await fs.writeFile(plistPath, plistContent);
        await execAsync(`launchctl load ${plistPath}`);
        console.log(chalk.green('✓ CORS proxy LaunchAgent configured'));
      } catch (error) {
        console.log(chalk.yellow(`✗ Could not set up LaunchAgent: ${error.message}`));
      }
    }
  }

  async setupWindowsAutoStart() {
    if (this.provider === 'ollama') {
      // Ollama Windows installer should handle its own startup
      console.log(chalk.dim('Note: Ollama Windows installer handles its own startup'));

      // Set up CORS proxy Task Scheduler
      console.log(chalk.dim('Setting up CORS proxy auto-start...'));
      const taskXml = `<?xml version="1.0" encoding="UTF-16"?>
<Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo>
    <Date>$(Get-Date -Format o)</Date>
    <Author>Moly Installer</Author>
    <Description>Moly CORS Proxy for Ollama</Description>
  </RegistrationInfo>
  <Triggers>
    <LogonTrigger>
      <Enabled>true</Enabled>
    </LogonTrigger>
  </Triggers>
  <Principals>
    <Principal id="Author">
      <UserId>${this.username}</UserId>
      <RunLevel>Limited</RunLevel>
    </Principal>
  </Principals>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>
    <RestartCount>3</RestartCount>
    <RestartInterval>PT10M</RestartInterval>
  </Settings>
  <Actions Context="Author">
    <Exec>
      <Command>moly-proxy</Command>
    </Exec>
  </Actions>
</Task>`;

      try {
        // Register task via PowerShell
        const psCommand = `Add-ScheduledTask -TaskName "Moly Proxy" -Trigger (New-ScheduledTaskTrigger -AtLogOn) -Action (New-ScheduledTaskAction -Execute "moly-proxy") -Principal (New-ScheduledTaskPrincipal -UserId "$env:USERNAME" -RunLevel Limited) -Force`;
        await execAsync(`powershell -Command "${psCommand}"`);
        console.log(chalk.green('✓ Task Scheduler auto-start configured'));
      } catch (error) {
        console.log(chalk.yellow(`✗ Could not set up Task Scheduler: ${error.message}`));
      }
    }
  }
}
