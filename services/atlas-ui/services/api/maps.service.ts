/**
 * Maps Service
 * 
 * Provides comprehensive map data management functionality including:
 * - Basic map CRUD operations
 * - Enhanced error handling and caching
 * - Batch operations and validation
 * - Map-specific query operations
 */

import { BaseService, type ServiceOptions, type QueryOptions, type ValidationError } from './base.service';
import type { Tenant } from '@/types/models/tenant';

// Map attributes interface
export interface MapAttributes {
  name: string;
  streetName: string;
}

// Map interface
export interface MapData {
  id: string;
  attributes: MapAttributes;
}

// Create map input types for JSON:API structure
interface CreateMapInput {
  data: {
    type: 'maps';
    attributes: MapAttributes;
  };
}

interface UpdateMapInput {
  data: {
    id: string;
    type: 'maps';
    attributes: Partial<MapAttributes>;
  };
}

/**
 * Maps service class extending BaseService with map-specific functionality
 */
class MapsService extends BaseService {
  protected basePath = '/api/data/maps';

  /**
   * Validate map data before API calls
   */
  protected override validate<T>(data: T): ValidationError[] {
    const errors: ValidationError[] = [];

    if (this.isMapAttributes(data)) {
      if (!data.name || data.name.trim().length === 0) {
        errors.push({ field: 'name', message: 'Map name is required' });
      }
      if (!data.streetName || data.streetName.trim().length === 0) {
        errors.push({ field: 'streetName', message: 'Street name is required' });
      }
    }

    return errors;
  }

  /**
   * Transform request data to proper API format
   */
  protected override transformRequest<T>(data: T): T {
    // For create/update operations, ensure proper JSON:API structure
    if (this.isCreateMapInput(data) || this.isUpdateMapInput(data)) {
      return data;
    }
    
    // For raw attributes, wrap in JSON:API structure
    if (this.isMapAttributes(data)) {
      return {
        data: {
          type: 'maps',
          attributes: data,
        },
      } as T;
    }

    return data;
  }

  /**
   * Sort maps by name
   */
  private sortMaps(maps: MapData[]): MapData[] {
    return maps.sort((a, b) => a.attributes.name.localeCompare(b.attributes.name));
  }

  /**
   * Get all maps with sorting
   */
  async getAllMaps(options?: QueryOptions): Promise<MapData[]> {
    const maps = await this.getAll<MapData>(options);
    return this.sortMaps(maps);
  }

  /**
   * Get map by ID
   */
  async getMapById(id: string, options?: ServiceOptions): Promise<MapData> {
    return this.getById<MapData>(id, options);
  }

  /**
   * Create a new map
   */
  async createMap(attributes: MapAttributes, options?: ServiceOptions): Promise<MapData> {
    const input: CreateMapInput = {
      data: {
        type: 'maps',
        attributes,
      },
    };

    return this.create<MapData, CreateMapInput>(input, options);
  }

  /**
   * Update an existing map
   */
  async updateMap(
    map: MapData,
    updatedAttributes: Partial<MapAttributes>,
    options?: ServiceOptions
  ): Promise<MapData> {
    const input: UpdateMapInput = {
      data: {
        id: map.id,
        type: 'maps',
        attributes: {
          ...map.attributes,
          ...updatedAttributes,
        },
      },
    };

    await this.patch<void, UpdateMapInput>(map.id, input, options);
    
    // Return updated map object
    return {
      ...map,
      attributes: { ...map.attributes, ...updatedAttributes },
    };
  }

  /**
   * Delete a map
   */
  async deleteMap(mapId: string, options?: ServiceOptions): Promise<void> {
    return this.delete(mapId, options);
  }

  /**
   * Search maps by name
   */
  async searchMapsByName(name: string, options?: ServiceOptions): Promise<MapData[]> {
    const searchOptions: QueryOptions = {
      ...options,
      search: name,
      filters: {
        name: name,
      },
    };
    
    const maps = await this.getAll<MapData>(searchOptions);
    return this.sortMaps(maps);
  }

  /**
   * Get maps by street name
   */
  async getMapsByStreetName(streetName: string, options?: ServiceOptions): Promise<MapData[]> {
    const searchOptions: QueryOptions = {
      ...options,
      filters: {
        streetName: streetName,
      },
    };
    
    const maps = await this.getAll<MapData>(searchOptions);
    return this.sortMaps(maps);
  }

  /**
   * Legacy method for backward compatibility with the old API
   * @deprecated Use getMapById instead
   */
  async fetchMap(tenant: Tenant, mapId: string): Promise<MapData> {
    // For backward compatibility, we'll use the tenant information if needed
    // The new API client handles tenant information automatically
    return this.getMapById(mapId);
  }

  // === TYPE GUARDS ===

  private isMapAttributes(data: unknown): data is MapAttributes {
    return (
      typeof data === 'object' &&
      data !== null &&
      'name' in data &&
      'streetName' in data
    );
  }

  private isCreateMapInput(data: unknown): data is CreateMapInput {
    return (
      typeof data === 'object' &&
      data !== null &&
      'data' in data &&
      typeof (data as any).data === 'object' &&
      'type' in (data as any).data &&
      (data as any).data.type === 'maps' &&
      'attributes' in (data as any).data
    );
  }

  private isUpdateMapInput(data: unknown): data is UpdateMapInput {
    return (
      typeof data === 'object' &&
      data !== null &&
      'data' in data &&
      typeof (data as any).data === 'object' &&
      'id' in (data as any).data &&
      'type' in (data as any).data &&
      (data as any).data.type === 'maps' &&
      'attributes' in (data as any).data
    );
  }
}

// Create and export a singleton instance
export const mapsService = new MapsService();

// For backward compatibility, export MapData as Map
export type Map = MapData;