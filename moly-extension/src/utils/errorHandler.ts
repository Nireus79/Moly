/**
 * Centralized error handling and user-facing error messages
 */

export class ApiError extends Error {
  constructor(
    public statusCode: number,
    public endpoint: string,
    message: string,
    public retryable: boolean = true
  ) {
    super(message);
    this.name = 'ApiError';
  }

  isRetryable(): boolean {
    // 5xx errors and timeout errors are retryable
    return this.retryable && (this.statusCode >= 500 || this.statusCode === 408);
  }

  getUserMessage(): string {
    switch (this.statusCode) {
      case 400:
        return 'Invalid request. Please check your input.';
      case 401:
      case 403:
        return 'Authentication failed. Please log in again.';
      case 404:
        return 'Resource not found.';
      case 408:
      case 504:
        return 'Request timed out. Please try again.';
      case 500:
      case 502:
      case 503:
        return 'Server error. Please try again later.';
      default:
        if (this.statusCode >= 500) {
          return 'Server error. Please try again later.';
        }
        return this.message;
    }
  }
}

/**
 * Validate API response structure
 */
export function validateResponse<T>(
  data: any,
  requiredFields: string[]
): data is T {
  if (!data || typeof data !== 'object') {
    return false;
  }

  for (const field of requiredFields) {
    if (!(field in data)) {
      console.warn(`[ApiError] Response missing required field: ${field}`);
      return false;
    }
  }

  return true;
}

/**
 * Handle API errors consistently
 */
export async function handleApiError(
  response: Response,
  endpoint: string
): Promise<never> {
  let errorMessage = `API Error: ${response.statusText}`;

  try {
    const data = await response.json();
    errorMessage = data.error || data.message || errorMessage;
  } catch {
    // Response wasn't JSON, use default message
  }

  throw new ApiError(response.status, endpoint, errorMessage);
}

/**
 * Retry logic for failed API calls
 */
export async function retryApiCall<T>(
  fn: () => Promise<T>,
  maxRetries: number = 3,
  delayMs: number = 1000
): Promise<T> {
  let lastError: Error | null = null;

  for (let attempt = 1; attempt <= maxRetries; attempt++) {
    try {
      console.log(`[Retry] Attempt ${attempt}/${maxRetries}`);
      return await fn();
    } catch (err) {
      lastError = err instanceof Error ? err : new Error(String(err));

      // Only retry if it's retryable
      if (err instanceof ApiError && !err.isRetryable()) {
        throw err;
      }

      // Wait before retrying (exponential backoff)
      if (attempt < maxRetries) {
        const wait = delayMs * Math.pow(2, attempt - 1);
        console.log(`[Retry] Waiting ${wait}ms before retry...`);
        await new Promise(resolve => setTimeout(resolve, wait));
      }
    }
  }

  throw lastError || new Error('Max retries exceeded');
}

/**
 * Log error with context
 */
export function logError(context: string, error: any): void {
  if (error instanceof ApiError) {
    console.error(`[${context}] ${error.endpoint}: ${error.message} (${error.statusCode})`);
  } else if (error instanceof Error) {
    console.error(`[${context}] ${error.name}: ${error.message}`);
  } else {
    console.error(`[${context}]`, error);
  }
}

/**
 * Safe JSON parse
 */
export function safeJsonParse<T>(json: string, fallback: T): T {
  try {
    return JSON.parse(json);
  } catch (err) {
    console.warn('[Parse] Invalid JSON, using fallback:', err);
    return fallback;
  }
}
