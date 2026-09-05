/**
 * Backend Manager - Auto-start and health check for Go backend
 * Ensures backend is running when extension is used
 */

const BACKEND_URL = 'http://127.0.0.1:11436';
const BACKEND_STARTUP_TIMEOUT = 5000; // 5 seconds

interface BackendStatus {
  running: boolean;
  healthy: boolean;
  lastCheck: number;
}

class BackendManager {
  private status: BackendStatus = {
    running: false,
    healthy: false,
    lastCheck: 0,
  };

  private startupPromise: Promise<boolean> | null = null;

  /**
   * Health check - ping backend status endpoint
   */
  private async healthCheck(): Promise<boolean> {
    try {
      const response = await fetch(`${BACKEND_URL}/api/status`, {
        method: 'GET',
        headers: { 'Content-Type': 'application/json' },
        timeout: 1000,
      });
      return response.ok;
    } catch (error) {
      return false;
    }
  }

  /**
   * Ensure backend is running and healthy
   * Called when extension icon is clicked
   */
  async ensureRunning(): Promise<boolean> {
    // If already running and recently checked, skip
    if (
      this.status.healthy &&
      Date.now() - this.status.lastCheck < 2000
    ) {
      return true;
    }

    // If startup already in progress, wait for it
    if (this.startupPromise) {
      return this.startupPromise;
    }

    // Start new check
    this.startupPromise = this.startBackendIfNeeded();
    try {
      const result = await this.startupPromise;
      return result;
    } finally {
      this.startupPromise = null;
    }
  }

  /**
   * Start backend if not running
   */
  private async startBackendIfNeeded(): Promise<boolean> {
    console.log('[BackendManager] Checking backend status...');

    // First, check if already running
    const isHealthy = await this.healthCheck();
    if (isHealthy) {
      console.log('[BackendManager] Backend is running');
      this.status = { running: true, healthy: true, lastCheck: Date.now() };
      return true;
    }

    console.log('[BackendManager] Backend not responding, attempting to start...');

    // Try to start via native message host (for spawning Go binary)
    try {
      const startResult = await this.requestNativeStartBackend();
      if (startResult && startResult.success) {
        console.log('[BackendManager] Backend startup requested via native host');

        // Wait for backend to become healthy
        const healthy = await this.waitForBackend();
        if (healthy) {
          this.status = { running: true, healthy: true, lastCheck: Date.now() };
          return true;
        }
      }
    } catch (error) {
      console.warn('[BackendManager] Native host not available:', error);
    }

    // If native host unavailable, try direct check (backend may auto-start via systemd)
    console.log('[BackendManager] Waiting for backend (systemd auto-start?)...');
    const healthy = await this.waitForBackend();

    if (healthy) {
      console.log('[BackendManager] Backend became healthy');
      this.status = { running: true, healthy: true, lastCheck: Date.now() };
      return true;
    }

    // Backend couldn't start
    console.error('[BackendManager] Backend failed to start');
    this.status = { running: false, healthy: false, lastCheck: Date.now() };

    // Show user instructions
    this.showBackendInstructions();
    return false;
  }

  /**
   * Wait for backend to become healthy (with retries)
   */
  private async waitForBackend(maxWait: number = BACKEND_STARTUP_TIMEOUT): Promise<boolean> {
    const startTime = Date.now();
    const retryInterval = 200; // Check every 200ms

    while (Date.now() - startTime < maxWait) {
      if (await this.healthCheck()) {
        return true;
      }
      await new Promise((resolve) => setTimeout(resolve, retryInterval));
    }

    return false;
  }

  /**
   * Request native host to start backend
   */
  private async requestNativeStartBackend(): Promise<{ success: boolean } | null> {
    return new Promise((resolve, reject) => {
      try {
        const port = chrome.runtime.connectNative('com.moly.backend_host');

        const timeout = setTimeout(() => {
          port.disconnect();
          reject(new Error('Native host timeout'));
        }, 2000);

        port.onMessage.addListener((response: any) => {
          clearTimeout(timeout);
          port.disconnect();
          resolve(response);
        });

        port.onDisconnect.addListener(() => {
          clearTimeout(timeout);
          if (chrome.runtime.lastError) {
            reject(chrome.runtime.lastError);
          } else {
            reject(new Error('Native host disconnected'));
          }
        });

        // Send start request
        port.postMessage({ action: 'start-backend' });
      } catch (error) {
        reject(error);
      }
    });
  }

  /**
   * Show user instructions for starting backend
   */
  private showBackendInstructions(): void {
    // Store message for UI to display
    chrome.storage.local.set({
      backendStatus: {
        running: false,
        message:
          'Go backend is not running. Start it with: cd moly-go && ./moly',
        timestamp: Date.now(),
      },
    });

    console.error(
      '[BackendManager] Backend not running. Start with: cd moly-go && ./moly'
    );
  }

  /**
   * Get current backend status
   */
  getStatus(): BackendStatus {
    return { ...this.status };
  }

  /**
   * Reset status (for testing)
   */
  reset(): void {
    this.status = { running: false, healthy: false, lastCheck: 0 };
    this.startupPromise = null;
  }
}

// Singleton instance
let manager: BackendManager | null = null;

export function getBackendManager(): BackendManager {
  if (!manager) {
    manager = new BackendManager();
  }
  return manager;
}

export default BackendManager;
