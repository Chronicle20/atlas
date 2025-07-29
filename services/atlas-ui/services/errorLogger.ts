/**
 * Error logging service for production error tracking and monitoring
 * 
 * This service handles:
 * - Sanitization of sensitive data from error reports
 * - Structured error logging with context
 * - Integration with monitoring services (ready for future implementation)
 * - Error rate limiting to prevent spam
 */

import type { ErrorInfo } from 'react';

interface ErrorContext {
  userId?: string;
  tenantId?: string;
  userAgent?: string;
  url?: string;
  timestamp?: string;
  sessionId?: string;
  buildVersion?: string;
}

interface ErrorReport {
  error: {
    name: string;
    message: string;
    stack?: string;
  };
  errorInfo?: {
    componentStack?: string;
  };
  context: ErrorContext;
  severity: 'low' | 'medium' | 'high' | 'critical';
  fingerprint: string;
}

interface LoggerOptions {
  maxReportsPerMinute?: number;
  enableConsoleLogging?: boolean;
  enableRemoteLogging?: boolean;
  remoteEndpoint?: string;
  apiKey?: string;
}

/**
 * Sanitizes sensitive data from error messages and stack traces
 */
function sanitizeError(error: Error): { name: string; message: string; stack?: string } {
  const sensitivePatterns = [
    // API keys and tokens
    /[Aa]pi[_-]?[Kk]ey[:\s=]+[^\s\n]+/g,
    /[Tt]oken[:\s=]+[^\s\n]+/g,
    /[Bb]earer\s+[^\s\n]+/g,
    // Passwords
    /[Pp]assword[:\s=]+[^\s\n]+/g,
    /[Pp]wd[:\s=]+[^\s\n]+/g,
    // Email addresses (partial sanitization)
    /[\w\.-]+@[\w\.-]+\.\w+/g,
    // Credit card patterns
    /\b\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b/g,
    // SSN patterns
    /\b\d{3}[-\s]?\d{2}[-\s]?\d{4}\b/g,
  ];

  let sanitizedMessage = error.message;
  let sanitizedStack = error.stack;

  // Apply sanitization patterns
  sensitivePatterns.forEach(pattern => {
    sanitizedMessage = sanitizedMessage.replace(pattern, '[REDACTED]');
    if (sanitizedStack) {
      sanitizedStack = sanitizedStack.replace(pattern, '[REDACTED]');
    }
  });

  return {
    name: error.name,
    message: sanitizedMessage,
    ...(sanitizedStack ? { stack: sanitizedStack } : {}),
  };
}

/**
 * Generates a fingerprint for error deduplication
 */
function generateErrorFingerprint(error: Error, errorInfo?: ErrorInfo): string {
  const components = [
    error.name,
    error.message.split('\n')[0], // First line only
    errorInfo?.componentStack?.split('\n')[0], // First component
  ].filter(Boolean);

  return btoa(components.join('|')).slice(0, 16);
}

/**
 * Gathers context information for error reporting
 */
function gatherContext(): ErrorContext {
  const context: ErrorContext = {
    timestamp: new Date().toISOString(),
    buildVersion: process.env.NEXT_PUBLIC_BUILD_VERSION || 'unknown',
  };

  // Add optional properties only if they exist
  if (typeof window !== 'undefined') {
    context.url = window.location.href;
  }

  if (typeof navigator !== 'undefined') {
    context.userAgent = navigator.userAgent;
  }

  // Add session ID if available (from localStorage, sessionStorage, or cookie)
  if (typeof window !== 'undefined') {
    try {
      const sessionId = window.sessionStorage.getItem('sessionId') || 
                       window.localStorage.getItem('sessionId');
      if (sessionId) {
        context.sessionId = sessionId;
      }
    } catch {
      // Ignore storage errors
    }
  }

  return context;
}

class ErrorLogger {
  private options: Required<LoggerOptions>;
  private reportCounts = new Map<string, { count: number; lastReset: number }>();

  constructor(options: LoggerOptions = {}) {
    this.options = {
      maxReportsPerMinute: 10,
      enableConsoleLogging: process.env.NODE_ENV === 'development',
      enableRemoteLogging: process.env.NODE_ENV === 'production',
      remoteEndpoint: process.env.NEXT_PUBLIC_ERROR_ENDPOINT || '',
      apiKey: process.env.NEXT_PUBLIC_ERROR_API_KEY || '',
      ...options,
    };
  }

  /**
   * Checks if we should rate limit this error report
   */
  private shouldRateLimit(fingerprint: string): boolean {
    const now = Date.now();
    const minute = Math.floor(now / 60000);
    
    const existing = this.reportCounts.get(fingerprint);
    
    if (!existing || existing.lastReset !== minute) {
      this.reportCounts.set(fingerprint, { count: 1, lastReset: minute });
      return false;
    }
    
    if (existing.count >= this.options.maxReportsPerMinute) {
      return true;
    }
    
    existing.count++;
    return false;
  }

  /**
   * Logs error to console (development mode)
   */
  private logToConsole(report: ErrorReport): void {
    if (!this.options.enableConsoleLogging) return;

    const { error, context, severity } = report;
    
    console.group(`🚨 Error Report [${severity.toUpperCase()}]`);
    console.error('Error:', error);
    console.log('Context:', context);
    console.log('Fingerprint:', report.fingerprint);
    
    if (report.errorInfo?.componentStack) {
      console.log('Component Stack:', report.errorInfo.componentStack);
    }
    
    console.groupEnd();
  }

  /**
   * Sends error report to remote monitoring service
   */
  private async logToRemote(report: ErrorReport): Promise<void> {
    if (!this.options.enableRemoteLogging || !this.options.remoteEndpoint) {
      return;
    }

    try {
      const response = await fetch(this.options.remoteEndpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(this.options.apiKey ? { 'Authorization': `Bearer ${this.options.apiKey}` } : {}),
        },
        body: JSON.stringify(report),
      });

      if (!response.ok) {
        throw new Error(`Failed to log error: ${response.status}`);
      }
    } catch (remoteError) {
      // Avoid infinite loops - just log to console
      console.warn('Failed to send error report to remote service:', remoteError);
    }
  }

  /**
   * Determines error severity based on error type and context
   */
  private determineSeverity(error: Error, errorInfo?: ErrorInfo): ErrorReport['severity'] {
    // Critical errors
    if (error.name === 'ChunkLoadError' || 
        error.message.includes('Loading chunk') ||
        error.message.includes('Loading CSS chunk')) {
      return 'critical';
    }

    // High severity errors
    if (error.name === 'TypeError' && error.message.includes('Cannot read prop')) {
      return 'high';
    }

    // Medium severity for component errors
    if (errorInfo?.componentStack) {
      return 'medium';
    }

    // Default to low
    return 'low';
  }

  /**
   * Main method to log an error with full context and sanitization
   */
  async logError(
    error: Error, 
    errorInfo?: ErrorInfo,
    additionalContext?: Partial<ErrorContext>
  ): Promise<void> {
    try {
      const sanitizedError = sanitizeError(error);
      const context = { ...gatherContext(), ...additionalContext };
      const fingerprint = generateErrorFingerprint(error, errorInfo);
      const severity = this.determineSeverity(error, errorInfo);

      // Check rate limiting
      if (this.shouldRateLimit(fingerprint)) {
        return;
      }

      const report: ErrorReport = {
        error: sanitizedError,
        context,
        severity,
        fingerprint,
        ...(errorInfo && errorInfo.componentStack 
          ? { errorInfo: { componentStack: errorInfo.componentStack } } 
          : {}),
      };

      // Log to console (development)
      this.logToConsole(report);

      // Log to remote service (production)
      await this.logToRemote(report);

    } catch (loggingError) {
      // Prevent infinite loops
      console.warn('Error while logging error:', loggingError);
    }
  }

  /**
   * Logs a user action for context in subsequent error reports
   */
  logUserAction(action: string, details?: Record<string, unknown>): void {
    if (this.options.enableConsoleLogging) {
      console.log('User Action:', action, details);
    }
    
    // Store recent actions for context in error reports
    if (typeof window !== 'undefined') {
      try {
        const actions = JSON.parse(window.sessionStorage.getItem('recentActions') || '[]');
        actions.push({
          action,
          details,
          timestamp: new Date().toISOString(),
        });
        
        // Keep only last 10 actions
        const recentActions = actions.slice(-10);
        window.sessionStorage.setItem('recentActions', JSON.stringify(recentActions));
      } catch {
        // Ignore storage errors
      }
    }
  }

  /**
   * Logs performance issues that might lead to errors
   */
  logPerformanceIssue(metric: string, value: number, threshold: number): void {
    if (value > threshold) {
      console.warn(`Performance issue detected: ${metric} = ${value} (threshold: ${threshold})`);
      
      // Could send to monitoring service
      if (this.options.enableRemoteLogging) {
        // This could be expanded to send performance data
      }
    }
  }
}

// Create singleton instance
export const errorLogger = new ErrorLogger();

// Export types for use in other modules
export type { ErrorContext, ErrorReport, LoggerOptions };