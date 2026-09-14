/**
 * Centralized logging system with structured output
 * Features: log levels, colors, timestamps, performance tracking, filtering
 */

export enum LogLevel {
  DEBUG = 0,
  INFO = 1,
  WARN = 2,
  ERROR = 3,
}

interface LogEntry {
  timestamp: Date;
  level: LogLevel;
  source: string;
  message: string;
  data?: any;
  duration?: number; // in ms
  error?: Error;
}

class Logger {
  private minLevel = LogLevel.DEBUG;
  private history: LogEntry[] = [];
  private maxHistorySize = 500;
  private performanceMarkers: Map<string, number> = new Map();

  constructor() {
    // Check localStorage for log level preference
    try {
      const stored = localStorage.getItem('moly_log_level');
      if (stored) {
        this.minLevel = parseInt(stored, 10);
      }
    } catch {
      // Ignore
    }
  }

  private getColor(level: LogLevel): string {
    switch (level) {
      case LogLevel.DEBUG:
        return '#999';
      case LogLevel.INFO:
        return '#0066cc';
      case LogLevel.WARN:
        return '#ff9900';
      case LogLevel.ERROR:
        return '#cc0000';
    }
  }

  private getLevelName(level: LogLevel): string {
    switch (level) {
      case LogLevel.DEBUG:
        return 'DEBUG';
      case LogLevel.INFO:
        return 'INFO';
      case LogLevel.WARN:
        return 'WARN';
      case LogLevel.ERROR:
        return 'ERROR';
    }
  }

  private formatTime(date: Date): string {
    const h = String(date.getHours()).padStart(2, '0');
    const m = String(date.getMinutes()).padStart(2, '0');
    const s = String(date.getSeconds()).padStart(2, '0');
    const ms = String(date.getMilliseconds()).padStart(3, '0');
    return `${h}:${m}:${s}.${ms}`;
  }

  private log(level: LogLevel, source: string, message: string, data?: any, duration?: number) {
    if (level < this.minLevel) {
      return;
    }

    const entry: LogEntry = {
      timestamp: new Date(),
      level,
      source,
      message,
      data,
      duration,
    };

    // Store in history
    this.history.push(entry);
    if (this.history.length > this.maxHistorySize) {
      this.history.shift();
    }

    // Format console output
    const time = this.formatTime(entry.timestamp);
    const levelName = this.getLevelName(level);
    const color = this.getColor(level);

    const prefix = `%c${time} [${levelName}] ${source}`;
    const args = [prefix, `color: ${color}; font-weight: bold`];

    if (duration !== undefined) {
      console.log(...args, `${message} (+${duration}ms)`, data || '');
    } else {
      console.log(...args, message, data || '');
    }

    // Also log to console.error for ERROR level
    if (level === LogLevel.ERROR) {
      console.error(`[${source}] ${message}`, data);
    }
  }

  // Public methods
  debug(source: string, message: string, data?: any) {
    this.log(LogLevel.DEBUG, source, message, data);
  }

  info(source: string, message: string, data?: any) {
    this.log(LogLevel.INFO, source, message, data);
  }

  warn(source: string, message: string, data?: any) {
    this.log(LogLevel.WARN, source, message, data);
  }

  error(source: string, message: string, error?: Error | any, data?: any) {
    const entry: LogEntry = {
      timestamp: new Date(),
      level: LogLevel.ERROR,
      source,
      message,
      data,
      error: error instanceof Error ? error : new Error(String(error)),
    };

    this.history.push(entry);
    if (this.history.length > this.maxHistorySize) {
      this.history.shift();
    }

    const time = this.formatTime(entry.timestamp);
    const prefix = `%c${time} [ERROR] ${source}`;
    console.error(prefix, `color: #cc0000; font-weight: bold`, message);
    if (error) {
      console.error('Error details:', error);
    }
    if (data) {
      console.error('Data:', data);
    }
  }

  // Performance tracking
  startTimer(label: string): void {
    this.performanceMarkers.set(label, performance.now());
  }

  endTimer(source: string, label: string, message?: string): number {
    const start = this.performanceMarkers.get(label);
    if (!start) {
      this.warn(source, `Timer "${label}" not found`);
      return 0;
    }

    const duration = Math.round(performance.now() - start);
    this.performanceMarkers.delete(label);

    this.log(
      duration > 1000 ? LogLevel.WARN : LogLevel.DEBUG,
      source,
      message || `${label} completed`,
      undefined,
      duration
    );

    return duration;
  }

  // User actions
  trackAction(source: string, action: string, details?: any) {
    this.info(source, `🎯 ACTION: ${action}`, details);
  }

  // API calls
  trackApiCall(method: string, endpoint: string, status?: number, duration?: number) {
    const statusEmoji = !status ? '🔄' : status < 400 ? '✅' : '❌';
    const message = `${statusEmoji} ${method} ${endpoint} (${status || 'pending'})`;

    if (duration) {
      this.log(
        status && status >= 400 ? LogLevel.WARN : LogLevel.DEBUG,
        'API',
        message,
        undefined,
        duration
      );
    } else {
      this.debug('API', message);
    }
  }

  // State changes
  trackStateChange(source: string, stateKey: string, oldValue: any, newValue: any) {
    this.debug(source, `📊 STATE: ${stateKey}`, {
      from: oldValue,
      to: newValue,
    });
  }

  // Error tracking
  trackError(source: string, errorType: string, message: string, error?: Error) {
    this.error(source, `🚨 ${errorType}: ${message}`, error);
  }

  // Get history
  getHistory(
    filter?: {
      level?: LogLevel;
      source?: string;
      limit?: number;
    }
  ): LogEntry[] {
    let filtered = [...this.history];

    if (filter?.level !== undefined) {
      filtered = filtered.filter(e => e.level >= filter.level!);
    }

    if (filter?.source) {
      filtered = filtered.filter(e => e.source.includes(filter.source!));
    }

    if (filter?.limit) {
      filtered = filtered.slice(-filter.limit);
    }

    return filtered;
  }

  // Clear history
  clearHistory() {
    this.history = [];
  }

  // Export history as JSON
  exportHistory(): string {
    return JSON.stringify(this.history, null, 2);
  }

  // Set log level
  setLogLevel(level: LogLevel) {
    this.minLevel = level;
    try {
      localStorage.setItem('moly_log_level', String(level));
    } catch {
      // Ignore
    }
    this.info('Logger', `Log level changed to ${this.getLevelName(level)}`);
  }

  // Get current log level
  getLogLevel(): LogLevel {
    return this.minLevel;
  }

  // Console commands
  static installGlobalCommands() {
    (window as any).molyLogger = {
      debug: (source: string, msg: string, data?: any) => logger.debug(source, msg, data),
      info: (source: string, msg: string, data?: any) => logger.info(source, msg, data),
      warn: (source: string, msg: string, data?: any) => logger.warn(source, msg, data),
      error: (source: string, msg: string, err?: any) => logger.error(source, msg, err),
      history: (filter?: any) => logger.getHistory(filter),
      clear: () => logger.clearHistory(),
      export: () => logger.exportHistory(),
      setLevel: (level: LogLevel) => logger.setLogLevel(level),
      getLevel: () => logger.getLogLevel(),
    };

    console.log(
      '%cMoly Logger Commands Available',
      'color: #0066cc; font-weight: bold; font-size: 14px',
      '\nUsage in browser console:',
      'molyLogger.info("Source", "message", {data})',
      'molyLogger.history({level: 1, limit: 50})',
      'molyLogger.export()',
      'molyLogger.setLevel(0) // 0=DEBUG, 1=INFO, 2=WARN, 3=ERROR'
    );
  }
}

// Export singleton instance
export const logger = new Logger();

// Install global commands on load
if (typeof window !== 'undefined') {
  Logger.installGlobalCommands();
}
