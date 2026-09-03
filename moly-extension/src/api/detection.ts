/**
 * Local Model Detection System
 * Detects what's installed, running, and available
 */

export interface LocalModelStatus {
  ollama: {
    running: boolean;
    installed: boolean;
    models: string[];
    baseUrl?: string;
    installMethod?: 'snap' | 'package' | 'manual';
  };
  lmStudio: {
    running: boolean;
    installed: boolean;
    models: string[];
  };
  corsProxy: {
    running: boolean;
    installed: boolean;
  };
  nativeHost: {
    installed: boolean;
  };
  cloudProviders: {
    claude: boolean;
    openai: boolean;
  };
  components: {
    allConfigured: boolean;
    needsSetup: string[];
  };
}

/**
 * Check if Ollama is running
 */
async function checkOllamaRunning(): Promise<boolean> {
  try {
    const response = await fetch('http://localhost:11434/api/tags', {
      signal: AbortSignal.timeout(2000),
    });
    return response.ok;
  } catch {
    return false;
  }
}

/**
 * Check if CORS proxy is running
 */
async function checkProxyRunning(): Promise<boolean> {
  try {
    const response = await fetch('http://localhost:11435/api/tags', {
      signal: AbortSignal.timeout(2000),
    });
    return response.ok;
  } catch {
    return false;
  }
}

/**
 * Discover Ollama models
 */
async function discoverOllamaModels(): Promise<string[]> {
  try {
    const response = await fetch('http://localhost:11434/api/tags', {
      signal: AbortSignal.timeout(3000),
    });

    if (!response.ok) return [];

    const data = await response.json();
    if (data.models && Array.isArray(data.models)) {
      return data.models
        .map((m: any) => m.name)
        .sort()
        .slice(0, 10); // Limit to 10 for display
    }
  } catch {
    // Ollama not running or unreachable
  }

  return [];
}

/**
 * Check if LM Studio is running
 */
async function checkLMStudioRunning(): Promise<boolean> {
  try {
    const response = await fetch('http://localhost:8000/api/models', {
      signal: AbortSignal.timeout(2000),
    });
    return response.ok;
  } catch {
    return false;
  }
}

/**
 * Discover LM Studio models
 */
async function discoverLMStudioModels(): Promise<string[]> {
  try {
    const response = await fetch('http://localhost:8000/api/models', {
      signal: AbortSignal.timeout(3000),
    });

    if (!response.ok) return [];

    const data = await response.json();
    if (data.data && Array.isArray(data.data)) {
      return data.data
        .map((m: any) => m.id)
        .sort()
        .slice(0, 10);
    }
  } catch {
    // LM Studio not running or unreachable
  }

  return [];
}

/**
 * Check if cloud provider is configured
 */
async function checkCloudProviders(): Promise<{
  claude: boolean;
  openai: boolean;
}> {
  try {
    const result = await chrome.storage.local.get('settings');
    const settings = result.settings;

    if (!settings) {
      return { claude: false, openai: false };
    }

    return {
      claude: !!settings.providers?.claude?.apiKey,
      openai: !!settings.providers?.openai?.apiKey,
    };
  } catch {
    return { claude: false, openai: false };
  }
}

/**
 * Detect if Ollama executable exists (browser can't directly check filesystem)
 * Instead, we infer from system by checking if user has run ollama before
 */
async function checkOllamaInstalled(): Promise<boolean> {
  try {
    const result = await chrome.storage.local.get('detectionCache');
    if (result.detectionCache?.ollamaInstalled !== undefined) {
      return result.detectionCache.ollamaInstalled;
    }

    // If Ollama was running before, it's likely installed
    // We'll detect this based on settings or prior successful connections
    const settingsResult = await chrome.storage.local.get('settings');
    const hasOllamaConfig =
      settingsResult.settings?.providers?.ollama?.enabled;

    return hasOllamaConfig || false;
  } catch {
    return false;
  }
}

/**
 * Check if native host is installed
 */
async function checkNativeHostInstalled(): Promise<boolean> {
  return new Promise((resolve) => {
    const timeout = setTimeout(() => resolve(false), 2000);
    chrome.runtime.sendNativeMessage(
      'com.moly.native_host',
      { action: 'ping' },
      (response) => {
        clearTimeout(timeout);
        if (chrome.runtime.lastError) {
          resolve(false);
        } else {
          resolve(response?.pong === true);
        }
      }
    );
  });
}

/**
 * Main detection function - get complete status
 */
export async function detectLocalModels(): Promise<LocalModelStatus> {
  console.log('[Detection] Starting local model detection...');

  const startTime = performance.now();

  // Run detections in parallel
  const [
    ollamaRunning,
    proxyRunning,
    lmStudioRunning,
    ollamaModels,
    lmStudioModels,
    cloudProviders,
    ollamaInstalled,
    nativeHostInstalled,
  ] = await Promise.all([
    checkOllamaRunning(),
    checkProxyRunning(),
    checkLMStudioRunning(),
    discoverOllamaModels(),
    discoverLMStudioModels(),
    checkCloudProviders(),
    checkOllamaInstalled(),
    checkNativeHostInstalled(),
  ]);

  // Determine what needs to be set up
  const needsSetup: string[] = [];

  if (!nativeHostInstalled) {
    needsSetup.push('native-host');
  }
  if (!ollamaInstalled && !ollamaRunning) {
    needsSetup.push('ollama');
  }
  if (ollamaInstalled || ollamaRunning) {
    if (!proxyRunning) {
      needsSetup.push('cors-proxy');
    }
  }
  if (!cloudProviders.claude && !cloudProviders.openai && (!ollamaInstalled && !ollamaRunning)) {
    needsSetup.push('llm-provider');
  }

  const status: LocalModelStatus = {
    ollama: {
      running: ollamaRunning,
      installed: ollamaInstalled || ollamaRunning,
      models: ollamaModels,
      baseUrl: proxyRunning
        ? 'http://localhost:11435'
        : ollamaRunning
          ? 'http://localhost:11434'
          : undefined,
    },
    lmStudio: {
      running: lmStudioRunning,
      installed: lmStudioRunning,
      models: lmStudioModels,
    },
    corsProxy: {
      running: proxyRunning,
      installed: proxyRunning, // Browser can't detect installation, only if running
    },
    nativeHost: {
      installed: nativeHostInstalled,
    },
    cloudProviders,
    components: {
      allConfigured: needsSetup.length === 0,
      needsSetup,
    },
  };

  const elapsed = performance.now() - startTime;
  console.log(`[Detection] Complete in ${elapsed.toFixed(0)}ms`, status);

  // Cache result for 30 seconds
  await chrome.storage.local.set({
    detectionCache: {
      ...status,
      timestamp: Date.now(),
    },
  });

  return status;
}

/**
 * Get cached detection result if recent enough
 */
export async function getCachedDetection(
  maxAge: number = 30000
): Promise<LocalModelStatus | null> {
  try {
    const result = await chrome.storage.local.get('detectionCache');
    const cache = result.detectionCache;

    if (!cache || !cache.timestamp) {
      return null;
    }

    const age = Date.now() - cache.timestamp;
    if (age > maxAge) {
      return null;
    }

    console.log(`[Detection] Using cached result (${age.toFixed(0)}ms old)`);
    return cache;
  } catch {
    return null;
  }
}

/**
 * Get detection with caching
 */
export async function getLocalModelStatus(): Promise<LocalModelStatus> {
  // Try cache first
  const cached = await getCachedDetection();
  if (cached) {
    return cached;
  }

  // Fall back to fresh detection
  return detectLocalModels();
}

/**
 * Format status for display
 */
export function formatStatus(status: LocalModelStatus): string[] {
  const lines: string[] = [];

  if (status.ollama.running && status.ollama.models.length > 0) {
    lines.push(
      `✓ Ollama Running (${status.ollama.models.length} model${status.ollama.models.length === 1 ? '' : 's'})`
    );
  } else if (status.ollama.installed) {
    lines.push('○ Ollama Installed (Not Running)');
  }

  if (status.lmStudio.running && status.lmStudio.models.length > 0) {
    lines.push(
      `✓ LM Studio Running (${status.lmStudio.models.length} model${status.lmStudio.models.length === 1 ? '' : 's'})`
    );
  } else if (status.lmStudio.installed) {
    lines.push('○ LM Studio Installed (Not Running)');
  }

  if (status.cloudProviders.claude) {
    lines.push('✓ Claude API Configured');
  }

  if (status.cloudProviders.openai) {
    lines.push('✓ OpenAI API Configured');
  }

  if (lines.length === 0) {
    lines.push('No local models or cloud providers configured');
  }

  return lines;
}
