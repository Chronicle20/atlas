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
  /** AbortController signal for request cancellation */
  signal?: AbortSignal;
  /** Disable request deduplication for this specific request (default: false) */
  skipDeduplication?: boolean;
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
 * Information about a pending request for deduplication
 */
interface PendingRequest<T = unknown> {
  promise: Promise<T>;
  abortController: AbortController;
  requestCount: number;
}

/**
 * API client class that provides centralized HTTP request handling
 */
class ApiClient {
  private config: ApiClientConfig;
  private pendingRequests: Map<string, PendingRequest> = new Map();

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
   * Generate a unique key for request deduplication based on method, URL, and data
   */
  private generateRequestKey(method: string, url: string, data?: unknown, headers?: Headers): string {
    // Create a hash-like key from the request components
    const tenantId = this.config.tenant?.id || 'no-tenant';
    const headerString = headers ? Array.from(headers.entries())
      .filter(([key]) => !['authorization', 'cookie'].includes(key.toLowerCase())) // Exclude sensitive headers
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([key, value]) => `${key}:${value}`)
      .join('|') : '';
    
    const components = [
      method.toUpperCase(),
      url,
      data ? JSON.stringify(data) : '',
      headerString,
      tenantId
    ];
    
    // Create a simple hash-like string (not cryptographically secure, but sufficient for deduplication)
    return btoa(components.join('::'));
  }

  /**
   * Get or create a deduplicated request
   */
  private getOrCreateDeduplicatedRequest<T>(
    key: string,
    requestFactory: () => Promise<T>,
    externalSignal?: AbortSignal
  ): Promise<T> {
    const existingRequest = this.pendingRequests.get(key) as PendingRequest<T> | undefined;
    
    if (existingRequest) {
      // Increment request count
      existingRequest.requestCount++;
      
      // If external signal is provided, listen for cancellation
      if (externalSignal) {
        const abortHandler = () => {
          existingRequest.requestCount--;
          if (existingRequest.requestCount <= 0) {
            existingRequest.abortController.abort();
            this.pendingRequests.delete(key);
          }
        };
        
        if (externalSignal.aborted) {
          // If signal is already aborted, reject immediately
          return Promise.reject(createApiErrorFromResponse(0, 'Request was cancelled'));
        } else {
          externalSignal.addEventListener('abort', abortHandler, { once: true });
        }
      }
      
      return existingRequest.promise;
    }

    // Create new request
    const abortController = new AbortController();
    
    // Create a promise that we can control manually
    let resolvePromise: (value: T | PromiseLike<T>) => void;
    let rejectPromise: (reason?: any) => void;
    
    const controlledPromise = new Promise<T>((resolve, reject) => {
      resolvePromise = resolve;
      rejectPromise = reject;
    });

    const pendingRequest: PendingRequest<T> = {
      promise: controlledPromise,
      abortController,
      requestCount: 1
    };

    this.pendingRequests.set(key, pendingRequest);
    
    // Execute the actual request
    // Note: requestFactory should already handle the external signal passed to the method
    requestFactory()
      .then(result => {
        resolvePromise(result);
      })
      .catch(error => {
        rejectPromise(error);
      })
      .finally(() => {
        // Clean up after request completion (success or failure)
        this.pendingRequests.delete(key);
      });

    // If external signal is provided, listen for cancellation
    if (externalSignal) {
      const abortHandler = () => {
        pendingRequest.requestCount--;
        if (pendingRequest.requestCount <= 0) {
          abortController.abort();
          this.pendingRequests.delete(key);
          // Reject the controlled promise with cancellation error
          rejectPromise(createApiErrorFromResponse(0, 'Request was cancelled'));
        }
      };
      
      if (externalSignal.aborted) {
        abortHandler();
        return controlledPromise; // Return early if already cancelled
      } else {
        externalSignal.addEventListener('abort', abortHandler, { once: true });
      }
    }

    return controlledPromise;
  }

  /**
   * Clear all pending requests (useful for cleanup or testing)
   */
  clearPendingRequests(): void {
    // Cancel all pending requests
    for (const [key, request] of this.pendingRequests.entries()) {
      request.abortController.abort();
    }
    this.pendingRequests.clear();
  }

  /**
   * Get the number of pending requests (useful for debugging)
   */
  getPendingRequestCount(): number {
    return this.pendingRequests.size;
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
   * Sleep for the specified number of milliseconds with cancellation support
   */
  private async sleepWithCancellation(ms: number, signal?: AbortSignal): Promise<void> {
    return new Promise((resolve, reject) => {
      // Check if already cancelled
      if (signal?.aborted) {
        reject(createApiErrorFromResponse(0, 'Request was cancelled'));
        return;
      }

      const timeoutId = setTimeout(() => {
        resolve();
      }, ms);

      // Listen for cancellation
      if (signal) {
        const abortHandler = () => {
          clearTimeout(timeoutId);
          reject(createApiErrorFromResponse(0, 'Request was cancelled'));
        };

        signal.addEventListener('abort', abortHandler, { once: true });

        // Clean up listener when timeout completes
        const originalResolve = resolve;
        resolve = () => {
          signal.removeEventListener('abort', abortHandler);
          originalResolve();
        };
      }
    });
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
   * Create a combined AbortController that handles both timeout and external cancellation
   */
  private createCombinedController(timeout: number, externalSignal?: AbortSignal): {
    controller: AbortController;
    timeoutId: NodeJS.Timeout | number | null;
  } {
    const controller = new AbortController();
    let timeoutId: NodeJS.Timeout | number | null = null;

    // Set up timeout abortion
    timeoutId = setTimeout(() => {
      if (!controller.signal.aborted) {
        controller.abort();
      }
    }, timeout);

    // Listen for external signal abortion
    if (externalSignal) {
      const abortHandler = () => {
        if (!controller.signal.aborted) {
          controller.abort();
        }
      };

      if (externalSignal.aborted) {
        // If already aborted, abort immediately
        controller.abort();
      } else {
        // Listen for abort event
        externalSignal.addEventListener('abort', abortHandler, { once: true });
        
        // Clean up listener when our controller is aborted
        controller.signal.addEventListener('abort', () => {
          externalSignal.removeEventListener('abort', abortHandler);
        }, { once: true });
      }
    }

    return { controller, timeoutId };
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
    const externalSignal = requestOptions?.signal;

    let lastError: unknown;

    for (let attempt = 0; attempt <= maxRetries; attempt++) {
      // Check if external signal is already aborted before starting
      if (externalSignal?.aborted) {
        throw createApiErrorFromResponse(0, 'Request was cancelled');
      }

      const { controller, timeoutId } = this.createCombinedController(timeout, externalSignal);

      try {
        const response = await fetch(url, {
          ...options,
          signal: controller.signal,
        });
        
        if (timeoutId) clearTimeout(timeoutId);
        
        // If the response is successful or if it's a non-retryable error, return it
        if (response.ok || !this.shouldRetryResponse(response.status)) {
          return response;
        }
        
        // For failed responses that are retryable, create an error and check if we should retry
        const error = createApiErrorFromResponse(response.status, `Request failed with status ${response.status}`);
        
        if (attempt === maxRetries || !isRetryableError(error)) {
          return response; // Return the response so it can be processed normally
        }
        
        // Wait before retrying (but check for cancellation during sleep)
        if (attempt < maxRetries) {
          const delay = this.calculateRetryDelay(attempt, retryDelay, maxRetryDelay, exponentialBackoff);
          await this.sleepWithCancellation(delay, externalSignal);
        }
        
      } catch (error) {
        if (timeoutId) clearTimeout(timeoutId);
        lastError = error;
        
        // Handle abort/timeout/cancellation errors
        if (error instanceof Error && error.name === 'AbortError') {
          // Check if it was external cancellation vs timeout
          if (externalSignal?.aborted) {
            throw createApiErrorFromResponse(0, 'Request was cancelled');
          } else {
            const timeoutError = createApiErrorFromResponse(408, 'Request timeout');
            
            if (attempt === maxRetries || !isRetryableError(timeoutError)) {
              throw timeoutError;
            }
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
        
        // Wait before retrying (but check for cancellation during sleep)
        if (attempt < maxRetries) {
          const delay = this.calculateRetryDelay(attempt, retryDelay, maxRetryDelay, exponentialBackoff);
          await this.sleepWithCancellation(delay, externalSignal);
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
    // Check for already aborted signal early
    if (options?.signal?.aborted) {
      console.log('Signal already aborted, throwing error');
      throw createApiErrorFromResponse(0, 'Request was cancelled');
    }
    console.log('Signal aborted?', options?.signal?.aborted);

    const headers = this.createHeaders(options);
    const fullUrl = `${this.config.baseUrl}${url}`;
    
    // Skip deduplication if explicitly requested
    if (options?.skipDeduplication) {
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

    // Use deduplication for GET requests
    const requestKey = this.generateRequestKey('GET', fullUrl, undefined, headers);
    
    return this.getOrCreateDeduplicatedRequest<T>(
      requestKey,
      async () => {
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
      },
      options?.signal
    );
  }

  /**
   * Perform a POST request
   */
  async post<T>(url: string, data?: unknown, options?: ApiRequestOptions): Promise<T> {
    // Check for already aborted signal early
    if (options?.signal?.aborted) {
      throw createApiErrorFromResponse(0, 'Request was cancelled');
    }

    const headers = this.createHeaders(options);
    const fullUrl = `${this.config.baseUrl}${url}`;
    
    // Skip deduplication if explicitly requested
    if (options?.skipDeduplication) {
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

    // Use deduplication for POST requests
    const requestKey = this.generateRequestKey('POST', fullUrl, data, headers);
    
    return this.getOrCreateDeduplicatedRequest<T>(
      requestKey,
      async () => {
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
      },
      options?.signal
    );
  }

  /**
   * Perform a PUT request
   */
  async put<T>(url: string, data?: unknown, options?: ApiRequestOptions): Promise<T> {
    const headers = this.createHeaders(options);
    const fullUrl = `${this.config.baseUrl}${url}`;
    
    // Skip deduplication if explicitly requested
    if (options?.skipDeduplication) {
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

    // Use deduplication for PUT requests
    const requestKey = this.generateRequestKey('PUT', fullUrl, data, headers);
    
    return this.getOrCreateDeduplicatedRequest<T>(
      requestKey,
      async () => {
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
      },
      options?.signal
    );
  }

  /**
   * Perform a PATCH request
   */
  async patch<T>(url: string, data?: unknown, options?: ApiRequestOptions): Promise<T> {
    const headers = this.createHeaders(options);
    const fullUrl = `${this.config.baseUrl}${url}`;
    
    // Skip deduplication if explicitly requested
    if (options?.skipDeduplication) {
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

    // Use deduplication for PATCH requests
    const requestKey = this.generateRequestKey('PATCH', fullUrl, data, headers);
    
    return this.getOrCreateDeduplicatedRequest<T>(
      requestKey,
      async () => {
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
      },
      options?.signal
    );
  }

  /**
   * Perform a DELETE request
   */
  async delete<T>(url: string, options?: ApiRequestOptions): Promise<T> {
    const headers = this.createHeaders(options);
    const fullUrl = `${this.config.baseUrl}${url}`;
    
    // Skip deduplication if explicitly requested
    if (options?.skipDeduplication) {
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

    // Use deduplication for DELETE requests
    const requestKey = this.generateRequestKey('DELETE', fullUrl, undefined, headers);
    
    return this.getOrCreateDeduplicatedRequest<T>(
      requestKey,
      async () => {
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
      },
      options?.signal
    );
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

  /**
   * Clear all pending requests (useful for cleanup)
   */
  clearPendingRequests: (): void => apiClient.clearPendingRequests(),

  /**
   * Get the number of pending requests (useful for debugging)
   */
  getPendingRequestCount: (): number => apiClient.getPendingRequestCount(),
};

/**
 * Utility functions for request cancellation
 */
export const cancellation = {
  /**
   * Create a new AbortController for request cancellation
   */
  createController: (): AbortController => new AbortController(),

  /**
   * Create an AbortController that automatically aborts after a timeout
   */
  createTimeoutController: (timeoutMs: number): AbortController => {
    const controller = new AbortController();
    setTimeout(() => {
      if (!controller.signal.aborted) {
        controller.abort();
      }
    }, timeoutMs);
    return controller;
  },

  /**
   * Combine multiple AbortSignals into one
   */
  combineSignals: (...signals: (AbortSignal | undefined)[]): AbortController => {
    const controller = new AbortController();
    const validSignals = signals.filter((signal): signal is AbortSignal => signal != null);

    if (validSignals.length === 0) {
      return controller;
    }

    const abortHandler = () => {
      if (!controller.signal.aborted) {
        controller.abort();
      }
    };

    // Check if any signal is already aborted
    if (validSignals.some(signal => signal.aborted)) {
      controller.abort();
      return controller;
    }

    // Listen to all signals
    validSignals.forEach(signal => {
      signal.addEventListener('abort', abortHandler, { once: true });
    });

    // Clean up listeners when our controller is aborted
    controller.signal.addEventListener('abort', () => {
      validSignals.forEach(signal => {
        signal.removeEventListener('abort', abortHandler);
      });
    }, { once: true });

    return controller;
  },

  /**
   * Check if an error is a cancellation error
   */
  isCancellationError: (error: unknown): boolean => {
    return error instanceof Error && 
           (error.name === 'AbortError' || 
            (typeof error.message === 'string' && error.message.includes('Request was cancelled')));
  }
};

export default api;