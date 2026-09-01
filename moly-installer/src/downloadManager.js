import fetch from 'node-fetch';
import { createWriteStream, promises as fs } from 'fs';
import { pipeline } from 'stream/promises';
import { dirname } from 'path';
import chalk from 'chalk';
import cliProgress from 'cli-progress';
import os from 'os';

export class DownloadManager {
  constructor() {
    this.downloadDir = `${os.homedir()}/.moly/downloads`;
    this.progressBar = null;
  }

  async ensureDownloadDir() {
    try {
      await fs.mkdir(this.downloadDir, { recursive: true });
    } catch (error) {
      console.error(chalk.red(`Failed to create download directory: ${error.message}`));
      throw error;
    }
  }

  async downloadFile(url, filename, description) {
    console.log(chalk.dim(`\nDownloading: ${description}`));
    console.log(chalk.dim(`From: ${url}\n`));

    const filePath = `${this.downloadDir}/${filename}`;

    try {
      const response = await fetch(url, { redirect: 'follow' });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const contentLength = parseInt(response.headers.get('content-length'), 10);

      this.progressBar = new cliProgress.SingleBar({
        format: 'Progress |{bar}| {percentage}% || {value}/{total} bytes',
        barCompleteChar: '█',
        barIncompleteChar: '░',
        hideCursor: true,
      });

      this.progressBar.start(contentLength, 0);

      let downloadedBytes = 0;

      // Create write stream with progress tracking
      const writeStream = createWriteStream(filePath);

      response.body.on('data', chunk => {
        downloadedBytes += chunk.length;
        this.progressBar.update(downloadedBytes);
      });

      await pipeline(response.body, writeStream);

      this.progressBar.stop();
      console.log(chalk.green(`✓ Downloaded: ${filename}`));

      return filePath;
    } catch (error) {
      if (this.progressBar) {
        this.progressBar.stop();
      }
      console.error(chalk.red(`✗ Download failed: ${error.message}`));
      throw error;
    }
  }

  async downloadOllamaInstaller() {
    await this.ensureDownloadDir();

    const platform = process.platform;
    const arch = process.arch;

    let url, filename;

    if (platform === 'darwin') {
      url = 'https://ollama.ai/download/Ollama-darwin.zip';
      filename = 'Ollama-darwin.zip';
    } else if (platform === 'win32') {
      url = 'https://ollama.ai/download/OllamaSetup.exe';
      filename = 'OllamaSetup.exe';
    } else if (platform === 'linux') {
      url = 'https://ollama.ai/download/ollama-linux-x86_64.tar.gz';
      filename = 'ollama-linux-x86_64.tar.gz';
    } else {
      throw new Error(`Unsupported platform: ${platform}`);
    }

    return await this.downloadFile(url, filename, 'Ollama Installer');
  }

  async downloadLMStudioInstaller() {
    await this.ensureDownloadDir();

    const platform = process.platform;

    let url, filename;

    if (platform === 'darwin') {
      url = 'https://lmstudio.ai/api/download/mac';
      filename = 'LM-Studio-darwin.dmg';
    } else if (platform === 'win32') {
      url = 'https://lmstudio.ai/api/download/windows';
      filename = 'LM-Studio-Setup.exe';
    } else if (platform === 'linux') {
      url = 'https://lmstudio.ai/api/download/linux';
      filename = 'LM-Studio-linux.AppImage';
    } else {
      throw new Error(`Unsupported platform: ${platform}`);
    }

    return await this.downloadFile(url, filename, 'LM Studio Installer');
  }

  async downloadMolyProxy() {
    await this.ensureDownloadDir();

    console.log(chalk.dim('\nInstalling CORS Proxy via npm...'));

    try {
      const { exec } = await import('child_process');
      const { promisify } = await import('util');
      const execAsync = promisify(exec);

      await execAsync('npm install -g moly-proxy');
      console.log(chalk.green('✓ CORS Proxy installed'));

      return true;
    } catch (error) {
      console.error(chalk.red(`✗ Failed to install CORS Proxy: ${error.message}`));
      throw error;
    }
  }

  async downloadNativeHost() {
    await this.ensureDownloadDir();

    const platform = process.platform;
    let url, filename, targetPath;

    if (platform === 'darwin') {
      url = 'https://github.com/user/moly-installer/releases/download/v1.0.0/moly-native-host-macos';
      filename = 'moly-native-host-macos';
      targetPath = '/usr/local/bin/moly-native-host';
    } else if (platform === 'linux') {
      url = 'https://github.com/user/moly-installer/releases/download/v1.0.0/moly-native-host-linux';
      filename = 'moly-native-host-linux';
      targetPath = '/usr/local/bin/moly-native-host';
    } else if (platform === 'win32') {
      url = 'https://github.com/user/moly-installer/releases/download/v1.0.0/moly-native-host-windows.exe';
      filename = 'moly-native-host-windows.exe';
      targetPath = `C:\\Program Files\\Moly\\moly-native-host.exe`;
    } else {
      throw new Error(`Unsupported platform: ${platform}`);
    }

    const filePath = await this.downloadFile(url, filename, 'Native Host Binary');

    try {
      // Copy to system location
      if (platform !== 'win32') {
        const { exec } = await import('child_process');
        const { promisify } = await import('util');
        const execAsync = promisify(exec);

        // Copy and make executable
        await execAsync(`sudo cp ${filePath} ${targetPath}`);
        await execAsync(`sudo chmod +x ${targetPath}`);
      } else {
        // Windows: Copy to Program Files
        const { exec } = await import('child_process');
        const { promisify } = await import('util');
        const execAsync = promisify(exec);

        await fs.mkdir('C:\\Program Files\\Moly', { recursive: true }).catch(() => {});
        await execAsync(`copy "${filePath}" "${targetPath}"`);
      }

      console.log(chalk.green('✓ Native host installed to system path'));
      return targetPath;
    } catch (error) {
      console.warn(
        chalk.yellow(`⚠️  Could not move native host to system path: ${error.message}`)
      );
      console.log(chalk.dim('Service control requires: sudo or admin privileges'));
      throw error;
    }
  }

  async cleanup() {
    try {
      // Optionally clean up downloads after successful installation
      // await fs.rm(this.downloadDir, { recursive: true, force: true });
    } catch (error) {
      console.warn(chalk.yellow(`Warning: Failed to cleanup: ${error.message}`));
    }
  }

  async verifyDownload(filePath) {
    try {
      const stats = await fs.stat(filePath);
      return stats.size > 0;
    } catch {
      return false;
    }
  }
}
