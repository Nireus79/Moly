/**
 * Centralized backend configuration
 * All API calls should use this instead of hardcoding URLs
 */

import { enginePort } from '@/utils/platform';

export const BACKEND_CONFIG = {
  // Base URL - can be overridden via environment or settings
  baseUrl: `http://localhost:${enginePort()}`,

  // API endpoints
  endpoints: {
    // Auth
    auth: {
      register: '/api/auth/register',
      login: '/api/auth/login',
      verify: '/api/auth/verify',
      logout: '/api/auth/logout',
    },

    // Phase 5 orchestration
    phase5: {
      process: '/api/phase5/process',
      clarificationRespond: '/api/clarification/respond',
    },

    // Context binding
    context: {
      aboutMe: '/api/about-me',
      conversations: '/api/conversations',
      contacts: '/api/contacts',
    },

    // User account management
    account: {
      deleteProfile: '/api/user/delete',
    },

    // Conflict resolution
    conflicts: {
      resolve: '/api/conflicts/resolve',
    },

    // Health check
    status: '/api/status',
  },
};

/**
 * Get full URL for an endpoint
 */
export function getEndpointUrl(endpoint: string): string {
  return `${BACKEND_CONFIG.baseUrl}${endpoint}`;
}

/**
 * Update backend base URL (e.g., from settings)
 */
export function setBackendBaseUrl(url: string): void {
  if (!url.startsWith('http')) {
    throw new Error('Backend URL must start with http:// or https://');
  }
  BACKEND_CONFIG.baseUrl = url;
}

/**
 * Get current backend base URL
 */
export function getBackendBaseUrl(): string {
  return BACKEND_CONFIG.baseUrl;
}

/**
 * Verify backend is reachable
 */
export async function verifyBackendConnection(): Promise<boolean> {
  try {
    const response = await fetch(getEndpointUrl(BACKEND_CONFIG.endpoints.status), {
      method: 'GET',
      signal: AbortSignal.timeout(5000), // fetch has no `timeout` option: the old line was ignored
    });
    return response.ok;
  } catch (err) {
    console.warn('[Backend] Connection check failed:', err);
    return false;
  }
}
