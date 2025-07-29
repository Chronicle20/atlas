/**
 * Base service class with common functionality for all API services
 * 
 * This class provides:
 * - Common CRUD operations with full API client feature support
 * - Consistent error handling and retry logic
 * - Request cancellation and deduplication support
 * - Response caching with configurable TTL
 * - Progress tracking for uploads/downloads
 * - Standard response transformations
 * - Batch operations and validation helpers
 */
import { apiClient, type ApiRequestOptions, type ProgressCallback, type CacheOptions } from '@/lib/api/client';
import { api } from '@/lib/api/client';

/**
 * Service configuration options that can be set per operation or service-wide
 */
export interface ServiceOptions extends ApiRequestOptions {
  /** Whether to use caching for this operation */
  useCache?: boolean;
  /** Custom cache configuration */
  cacheConfig?: CacheOptions;
  /** Whether to validate data before sending */
  validate?: boolean;
}

/**
 * Batch operation configuration
 */
export interface BatchOptions {
  /** Maximum number of concurrent requests */
  concurrency?: number;
  /** Whether to fail fast on first error or collect all errors */
  failFast?: boolean;
  /** Delay between batch requests in milliseconds */
  delay?: number;
}

/**
 * Search/filter options for getAll operations
 */
export interface QueryOptions extends ServiceOptions {
  /** Search query string */
  search?: string;
  /** Field to sort by */
  sortBy?: string;
  /** Sort direction */
  sortOrder?: 'asc' | 'desc';
  /** Number of items per page */
  limit?: number;
  /** Page offset or page number */
  offset?: number;
  /** Additional filters as key-value pairs */
  filters?: Record<string, unknown>;
}

/**
 * Result type for batch operations
 */
export interface BatchResult<T> {
  /** Successfully processed items */
  successes: T[];
  /** Failed items with their errors */
  failures: Array<{ item: unknown; error: Error }>;
  /** Total number of items processed */
  total: number;
  /** Number of successful operations */
  successCount: number;
  /** Number of failed operations */
  failureCount: number;
}

/**
 * Validation error details
 */
export interface ValidationError {
  field: string;
  message: string;
  value?: unknown;
}

/**
 * Abstract base service class with comprehensive functionality
 */
export abstract class BaseService {
  /** Base API path for this service (e.g., '/api/v1/users') */
  protected abstract basePath: string;
  
  /** Default cache configuration for this service */
  protected defaultCacheConfig: CacheOptions = {
    ttl: 5 * 60 * 1000, // 5 minutes
    staleWhileRevalidate: true,
    maxStaleTime: 60 * 1000, // 1 minute
  };

  /**
   * Validate data before sending to API (override in subclasses)
   */
  protected validate<T>(data: T): ValidationError[] {
    // Default implementation - no validation
    // Subclasses should override this method
    return [];
  }

  /**
   * Transform data before sending to API (override in subclasses)
   */
  protected transformRequest<T>(data: T): T {
    // Default implementation - no transformation
    // Subclasses can override this method
    return data;
  }

  /**
   * Transform data after receiving from API (override in subclasses)
   */
  protected transformResponse<T>(data: T): T {
    // Default implementation - no transformation
    // Subclasses can override this method
    return data;
  }

  /**
   * Build query string from QueryOptions
   */
  private buildQueryString(options?: QueryOptions): string {
    const params = new URLSearchParams();
    
    if (options?.search) params.append('search', options.search);
    if (options?.sortBy) params.append('sortBy', options.sortBy);
    if (options?.sortOrder) params.append('sortOrder', options.sortOrder);
    if (options?.limit) params.append('limit', options.limit.toString());
    if (options?.offset) params.append('offset', options.offset.toString());
    
    if (options?.filters) {
      Object.entries(options.filters).forEach(([key, value]) => {
        if (value !== null && value !== undefined) {
          params.append(`filter[${key}]`, String(value));
        }
      });
    }
    
    const queryString = params.toString();
    return queryString ? `?${queryString}` : '';
  }

  /**
   * Process service options and merge with defaults
   */
  private processOptions(options?: ServiceOptions): ApiRequestOptions {
    const processedOptions: ApiRequestOptions = { ...options };
    
    // Configure caching
    if (options?.useCache !== false) {
      processedOptions.cache = options?.cacheConfig || this.defaultCacheConfig;
    } else {
      processedOptions.cache = false;
    }
    
    return processedOptions;
  }

  /**
   * Generic GET request for retrieving all resources with advanced query support
   */
  protected async getAll<T>(options?: QueryOptions): Promise<T[]> {
    const queryString = this.buildQueryString(options);
    const url = `${this.basePath}${queryString}`;
    const processedOptions = this.processOptions(options);
    
    const data = await api.getList<T>(url, processedOptions);
    return data.map(item => this.transformResponse(item));
  }

  /**
   * Generic GET request for retrieving a single resource by ID
   */
  protected async getById<T>(id: string, options?: ServiceOptions): Promise<T> {
    const processedOptions = this.processOptions(options);
    const data = await api.getOne<T>(`${this.basePath}/${id}`, processedOptions);
    return this.transformResponse(data);
  }

  /**
   * Generic GET request for checking if a resource exists
   */
  protected async exists(id: string, options?: ServiceOptions): Promise<boolean> {
    try {
      await this.getById(id, options);
      return true;
    } catch (error) {
      // If it's a 404 error, the resource doesn't exist
      if (error && typeof error === 'object' && 'status' in error && error.status === 404) {
        return false;
      }
      // Re-throw other errors (network issues, etc.)
      throw error;
    }
  }

  /**
   * Generic POST request for creating a new resource with validation
   */
  protected async create<T, D = unknown>(data: D, options?: ServiceOptions): Promise<T> {
    // Validate data if validation is enabled
    if (options?.validate !== false) {
      const validationErrors = this.validate(data);
      if (validationErrors.length > 0) {
        throw new Error(`Validation failed: ${validationErrors.map(e => e.message).join(', ')}`);
      }
    }
    
    const transformedData = this.transformRequest(data);
    const processedOptions = this.processOptions(options);
    
    const result = await api.post<T>(this.basePath, transformedData, processedOptions);
    return this.transformResponse(result);
  }

  /**
   * Generic PUT request for updating a resource with validation
   */
  protected async update<T, D = unknown>(id: string, data: D, options?: ServiceOptions): Promise<T> {
    // Validate data if validation is enabled
    if (options?.validate !== false) {
      const validationErrors = this.validate(data);
      if (validationErrors.length > 0) {
        throw new Error(`Validation failed: ${validationErrors.map(e => e.message).join(', ')}`);
      }
    }
    
    const transformedData = this.transformRequest(data);
    const processedOptions = this.processOptions(options);
    
    const result = await api.put<T>(`${this.basePath}/${id}`, transformedData, processedOptions);
    return this.transformResponse(result);
  }

  /**
   * Generic PATCH request for partial updates
   */
  protected async patch<T, D = unknown>(id: string, data: D, options?: ServiceOptions): Promise<T> {
    const transformedData = this.transformRequest(data);
    const processedOptions = this.processOptions(options);
    
    const result = await api.patch<T>(`${this.basePath}/${id}`, transformedData, processedOptions);
    return this.transformResponse(result);
  }

  /**
   * Generic DELETE request for removing a resource
   */
  protected async delete(id: string, options?: ServiceOptions): Promise<void> {
    const processedOptions = this.processOptions(options);
    return api.delete(`${this.basePath}/${id}`, processedOptions);
  }

  /**
   * Batch create multiple resources
   */
  protected async createBatch<T, D = unknown>(
    items: D[], 
    options?: ServiceOptions,
    batchOptions?: BatchOptions
  ): Promise<BatchResult<T>> {
    const concurrency = batchOptions?.concurrency || 5;
    const failFast = batchOptions?.failFast || false;
    const delay = batchOptions?.delay || 0;
    
    const successes: T[] = [];
    const failures: Array<{ item: D; error: Error }> = [];
    
    // Process items in batches
    for (let i = 0; i < items.length; i += concurrency) {
      const batch = items.slice(i, i + concurrency);
      
      const promises = batch.map(async (item) => {
        try {
          const result = await this.create<T, D>(item, options);
          return { success: true, item, result };
        } catch (error) {
          return { success: false, item, error: error as Error };
        }
      });
      
      const results = await Promise.all(promises);
      
      for (const result of results) {
        if (result.success) {
          successes.push(result.result);
        } else {
          failures.push({ item: result.item, error: result.error });
          
          // Fail fast if enabled
          if (failFast) {
            throw result.error;
          }
        }
      }
      
      // Add delay between batches if specified
      if (delay > 0 && i + concurrency < items.length) {
        await new Promise(resolve => setTimeout(resolve, delay));
      }
    }
    
    return {
      successes,
      failures,
      total: items.length,
      successCount: successes.length,
      failureCount: failures.length,
    };
  }

  /**
   * Batch update multiple resources
   */
  protected async updateBatch<T, D = unknown>(
    updates: Array<{ id: string; data: D }>,
    options?: ServiceOptions,
    batchOptions?: BatchOptions
  ): Promise<BatchResult<T>> {
    const concurrency = batchOptions?.concurrency || 5;
    const failFast = batchOptions?.failFast || false;
    const delay = batchOptions?.delay || 0;
    
    const successes: T[] = [];
    const failures: Array<{ item: { id: string; data: D }; error: Error }> = [];
    
    // Process items in batches
    for (let i = 0; i < updates.length; i += concurrency) {
      const batch = updates.slice(i, i + concurrency);
      
      const promises = batch.map(async (update) => {
        try {
          const result = await this.update<T, D>(update.id, update.data, options);
          return { success: true, item: update, result };
        } catch (error) {
          return { success: false, item: update, error: error as Error };
        }
      });
      
      const results = await Promise.all(promises);
      
      for (const result of results) {
        if (result.success) {
          successes.push(result.result);
        } else {
          failures.push({ item: result.item, error: result.error });
          
          // Fail fast if enabled
          if (failFast) {
            throw result.error;
          }
        }
      }
      
      // Add delay between batches if specified
      if (delay > 0 && i + concurrency < updates.length) {
        await new Promise(resolve => setTimeout(resolve, delay));
      }
    }
    
    return {
      successes,
      failures,
      total: updates.length,
      successCount: successes.length,
      failureCount: failures.length,
    };
  }

  /**
   * Batch delete multiple resources
   */
  protected async deleteBatch(
    ids: string[],
    options?: ServiceOptions,
    batchOptions?: BatchOptions
  ): Promise<BatchResult<string>> {
    const concurrency = batchOptions?.concurrency || 5;
    const failFast = batchOptions?.failFast || false;
    const delay = batchOptions?.delay || 0;
    
    const successes: string[] = [];
    const failures: Array<{ item: string; error: Error }> = [];
    
    // Process items in batches
    for (let i = 0; i < ids.length; i += concurrency) {
      const batch = ids.slice(i, i + concurrency);
      
      const promises = batch.map(async (id) => {
        try {
          await this.delete(id, options);
          return { success: true, item: id };
        } catch (error) {
          return { success: false, item: id, error: error as Error };
        }
      });
      
      const results = await Promise.all(promises);
      
      for (const result of results) {
        if (result.success) {
          successes.push(result.item);
        } else {
          failures.push({ item: result.item, error: result.error });
          
          // Fail fast if enabled
          if (failFast) {
            throw result.error;
          }
        }
      }
      
      // Add delay between batches if specified
      if (delay > 0 && i + concurrency < ids.length) {
        await new Promise(resolve => setTimeout(resolve, delay));
      }
    }
    
    return {
      successes,
      failures,
      total: ids.length,
      successCount: successes.length,
      failureCount: failures.length,
    };
  }

  /**
   * Upload a file with progress tracking
   */
  protected async uploadFile<T>(
    file: File | FormData, 
    endpoint?: string,
    onProgress?: ProgressCallback,
    options?: ServiceOptions
  ): Promise<T> {
    const url = endpoint ? `${this.basePath}${endpoint}` : `${this.basePath}/upload`;
    const processedOptions = this.processOptions(options);
    if (onProgress) {
      processedOptions.onProgress = onProgress;
    }
    
    return api.upload<T>(url, file, processedOptions);
  }

  /**
   * Download a file with progress tracking
   */
  protected async downloadFile(
    endpoint: string,
    onProgress?: ProgressCallback,
    options?: ServiceOptions
  ): Promise<Blob> {
    const url = `${this.basePath}${endpoint}`;
    const processedOptions = this.processOptions(options);
    if (onProgress) {
      processedOptions.onProgress = onProgress;
    }
    
    return api.download(url, processedOptions);
  }

  /**
   * Clear cache for this service
   */
  protected clearServiceCache(): void {
    // Create a pattern that matches this service's cache keys
    const pattern = btoa(`.*${this.basePath.replace(/\//g, '\\/')}.*`);
    api.clearCacheByPattern(pattern);
  }

  /**
   * Get cache statistics for this service
   */
  protected getServiceCacheStats() {
    const allStats = api.getCacheStats();
    const servicePattern = new RegExp(btoa(`.*${this.basePath.replace(/\//g, '\\/')}.*`));
    
    const serviceEntries = allStats.entries.filter(entry => 
      servicePattern.test(entry.key)
    );
    
    return {
      size: serviceEntries.length,
      entries: serviceEntries
    };
  }
}