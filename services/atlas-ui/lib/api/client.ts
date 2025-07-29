/**
 * Centralized API client for Atlas UI
 * 
 * This module provides a type-safe, consistent API client that handles:
 * - Automatic tenant header injection
 * - Standardized error handling
 * - Request/response interceptors
 * - Type-safe HTTP methods
 */

import type { Tenant } from '@/types/models/tenant';
import type { ApiListResponse, ApiSingleResponse } from '@/types/api/responses';
import { tenantHeaders } from '@/lib/headers';
import { createApiErrorFromResponse } from '@/types/api/errors';
import { isRetryableError, sanitizeErrorData } from '@/lib/api/errors';

/**
 * Configuration options for API requests
 */
export interface ApiRequestOptions extends Omit<RequestInit, 'method' | 'body'> {
  /** Override default timeout (default: 30000ms) */
  timeout?: number;
  /** Skip automatic tenant header injection */
  skipTenantHeaders?: boolean;
  /** Additional headers to merge with defaults */
  headers?: HeadersInit;
  /** Number of retry attempts for failed requests (default: 3) */
  maxRetries?: number;
  /** Base delay between retries in milliseconds (default: 1000) */
  retryDelay?: number;
  /** Maximum delay between retries in milliseconds (default: 10000) */
  maxRetryDelay?: number;
  /** Whether to use exponential backoff for retry delays (default: true) */
  exponentialBackoff?: boolean;
}

/**
 * Internal configuration for API client
 */
interface ApiClientConfig {
  baseUrl: string;
  defaultTimeout: number;
  tenant: Tenant | null;
  defaultMaxRetries: number;
  defaultRetryDelay: number;
  defaultMaxRetryDelay: number;
  defaultExponentialBackoff: boolean;
}

/**
 * API client class that provides centralized HTTP request handling
 */
class ApiClient {
  private config: ApiClientConfig;

  constructor() {
    this.config = {
      baseUrl: process.env.NEXT_PUBLIC_ROOT_API_URL || (typeof window !== 'undefined' ? window.location.origin : ''),
      defaultTimeout: 30000,
      tenant: null,
      defaultMaxRetries: 3,
      defaultRetryDelay: 1000,
      defaultMaxRetryDelay: 10000,
      defaultExponentialBackoff: true,
    };
  }

  /**
   * Set the tenant for all subsequent requests
   */
  setTenant(tenant: Tenant | null): void {
    this.config.tenant = tenant;
  }

  /**
   * Get the current tenant
   */
  getTenant(): Tenant | null {
    return this.config.tenant;
  }

  /**
   * Calculate retry delay with exponential backoff
   */
  private calculateRetryDelay(
    attempt: number,
    baseDelay: number,
    maxDelay: number,
    exponentialBackoff: boolean
  ): number {
    if (!exponentialBackoff) {
      return Math.min(baseDelay, maxDelay);
    }
    
    // Exponential backoff: baseDelay * 2^attempt with jitter
    const exponentialDelay = baseDelay * Math.pow(2, attempt);
    const jitter = Math.random() * 0.1 * exponentialDelay; // Add 10% jitter
    const delayWithJitter = exponentialDelay + jitter;
    
    return Math.min(delayWithJitter, maxDelay);
  }

  /**
   * Sleep for the specified number of milliseconds
   */
  private async sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  /**
   * Create request headers with tenant information
   */
  private createHeaders(options?: ApiRequestOptions): Headers {
    const headers = new Headers();

    // Add tenant headers if available and not explicitly skipped
    if (this.config.tenant && !options?.skipTenantHeaders) {
      const tenantHeadersObj = tenantHeaders(this.config.tenant);
      tenantHeadersObj.forEach((value, key) => {
        headers.set(key, value);
      });
    }

    // Add default content type for JSON requests
    headers.set('Content-Type', 'application/json');

    // Merge with any additional headers
    if (options?.headers) {
      const additionalHeaders = new Headers(options.headers);
      additionalHeaders.forEach((value, key) => {
        headers.set(key, value);
      });
    }

    return headers;
  }

  /**
   * Create a fetch request with timeout and retry support
   */
  private async fetchWithTimeout(
    url: string, 
    options: RequestInit, 
    timeout: number = this.config.defaultTimeout,
    requestOptions?: ApiRequestOptions
  ): Promise<Response> {
    const maxRetries = requestOptions?.maxRetries ?? this.config.defaultMaxRetries;
    const retryDelay = requestOptions?.retryDelay ?? this.config.defaultRetryDelay;
    const maxRetryDelay = requestOptions?.maxRetryDelay ?? this.config.defaultMaxRetryDelay;
    const exponentialBackoff = requestOptions?.exponentialBackoff ?? this.config.defaultExponentialBackoff;

    let lastError: unknown;

    for (let attempt = 0; attempt <= maxRetries; attempt++) {
      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), timeout);

      try {
        const response = await fetch(url, {
          ...options,
          signal: controller.signal,
        });
        
        clearTimeout(timeoutId);
        
        // If the response is successful or if it's a non-retryable error, return it
        if (response.ok || !this.shouldRetryResponse(response.status)) {
          return response;
        }
        
        // For failed responses that are retryable, create an error and check if we should retry
        const error = createApiErrorFromResponse(response.status, `Request failed with status ${response.status}`);
        
        if (attempt === maxRetries || !isRetryableError(error)) {
          return response; // Return the response so it can be processed normally
        }
        
        // Wait before retrying
        if (attempt < maxRetries) {
          const delay = this.calculateRetryDelay(attempt, retryDelay, maxRetryDelay, exponentialBackoff);
          await this.sleep(delay);
        }
        
      } catch (error) {
        clearTimeout(timeoutId);
        lastError = error;
        
        // Handle abort/timeout errors
        if (error instanceof Error && error.name === 'AbortError') {
          const timeoutError = createApiErrorFromResponse(408, 'Request timeout');
          
          if (attempt === maxRetries || !isRetryableError(timeoutError)) {
            throw timeoutError;
          }
        }
        // Handle network errors
        else if (error instanceof TypeError && error.message.includes('fetch')) {
          const networkError = createApiErrorFromResponse(0, 'Network error: Unable to connect to server');
          
          if (attempt === maxRetries || !isRetryableError(networkError)) {
            throw networkError;
          }
        }
        // Handle other unknown errors
        else if (attempt === maxRetries) {
          throw error;
        }
        
        // Wait before retrying
        if (attempt < maxRetries) {
          const delay = this.calculateRetryDelay(attempt, retryDelay, maxRetryDelay, exponentialBackoff);
          await this.sleep(delay);
        }
      }
    }

    // This should never be reached, but just in case
    throw lastError || createApiErrorFromResponse(500, 'Max retries exceeded');
  }

  /**
   * Check if a response status code should trigger a retry
   */
  private shouldRetryResponse(status: number): boolean {
    // Retry on server errors and specific client errors
    const retryableStatusCodes = [408, 429, 500, 502, 503, 504];
    return retryableStatusCodes.includes(status);
  }

  /**
   * Process response and handle errors
   */
  private async processResponse<T>(response: Response): Promise<T> {
    if (!response.ok) {
      let errorMessage = 'Request failed';
      let sanitizedErrorData: Record<string, unknown> = {};
      
      try {
        const errorData = await response.json();
        
        // Sanitize error data to prevent sensitive data leaks
        sanitizedErrorData = sanitizeErrorData(errorData);
        
        // Handle different error response formats (using sanitized data)
        if (sanitizedErrorData.error && typeof sanitizedErrorData.error === 'object') {
          const errorObj = sanitizedErrorData.error as Record<string, unknown>;
          if (typeof errorObj.detail === 'string') {
            errorMessage = errorObj.detail;
          }
        } else if (typeof sanitizedErrorData.message === 'string') {
          errorMessage = sanitizedErrorData.message;
        }
      } catch {
        // Fallback to status-based messages if response body can't be parsed
        switch (response.status) {
          case 400:
            errorMessage = 'Invalid request parameters';
            break;
          case 401:
            errorMessage = 'Authentication required';
            break;
          case 403:
            errorMessage = 'Permission denied';
            break;
          case 404:
            errorMessage = 'Resource not found';
            break;
          case 409:
            errorMessage = 'Request conflicts with current state';
            break;
          case 429:
            errorMessage = 'Too many requests. Please try again later';
            break;
          case 500:
            errorMessage = 'Internal server error';
            break;
          case 502:
            errorMessage = 'Bad gateway';
            break;
          case 503:
            errorMessage = 'Service unavailable';
            break;
          case 504:
            errorMessage = 'Gateway timeout';
            break;
          default:
            errorMessage = `Request failed with status ${response.status}`;
        }
      }
      
      throw createApiErrorFromResponse(response.status, errorMessage);
    }

    // Handle 204 No Content responses
    if (response.status === 204) {
      return undefined as T;
    }

    try {
      return await response.json();
    } catch {
      throw createApiErrorFromResponse(500, 'Invalid JSON response from server');
    }
  }

  /**
   * Perform a GET request
   */
  async get<T>(url: string, options?: ApiRequestOptions): Promise<T> {
    const headers = this.createHeaders(options);
    const fullUrl = `${this.config.baseUrl}${url}`;
    
    const response = await this.fetchWithTimeout(
      fullUrl,
      {
        method: 'GET',
        headers,
        ...options,
      },
      options?.timeout,
      options
    );

    return this.processResponse<T>(response);
  }

  /**
   * Perform a POST request
   */
  async post<T>(url: string, data?: unknown, options?: ApiRequestOptions): Promise<T> {
    const headers = this.createHeaders(options);
    const fullUrl = `${this.config.baseUrl}${url}`;
    
    const response = await this.fetchWithTimeout(
      fullUrl,
      {
        method: 'POST',
        headers,
        body: data ? JSON.stringify(data) : null,
        ...options,
      },
      options?.timeout,
      options
    );

    return this.processResponse<T>(response);
  }

  /**
   * Perform a PUT request
   */
  async put<T>(url: string, data?: unknown, options?: ApiRequestOptions): Promise<T> {
    const headers = this.createHeaders(options);
    const fullUrl = `${this.config.baseUrl}${url}`;
    
    const response = await this.fetchWithTimeout(
      fullUrl,
      {
        method: 'PUT',
        headers,
        body: data ? JSON.stringify(data) : null,
        ...options,
      },
      options?.timeout,
      options
    );

    return this.processResponse<T>(response);
  }

  /**
   * Perform a PATCH request
   */
  async patch<T>(url: string, data?: unknown, options?: ApiRequestOptions): Promise<T> {
    const headers = this.createHeaders(options);
    const fullUrl = `${this.config.baseUrl}${url}`;
    
    const response = await this.fetchWithTimeout(
      fullUrl,
      {
        method: 'PATCH',
        headers,
        body: data ? JSON.stringify(data) : null,
        ...options,
      },
      options?.timeout,
      options
    );

    return this.processResponse<T>(response);
  }

  /**
   * Perform a DELETE request
   */
  async delete<T>(url: string, options?: ApiRequestOptions): Promise<T> {
    const headers = this.createHeaders(options);
    const fullUrl = `${this.config.baseUrl}${url}`;
    
    const response = await this.fetchWithTimeout(
      fullUrl,
      {
        method: 'DELETE',
        headers,
        ...options,
      },
      options?.timeout,
      options
    );

    return this.processResponse<T>(response);
  }
}

/**
 * Singleton API client instance
 */
export const apiClient = new ApiClient();

/**
 * Convenience functions for common API response patterns
 */
export const api = {
  /**
   * GET request that expects a list response
   */
  getList: <T>(url: string, options?: ApiRequestOptions): Promise<T[]> =>
    apiClient.get<ApiListResponse<T>>(url, options).then(response => response.data),

  /**
   * GET request that expects a single item response
   */
  getOne: <T>(url: string, options?: ApiRequestOptions): Promise<T> =>
    apiClient.get<ApiSingleResponse<T>>(url, options).then(response => response.data),

  /**
   * GET request that expects a raw response (no data wrapper)
   */
  get: <T>(url: string, options?: ApiRequestOptions): Promise<T> =>
    apiClient.get<T>(url, options),

  /**
   * POST request with JSON:API format
   */
  post: <T>(url: string, data?: unknown, options?: ApiRequestOptions): Promise<T> =>
    apiClient.post<T>(url, data, options),

  /**
   * PUT request with JSON:API format
   */
  put: <T>(url: string, data?: unknown, options?: ApiRequestOptions): Promise<T> =>
    apiClient.put<T>(url, data, options),

  /**
   * PATCH request with JSON:API format
   */
  patch: <T>(url: string, data?: unknown, options?: ApiRequestOptions): Promise<T> =>
    apiClient.patch<T>(url, data, options),

  /**
   * DELETE request
   */
  delete: <T>(url: string, options?: ApiRequestOptions): Promise<T> =>
    apiClient.delete<T>(url, options),

  /**
   * Set tenant for all subsequent requests
   */
  setTenant: (tenant: Tenant | null): void => apiClient.setTenant(tenant),

  /**
   * Get current tenant
   */
  getTenant: (): Tenant | null => apiClient.getTenant(),
};

export default api;