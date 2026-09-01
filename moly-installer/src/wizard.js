import prompts from 'prompts';
import chalk from 'chalk';
import os from 'os';

export class InstallerWizard {
  constructor() {
    this.choices = {
      provider: null,
      model: null,
      installProxy: true,
      autoStart: true,
      customPaths: false,
      installDir: null,
    };
  }

  async run() {
    await this.showWelcome();
    await this.chooseProvider();
    await this.chooseModel();
    await this.chooseOptions();
    await this.confirm();

    return this.choices;
  }

  async showWelcome() {
    console.clear();
    console.log(chalk.cyan.bold(`
╔═════════════════════════════════════════╗
║   Moly AI Coaching - Installation       ║
║   Privacy-First Local AI Models          ║
╚═════════════════════════════════════════╝
`));

    console.log(chalk.dim('This installer will set up:'));
    console.log(chalk.dim('  • Local AI model (Ollama or LM Studio)'));
    console.log(chalk.dim('  • CORS proxy for browser access'));
    console.log(chalk.dim('  • Automatic service startup'));
    console.log(chalk.dim('  • Moly browser extension\n'));

    await prompts({
      type: 'confirm',
      name: 'continue',
      message: 'Continue with installation?',
      initial: true,
    });
  }

  async chooseProvider() {
    console.log(chalk.bold('\n1️⃣  Choose Local AI Platform\n'));

    const response = await prompts({
      type: 'select',
      name: 'provider',
      message: 'Which platform do you prefer?',
      choices: [
        {
          title: 'Ollama (Recommended for developers)',
          description: 'CLI-based, lightweight, excellent for coding tasks',
          value: 'ollama',
        },
        {
          title: 'LM Studio (Recommended for non-programmers)',
          description: 'GUI application, easier to use, good visual interface',
          value: 'lm-studio',
        },
        {
          title: 'Cloud-Only (No local model)',
          description: 'Use Claude or OpenAI in the cloud (less private)',
          value: 'cloud-only',
        },
      ],
      initial: 0,
    });

    this.choices.provider = response.provider;

    if (this.choices.provider !== 'cloud-only') {
      console.log(chalk.green(`\n✓ Selected: ${this.choices.provider}`));
    }
  }

  async chooseModel() {
    if (this.choices.provider === 'cloud-only') {
      console.log(chalk.yellow('\nℹ️  Cloud-only mode selected. No local model needed.'));
      this.choices.model = null;
      return;
    }

    console.log(chalk.bold('\n2️⃣  Choose AI Model\n'));

    const modelChoices = {
      ollama: [
        {
          title: 'Mistral 7B (Recommended)',
          description: '4GB, excellent for conversation, fast inference',
          value: 'mistral',
        },
        {
          title: 'Llama 2 7B',
          description: '4GB, good general model, well-tuned',
          value: 'llama2',
        },
        {
          title: 'Neural Chat 7B',
          description: '4GB, specialized for chat, optimized responses',
          value: 'neural-chat',
        },
      ],
      'lm-studio': [
        {
          title: 'Mistral 7B (Recommended)',
          description: '4GB GGUF, fast and efficient',
          value: 'mistral-7b-gguf',
        },
        {
          title: 'Llama 2 7B Chat',
          description: '4GB GGUF, conversational model',
          value: 'llama-2-7b-chat-gguf',
        },
      ],
    };

    const response = await prompts({
      type: 'select',
      name: 'model',
      message: 'Which model do you want to download?',
      choices: modelChoices[this.choices.provider] || modelChoices.ollama,
      initial: 0,
    });

    this.choices.model = response.model;
    console.log(chalk.green(`\n✓ Selected: ${this.choices.model}`));
  }

  async chooseOptions() {
    console.log(chalk.bold('\n3️⃣  Configuration Options\n'));

    if (this.choices.provider === 'ollama') {
      const proxyResponse = await prompts({
        type: 'confirm',
        name: 'installProxy',
        message: 'Install CORS proxy for browser access?',
        initial: true,
      });
      this.choices.installProxy = proxyResponse.installProxy;
    }

    const autoStartResponse = await prompts({
      type: 'confirm',
      name: 'autoStart',
      message: 'Enable automatic startup on system boot?',
      initial: true,
    });
    this.choices.autoStart = autoStartResponse.autoStart;

    const customPathResponse = await prompts({
      type: 'confirm',
      name: 'customPaths',
      message: 'Use custom installation paths?',
      initial: false,
    });

    if (customPathResponse.customPaths) {
      const pathResponse = await prompts({
        type: 'text',
        name: 'installDir',
        message: 'Installation directory:',
        initial: this.getDefaultInstallDir(),
      });
      this.choices.installDir = pathResponse.installDir;
    } else {
      this.choices.installDir = this.getDefaultInstallDir();
    }
  }

  async confirm() {
    console.log(chalk.bold('\n4️⃣  Review Settings\n'));

    const providerLabel = {
      ollama: 'Ollama (CLI)',
      'lm-studio': 'LM Studio (GUI)',
      'cloud-only': 'Cloud Only',
    };

    console.log(`Provider:        ${chalk.cyan(providerLabel[this.choices.provider])}`);
    console.log(`Model:           ${chalk.cyan(this.choices.model || 'None')}`);
    console.log(`Auto-Start:      ${chalk.cyan(this.choices.autoStart ? 'Yes' : 'No')}`);
    if (this.choices.provider === 'ollama') {
      console.log(`CORS Proxy:      ${chalk.cyan(this.choices.installProxy ? 'Yes' : 'No')}`);
    }
    console.log(`Install Dir:     ${chalk.cyan(this.choices.installDir)}`);

    const response = await prompts({
      type: 'confirm',
      name: 'proceed',
      message: '\nProceed with installation?',
      initial: true,
    });

    if (!response.proceed) {
      console.log(chalk.yellow('\nInstallation cancelled.'));
      process.exit(0);
    }
  }

  getDefaultInstallDir() {
    const homeDir = os.homedir();

    if (process.platform === 'win32') {
      return `${homeDir}\\AppData\\Local\\Moly`;
    } else if (process.platform === 'darwin') {
      return `${homeDir}/Library/Application Support/Moly`;
    } else {
      return `${homeDir}/.local/share/moly`;
    }
  }
}
