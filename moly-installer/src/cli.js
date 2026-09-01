#!/usr/bin/env node

import chalk from 'chalk';
import fs from 'fs/promises';
import path from 'path';
import { fileURLToPath } from 'url';
import { SystemRequirements } from './systemRequirements.js';
import { InstallerWizard } from './wizard.js';
import { DownloadManager } from './downloadManager.js';
import { ModelDownloader } from './modelDownloader.js';
import { ServiceManager } from './serviceManager.js';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

async function setupNativeMessaging(extensionId) {
  /**
   * Set up native messaging host registry for current platform
   * This enables Moly extension to control services via native host
   */
  const platform = process.platform;

  try {
    if (platform === 'darwin') {
      // macOS: Create plist in ~/Library/Application Support/Google/Chrome/NativeMessagingHosts/
      const nativeHostDir = path.join(
        process.env.HOME,
        'Library/Application Support/Google/Chrome/NativeMessagingHosts'
      );
      await fs.mkdir(nativeHostDir, { recursive: true });

      const configPath = path.join(nativeHostDir, 'com.moly.installer.json');
      const config = {
        name: 'com.moly.installer',
        description: 'Moly Installer Launcher',
        path: '/usr/local/bin/moly-native-host',
        type: 'stdio',
        allowed_origins: [`chrome-extension://${extensionId}/`],
      };

      await fs.writeFile(configPath, JSON.stringify(config, null, 2));
      console.log(chalk.green('✓ Native messaging configured (macOS)'));
    } else if (platform === 'linux') {
      // Linux: Create config in ~/.config/google-chrome/NativeMessagingHosts/
      const nativeHostDir = path.join(
        process.env.HOME,
        '.config/google-chrome/NativeMessagingHosts'
      );
      await fs.mkdir(nativeHostDir, { recursive: true });

      const configPath = path.join(nativeHostDir, 'com.moly.installer.json');
      const config = {
        name: 'com.moly.installer',
        description: 'Moly Installer Launcher',
        path: '/usr/local/bin/moly-native-host',
        type: 'stdio',
        allowed_origins: [`chrome-extension://${extensionId}/`],
      };

      await fs.writeFile(configPath, JSON.stringify(config, null, 2));
      console.log(chalk.green('✓ Native messaging configured (Linux)'));
    } else if (platform === 'win32') {
      // Windows: Create registry entry (requires admin)
      console.log(
        chalk.yellow('ℹ️  Windows native messaging setup requires administrator privileges')
      );
      console.log(chalk.yellow('    This will be handled during proxy installation'));
    }
  } catch (error) {
    console.log(chalk.yellow(`⚠️  Could not set up native messaging: ${error.message}`));
    console.log(chalk.dim('    Fallback: Use direct Ollama connection (no native control)'));
  }
}

async function verifyInstallation(choices) {
  /**
   * Verify that all services are properly installed and can start
   */
  console.log(chalk.bold.cyan('\nVerifying Installation...\n'));

  let allOk = true;

  try {
    // Check Ollama
    if (choices.provider === 'ollama') {
      const platform = process.platform;
      let ollamaPath;

      if (platform === 'darwin') {
        ollamaPath = '/Applications/Ollama.app';
      } else if (platform === 'linux') {
        ollamaPath = '/usr/local/bin/ollama';
      } else if (platform === 'win32') {
        const username = process.env.USERNAME;
        ollamaPath = `C:\\Users\\${username}\\AppData\\Local\\Programs\\Ollama\\ollama.exe`;
      }

      try {
        await fs.access(ollamaPath);
        console.log(chalk.green('✓ Ollama installed and verified'));
      } catch {
        console.log(chalk.yellow('⚠️  Ollama installation not found'));
        allOk = false;
      }
    }

    // Check for native host binary (will be installed by downloader)
    if (process.platform !== 'win32') {
      try {
        await fs.access('/usr/local/bin/moly-native-host');
        console.log(chalk.green('✓ Native host installed and verified'));
      } catch {
        console.log(chalk.yellow('ℹ️  Native host not yet installed (optional)'));
      }
    }
  } catch (error) {
    console.log(chalk.yellow(`⚠️  Verification check failed: ${error.message}`));
  }

  return allOk;
}

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

    // Step 3: Download components (Ollama, Proxy, Native Host)
    console.log(chalk.bold.cyan('\nSTEP 3: Downloading Components\n'));

    const downloadManager = new DownloadManager();

    if (choices.provider === 'ollama') {
      try {
        await downloadManager.downloadOllamaInstaller();
        console.log(chalk.green('✓ Ollama downloaded'));
      } catch (error) {
        console.log(chalk.yellow(`⚠️  Could not auto-download Ollama`));
        console.log(chalk.dim('  Visit: https://ollama.ai and install manually'));
      }

      // Always install proxy + native host for service control
      try {
        await downloadManager.downloadMolyProxy();
        console.log(chalk.green('✓ CORS proxy installed'));
      } catch (error) {
        console.log(chalk.yellow(`⚠️  Could not install proxy: ${error.message}`));
      }

      try {
        await downloadManager.downloadNativeHost();
        console.log(chalk.green('✓ Native host installed'));
      } catch (error) {
        console.log(chalk.yellow(`⚠️  Could not install native host (optional)`));
      }
    } else if (choices.provider === 'lm-studio') {
      try {
        await downloadManager.downloadLMStudioInstaller();
        console.log(chalk.green('✓ LM Studio downloaded'));
      } catch (error) {
        console.log(chalk.yellow(`⚠️  Could not auto-download LM Studio`));
        console.log(chalk.dim('  Visit: https://lmstudio.ai and install manually'));
      }
    }

    // Step 4: Set up native messaging (for service control)
    console.log(chalk.bold.cyan('\nSTEP 4: Setting Up Service Control\n'));
    // NOTE: Extension ID will be generated when published to Chrome Web Store
    // Update this value after Web Store submission
    const extensionId = process.env.MOLY_EXTENSION_ID || 'nonheafhmdhjpbggfpdhjeoanofnkijc';
    await setupNativeMessaging(extensionId);

    // Step 5: Download model
    if (choices.model) {
      console.log(chalk.bold.cyan('\nSTEP 5: Downloading Default Model\n'));

      const modelDownloader = new ModelDownloader(choices.provider, choices.model);
      const modelDownloaded = await modelDownloader.download();

      if (!modelDownloaded && choices.provider === 'ollama') {
        console.log(chalk.yellow('\n⚠️  Model download needs manual attention.'));
        console.log(chalk.dim('Please run: ollama pull ' + choices.model));
      }
    }

    // Step 6: Setup auto-start for all services
    if (choices.autoStart) {
      console.log(chalk.bold.cyan('\nSTEP 6: Setting Up Auto-Start\n'));

      const serviceManager = new ServiceManager(choices.provider);
      await serviceManager.setupAutoStart();
      console.log(chalk.green('✓ Auto-start configured'));
    }

    // Step 7: Verify everything
    await verifyInstallation(choices);

    // Step 8: Summary and next steps
    console.log(chalk.bold.cyan('\n✓ INSTALLATION COMPLETE!\n'));

    console.log(chalk.green.bold('╔════════════════════════════════════════════════════════╗'));
    console.log(chalk.green.bold('║   Moly One-Click Setup Complete!                       ║'));
    console.log(chalk.green.bold('║   All services are configured and ready                ║'));
    console.log(chalk.green.bold('╚════════════════════════════════════════════════════════╝\n'));

    console.log(chalk.bold('NEXT STEPS:\n'));

    console.log(chalk.cyan('1. Services Auto-Start'));
    if (choices.autoStart) {
      if (choices.provider === 'ollama') {
        console.log(chalk.green('   ✓ Ollama will start automatically on reboot'));
        console.log(chalk.green('   ✓ CORS proxy will start automatically on reboot'));
        console.log(chalk.green('   ✓ Native host configured for service control'));
      }
    } else {
      console.log(chalk.yellow('   Manual: Restart your computer to enable auto-start'));
    }

    console.log(chalk.cyan('\n2. Control Services from Moly'));
    console.log(chalk.dim('   • Open Settings → Local Models Status → Service Control'));
    console.log(chalk.dim('   • Click "Start" / "Stop" buttons (no terminal needed!)'));

    console.log(chalk.cyan('\n3. Install Moly Extension'));
    const webStoreUrl = process.env.MOLY_WEBSTORE_URL || 'https://chromewebstore.google.com/detail/moly-messaging-coach/[UPDATE_AFTER_PUBLICATION]';
    console.log(chalk.dim(`   Chrome Web Store: ${webStoreUrl}`));
    console.log(chalk.dim('   Or load manually: chrome://extensions > Load unpacked'));

    console.log(chalk.cyan('\n4. Use Moly'));
    console.log(chalk.dim('   • Open any dating/messaging app'));
    console.log(chalk.dim('   • Moly will auto-detect your local setup'));
    console.log(chalk.dim('   • Start getting AI-powered suggestions!'));

    console.log('\n' + chalk.bold('💡 You No Longer Need the Terminal\n'));
    console.log(chalk.dim('Documentation: https://github.com/Nireus79/Moly'));
    console.log(chalk.dim('Support: https://github.com/Nireus79/Moly/issues\n'));
  } catch (error) {
    console.error(chalk.red.bold('\n❌ Installation failed:'));
    console.error(chalk.red(`   ${error.message}`));
    console.log(chalk.yellow('\nFor support, visit: https://github.com/Nireus79/Moly/issues'));
    process.exit(1);
  }
}

main();
