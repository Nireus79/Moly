/**
 * Feature Flags - Control gradual rollout of V2 agents
 */

export interface FeatureFlagConfig {
  enableV2Agents: boolean;
  rolloutPercentage: number; // 0-100
  userOverrides: Record<string, boolean>; // userId -> enabled/disabled
  backendUrl: string;
}

export interface FeatureFlagState {
  isEnabled: boolean;
  rolloutPercentage: number;
  reason: string; // Why is it enabled/disabled
}

/**
 * Feature flags management
 */
export class FeatureFlags {
  private config: FeatureFlagConfig;
  private userBucket: number;

  constructor(config: FeatureFlagConfig, userId?: string) {
    this.config = config;

    // Assign user to consistent bucket (0-99) based on userId hash
    if (userId) {
      this.userBucket = this.hashUserId(userId) % 100;
    } else {
      this.userBucket = Math.floor(Math.random() * 100);
    }
  }

  /**
   * Check if V2 agents are enabled for this user
   */
  isV2AgentsEnabled(userId?: string): FeatureFlagState {
    // Check user-specific override
    if (userId && userId in this.config.userOverrides) {
      const enabled = this.config.userOverrides[userId];
      return {
        isEnabled: enabled,
        rolloutPercentage: this.config.rolloutPercentage,
        reason: enabled ? 'user_override_enabled' : 'user_override_disabled',
      };
    }

    // Check global flag
    if (!this.config.enableV2Agents) {
      return {
        isEnabled: false,
        rolloutPercentage: this.config.rolloutPercentage,
        reason: 'global_flag_disabled',
      };
    }

    // Check rollout percentage
    const isInRollout = this.userBucket < this.config.rolloutPercentage;

    return {
      isEnabled: isInRollout,
      rolloutPercentage: this.config.rolloutPercentage,
      reason: isInRollout ? 'in_rollout_bucket' : 'outside_rollout_bucket',
    };
  }

  /**
   * Get backend URL for V2 agents
   */
  getBackendUrl(): string {
    return this.config.backendUrl;
  }

  /**
   * Update rollout percentage
   */
  setRolloutPercentage(percentage: number) {
    if (percentage < 0 || percentage > 100) {
      throw new Error('Rollout percentage must be 0-100');
    }
    this.config.rolloutPercentage = percentage;
  }

  /**
   * Set user override
   */
  setUserOverride(userId: string, enabled: boolean) {
    this.config.userOverrides[userId] = enabled;
  }

  /**
   * Remove user override
   */
  removeUserOverride(userId: string) {
    delete this.config.userOverrides[userId];
  }

  /**
   * Simple hash function for user ID to bucket
   */
  private hashUserId(userId: string): number {
    let hash = 0;
    for (let i = 0; i < userId.length; i++) {
      const char = userId.charCodeAt(i);
      hash = ((hash << 5) - hash) + char;
      hash = hash & hash; // Convert to 32-bit integer
    }
    return Math.abs(hash);
  }

  /**
   * Get diagnostics info
   */
  getDiagnostics() {
    return {
      globalEnabled: this.config.enableV2Agents,
      rolloutPercentage: this.config.rolloutPercentage,
      userBucket: this.userBucket,
      userOverrideCount: Object.keys(this.config.userOverrides).length,
      backendUrl: this.config.backendUrl,
    };
  }
}

/**
 * Create feature flags from Chrome storage or defaults
 */
export async function createFeatureFlags(userId?: string): Promise<FeatureFlags> {
  // Get config from Chrome storage with defaults
  const config: FeatureFlagConfig = await new Promise((resolve) => {
    chrome.storage.local.get(
      {
        featureFlags: {
          enableV2Agents: true, // Enabled for MVP
          rolloutPercentage: 100, // 100% rollout
          userOverrides: {},
          backendUrl: 'http://127.0.0.1:11436', // Backend running locally
        },
      },
      (result) => {
        resolve(result.featureFlags as FeatureFlagConfig);
      }
    );
  });

  return new FeatureFlags(config, userId);
}

/**
 * Update feature flags in Chrome storage
 */
export async function updateFeatureFlags(flags: FeatureFlags): Promise<void> {
  return new Promise((resolve) => {
    chrome.storage.local.set({
      featureFlags: {
        enableV2Agents: flags['config'].enableV2Agents,
        rolloutPercentage: flags['config'].rolloutPercentage,
        userOverrides: flags['config'].userOverrides,
        backendUrl: flags['config'].backendUrl,
      },
    }, resolve);
  });
}
