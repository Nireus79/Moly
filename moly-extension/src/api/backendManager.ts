/**
 * Backend Manager for Moly Extension
 * Handles backend startup, health checks, and communication
 */

const BACKEND_HOST = 'http://127.0.0.1';
const BACKEND_PORT = 11436;
const BACKEND_URL = `${BACKEND_HOST}:${BACKEND_PORT}`;
const HEALTH_CHECK_INTERVAL = 5000; // 5 seconds
const START_TIMEOUT = 30000; // 30 seconds
const PRODUCTION_EXTENSION_ID = 'jkvuyxvgeivlakjahixagdztxvrcpzbc';

/**
 * Detect if running in development mode (unpacked extension)
 */
async function isDevelopmentMode(): Promise<boolean> {
  try {
    const self = await chrome.management.getSelf();
    // installType is 'development' for unpacked extensions, 'normal' for Web Store
    return self.installType === 'development';
  } catch {
    return false;
  }
}

export interface BackendStatus {
  running: boolean;
  url: string;
  version?: string;
  error?: string;
}

class BackendManager {
  private healthCheckInterval?: NodeJS.Timeout;
  private isStarting = false;
  private statusCallbacks: ((status: BackendStatus) => void)[] = [];

  /**
   * Initialize backend manager and start backend if needed
   */
  async initialize(): Promise<BackendStatus> {
    console.info('[BackendManager] Initializing...');

    // Check if backend is already running
    const status = await this.checkHealth();
    if (status.running) {
      console.info('[BackendManager] Backend already running');
      this.startHealthChecks();
      return status;
    }

    // Try to start backend
    console.info('[BackendManager] Starting backend...');
    const started = await this.startBackend();

    if (started) {
      console.info('[BackendManager] Backend started successfully');
      this.startHealthChecks();
      return { running: true, url: BACKEND_URL };
    } else {
      console.error('[BackendManager] Failed to start backend');
      return {
        running: false,
        url: BACKEND_URL,
        error: 'Failed to start backend service'
      };
    }
  }

  /**
   * Check if backend is healthy
   */
  async checkHealth(): Promise<BackendStatus> {
    try {
      const response = await fetch(`${BACKEND_URL}/api/status`, {
        method: 'GET',
        headers: { 'Content-Type': 'application/json' },
      });

      console.info('[BackendManager] Health check response:', response.status, response.statusText);

      if (response.ok) {
        const data = await response.json();
        console.info('[BackendManager] Backend healthy:', data);
        return {
          running: true,
          url: BACKEND_URL,
          version: data.version,
        };
      } else {
        console.warn('[BackendManager] Health check failed:', response.status, response.statusText);
      }
    } catch (error) {
      console.error('[BackendManager] Health check error:', error instanceof Error ? error.message : error);
    }

    return {
      running: false,
      url: BACKEND_URL,
      error: 'Backend not responding',
    };
  }

  /**
   * Start backend via native messaging
   */
  private async startBackend(): Promise<boolean> {
    if (this.isStarting) {
      console.warn('[BackendManager] Backend start already in progress');
      return false;
    }

    this.isStarting = true;

    try {
      // Check if in development mode
      const devMode = await isDevelopmentMode();

      if (devMode) {
        // In development: user must start backend manually (BackendStatus component shows instructions)
        console.info('[BackendManager] Development mode detected - user must start backend manually');
        return false;
      }

      // In production: use native messaging
      const result = await this.sendNativeMessage({
        action: 'start-backend',
        timeout: START_TIMEOUT,
      });

      return await this.waitForBackendReady();
    } catch (error) {
      console.error('[BackendManager] Failed to start backend:', error);
      return false;
    } finally {
      this.isStarting = false;
    }
  }

  /**
   * Wait for backend to be ready
   */
  private async waitForBackendReady(maxAttempts = 60): Promise<boolean> {
    for (let attempt = 0; attempt < maxAttempts; attempt++) {
      const status = await this.checkHealth();
      if (status.running) {
        console.info('[BackendManager] Backend is ready');
        return true;
      }

      // Wait 500ms before next attempt
      await new Promise(resolve => setTimeout(resolve, 500));
    }

    console.error('[BackendManager] Timeout waiting for backend');
    return false;
  }

  /**
   * Send message via native messaging
   */
  private sendNativeMessage(message: any): Promise<any> {
    return new Promise((resolve, reject) => {
      try {
        chrome.runtime.sendNativeMessage(
          'com.moly.backend_host',
          message,
          (response) => {
            if (chrome.runtime.lastError) {
              reject(new Error(chrome.runtime.lastError.message));
            } else {
              resolve(response);
            }
          }
        );
      } catch (error) {
        reject(error);
      }
    });
  }

  /**
   * Start periodic health checks
   */
  private startHealthChecks(): void {
    if (this.healthCheckInterval) {
      clearInterval(this.healthCheckInterval);
    }

    this.healthCheckInterval = setInterval(async () => {
      const status = await this.checkHealth();
      this.notifyStatusChange(status);

      if (!status.running) {
        console.warn('[BackendManager] Backend health check failed, attempting restart');
        // Try to restart if backend goes down
        this.startBackend();
      }
    }, HEALTH_CHECK_INTERVAL);
  }

  /**
   * Stop health checks
   */
  stopHealthChecks(): void {
    if (this.healthCheckInterval) {
      clearInterval(this.healthCheckInterval);
      this.healthCheckInterval = undefined;
    }
  }

  /**
   * Register callback for status changes
   */
  onStatusChange(callback: (status: BackendStatus) => void): void {
    this.statusCallbacks.push(callback);
  }

  /**
   * Notify all listeners of status change
   */
  private notifyStatusChange(status: BackendStatus): void {
    this.statusCallbacks.forEach(callback => {
      try {
        callback(status);
      } catch (error) {
        console.error('[BackendManager] Error in status callback:', error);
      }
    });
  }

  /**
   * Get backend URL
   */
  getBackendUrl(): string {
    return BACKEND_URL;
  }

  /**
   * Get current status
   */
  async getStatus(): Promise<BackendStatus> {
    return this.checkHealth();
  }

}

// Singleton instance
let instance: BackendManager | null = null;

// Export factory function for singleton
export function getBackendManager(): BackendManager {
  if (!instance) {
    instance = new BackendManager();
  }
  return instance;
}
