const { spawn } = require('child_process');
const path = require('path');
const os = require('os');
const fs = require('fs');

const HOME = os.homedir();
const CONFIG_DIR = path.join(HOME, '.config', 'moly');
const CONFIG_FILE = path.join(CONFIG_DIR, 'config.json');

class ModelManager {
  constructor() {
    this.nativeHostPath = path.join(HOME, '.local', 'bin', 'moly-native-host');
    this.pythonNativeHost = path.join(__dirname, '..', '..', 'moly-installer', 'native-host', 'moly-host.py');
  }

  ensureConfigDir() {
    if (!fs.existsSync(CONFIG_DIR)) {
      fs.mkdirSync(CONFIG_DIR, { recursive: true });
    }
  }

  loadConfig() {
    try {
      this.ensureConfigDir();
      if (fs.existsSync(CONFIG_FILE)) {
        return JSON.parse(fs.readFileSync(CONFIG_FILE, 'utf8'));
      }
    } catch (e) {
      console.log('[ModelManager] Error loading config:', e.message);
    }
    return this.getDefaultConfig();
  }

  saveConfig(config) {
    try {
      this.ensureConfigDir();
      fs.writeFileSync(CONFIG_FILE, JSON.stringify(config, null, 2));
    } catch (e) {
      console.log('[ModelManager] Error saving config:', e.message);
    }
  }

  getDefaultConfig() {
    return {
      version: '1.0',
      provider: 'cloud',
      model: 'claude-3-sonnet',
      ollama_installed: false,
      ollama_running: false,
      installed_models: [],
      api_keys: {},
      first_run_complete: false,
      created_at: new Date().toISOString()
    };
  }

  callNativeHost(action, params = {}) {
    return new Promise((resolve, reject) => {
      const request = { action, ...params };
      const pythonPath = process.env.PYTHON || 'python3';

      const proc = spawn(pythonPath, [this.pythonNativeHost], {
        stdio: ['pipe', 'pipe', 'pipe'],
        timeout: 60000
      });

      let stdoutBuffer = Buffer.alloc(0);
      let stderr = '';

      proc.stdout.on('data', (data) => {
        stdoutBuffer = Buffer.concat([stdoutBuffer, data]);
      });

      proc.stderr.on('data', (data) => {
        stderr += data.toString();
      });

      proc.on('close', (code) => {
        if (code === 0) {
          try {
            if (stdoutBuffer.length < 4) {
              reject(new Error('Invalid response from native host'));
              return;
            }

            const messageLength = stdoutBuffer.readUInt32LE(0);
            const messageData = stdoutBuffer.subarray(4, 4 + messageLength).toString('utf-8');
            const result = JSON.parse(messageData);
            resolve(result);
          } catch (e) {
            reject(new Error(`Failed to parse native host response: ${e.message}`));
          }
        } else {
          reject(new Error(`Native host error: ${stderr || 'Unknown error'}`));
        }
      });

      proc.on('error', (err) => {
        reject(new Error(`Failed to spawn native host: ${err.message}`));
      });

      const requestJson = JSON.stringify(request);
      const requestBytes = Buffer.from(requestJson, 'utf-8');
      const lengthBytes = Buffer.alloc(4);
      lengthBytes.writeUInt32LE(requestBytes.length, 0);

      proc.stdin.write(lengthBytes);
      proc.stdin.write(requestBytes);
      proc.stdin.end();
    });
  }

  async firstRunCheck() {
    try {
      const result = await this.callNativeHost('check-ollama');
      return {
        ollama_installed: result.installed || false,
        ollama_running: result.running || false
      };
    } catch (e) {
      console.log('[ModelManager] First run check error:', e.message);
      return { ollama_installed: false, ollama_running: false };
    }
  }

  async getInstalledModels() {
    try {
      const result = await this.callNativeHost('get-models');
      if (result.models) {
        return { models: result.models, error: null };
      }
      return { models: [], error: result.error };
    } catch (e) {
      console.log('[ModelManager] Error getting models:', e.message);
      return { models: [], error: e.message };
    }
  }

  async pullModel(modelName) {
    try {
      const result = await this.callNativeHost('pull-model', { model: modelName });
      return result;
    } catch (e) {
      console.log('[ModelManager] Error pulling model:', e.message);
      return { success: false, error: e.message };
    }
  }

  async removeModel(modelName) {
    try {
      const result = await this.callNativeHost('remove-model', { model: modelName });
      return result;
    } catch (e) {
      console.log('[ModelManager] Error removing model:', e.message);
      return { success: false, error: e.message };
    }
  }

  async startOllama() {
    try {
      const result = await this.callNativeHost('start-ollama');
      return result;
    } catch (e) {
      console.log('[ModelManager] Error starting Ollama:', e.message);
      return { success: false, error: e.message };
    }
  }

  async stopOllama() {
    try {
      const result = await this.callNativeHost('stop-ollama');
      return result;
    } catch (e) {
      console.log('[ModelManager] Error stopping Ollama:', e.message);
      return { success: false, error: e.message };
    }
  }
}

module.exports = ModelManager;
