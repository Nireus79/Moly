import fetch from 'node-fetch';
import chalk from 'chalk';
import cliProgress from 'cli-progress';

export class ModelDownloader {
  constructor(provider, model) {
    this.provider = provider;
    this.model = model;
    this.progressBar = null;
  }

  async downloadWithOllama() {
    console.log(chalk.bold(`\n📦 Pulling model: ${this.model}\n`));
    console.log(chalk.dim('This may take several minutes depending on your internet speed...'));

    const { exec } = await import('child_process');
    const { promisify } = await import('util');
    const execAsync = promisify(exec);

    try {
      // Check if Ollama is running
      try {
        const response = await fetch('http://localhost:11434/api/tags', { method: 'HEAD' });
        if (!response.ok) throw new Error('Ollama not responding');
      } catch {
        console.log(chalk.yellow('\n⚠️  Ollama does not appear to be running yet.'));
        console.log(chalk.dim('Please ensure Ollama was installed and start it with: ollama serve'));
        return false;
      }

      // Pull the model
      console.log(chalk.dim(`Pulling ${this.model}...`));
      await execAsync(`ollama pull ${this.model}`);

      console.log(chalk.green(`\n✓ Model downloaded: ${this.model}`));

      // Verify model was pulled
      const tagsResponse = await fetch('http://localhost:11434/api/tags');
      const tagsData = await tagsResponse.json();

      if (tagsData.models && tagsData.models.some(m => m.name.includes(this.model))) {
        console.log(chalk.green('✓ Model verified in Ollama'));
        return true;
      }

      console.error(chalk.red('✗ Model not found after download'));
      return false;
    } catch (error) {
      console.error(chalk.red(`\n✗ Failed to download model: ${error.message}`));
      console.log(chalk.dim('\nManual installation:'));
      console.log(chalk.dim(`  1. Open terminal/PowerShell`));
      console.log(chalk.dim(`  2. Run: ollama pull ${this.model}`));
      console.log(chalk.dim(`  3. Wait for download to complete`));
      return false;
    }
  }

  async downloadWithLMStudio() {
    console.log(chalk.bold(`\n📦 Model for LM Studio: ${this.model}\n`));
    console.log(chalk.dim('LM Studio downloads models through its GUI.'));
    console.log(chalk.dim('Manual steps:'));
    console.log(chalk.dim('  1. Launch LM Studio application'));
    console.log(chalk.dim('  2. Go to Search tab'));
    console.log(chalk.dim(`  3. Search for "${this.getModelLabel()}"`));
    console.log(chalk.dim('  4. Click Download'));
    console.log(chalk.dim('  5. Wait for model to download\n'));

    // LM Studio handles downloads through its GUI
    // We can't automate this, so we return true to continue
    return true;
  }

  getModelLabel() {
    const labels = {
      'mistral-7b-gguf': 'Mistral 7B',
      'llama-2-7b-chat-gguf': 'Llama 2 7B Chat',
    };
    return labels[this.model] || this.model;
  }

  async download() {
    if (this.provider === 'ollama') {
      return await this.downloadWithOllama();
    } else if (this.provider === 'lm-studio') {
      return await this.downloadWithLMStudio();
    } else {
      console.log(chalk.yellow('ℹ️  Cloud-only mode: No local model download needed'));
      return true;
    }
  }

  async verifyModel() {
    try {
      // Try to connect to Ollama and verify model exists
      const response = await fetch('http://localhost:11434/api/tags', {
        signal: AbortSignal.timeout(5000),
      });

      if (!response.ok) return false;

      const data = await response.json();
      if (data.models && Array.isArray(data.models)) {
        return data.models.some(m => m.name.includes(this.model));
      }

      return false;
    } catch {
      // LM Studio verification would be different
      return this.provider !== 'ollama';
    }
  }
}
