/**
 * Inventory management service
 * Handles all inventory-related API operations
 */
import { BaseService } from './base.service';

class InventoryService extends BaseService {
  protected basePath = '/inventory';

  /**
   * Get all inventory items
   */
  async getAll(signal?: AbortSignal): Promise<any[]> {
    return super.getAll<any>(signal);
  }

  /**
   * Get inventory item by ID
   */
  async getById(id: string, signal?: AbortSignal): Promise<any> {
    return super.getById<any>(id, signal);
  }

  /**
   * Create new inventory item
   */
  async create(data: any, signal?: AbortSignal): Promise<any> {
    return super.create<any, any>(data, signal);
  }

  /**
   * Update existing inventory item
   */
  async update(id: string, data: any, signal?: AbortSignal): Promise<any> {
    return super.update<any, any>(id, data, signal);
  }

  /**
   * Delete inventory item
   */
  async delete(id: string, signal?: AbortSignal): Promise<void> {
    return super.delete(id, signal);
  }
}

export const inventoryService = new InventoryService();