/**
 * Error Reporter for Moly Extension
 * Captures, formats, and logs errors with full context
 * Stores errors locally for debugging and analytics
 */

export interface ErrorLog {
  id: string;
  timestamp: string;
  level: 'error' | 'warning' | 'info';
  component: string;
  message: string;
  stack?: string;
  context: Record<string, any>;
  userAgent: string;
  url: string;
  session: string;
}

export interface ErrorReport {
  errors: ErrorLog[];
  totalCount: number;
  lastError?: ErrorLog;
  lastClearTime?: string;
}

class ErrorReporter {
  private storageKey = 'moly_error_logs';
  private sessionId: string;
  private maxErrors = 100; // Keep last 100 errors

  constructor() {
    this.sessionId = this.generateSessionId();
    this.initializeErrorHandlers();
  }

  /**
   * Generate unique session ID for this extension session
   */
  private generateSessionId(): string {
    return `session_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  /**
   * Initialize global error handlers
   */
  private initializeErrorHandlers(): void {
    // Handle uncaught errors
    window.addEventListener('error', (event) => {
      this.captureError({
        component: 'window',
        message: event.message,
        stack: event.error?.stack,
        context: {
          filename: event.filename,
          lineno: event.lineno,
          colno: event.colno,
        },
        level: 'error',
      });
    });

    // Handle unhandled promise rejections
    window.addEventListener('unhandledrejection', (event) => {
      this.captureError({
        component: 'promise',
        message: event.reason?.message || String(event.reason),
        stack: event.reason?.stack,
        context: {
          reason: event.reason,
        },
        level: 'error',
      });
    });

    console.info('[ErrorReporter] Initialized with session:', this.sessionId);
  }

  /**
   * Capture an error with full context
   */
  public captureError(options: {
    component: string;
    message: string;
    stack?: string;
    context?: Record<string, any>;
    level?: 'error' | 'warning' | 'info';
  }): void {
    const errorLog: ErrorLog = {
      id: this.generateErrorId(),
      timestamp: new Date().toISOString(),
      level: options.level || 'error',
      component: options.component,
      message: options.message,
      stack: options.stack,
      context: options.context || {},
      userAgent: navigator.userAgent,
      url: window.location.href,
      session: this.sessionId,
    };

    this.storeError(errorLog);
    this.logToConsole(errorLog);
  }

  /**
   * Capture API errors specifically
   */
  public captureApiError(options: {
    endpoint: string;
    method: string;
    status?: number;
    statusText?: string;
    error: any;
    context?: Record<string, any>;
  }): void {
    this.captureError({
      component: 'api',
      message: `${options.method} ${options.endpoint} failed${options.status ? ` (${options.status} ${options.statusText})` : ''}`,
      stack: options.error?.stack,
      context: {
        endpoint: options.endpoint,
        method: options.method,
        status: options.status,
        statusText: options.statusText,
        errorMessage: options.error?.message,
        errorName: options.error?.name,
        ...options.context,
      },
      level: 'error',
    });
  }

  /**
   * Capture provider errors (Claude, OpenAI, Ollama)
   */
  public captureProviderError(options: {
    provider: string;
    error: any;
    context?: Record<string, any>;
  }): void {
    this.captureError({
      component: `provider_${options.provider.toLowerCase()}`,
      message: `Provider error: ${options.error?.message || String(options.error)}`,
      stack: options.error?.stack,
      context: {
        provider: options.provider,
        errorName: options.error?.name,
        errorCode: options.error?.code,
        ...options.context,
      },
      level: 'error',
    });
  }

  /**
   * Store error in local storage
   */
  private storeError(errorLog: ErrorLog): void {
    try {
      const report = this.getErrorReport();

      // Add new error to beginning
      report.errors.unshift(errorLog);
      report.lastError = errorLog;
      report.totalCount++;

      // Keep only last N errors
      if (report.errors.length > this.maxErrors) {
        report.errors = report.errors.slice(0, this.maxErrors);
      }

      localStorage.setItem(this.storageKey, JSON.stringify(report));
    } catch (e) {
      // Storage quota exceeded or localStorage not available
      console.error('[ErrorReporter] Failed to store error:', e);
    }
  }

  /**
   * Get all stored errors
   */
  public getErrorReport(): ErrorReport {
    try {
      const stored = localStorage.getItem(this.storageKey);
      if (stored) {
        return JSON.parse(stored);
      }
    } catch (e) {
      console.error('[ErrorReporter] Failed to retrieve error report:', e);
    }

    return {
      errors: [],
      totalCount: 0,
    };
  }

  /**
   * Get errors for a specific component
   */
  public getErrorsByComponent(component: string): ErrorLog[] {
    const report = this.getErrorReport();
    return report.errors.filter((e) => e.component === component);
  }

  /**
   * Get errors since a specific time
   */
  public getErrorsSince(timestamp: string): ErrorLog[] {
    const report = this.getErrorReport();
    return report.errors.filter((e) => new Date(e.timestamp) >= new Date(timestamp));
  }

  /**
   * Get recent errors
   */
  public getRecentErrors(count: number = 10): ErrorLog[] {
    const report = this.getErrorReport();
    return report.errors.slice(0, count);
  }

  /**
   * Clear all stored errors
   */
  public clearErrors(): void {
    try {
      const report: ErrorReport = {
        errors: [],
        totalCount: 0,
        lastClearTime: new Date().toISOString(),
      };
      localStorage.setItem(this.storageKey, JSON.stringify(report));
      console.info('[ErrorReporter] Errors cleared');
    } catch (e) {
      console.error('[ErrorReporter] Failed to clear errors:', e);
    }
  }

  /**
   * Export errors as JSON for debugging
   */
  public exportErrors(): string {
    const report = this.getErrorReport();
    return JSON.stringify(report, null, 2);
  }

  /**
   * Log error to console with formatting
   */
  private logToConsole(errorLog: ErrorLog): void {
    const style = this.getConsoleStyle(errorLog.level);
    console.group(
      `%c[${errorLog.level.toUpperCase()}] ${errorLog.component}`,
      style,
    );
    console.log('Message:', errorLog.message);
    if (errorLog.stack) {
      console.log('Stack:', errorLog.stack);
    }
    if (Object.keys(errorLog.context).length > 0) {
      console.log('Context:', errorLog.context);
    }
    console.log('Time:', errorLog.timestamp);
    console.log('Session:', errorLog.session);
    console.groupEnd();
  }

  /**
   * Get console styling for error level
   */
  private getConsoleStyle(level: string): string {
    const styles = {
      error: 'color: #d32f2f; font-weight: bold;',
      warning: 'color: #f57c00; font-weight: bold;',
      info: 'color: #1976d2; font-weight: bold;',
    };
    return styles[level as keyof typeof styles] || styles.info;
  }

  /**
   * Generate unique error ID
   */
  private generateErrorId(): string {
    return `error_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  /**
   * Send errors to backend for persistent logging (optional)
   */
  public async sendToBackend(maxErrors: number = 10): Promise<void> {
    try {
      const report = this.getErrorReport();
      const recentErrors = report.errors.slice(0, maxErrors);

      if (recentErrors.length === 0) {
        return;
      }

      const response = await fetch('http://127.0.0.1:11436/api/frontend-errors', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          errors: recentErrors,
          session: this.sessionId,
          extension_version: chrome.runtime.getManifest().version,
        }),
      });

      if (!response.ok) {
        console.warn('[ErrorReporter] Failed to send errors to backend:', response.status);
      }
    } catch (e) {
      console.error('[ErrorReporter] Failed to send errors to backend:', e);
    }
  }

  /**
   * Get error statistics
   */
  public getStatistics(): {
    totalErrors: number;
    byLevel: Record<string, number>;
    byComponent: Record<string, number>;
    oldestError?: string;
    newestError?: string;
  } {
    const report = this.getErrorReport();
    const stats = {
      totalErrors: report.totalCount,
      byLevel: {} as Record<string, number>,
      byComponent: {} as Record<string, number>,
    };

    for (const error of report.errors) {
      stats.byLevel[error.level] = (stats.byLevel[error.level] || 0) + 1;
      stats.byComponent[error.component] = (stats.byComponent[error.component] || 0) + 1;
    }

    if (report.errors.length > 0) {
      stats.newestError = report.errors[0].timestamp;
      stats.oldestError = report.errors[report.errors.length - 1].timestamp;
    }

    return stats;
  }
}

// Export singleton instance
export const errorReporter = new ErrorReporter();
