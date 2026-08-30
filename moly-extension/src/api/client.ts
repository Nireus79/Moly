/**
 * HTTP Client for API calls
 */

export interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE';
  headers?: Record<string, string>;
  body?: any;
  params?: Record<string, any>;
  timeout?: number;
}

export interface ApiError extends Error {
  status?: number;
  response?: any;
}

const DEFAULT_TIMEOUT = 30000; // 30 seconds

/**
 * Make HTTP request to Claude API
 */
export async function makeApiRequest<T>(
  url: string,
  options: RequestOptions = {},
): Promise<T> {
  const {
    method = 'GET',
    headers = {},
    body,
    params,
    timeout = DEFAULT_TIMEOUT,
  } = options;

  // Build URL with query parameters
  let finalUrl = url;
  if (params) {
    const queryString = new URLSearchParams(
      Object.entries(params).reduce((acc, [key, val]) => {
        acc[key] = String(val);
        return acc;
      }, {} as Record<string, string>),
    ).toString();
    if (queryString) {
      finalUrl += `?${queryString}`;
    }
  }

  // Prepare request options
  const fetchOptions: RequestInit = {
    method,
    headers: {
      'Content-Type': 'application/json',
      ...headers,
    },
  };

  if (body && (method === 'POST' || method === 'PUT')) {
    fetchOptions.body = JSON.stringify(body);
  }

  // Create abort controller for timeout
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), timeout);

  try {
    const response = await fetch(finalUrl, {
      ...fetchOptions,
      signal: controller.signal,
    });

    clearTimeout(timeoutId);

    if (!response.ok) {
      const error: ApiError = new Error(`HTTP ${response.status}`);
      error.status = response.status;
      error.response = await response.json().catch(() => null);
      throw error;
    }

    return await response.json();
  } catch (error) {
    clearTimeout(timeoutId);

    if (error instanceof TypeError && error.message === 'Failed to fetch') {
      throw new Error('Network error - check your connection');
    }

    if (error instanceof Error && error.message === 'AbortError') {
      throw new Error('Request timeout - please try again');
    }

    throw error;
  }
}

/**
 * Simple HTTP client
 */
export const apiClient = {
  async get<T>(url: string, options?: RequestOptions): Promise<T> {
    return makeApiRequest<T>(url, { ...options, method: 'GET' });
  },

  async post<T>(url: string, body?: any, options?: RequestOptions): Promise<T> {
    return makeApiRequest<T>(url, { ...options, method: 'POST', body });
  },

  async put<T>(url: string, body?: any, options?: RequestOptions): Promise<T> {
    return makeApiRequest<T>(url, { ...options, method: 'PUT', body });
  },

  async delete<T>(url: string, options?: RequestOptions): Promise<T> {
    return makeApiRequest<T>(url, { ...options, method: 'DELETE' });
  },
};
