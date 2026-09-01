import os from 'os';
import fs from 'fs';
import { exec } from 'child_process';
import { promisify } from 'util';
import chalk from 'chalk';

const execAsync = promisify(exec);

export class SystemRequirements {
  constructor() {
    this.platform = process.platform;
    this.arch = process.arch;
    this.nodeVersion = process.version;
    this.requirements = {
      os: null,
      disk: null,
      ram: null,
      cpu: null,
      git: null,
    };
  }

  async check() {
    console.log(chalk.blue('\n🔍 Checking system requirements...\n'));

    await this.checkOS();
    await this.checkDisk();
    await this.checkRAM();
    await this.checkCPU();
    await this.checkGit();

    return this.getResults();
  }

  async checkOS() {
    const osMap = {
      win32: 'Windows',
      darwin: 'macOS',
      linux: 'Linux',
    };

    const osName = osMap[this.platform] || this.platform;

    if (!['win32', 'darwin', 'linux'].includes(this.platform)) {
      this.requirements.os = {
        passed: false,
        name: osName,
        message: `Unsupported OS: ${osName}. Requires Windows 10+, macOS 10.15+, or Linux.`,
      };
      return;
    }

    if (this.platform === 'darwin') {
      const osVersion = os.release();
      const [major] = osVersion.split('.');
      if (parseInt(major) < 15) {
        this.requirements.os = {
          passed: false,
          name: `${osName} ${osVersion}`,
          message: 'macOS 10.15 or newer required',
        };
        return;
      }
    }

    if (this.platform === 'linux') {
      try {
        const { stdout } = await execAsync('lsb_release -si');
        const distro = stdout.trim();
        this.requirements.os = {
          passed: true,
          name: `${osName} (${distro})`,
          message: 'Supported',
        };
      } catch {
        this.requirements.os = {
          passed: true,
          name: osName,
          message: 'Supported',
        };
      }
      return;
    }

    if (this.platform === 'win32') {
      const osVersion = os.release();
      const [major] = osVersion.split('.');
      if (parseInt(major) < 10) {
        this.requirements.os = {
          passed: false,
          name: `${osName} ${osVersion}`,
          message: 'Windows 10 or newer required',
        };
        return;
      }
    }

    this.requirements.os = {
      passed: true,
      name: `${osName} (${this.arch})`,
      message: 'Supported',
    };
  }

  async checkDisk() {
    const diskPath = this.platform === 'win32' ? 'C:' : '/';

    try {
      if (this.platform === 'win32') {
        const { stdout } = await execAsync(`fsutil volume diskfree ${diskPath}`);
        const lines = stdout.split('\n');
        const freeLine = lines.find(l => l.includes('free bytes'));
        if (freeLine) {
          const match = freeLine.match(/:\s*(\d+)/);
          if (match) {
            const freeBytes = BigInt(match[1]);
            const freeGB = Number(freeBytes) / (1024 ** 3);

            if (freeGB < 10) {
              this.requirements.disk = {
                passed: false,
                size: `${freeGB.toFixed(1)} GB free`,
                message: 'At least 10 GB free required',
              };
            } else {
              this.requirements.disk = {
                passed: true,
                size: `${freeGB.toFixed(1)} GB free`,
                message: 'Sufficient',
              };
            }
            return;
          }
        }
      } else {
        const { stdout } = await execAsync(`df -k ${diskPath} | tail -1`);
        const parts = stdout.split(/\s+/);
        if (parts.length >= 4) {
          const freeKB = parseInt(parts[3], 10);
          const freeGB = freeKB / (1024 * 1024);

          if (freeGB < 10) {
            this.requirements.disk = {
              passed: false,
              size: `${freeGB.toFixed(1)} GB free`,
              message: 'At least 10 GB free required',
            };
          } else {
            this.requirements.disk = {
              passed: true,
              size: `${freeGB.toFixed(1)} GB free`,
              message: 'Sufficient',
            };
          }
          return;
        }
      }
    } catch (error) {
      console.warn('Could not check disk space');
    }

    this.requirements.disk = {
      passed: true,
      size: 'Unknown',
      message: 'Assume sufficient',
    };
  }

  async checkRAM() {
    const totalMem = os.totalmem();
    const freeMem = os.freemem();
    const totalGB = totalMem / (1024 ** 3);
    const freeGB = freeMem / (1024 ** 3);

    if (totalGB < 4) {
      this.requirements.ram = {
        passed: false,
        size: `${totalGB.toFixed(1)} GB total`,
        message: 'Minimum 4 GB required (recommended 8 GB)',
      };
    } else if (totalGB < 8) {
      this.requirements.ram = {
        passed: true,
        size: `${totalGB.toFixed(1)} GB total (${freeGB.toFixed(1)} GB free)`,
        message: 'Minimum met, but 8 GB recommended',
      };
    } else {
      this.requirements.ram = {
        passed: true,
        size: `${totalGB.toFixed(1)} GB total (${freeGB.toFixed(1)} GB free)`,
        message: 'Sufficient',
      };
    }
  }

  async checkCPU() {
    const cpus = os.cpus();
    const cpuCount = cpus.length;
    const model = cpus[0]?.model || 'Unknown';

    this.requirements.cpu = {
      passed: cpuCount >= 2,
      cores: cpuCount,
      model: model.substring(0, 40),
      message: cpuCount >= 2 ? 'Sufficient' : 'Minimum 2 cores recommended',
    };
  }

  async checkGit() {
    try {
      await execAsync('git --version');
      this.requirements.git = {
        passed: true,
        message: 'Installed',
      };
    } catch {
      this.requirements.git = {
        passed: false,
        message: 'Not found (optional for manual installation)',
      };
    }
  }

  getResults() {
    const allPassed = Object.values(this.requirements).every(req => req?.passed !== false);

    const results = {
      passed: allPassed,
      critical: [],
      warnings: [],
    };

    if (this.requirements.os?.passed === false) {
      results.critical.push(`OS: ${this.requirements.os.message}`);
    }

    if (this.requirements.disk?.passed === false) {
      results.critical.push(`Disk: ${this.requirements.disk.message}`);
    }

    if (this.requirements.ram?.passed === false) {
      results.critical.push(`RAM: ${this.requirements.ram.message}`);
    }

    if (this.requirements.ram?.message.includes('recommended')) {
      results.warnings.push(`RAM: ${this.requirements.ram.message}`);
    }

    if (this.requirements.cpu?.passed === false) {
      results.warnings.push(`CPU: ${this.requirements.cpu.message}`);
    }

    if (this.requirements.git?.passed === false) {
      results.warnings.push('Git: Not found (optional)');
    }

    return results;
  }

  printResults() {
    const results = this.getResults();

    console.log(chalk.bold('\n📊 System Requirements Check\n'));

    // OS
    const osIcon = this.requirements.os?.passed ? '✓' : '✗';
    console.log(`${osIcon} OS: ${this.requirements.os?.name} - ${this.requirements.os?.message}`);

    // Disk
    const diskIcon = this.requirements.disk?.passed ? '✓' : '✗';
    console.log(`${diskIcon} Disk: ${this.requirements.disk?.size} - ${this.requirements.disk?.message}`);

    // RAM
    const ramIcon = this.requirements.ram?.passed ? '✓' : '✗';
    console.log(`${ramIcon} RAM: ${this.requirements.ram?.size} - ${this.requirements.ram?.message}`);

    // CPU
    const cpuIcon = this.requirements.cpu?.passed ? '✓' : '✗';
    console.log(`${cpuIcon} CPU: ${this.requirements.cpu?.cores} cores - ${this.requirements.cpu?.message}`);

    // Git
    const gitIcon = this.requirements.git?.passed ? '✓' : '✗';
    console.log(`${gitIcon} Git: ${this.requirements.git?.message}`);

    if (results.critical.length > 0) {
      console.log(chalk.red.bold('\n❌ Critical Issues:'));
      results.critical.forEach(issue => console.log(`  - ${issue}`));
      return false;
    }

    if (results.warnings.length > 0) {
      console.log(chalk.yellow.bold('\n⚠️  Warnings:'));
      results.warnings.forEach(warning => console.log(`  - ${warning}`));
    }

    console.log(chalk.green.bold('\n✓ System meets requirements!'));
    return true;
  }
}
