/**
 * Centralized backend configuration
 * All API calls should use this instead of hardcoding URLs
 */

export const BACKEND_CONFIG = {
  // Base URL - can be overridden via environment or settings
  baseUrl: 'http://localhost:8080',

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
      process: '/api/v2/phase5/process',
      clarificationRespond: '/api/v2/clarification/respond',
    },

    // Context binding
    context: {
      aboutMe: '/api/v2/about-me',
      conversations: '/api/v2/conversations',
      contacts: '/api/v2/contacts',
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
      timeout: 5000,
    });
    return response.ok;
  } catch (err) {
    console.warn('[Backend] Connection check failed:', err);
    return false;
  }
}
