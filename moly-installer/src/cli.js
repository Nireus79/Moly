#!/usr/bin/env node

import chalk from 'chalk';
import { SystemRequirements } from './systemRequirements.js';
import { InstallerWizard } from './wizard.js';
import { DownloadManager } from './downloadManager.js';
import { ModelDownloader } from './modelDownloader.js';
import { ServiceManager } from './serviceManager.js';

async function main() {
  console.clear();

  try {
    // Step 1: Check system requirements
    console.log(chalk.bold.cyan('STEP 1: Checking System Requirements\n'));

    const sysReq = new SystemRequirements();
    await sysReq.check();
    const canContinue = sysReq.printResults();

    if (!canContinue) {
      console.log(chalk.red.bold('\n❌ Installation cannot continue due to unmet requirements.'));
      console.log(chalk.yellow('\nOptions:'));
      console.log(chalk.yellow('  1. Upgrade your system to meet the requirements'));
      console.log(chalk.yellow('  2. Install cloud-only version (use Claude/OpenAI)'));
      process.exit(1);
    }

    // Step 2: Run wizard
    console.log(chalk.bold.cyan('\nSTEP 2: Configuration Wizard\n'));

    const wizard = new InstallerWizard();
    const choices = await wizard.run();

    // Step 3: Download components
    console.log(chalk.bold.cyan('\nSTEP 3: Downloading Components\n'));

    const downloadManager = new DownloadManager();

    if (choices.provider === 'ollama') {
      try {
        await downloadManager.downloadOllamaInstaller();
        console.log(chalk.dim('\nℹ️  Ollama installer downloaded. Run it to complete installation.'));
      } catch (error) {
        console.log(chalk.yellow(`\n⚠️  Could not auto-download Ollama. Manual installation:`));
        console.log(chalk.dim('  Visit: https://ollama.ai'));
        console.log(chalk.dim('  Download and run the installer for your OS'));
      }

      if (choices.installProxy) {
        try {
          await downloadManager.downloadMolyProxy();
        } catch (error) {
          console.log(chalk.yellow(`\n⚠️  Could not install proxy. Manual installation:`));
          console.log(chalk.dim('  npm install -g moly-proxy'));
        }
      }
    } else if (choices.provider === 'lm-studio') {
      try {
        await downloadManager.downloadLMStudioInstaller();
        console.log(chalk.dim('\nℹ️  LM Studio installer downloaded. Run it to complete installation.'));
      } catch (error) {
        console.log(chalk.yellow(`\n⚠️  Could not auto-download LM Studio. Manual installation:`));
        console.log(chalk.dim('  Visit: https://lmstudio.ai'));
        console.log(chalk.dim('  Download and run the installer for your OS'));
      }
    }

    // Step 4: Download model
    if (choices.model) {
      console.log(chalk.bold.cyan('\nSTEP 4: Downloading Model\n'));

      const modelDownloader = new ModelDownloader(choices.provider, choices.model);
      const modelDownloaded = await modelDownloader.download();

      if (!modelDownloaded && choices.provider === 'ollama') {
        console.log(chalk.yellow('\n⚠️  Model download needs manual attention.'));
        console.log(chalk.dim('Please run: ollama pull ' + choices.model));
      }
    }

    // Step 5: Setup auto-start
    if (choices.autoStart) {
      console.log(chalk.bold.cyan('\nSTEP 5: Setting Up Auto-Start\n'));

      const serviceManager = new ServiceManager(choices.provider);
      await serviceManager.setupAutoStart();
    }

    // Step 6: Summary and next steps
    console.log(chalk.bold.cyan('\n✓ INSTALLATION COMPLETE!\n'));

    console.log(chalk.green.bold('╔════════════════════════════════════════╗'));
    console.log(chalk.green.bold('║   Moly is ready to use!                 ║'));
    console.log(chalk.green.bold('╚════════════════════════════════════════╝\n'));

    console.log(chalk.bold('Next Steps:\n'));

    console.log('1. Start the services (if not auto-started):');
    if (choices.provider === 'ollama') {
      console.log(chalk.dim('   Terminal 1: ollama serve'));
      if (choices.installProxy) {
        console.log(chalk.dim('   Terminal 2: moly-proxy'));
      }
    } else if (choices.provider === 'lm-studio') {
      console.log(chalk.dim('   Launch LM Studio application'));
    }

    console.log('\n2. Install Moly browser extension:');
    console.log(chalk.dim('   Chrome: https://chromewebstore.google.com/...'));
    console.log(chalk.dim('   Load manually: chrome://extensions > Load unpacked'));

    console.log('\n3. Moly will auto-detect your local setup on first use');

    console.log('\n' + chalk.dim('Documentation: https://github.com/Nireus79/Moly'));
    console.log(chalk.dim('Support: https://github.com/Nireus79/Moly/issues\n'));

  } catch (error) {
    console.error(chalk.red.bold('\n❌ Installation failed:'));
    console.error(chalk.red(`   ${error.message}`));
    console.log(chalk.yellow('\nFor support, visit: https://github.com/Nireus79/Moly/issues'));
    process.exit(1);
  }
}

main();
