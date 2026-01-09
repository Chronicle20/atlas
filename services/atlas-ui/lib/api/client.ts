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
 * Progress information for file uploads/downloads
 */
export interface ProgressInfo {
  /** Total number of bytes (if known) */
  total: number | undefined;
  /** Number of bytes transferred */
  loaded: number;
  /** Progress percentage (0-100, if total is known) */
  percentage: number | undefined;
  /** Transfer rate in bytes per second */
  rate: number | undefined;
  /** Estimated time remaining in milliseconds */
  timeRemaining: number | undefined;
  /** Whether the operation is complete */
  done: boolean;
}

/**
 * Progress callback function
 */
export type ProgressCallback = (progress: ProgressInfo) => void;

/**
 * Cache configuration options
 */
export interface CacheOptions {
  /** Cache time-to-live in milliseconds (default: 300000ms = 5 minutes) */
  ttl?: number;
  /** Cache key prefix for namespacing (default: auto-generated) */
  keyPrefix?: string;
  /** Whether to use stale data while revalidating (default: false) */
  staleWhileRevalidate?: boolean;
  /** Maximum stale time in milliseconds when using stale-while-revalidate (default: 60000ms = 1 minute) */
  maxStaleTime?: number;
}

/**
 * Configuration options for API requests
 */
export interface ApiRequestOptions extends Omit<RequestInit, 'method' | 'body' | 'cache'> {
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
  /** Progress callback for tracking upload/download progress */
  onProgress?: ProgressCallback;
  /** Cache configuration for this request */
  cacheConfig?: CacheOptions | false;
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
 * Cached response entry with metadata
 */
interface CacheEntry<T = unknown> {
  /** The cached response data */
  data: T;
  /** Timestamp when the entry was created */
  timestamp: number;
  /** Time-to-live in milliseconds */
  ttl: number;
  /** Whether this entry supports stale-while-revalidate */
  staleWhileRevalidate: boolean;
  /** Maximum stale time in milliseconds */
  maxStaleTime: number;
  /** HTTP headers from the original response (for conditional requests) */
  headers?: Record<string, string>;
  /** ETag from the original response (for conditional requests) */
  etag?: string;
  /** Last-Modified from the original response (for conditional requests) */
  lastModified?: string;
}

/**
 * Progress tracker utility for monitoring upload/download progress
 */
class ProgressTracker {
  private startTime: number;
  private lastTime: number;
  private lastLoaded: number;
  private samples: Array<{ time: number; loaded: number }> = [];
  private readonly maxSamples = 10;

  constructor() {
    this.startTime = Date.now();
    this.lastTime = this.startTime;
    this.lastLoaded = 0;
  }

  /**
   * Update progress and calculate metrics
   */
  update(loaded: number, total?: number): ProgressInfo {
    const now = Date.now();

    // Add sample for rate calculation
    this.samples.push({ time: now, loaded });
    if (this.samples.length > this.maxSamples) {
      this.samples.shift();
    }

    // Calculate transfer rate (bytes per second)
    let rate = 0;
    if (this.samples.length >= 2) {
      const oldestSample = this.samples[0]!;
      const newestSample = this.samples[this.samples.length - 1]!;
      const timeDiff = newestSample.time - oldestSample.time;
      const loadedDiff = newestSample.loaded - oldestSample.loaded;
      
      if (timeDiff > 0) {
        rate = (loadedDiff / timeDiff) * 1000; // Convert to bytes per second
      }
    }

    // Calculate percentage if total is known
    let percentage: number | undefined;
    if (total && total > 0) {
      percentage = Math.min(100, (loaded / total) * 100);
    }

    // Calculate estimated time remaining
    let timeRemaining: number | undefined;
    if (total && rate > 0 && total > loaded) {
      const remainingBytes = total - loaded;
      timeRemaining = (remainingBytes / rate) * 1000; // Convert to milliseconds
    }

    // Update tracking variables
    this.lastTime = now;
    this.lastLoaded = loaded;

    return {
      total: total,
      loaded,
      percentage: percentage,
      rate,
      timeRemaining: timeRemaining,
      done: false
    };
  }

  /**
   * Mark progress as complete
   */
  complete(total?: number): ProgressInfo {
    return {
      total: total || this.lastLoaded,
      loaded: total || this.lastLoaded,
      percentage: 100,
      rate: 0,
      timeRemaining: 0,
      done: true
    };
  }
}

/**
 * API client class that provides centralized HTTP request handling
 */
class ApiClient {
  private config: ApiClientConfig;
  private pendingRequests: Map<string, PendingRequest> = new Map();
  private responseCache: Map<string, CacheEntry> = new Map();
  private cacheCleanupInterval: NodeJS.Timeout | number | null = null;

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

    // Start cache cleanup interval (every 5 minutes)
    this.startCacheCleanup();
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
   * Start automatic cache cleanup interval
   */
  private startCacheCleanup(): void {
    // Only run cleanup in browser environment
    if (typeof window === 'undefined') return;

    // Clean up expired cache entries every 5 minutes
    this.cacheCleanupInterval = setInterval(() => {
      this.cleanupExpiredCache();
    }, 5 * 60 * 1000); // 5 minutes
  }

  /**
   * Stop the cache cleanup interval
   */
  private stopCacheCleanup(): void {
    if (this.cacheCleanupInterval) {
      clearInterval(this.cacheCleanupInterval);
      this.cacheCleanupInterval = null;
    }
  }

  /**
   * Clean up expired cache entries
   */
  private cleanupExpiredCache(): void {
    const now = Date.now();
    const entriesToDelete: string[] = [];

    for (const [key, entry] of this.responseCache.entries()) {
      const age = now - entry.timestamp;
      const maxAge = entry.staleWhileRevalidate 
        ? entry.ttl + entry.maxStaleTime 
        : entry.ttl;

      if (age > maxAge) {
        entriesToDelete.push(key);
      }
    }

    entriesToDelete.forEach(key => {
      this.responseCache.delete(key);
    });
  }

  /**
   * Generate a cache key for a request
   */
  private generateCacheKey(
    method: string, 
    url: string, 
    data?: unknown, 
    headers?: Headers,
    cacheOptions?: CacheOptions
  ): string {
    const tenantId = this.config.tenant?.id || 'no-tenant';
    const keyPrefix = cacheOptions?.keyPrefix || 'api';
    
    // Include relevant headers that might affect the response
    const relevantHeaders = headers ? Array.from(headers.entries())
      .filter(([key]) => ['accept', 'accept-language', 'content-type'].includes(key.toLowerCase()))
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([key, value]) => `${key}:${value}`)
      .join('|') : '';
    
    const components = [
      keyPrefix,
      method.toUpperCase(),
      url,
      data ? JSON.stringify(data) : '',
      relevantHeaders,
      tenantId
    ];
    
    // Create a simple hash-like string for the cache key
    return btoa(components.join('::'));
  }

  /**
   * Get cached response if available and valid
   */
  private getCachedResponse<T>(key: string): { data: T; isStale: boolean } | null {
    const entry = this.responseCache.get(key) as CacheEntry<T> | undefined;
    
    if (!entry) {
      return null;
    }

    const now = Date.now();
    const age = now - entry.timestamp;

    // Check if entry is fresh (within TTL)
    if (age <= entry.ttl) {
      return { data: entry.data, isStale: false };
    }

    // Check if entry is stale but within stale-while-revalidate period
    if (entry.staleWhileRevalidate && age <= entry.ttl + entry.maxStaleTime) {
      return { data: entry.data, isStale: true };
    }

    // Entry is expired, remove it
    this.responseCache.delete(key);
    return null;
  }

  /**
   * Store response in cache
   */
  private setCachedResponse<T>(
    key: string, 
    data: T, 
    response: Response, 
    cacheOptions: CacheOptions
  ): void {
    const now = Date.now();
    const ttl = cacheOptions.ttl ?? 5 * 60 * 1000; // Default 5 minutes
    const staleWhileRevalidate = cacheOptions.staleWhileRevalidate ?? false;
    const maxStaleTime = cacheOptions.maxStaleTime ?? 60 * 1000; // Default 1 minute

    // Extract relevant headers for conditional requests
    const headers: Record<string, string> = {};
    const etag = response.headers.get('etag');
    const lastModified = response.headers.get('last-modified');
    const cacheControl = response.headers.get('cache-control');

    if (cacheControl) headers['cache-control'] = cacheControl;

    const entry: CacheEntry<T> = {
      data,
      timestamp: now,
      ttl,
      staleWhileRevalidate,
      maxStaleTime,
      headers,
      ...(etag && { etag }),
      ...(lastModified && { lastModified })
    };

    this.responseCache.set(key, entry);
  }

  /**
   * Clear all cached responses
   */
  clearCache(): void {
    this.responseCache.clear();
  }

  /**
   * Clear cached responses by key pattern
   */
  clearCacheByPattern(pattern: string): void {
    const regex = new RegExp(pattern);
    const keysToDelete: string[] = [];

    for (const key of this.responseCache.keys()) {
      if (regex.test(key)) {
        keysToDelete.push(key);
      }
    }

    keysToDelete.forEach(key => {
      this.responseCache.delete(key);
    });
  }

  /**
   * Get cache statistics
   */
  getCacheStats(): { size: number; entries: Array<{ key: string; age: number; ttl: number; isStale: boolean }> } {
    const now = Date.now();
    const entries: Array<{ key: string; age: number; ttl: number; isStale: boolean }> = [];

    for (const [key, entry] of this.responseCache.entries()) {
      const age = now - entry.timestamp;
      const isStale = age > entry.ttl;
      entries.push({ key, age, ttl: entry.ttl, isStale });
    }

    return {
      size: this.responseCache.size,
      entries
    };
  }

  /**
   * Revalidate cache entry in the background for stale-while-revalidate
   */
  private async revalidateInBackground<T>(
    fullUrl: string,
    headers: Headers,
    options: ApiRequestOptions | undefined,
    cacheKey: string,
    cacheOptions: CacheOptions
  ): Promise<void> {
    try {
      const response = await this.fetchWithTimeout(
        fullUrl,
        {
          method: 'GET',
          headers,
          ...options,
        },
        options?.timeout,
{
          ...(options ? (({ onProgress: _onProgress, ...rest }) => rest)(options) : {}),
        }
      );

      const data = await this.processResponse<T>(response);
      
      // Update the cache with fresh data
      this.setCachedResponse(cacheKey, data, response, cacheOptions);
    } catch (error) {
      // Silently handle revalidation errors - the user already has stale data
      // We could log this error for debugging purposes if needed
      console.debug('Background revalidation failed:', error);
    }
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
    let rejectPromise: (reason?: unknown) => void;
    
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
    for (const [, request] of this.pendingRequests.entries()) {
      request.abortController.abort();
    }
    this.pendingRequests.clear();
  }

  /**
   * Cleanup method to prevent memory leaks
   * Call this when the API client is no longer needed
   */
  cleanup(): void {
    this.stopCacheCleanup();
    this.clearCache();
    this.clearPendingRequests();
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
   * Create a progress-tracking readable stream for upload monitoring
   */
  private createProgressTrackingUploadBody(
    body: BodyInit | null | undefined,
    onProgress?: ProgressCallback
  ): BodyInit | null | undefined {
    if (!body || !onProgress) {
      return body;
    }

    // Handle different body types
    if (body instanceof FormData) {
      // For FormData, we can't easily track progress during the upload
      // The browser will handle this internally, so we just call the callback immediately
      // Real progress tracking would need to be handled at the xhr level or with a custom implementation
      const tracker = new ProgressTracker();
      onProgress(tracker.update(0));
      return body;
    }

    if (body instanceof ArrayBuffer || body instanceof Uint8Array) {
      const tracker = new ProgressTracker();
      const size = body instanceof ArrayBuffer ? body.byteLength : body.length;
      onProgress(tracker.update(0, size));
      // For binary data, we can't easily wrap it with progress tracking in fetch
      // This would require using XMLHttpRequest instead
      return body;
    }

    if (typeof body === 'string') {
      const tracker = new ProgressTracker();
      const size = new TextEncoder().encode(body).length;
      onProgress(tracker.update(0, size));
      // For string data, we can't easily wrap it with progress tracking in fetch
      return body;
    }

    // For other body types, return as-is
    return body;
  }

  /**
   * Create a progress-tracking response for download monitoring
   */
  private async createProgressTrackingResponse(
    response: Response,
    onProgress?: ProgressCallback
  ): Promise<Response> {
    if (!onProgress || !response.body) {
      return response;
    }

    const contentLength = response.headers.get('content-length');
    const total = contentLength ? parseInt(contentLength, 10) : undefined;
    const tracker = new ProgressTracker();
    
    // Create a new ReadableStream that tracks progress
    const progressStream = new ReadableStream({
      start(controller) {
        const reader = response.body!.getReader();
        let loaded = 0;

        function pump(): Promise<void> {
          return reader.read().then(({ done, value }) => {
            if (done) {
              // Complete the progress tracking
              onProgress!(tracker.complete(total));
              controller.close();
              return;
            }

            // Update progress
            loaded += value.length;
            onProgress!(tracker.update(loaded, total));

            // Enqueue the chunk
            controller.enqueue(value);
            return pump();
          }).catch(error => {
            controller.error(error);
          });
        }

        // Start initial progress update
        onProgress!(tracker.update(0, total));
        return pump();
      }
    });

    // Create a new response with the progress-tracking stream
    return new Response(progressStream, {
      status: response.status,
      statusText: response.statusText,
      headers: response.headers
    });
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
        // Add progress tracking to request body if needed
        const trackedBody = this.createProgressTrackingUploadBody(options.body, requestOptions?.onProgress);
        
        const response = await fetch(url, {
          ...options,
          body: trackedBody || null,
          signal: controller.signal,
        });
        
        if (timeoutId) clearTimeout(timeoutId);
        
        // Add progress tracking to response if needed and response is successful
        let finalResponse = response;
        if (response.ok && requestOptions?.onProgress) {
          finalResponse = await this.createProgressTrackingResponse(response, requestOptions.onProgress);
        }
        
        // If the response is successful or if it's a non-retryable error, return it
        if (response.ok || !this.shouldRetryResponse(response.status)) {
          return finalResponse;
        }
        
        // For failed responses that are retryable, create an error and check if we should retry
        const error = createApiErrorFromResponse(response.status, `Request failed with status ${response.status}`);
        
        if (attempt === maxRetries || !isRetryableError(error)) {
          return finalResponse; // Return the response so it can be processed normally
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
      throw createApiErrorFromResponse(0, 'Request was cancelled');
    }

    const headers = this.createHeaders(options);
    const fullUrl = `${this.config.baseUrl}${url}`;
    
    // Check if caching is enabled for this request
    const cacheOptions = options?.cacheConfig;
    const shouldCache = cacheOptions !== false && cacheOptions !== undefined;
    
    // Try cache first if enabled
    if (shouldCache && cacheOptions) {
      const cacheKey = this.generateCacheKey('GET', fullUrl, undefined, headers, cacheOptions);
      const cachedResponse = this.getCachedResponse<T>(cacheKey);
      
      if (cachedResponse) {
        // If we have fresh data, return it
        if (!cachedResponse.isStale) {
          return cachedResponse.data;
        }
        
        // If data is stale but stale-while-revalidate is enabled, return it and revalidate in background
        if (cachedResponse.isStale && cacheOptions.staleWhileRevalidate) {
          // Start background revalidation (don't await it)
          this.revalidateInBackground(fullUrl, headers, options, cacheKey, cacheOptions).catch(() => {
            // Silently handle revalidation errors - user already has stale data
          });
          
          return cachedResponse.data;
        }
      }
    }
    
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

      const data = await this.processResponse<T>(response);
      
      // Cache the response if caching is enabled
      if (shouldCache && cacheOptions) {
        const cacheKey = this.generateCacheKey('GET', fullUrl, undefined, headers, cacheOptions);
        this.setCachedResponse(cacheKey, data, response, cacheOptions);
      }
      
      return data;
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

        const data = await this.processResponse<T>(response);
        
        // Cache the response if caching is enabled
        if (shouldCache && cacheOptions) {
          const cacheKey = this.generateCacheKey('GET', fullUrl, undefined, headers, cacheOptions);
          this.setCachedResponse(cacheKey, data, response, cacheOptions);
        }
        
        return data;
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
   * Upload a file with progress tracking
   */
  async upload<T>(
    url: string,
    file: File | FormData,
    options?: ApiRequestOptions
  ): Promise<T> {
    const formData = file instanceof FormData ? file : new FormData();
    if (file instanceof File) {
      formData.append('file', file);
    }

    // Remove Content-Type header to let the browser set it with boundary for FormData
    const headers = this.createHeaders(options);
    headers.delete('Content-Type');

    const fullUrl = `${this.config.baseUrl}${url}`;
    
    // Progress tracking for uploads works better with XMLHttpRequest
    // For now, we'll use fetch with basic progress indication
    if (options?.onProgress) {
      const tracker = new ProgressTracker();
      const fileSize = file instanceof File ? file.size : undefined;
      
      // Initial progress
      options.onProgress(tracker.update(0, fileSize));
      
      // Since fetch doesn't provide upload progress, we'll simulate it
      // In a real implementation, you might want to use XMLHttpRequest for true upload progress
    }

    const response = await this.fetchWithTimeout(
      fullUrl,
      {
        method: 'POST',
        headers,
        body: formData,
        ...options,
      },
      options?.timeout,
      options
    );

    return this.processResponse<T>(response);
  }

  /**
   * Download a file with progress tracking
   */
  async download(
    url: string,
    options?: ApiRequestOptions
  ): Promise<Blob> {
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

    if (!response.ok) {
      // Use the regular error processing
      await this.processResponse(response);
    }

    // For downloads, we want to return the blob directly
    // Progress tracking is already handled in fetchWithTimeout via createProgressTrackingResponse
    return response.blob();
  }

  /**
   * Stream download with custom processing
   */
  async streamDownload(
    url: string,
    onChunk: (chunk: Uint8Array, progress: ProgressInfo) => void,
    options?: ApiRequestOptions
  ): Promise<void> {
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
// Don't use the built-in progress tracking since we're handling it manually
      options ? (({ onProgress: _onProgress, ...rest }) => rest)(options) : {}
    );

    if (!response.ok) {
      await this.processResponse(response);
      return;
    }

    if (!response.body) {
      throw createApiErrorFromResponse(500, 'Response body is empty');
    }

    const contentLength = response.headers.get('content-length');
    const total = contentLength ? parseInt(contentLength, 10) : undefined;
    const tracker = new ProgressTracker();
    
    const reader = response.body.getReader();
    let loaded = 0;

    try {
      while (true) {
        const { done, value } = await reader.read();
        
        if (done) {
          // Complete the progress tracking
          if (options?.onProgress) {
            options.onProgress(tracker.complete(total));
          }
          break;
        }

        // Update progress
        loaded += value.length;
        const progress = tracker.update(loaded, total);
        
        // Call progress callback
        if (options?.onProgress) {
          options.onProgress(progress);
        }

        // Call chunk callback
        onChunk(value, progress);
      }
    } finally {
      reader.releaseLock();
    }
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

  /**
   * Upload a file with progress tracking
   */
  upload: <T>(url: string, file: File | FormData, options?: ApiRequestOptions): Promise<T> =>
    apiClient.upload<T>(url, file, options),

  /**
   * Download a file with progress tracking
   */
  download: (url: string, options?: ApiRequestOptions): Promise<Blob> =>
    apiClient.download(url, options),

  /**
   * Stream download with custom chunk processing
   */
  streamDownload: (
    url: string,
    onChunk: (chunk: Uint8Array, progress: ProgressInfo) => void,
    options?: ApiRequestOptions
  ): Promise<void> => apiClient.streamDownload(url, onChunk, options),

  /**
   * Clear all cached responses
   */
  clearCache: (): void => apiClient.clearCache(),

  /**
   * Clear cached responses by key pattern
   */
  clearCacheByPattern: (pattern: string): void => apiClient.clearCacheByPattern(pattern),

  /**
   * Get cache statistics
   */
  getCacheStats: () => apiClient.getCacheStats(),

  /**
   * Cleanup method to prevent memory leaks
   */
  cleanup: (): void => apiClient.cleanup(),
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

/**
 * Utility functions for response caching
 */
export const cache = {
  /**
   * Create cache options with default TTL (5 minutes)
   */
  defaultOptions: (): CacheOptions => ({
    ttl: 5 * 60 * 1000, // 5 minutes
    staleWhileRevalidate: false,
    maxStaleTime: 60 * 1000, // 1 minute
  }),

  /**
   * Create cache options for short-lived data (1 minute)
   */
  shortLived: (): CacheOptions => ({
    ttl: 60 * 1000, // 1 minute
    staleWhileRevalidate: false,
  }),

  /**
   * Create cache options for long-lived data (30 minutes)
   */
  longLived: (): CacheOptions => ({
    ttl: 30 * 60 * 1000, // 30 minutes
    staleWhileRevalidate: true,
    maxStaleTime: 5 * 60 * 1000, // 5 minutes stale time
  }),

  /**
   * Create cache options with stale-while-revalidate enabled
   */
  staleWhileRevalidate: (ttl: number = 5 * 60 * 1000, maxStaleTime: number = 60 * 1000): CacheOptions => ({
    ttl,
    staleWhileRevalidate: true,
    maxStaleTime,
  }),

  /**
   * Create cache options with custom TTL
   */
  withTTL: (ttlMinutes: number): CacheOptions => ({
    ttl: ttlMinutes * 60 * 1000,
    staleWhileRevalidate: false,
  }),

  /**
   * Create cache options with custom key prefix for namespacing
   */
  withPrefix: (keyPrefix: string, ttl: number = 5 * 60 * 1000): CacheOptions => ({
    ttl,
    keyPrefix,
    staleWhileRevalidate: false,
  }),

  /**
   * Disable caching for a request
   */
  disable: () => false as const,
};

/**
 * Utility functions for progress tracking
 */
export const progress = {
  /**
   * Create a FormData object with file information for progress tracking
   */
  createFormData: (files: File[] | FileList, fieldName: string = 'files'): FormData => {
    const formData = new FormData();
    const fileArray = Array.from(files);
    
    fileArray.forEach((file, index) => {
      const name = fileArray.length === 1 ? fieldName : `${fieldName}[${index}]`;
      formData.append(name, file);
    });
    
    return formData;
  },

  /**
   * Calculate total size of files for progress tracking
   */
  getTotalSize: (files: File[] | FileList): number => {
    return Array.from(files).reduce((total, file) => total + file.size, 0);
  },

  /**
   * Format bytes into human-readable string
   */
  formatBytes: (bytes: number, decimals: number = 2): string => {
    if (bytes === 0) return '0 Bytes';

    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'];

    const i = Math.floor(Math.log(bytes) / Math.log(k));

    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
  },

  /**
   * Format transfer rate into human-readable string
   */
  formatRate: (bytesPerSecond: number): string => {
    return `${progress.formatBytes(bytesPerSecond)}/s`;
  },

  /**
   * Format time remaining into human-readable string
   */
  formatTimeRemaining: (milliseconds: number): string => {
    const seconds = Math.floor(milliseconds / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);

    if (hours > 0) {
      return `${hours}h ${minutes % 60}m`;
    } else if (minutes > 0) {
      return `${minutes}m ${seconds % 60}s`;
    } else {
      return `${seconds}s`;
    }
  }
};

export default api;