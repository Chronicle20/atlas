/**
 * Map data service
 * Handles all map-related API operations
 */
import { BaseService } from './base.service';
import type { MapData } from '@/types/models/map';

class MapsService extends BaseService {
  protected basePath = '/maps';

  /**
   * Get all maps
   */
  async getAll(signal?: AbortSignal): Promise<MapData[]> {
    return super.getAll<MapData>(signal);
  }

  /**
   * Get map by ID
   */
  async getById(id: string, signal?: AbortSignal): Promise<MapData> {
    return super.getById<MapData>(id, signal);
  }

  /**
   * Create new map
   */
  async create(data: Partial<MapData>, signal?: AbortSignal): Promise<MapData> {
    return super.create<MapData, Partial<MapData>>(data, signal);
  }

  /**
   * Update existing map
   */
  async update(id: string, data: Partial<MapData>, signal?: AbortSignal): Promise<MapData> {
    return super.update<MapData, Partial<MapData>>(id, data, signal);
  }

  /**
   * Delete map
   */
  async delete(id: string, signal?: AbortSignal): Promise<void> {
    return super.delete(id, signal);
  }
}

export const mapsService = new MapsService();