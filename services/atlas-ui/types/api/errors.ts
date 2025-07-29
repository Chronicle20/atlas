/**
 * Error types and utilities for API error handling.
 * 
 * This module provides standardized error types and utilities for handling
 * errors throughout the Atlas UI application, particularly for API calls
 * and async operations.
 */

/**
 * Base API error interface with common error properties
 */
export interface ApiError {
  /** Human-readable error message */
  message: string;
  /** HTTP status code */
  statusCode: number;
  /** Error code for programmatic handling */
  code: string;
  /** Optional additional error details */
  details?: Record<string, unknown>;
}

/**
 * Network-related error for failed HTTP requests
 */
export interface NetworkError extends ApiError {
  code: 'NETWORK_ERROR';
  /** The original error that caused the network failure */
  cause?: Error;
}

/**
 * Validation error for invalid request data
 */
export interface ValidationError extends ApiError {
  code: 'VALIDATION_ERROR';
  statusCode: 400 | 422;
  /** Field-specific validation errors */
  fieldErrors?: Record<string, string[]>;
}

/**
 * Authentication error for unauthorized requests
 */
export interface AuthenticationError extends ApiError {
  code: 'AUTHENTICATION_ERROR';
  statusCode: 401;
}

/**
 * Authorization error for forbidden resources
 */
export interface AuthorizationError extends ApiError {
  code: 'AUTHORIZATION_ERROR';
  statusCode: 403;
}

/**
 * Not found error for missing resources
 */
export interface NotFoundError extends ApiError {
  code: 'NOT_FOUND';
  statusCode: 404;
  /** The resource that was not found */
  resource?: string;
}

/**
 * Server error for internal server problems
 */
export interface ServerError extends ApiError {
  code: 'SERVER_ERROR';
  statusCode: 500 | 502 | 503 | 504;
}

/**
 * Union type for all possible API errors
 */
export type ApiErrorType = 
  | NetworkError 
  | ValidationError 
  | AuthenticationError 
  | AuthorizationError 
  | NotFoundError 
  | ServerError;

/**
 * Error that wraps unknown errors from catch blocks
 */
export interface UnknownError {
  message: string;
  originalError: unknown;
  code: 'UNKNOWN_ERROR';
}

/**
 * Utility function to create a standardized error from an unknown error
 * This is particularly useful for typing catch blocks properly
 */
export function createErrorFromUnknown(error: unknown, context?: string): UnknownError {
  let message = 'An unknown error occurred';
  
  if (error instanceof Error) {
    message = error.message;
  } else if (typeof error === 'string') {
    message = error;
  } else if (error && typeof error === 'object' && 'message' in error) {
    message = String(error.message);
  }
  
  if (context) {
    message = `${context}: ${message}`;
  }
  
  return {
    message,
    originalError: error,
    code: 'UNKNOWN_ERROR'
  };
}

/**
 * Type guard to check if an error is an ApiError
 */
export function isApiError(error: unknown): error is ApiError {
  return typeof error === 'object' && 
         error !== null && 
         'message' in error &&
         'statusCode' in error &&
         'code' in error &&
         typeof (error as ApiError).message === 'string' &&
         typeof (error as ApiError).statusCode === 'number' &&
         typeof (error as ApiError).code === 'string';
}

/**
 * Type guard to check if an error is a NetworkError
 */
export function isNetworkError(error: unknown): error is NetworkError {
  return isApiError(error) && error.code === 'NETWORK_ERROR';
}

/**
 * Type guard to check if an error is a ValidationError
 */
export function isValidationError(error: unknown): error is ValidationError {
  return isApiError(error) && error.code === 'VALIDATION_ERROR';
}

/**
 * Type guard to check if an error is an AuthenticationError
 */
export function isAuthenticationError(error: unknown): error is AuthenticationError {
  return isApiError(error) && error.code === 'AUTHENTICATION_ERROR';
}

/**
 * Type guard to check if an error is an AuthorizationError
 */
export function isAuthorizationError(error: unknown): error is AuthorizationError {
  return isApiError(error) && error.code === 'AUTHORIZATION_ERROR';
}

/**
 * Type guard to check if an error is a NotFoundError
 */
export function isNotFoundError(error: unknown): error is NotFoundError {
  return isApiError(error) && error.code === 'NOT_FOUND';
}

/**
 * Type guard to check if an error is a ServerError
 */
export function isServerError(error: unknown): error is ServerError {
  return isApiError(error) && error.code === 'SERVER_ERROR';
}

/**
 * Utility function to create an ApiError from an HTTP response
 */
export function createApiErrorFromResponse(
  statusCode: number, 
  message: string, 
  code?: string
): ApiErrorType {
  switch (statusCode) {
    case 400:
    case 422:
      return {
        message,
        statusCode,
        code: 'VALIDATION_ERROR'
      } as ValidationError;
    case 401:
      return { 
        message, 
        statusCode, 
        code: 'AUTHENTICATION_ERROR' 
      } as AuthenticationError;
    case 403:
      return { 
        message, 
        statusCode, 
        code: 'AUTHORIZATION_ERROR' 
      } as AuthorizationError;
    case 404:
      return { 
        message, 
        statusCode, 
        code: 'NOT_FOUND' 
      } as NotFoundError;
    case 500:
    case 502:
    case 503:
    case 504:
      return { 
        message, 
        statusCode, 
        code: 'SERVER_ERROR' 
      } as ServerError;
    default:
      // For other status codes, default to network error
      return { 
        message, 
        statusCode, 
        code: 'NETWORK_ERROR' 
      } as NetworkError;
  }
}

/**
 * Get a default error code based on HTTP status code
 */
function getDefaultErrorCode(statusCode: number): string {
  if (statusCode >= 400 && statusCode < 500) {
    return 'CLIENT_ERROR';
  } else if (statusCode >= 500) {
    return 'SERVER_ERROR';
  }
  return 'UNKNOWN_ERROR';
}

/**
 * Result type for operations that can fail
 * This is useful for functions that need to return either a value or an error
 */
export type Result<T, E = ApiErrorType | UnknownError> = {
  success: true;
  data: T;
} | {
  success: false;
  error: E;
};

/**
 * Utility function to create a successful result
 */
export function createSuccessResult<T>(data: T): Result<T> {
  return { success: true, data };
}

/**
 * Utility function to create an error result
 */
export function createErrorResult<E>(error: E): Result<never, E> {
  return { success: false, error };
}