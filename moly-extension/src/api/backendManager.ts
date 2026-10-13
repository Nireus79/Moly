/**
 * Backend Manager for Moly Extension
 * Handles backend startup, health checks, and dual-path routing (V1 + agents)
 */

import { FeatureFlags, createFeatureFlags } from '../config/featureFlags';
import { AgentClient, createAgentClient } from './agentClient';
import { ensureBackendConnection } from './backendAuth';
import { classifyHostError, needsUserAction } from './hostErrors';

// Default backend configuration (used only as fallback after detection fails)
const BACKEND_HOST = 'http://127.0.0.1';
const BACKEND_PORT = 11436;
const BACKEND_URL = `${BACKEND_HOST}:${BACKEND_PORT}`;

// Alternative ports to try if primary detection fails
const ALTERNATIVE_BACKEND_PORTS = [11436];

const HEALTH_CHECK_INTERVAL = 5000; // 5 seconds

// Agent configuration
const AGENTS_API_ENABLED = true; // Feature flag for agents
const AGENTS_API_TIMEOUT = 2000; // 2 second timeout for requests

export interface BackendStatus {
  running: boolean;
  url: string;
  version?: string;
  error?: string;
}

interface AgentBackendStatus extends BackendStatus {
  agentsEnabled?: boolean;
  agentsErrorRate?: number;
  agentsHealthy?: boolean;
}

class BackendManager {
  private healthCheckInterval?: NodeJS.Timeout;
  private isStarting = false;
  private statusCallbacks: ((status: BackendStatus) => void)[] = [];
  private detectedBackendUrl: string | null = null;
  private hostError: string | undefined;
  private hostNeedsUser = false; // the Moly app is missing or does not know this extension: retrying alone cannot help

  // agent support
  private agentClient: AgentClient | null = null;
  private featureFlags: FeatureFlags | null = null;
  private userId: string | null = null;
  private agentMetrics = { requests: 0, errors: 0 };

  /**
   * Initialize backend manager and start backend if needed
   */
  async initialize(): Promise<BackendStatus> {
    console.info('[BackendManager] Initializing...');

    // Get URL and token from the Moly app (starts the backend if needed); a development backend has no app
    const viaHost = await this.connectViaHost();

    // Check if backend is already running
    const status = await this.checkHealth();
    if (status.running) {
      console.info('[BackendManager] Backend already running');
      this.startHealthChecks();
      return status;
    }

    // Try to start backend (without the Moly app there is nothing to start: a development backend is started by hand)
    console.info('[BackendManager] Starting backend...');
    const started = viaHost ? await this.startBackend() : false;

    if (started) {
      console.info('[BackendManager] Backend started successfully');
      this.startHealthChecks();
      // Verify detection after starting
      const finalStatus = await this.checkHealth();
      return finalStatus;
    } else {
      console.error('[BackendManager] Failed to start backend');
      return {
        running: false,
        url: this.detectedBackendUrl || 'unknown',
        error: this.hostError || 'Failed to start backend service'
      };
    }
  }

  /**
   * Auto-detect backend by trying common ports
   * Priority: Cached result → Check alternative ports → Fallback to default
   */
  private async detectBackendUrl(): Promise<string | null> {
    // Return cached result if available
    if (this.detectedBackendUrl) {
      return this.detectedBackendUrl;
    }

    // Check if cached in storage from previous session
    try {
      const stored = await chrome.storage.local.get('detectedBackendUrl');
      if (stored.detectedBackendUrl) {
        console.info('[BackendManager] Using cached backend URL:', stored.detectedBackendUrl);
        this.detectedBackendUrl = stored.detectedBackendUrl;
        return stored.detectedBackendUrl;
      }
    } catch (err) {
      console.warn('[BackendManager] Failed to read cached backend URL:', err);
    }

    // Try to detect by testing alternative ports
    console.info('[BackendManager] Auto-detecting backend on common ports:', ALTERNATIVE_BACKEND_PORTS);

    for (const port of ALTERNATIVE_BACKEND_PORTS) {
      const url = `${BACKEND_HOST}:${port}`;
      try {
        const response = await fetch(`${url}/api/status`, {
          method: 'GET',
          signal: AbortSignal.timeout(1000),
          headers: { 'Content-Type': 'application/json' },
        });

        if (response.ok) {
          console.info(`[BackendManager] ✓ Backend detected on port ${port}`);
          this.detectedBackendUrl = url;
          // Cache detected URL for next session
          chrome.storage.local.set({ detectedBackendUrl: url }).catch(err => {
            console.warn('[BackendManager] Failed to cache backend URL:', err);
          });
          return url;
        }
      } catch (error) {
        // Try next port
        continue;
      }
    }

    console.warn('[BackendManager] Could not auto-detect backend on any port, using fallback');
    return null;
  }

  /**
   * Check if backend is healthy
   */
  async checkHealth(): Promise<BackendStatus> {
    // Try to get cached URL first, then auto-detect
    if (!this.detectedBackendUrl) {
      const stored = await chrome.storage.local.get('detectedBackendUrl');
      this.detectedBackendUrl = stored.detectedBackendUrl || null;
    }

    // If no cached URL, try to detect
    if (!this.detectedBackendUrl) {
      this.detectedBackendUrl = await this.detectBackendUrl();
    }

    if (!this.detectedBackendUrl) {
      return {
        running: false,
        url: 'unknown',
        error: 'Backend not found on any port',
      };
    }

    try {
      const response = await fetch(`${this.detectedBackendUrl}/api/status`, {
        method: 'GET',
        headers: { 'Content-Type': 'application/json' },
      });

      console.info('[BackendManager] Health check response:', response.status, response.statusText);

      if (response.ok) {
        const data = await response.json();
        console.info('[BackendManager] Backend healthy:', data);
        return {
          running: true,
          url: this.detectedBackendUrl,
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
      url: this.detectedBackendUrl,
      error: 'Backend not responding',
    };
  }

  /**
   * Ask the Moly app (native host, through the service worker) for the backend: it starts it if needed and returns the
   * URL and token. Without the app (a development backend started by hand) this fails and the caller falls back.
   */
  private async connectViaHost(): Promise<boolean> {
    const result = await ensureBackendConnection();
    if (result.ok && result.url) {
      this.hostError = undefined;
      this.hostNeedsUser = false;
      this.detectedBackendUrl = result.url;
      chrome.storage.local.set({ detectedBackendUrl: result.url }).catch(() => undefined);
      return true;
    }
    this.hostError = result.error;
    this.hostNeedsUser = needsUserAction(classifyHostError(result.error));
    console.info('[BackendManager] Moly app not available:', result.error);
    return false;
  }

  /**
   * Start the backend through the native host, then wait until it answers.
   */
  private async startBackend(): Promise<boolean> {
    if (this.isStarting) {
      console.warn('[BackendManager] Backend start already in progress');
      return false;
    }

    this.isStarting = true;

    try {
      if (!(await this.connectViaHost())) {
        // No Moly app: in development the backend is started by hand (BackendStatus shows the instructions).
        return false;
      }
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
   * Start periodic health checks
   */
  private startHealthChecks(): void {
    if (this.healthCheckInterval) {
      clearInterval(this.healthCheckInterval);
    }

    this.healthCheckInterval = setInterval(async () => {
      const status = await this.checkHealth();
      this.notifyStatusChange(status);

      if (!status.running && !this.hostNeedsUser) {
        console.warn('[BackendManager] Backend health check failed, attempting restart');
        // Try to restart if backend goes down
        this.startBackend();
      }
    }, HEALTH_CHECK_INTERVAL);
  }

  /** The last message from the Moly app (native host), if reaching it failed. */
  getLastHostError(): string | undefined {
    return this.hostError;
  }

  /** The user fixed something (installed the app): forget the failure and start again. */
  async retry(): Promise<BackendStatus> {
    this.hostNeedsUser = false;
    this.hostError = undefined;
    return this.initialize();
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
   * Get backend URL (auto-detected or cached)
   */
  getBackendUrl(): string {
    return this.detectedBackendUrl || BACKEND_URL;
  }

  /**
   * Get current status
   */
  async getStatus(): Promise<BackendStatus> {
    return this.checkHealth();
  }

  /**
   * Initialize agents support
   */
  async initializeAgents(userId?: string): Promise<void> {
    this.userId = userId || 'anonymous';

    try {
      // Initialize feature flags
      this.featureFlags = await createFeatureFlags(this.userId);

      // Check if is enabled
      const flagState = this.featureFlags.isAgentsEnabled(this.userId);

      if (!flagState.isEnabled) {
        console.info('[BackendManager] agents disabled:', flagState.reason);
        return;
      }

      // Initialize client
      const backendUrl = this.featureFlags.getBackendUrl();
      this.agentClient = createAgentClient(backendUrl);

      // Health check backend
      const isHealthy = await this.agentClient.healthCheck();

      if (isHealthy) {
        console.info('[BackendManager] agents initialized successfully');
      } else {
        console.warn('[BackendManager] backend health check failed');
        this.agentClient = null;
      }
    } catch (error) {
      console.error('[BackendManager] Failed to initialize agents:', error);
      this.agentClient = null;
    }
  }

  /**
   * Check if agents are enabled for this user
   */
  isAgentsEnabled(): boolean {
    if (!this.featureFlags || !this.userId) {
      return false;
    }

    const state = this.featureFlags.isAgentsEnabled(this.userId);
    return state.isEnabled;
  }

  /**
   * Get agent client (if enabled)
   */
  getClient(): AgentClient | null {
    return this.isAgentsEnabled() ? this.agentClient : null;
  }

  /**
   * Record request metrics
   */
  recordRequest(success: boolean): void {
    this.agentMetrics.requests++;
    if (!success) {
      this.agentMetrics.errors++;
    }
  }

  /**
   * Get metrics
   */
  getMetrics() {
    return {
      ...this.agentMetrics,
      errorRate: this.agentMetrics.requests > 0 ? this.agentMetrics.errors / this.agentMetrics.requests : 0,
    };
  }

  /**
   * Get diagnostic info (for debugging)
   */
  getDiagnostics() {
    return {
      backendUrl: this.detectedBackendUrl,
      agentsEnabled: this.isAgentsEnabled(),
      agentsClientHealthy: this.agentClient !== null,
      agentMetrics: this.getMetrics(),
      featureFlagsDiagnostics: this.featureFlags?.getDiagnostics(),
    };
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
