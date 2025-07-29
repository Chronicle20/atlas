/**
 * Base service class with common functionality for all API services
 * 
 * This class provides:
 * - Common CRUD operations
 * - Consistent error handling
 * - Request cancellation support
 * - Standard response transformations
 */
import { apiClient } from '@/lib/api/client';

export abstract class BaseService {
  protected abstract basePath: string;

  /**
   * Generic GET request for retrieving all resources
   */
  protected async getAll<T>(signal?: AbortSignal): Promise<T[]> {
    return apiClient.get<T[]>(this.basePath, { signal });
  }

  /**
   * Generic GET request for retrieving a single resource by ID
   */
  protected async getById<T>(id: string, signal?: AbortSignal): Promise<T> {
    return apiClient.get<T>(`${this.basePath}/${id}`, { signal });
  }

  /**
   * Generic POST request for creating a new resource
   */
  protected async create<T, D = unknown>(data: D, signal?: AbortSignal): Promise<T> {
    return apiClient.post<T>(this.basePath, data, { signal });
  }

  /**
   * Generic PUT request for updating a resource
   */
  protected async update<T, D = unknown>(id: string, data: D, signal?: AbortSignal): Promise<T> {
    return apiClient.put<T>(`${this.basePath}/${id}`, data, { signal });
  }

  /**
   * Generic DELETE request for removing a resource
   */
  protected async delete(id: string, signal?: AbortSignal): Promise<void> {
    return apiClient.delete(`${this.basePath}/${id}`, { signal });
  }
}