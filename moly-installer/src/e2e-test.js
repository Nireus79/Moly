#!/usr/bin/env node

import chalk from 'chalk';
import { SystemRequirements } from './systemRequirements.js';
import { InstallerWizard } from './wizard.js';

async function runE2ETest() {
  console.log(chalk.bold.cyan(`
╔════════════════════════════════════════╗
║   Moly Installer - End-to-End Test     ║
║   (Simulated Installation)             ║
╚════════════════════════════════════════╝
`));

  console.log(chalk.bold('\nPhase 1: System Requirements\n'));

  const sysReq = new SystemRequirements();
  await sysReq.check();
  const canContinue = sysReq.printResults();

  if (!canContinue) {
    console.log(chalk.red('\n❌ Installation blocked by system requirements'));
    process.exit(1);
  }

  console.log(chalk.bold.cyan('\nPhase 2: Configuration Wizard\n'));

  const wizard = new InstallerWizard();
  const choices = await wizard.run();

  console.log(chalk.bold.cyan('\n✓ Configuration Complete\n'));

  console.log(chalk.bold('User Choices:\n'));
  console.log(`  Provider:      ${chalk.cyan(choices.provider)}`);
  console.log(`  Model:         ${chalk.cyan(choices.model || 'None')}`);
  console.log(`  Auto-Start:    ${chalk.cyan(choices.autoStart ? 'Yes' : 'No')}`);
  console.log(`  Install Dir:   ${chalk.cyan(choices.installDir)}`);

  if (choices.provider === 'ollama') {
    console.log(`  CORS Proxy:    ${chalk.cyan(choices.installProxy ? 'Yes' : 'No')}`);
  }

  console.log(chalk.bold.cyan('\nPhase 3: Component Download (Simulated)\n'));

  const downloadSteps = [];

  if (choices.provider === 'ollama') {
    downloadSteps.push({
      name: 'Ollama Installer',
      size: '120 MB',
      platform: process.platform,
    });

    if (choices.installProxy) {
      downloadSteps.push({
        name: 'CORS Proxy (npm install -g moly-proxy)',
        size: '2 MB',
        platform: 'all',
      });
    }
  } else if (choices.provider === 'lm-studio') {
    downloadSteps.push({
      name: 'LM Studio',
      size: '350 MB',
      platform: process.platform,
    });
  }

  for (const step of downloadSteps) {
    console.log(`  ⬇️  ${step.name} (${step.size})`);
  }

  console.log(chalk.bold.cyan('\nPhase 4: Model Download (Simulated)\n'));

  if (choices.model) {
    console.log(`  📦 Pulling ${choices.model}...`);
    console.log(`  ⏳ Estimated time: 5-15 minutes depending on internet speed`);
    console.log(`  📊 Download size: ~4 GB\n`);
  } else {
    console.log('  (No model download needed for cloud-only mode)\n');
  }

  console.log(chalk.bold.cyan('Phase 5: Service Auto-Start Setup\n'));

  if (choices.autoStart) {
    let serviceSetup = '';

    if (process.platform === 'linux') {
      serviceSetup = 'systemd service (auto-start on boot)';
    } else if (process.platform === 'darwin') {
      serviceSetup = 'LaunchAgent (auto-start on login)';
    } else if (process.platform === 'win32') {
      serviceSetup = 'Task Scheduler (auto-start on logon)';
    }

    console.log(`  🔧 Setting up ${serviceSetup}`);
    console.log(`  ✓ Service configuration complete\n`);
  } else {
    console.log('  (Manual startup required)\n');
  }

  console.log(chalk.bold.cyan('\n✓ SIMULATED INSTALLATION COMPLETE!\n'));

  console.log(chalk.green.bold('╔════════════════════════════════════════╗'));
  console.log(chalk.green.bold('║   All phases completed successfully!    ║'));
  console.log(chalk.green.bold('╚════════════════════════════════════════╝\n'));

  console.log('Next Steps:');
  console.log('1. Run actual installation: moly-installer');
  console.log('2. Install browser extension');
  console.log('3. Moly will auto-detect local setup\n');

  console.log(chalk.dim('Test completed successfully!'));
}

runE2ETest().catch(error => {
  console.error(chalk.red('E2E test failed:', error));
  process.exit(1);
});
