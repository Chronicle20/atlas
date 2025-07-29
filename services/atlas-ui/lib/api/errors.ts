/**
 * Error transformation utilities for Atlas UI
 * 
 * This module provides utilities to transform API errors into user-friendly messages
 * and handle different error scenarios consistently across the application.
 */

import type { 
  ApiError, 
  ApiErrorType, 
  NetworkError, 
  ValidationError
} from '@/types/api/errors';
import {
  isApiError,
  isNetworkError,
  isValidationError,
  isAuthenticationError,
  isAuthorizationError,
  isNotFoundError,
  isServerError,
  createErrorFromUnknown
} from '@/types/api/errors';

/**
 * Configuration for error message transformation
 */
export interface ErrorTransformConfig {
  /** Whether to include technical details in user messages */
  includeTechnicalDetails?: boolean;
  /** Default message for unknown errors */
  defaultMessage?: string;
  /** Context information to include in error messages */
  context?: string;
}

/**
 * Default error messages mapped by error code
 */
const DEFAULT_ERROR_MESSAGES: Record<string, string> = {
  // Network and connectivity errors
  NETWORK_ERROR: 'Unable to connect to the server. Please check your internet connection and try again.',
  
  // Authentication and authorization errors
  AUTHENTICATION_ERROR: 'Please sign in to continue.',
  AUTHORIZATION_ERROR: 'You don\'t have permission to perform this action.',
  
  // Data and validation errors
  VALIDATION_ERROR: 'Please check your input and try again.',
  NOT_FOUND: 'The requested resource was not found.',
  
  // Server errors
  SERVER_ERROR: 'An unexpected error occurred. Please try again later.',
  
  // Unknown errors
  UNKNOWN_ERROR: 'An unexpected error occurred. Please try again.',
};

/**
 * HTTP status code to user-friendly message mapping
 */
const HTTP_STATUS_MESSAGES: Record<number, string> = {
  // Client errors (4xx)
  400: 'Invalid request. Please check your input and try again.',
  401: 'Please sign in to continue.',
  403: 'You don\'t have permission to perform this action.',
  404: 'The requested resource was not found.',
  405: 'This action is not allowed.',
  408: 'Request timeout. Please try again.',
  409: 'This action conflicts with the current state. Please refresh and try again.',
  410: 'This resource is no longer available.',
  422: 'Unable to process your request. Please check your input.',
  429: 'Too many requests. Please wait a moment and try again.',
  
  // Server errors (5xx)
  500: 'An internal server error occurred. Please try again later.',
  501: 'This feature is not currently supported.',
  502: 'Server is temporarily unavailable. Please try again later.',
  503: 'Service is temporarily unavailable. Please try again later.',
  504: 'Request timeout. Please try again later.',
  505: 'HTTP version not supported.',
};

/**
 * Transform an API error into a user-friendly message
 */
export function transformApiError(
  error: ApiError | ApiErrorType,
  config: ErrorTransformConfig = {}
): string {
  const { 
    includeTechnicalDetails = false, 
    defaultMessage = DEFAULT_ERROR_MESSAGES.UNKNOWN_ERROR,
    context 
  } = config;

  let message = '';

  // First try to get a message based on the error code
  if (error.code && DEFAULT_ERROR_MESSAGES[error.code]) {
    message = DEFAULT_ERROR_MESSAGES[error.code]!;
  }
  // Then try to get a message based on HTTP status code
  else if (HTTP_STATUS_MESSAGES[error.statusCode]) {
    message = HTTP_STATUS_MESSAGES[error.statusCode]!;
  }
  // Fall back to the error's own message or default
  else {
    message = error.message ?? defaultMessage;
  }

  // Add context if provided
  if (context) {
    message = `${context}: ${message}`;
  }

  // Include technical details if requested (for debugging)
  if (includeTechnicalDetails && error.statusCode) {
    message += ` (Status: ${error.statusCode})`;
  }

  return message;
}

/**
 * Transform a validation error into a user-friendly message with field details
 */
export function transformValidationError(
  error: ValidationError,
  config: ErrorTransformConfig = {}
): string {
  let message = transformApiError(error, config);

  // If we have field errors, create a more specific message
  if (error.fieldErrors && Object.keys(error.fieldErrors).length > 0) {
    const fieldMessages = Object.entries(error.fieldErrors)
      .map(([field, errors]) => `${field}: ${errors.join(', ')}`)
      .join('; ');
    
    message = `Validation failed: ${fieldMessages}`;
  }

  return message;
}

/**
 * Transform a network error into a user-friendly message
 */
export function transformNetworkError(
  error: NetworkError,
  config: ErrorTransformConfig = {}
): string {
  const baseMessage = transformApiError(error, config);

  // Handle specific network scenarios
  if (error.statusCode === 0) {
    return 'Unable to connect to the server. Please check your internet connection.';
  }
  
  if (error.statusCode === 408) {
    return 'Request timed out. Please try again.';
  }

  return baseMessage;
}

/**
 * Transform any error (including unknown errors) into a user-friendly message
 */
export function transformError(
  error: unknown,
  config: ErrorTransformConfig = {}
): string {
  // Handle known API errors
  if (isApiError(error)) {
    if (isValidationError(error)) {
      return transformValidationError(error, config);
    }
    if (isNetworkError(error)) {
      return transformNetworkError(error, config);
    }
    return transformApiError(error, config);
  }

  // Handle unknown errors
  const unknownError = createErrorFromUnknown(error, config.context);
  return config.defaultMessage || unknownError.message;
}

/**
 * Get a user-friendly error message from an HTTP status code
 */
export function getMessageFromStatusCode(
  statusCode: number,
  fallbackMessage?: string
): string {
  return HTTP_STATUS_MESSAGES[statusCode] || fallbackMessage || 'An error occurred';
}

/**
 * Check if an error indicates a temporary issue that might be resolved by retrying
 */
export function isRetryableError(error: unknown): boolean {
  if (!isApiError(error)) {
    return false;
  }

  // Network errors might be retryable
  if (isNetworkError(error)) {
    return true;
  }

  // Specific HTTP status codes that indicate temporary issues
  const retryableStatusCodes = [408, 429, 500, 502, 503, 504];
  return retryableStatusCodes.includes(error.statusCode);
}

/**
 * Check if an error indicates the user needs to authenticate
 */
export function requiresAuthentication(error: unknown): boolean {
  return isAuthenticationError(error);
}

/**
 * Check if an error indicates insufficient permissions
 */
export function requiresAuthorization(error: unknown): boolean {
  return isAuthorizationError(error);
}

/**
 * Get suggested actions for an error
 */
export function getErrorActions(error: unknown): string[] {
  const actions: string[] = [];

  if (requiresAuthentication(error)) {
    actions.push('Sign in to your account');
  } else if (requiresAuthorization(error)) {
    actions.push('Contact an administrator for access');
  } else if (isNotFoundError(error)) {
    actions.push('Check the URL and try again');
    actions.push('Return to the previous page');
  } else if (isValidationError(error)) {
    actions.push('Review your input and try again');
  } else if (isRetryableError(error)) {
    actions.push('Try again in a few moments');
    actions.push('Check your internet connection');
  } else if (isServerError(error)) {
    actions.push('Try again later');
    actions.push('Contact support if the problem persists');
  }

  return actions;
}

/**
 * Error context for consistent error handling across components
 */
export interface ErrorContext {
  /** Component or feature where the error occurred */
  component?: string;
  /** Action being performed when error occurred */
  action?: string;
  /** Additional context data */
  data?: Record<string, unknown>;
}

/**
 * Create error transformation config with context
 */
export function createErrorConfig(
  context: ErrorContext,
  options: Partial<ErrorTransformConfig> = {}
): ErrorTransformConfig {
  const contextString = [
    context.component && `in ${context.component}`,
    context.action && `while ${context.action}`
  ].filter(Boolean).join(' ');

  const result: ErrorTransformConfig = {
    includeTechnicalDetails: process.env.NODE_ENV === 'development',
    defaultMessage: 'An unexpected error occurred',
    ...options
  };
  
  if (contextString) {
    result.context = contextString;
  }
  
  return result;
}

/**
 * Utility for logging errors with context
 */
export function logError(
  error: unknown,
  context: ErrorContext = {}
): void {
  if (process.env.NODE_ENV === 'development') {
    console.error('Error occurred:', {
      error,
      context,
      message: transformError(error),
      timestamp: new Date().toISOString()
    });
  }

  // In production, you might want to send this to an error tracking service
  // Example: Sentry, LogRocket, etc.
}

/**
 * Sanitize error data to remove sensitive information before logging
 */
export function sanitizeErrorData(data: Record<string, unknown>): Record<string, unknown> {
  const sanitized = { ...data };
  
  // Remove common sensitive fields
  const sensitiveFields = [
    'password', 
    'token', 
    'secret', 
    'key', 
    'authorization',
    'cookie',
    'session'
  ];

  for (const field of sensitiveFields) {
    if (field in sanitized) {
      sanitized[field] = '[REDACTED]';
    }
  }

  return sanitized;
}