/**
 * Inventory management service
 * Handles all inventory-related API operations with tenant support
 */
import type { ServiceOptions } from './base.service';
import type { Tenant } from '@/types/models/tenant';
import { api } from '@/lib/api/client';

/**
 * Inventory data structure representing a character's inventory
 */
export interface Inventory {
  type: string;
  id: string;
  attributes: {
    characterId: number;
  };
  relationships: {
    compartments: {
      links: {
        related: string;
        self: string;
      };
      data: Array<{
        type: string;
        id: string;
      }>;
    };
  };
}

/**
 * Compartment data structure representing an inventory section
 */
export interface Compartment {
  type: string;
  id: string;
  attributes: {
    type: number;
    capacity: number;
  };
  relationships: {
    assets: {
      links: {
        related: string;
        self: string;
      };
      data: Array<{
        type: string;
        id: string;
      }>;
    };
  };
}

/**
 * Asset data structure representing an inventory item
 */
export interface Asset {
  type: string;
  id: string;
  attributes: {
    slot: number;
    templateId: number;
    expiration: string;
    referenceId: number;
    referenceType: string;
    referenceData: unknown;
  };
}

/**
 * API response structure for inventory requests
 */
export interface InventoryResponse {
  data: Inventory;
  included: Array<Compartment | Asset>;
}

/**
 * Compartment type enumeration mapping
 */
export enum CompartmentType {
  EQUIPABLES = 1,
  CONSUMABLES = 2,
  SETUP = 3,
  ETC = 4,
  CASH = 5,
}

class InventoryService {
  private basePath = '/api/characters';

  /**
   * Helper function to get compartment type name
   */
  getCompartmentTypeName(type: number): string {
    switch (type) {
      case CompartmentType.EQUIPABLES:
        return "Equipables";
      case CompartmentType.CONSUMABLES:
        return "Consumables";
      case CompartmentType.SETUP:
        return "Setup";
      case CompartmentType.ETC:
        return "Etc";
      case CompartmentType.CASH:
        return "Cash";
      default:
        return `Type ${type}`;
    }
  }

  /**
   * Helper function to get assets for a compartment
   */
  getAssetsForCompartment(compartment: Compartment, included: Array<Compartment | Asset>): Asset[] {
    return compartment.relationships.assets.data
      .map(assetRef => included.find(item => item.type === assetRef.type && item.id === assetRef.id))
      .filter((asset): asset is Asset => {
        return (
          asset !== undefined &&
          asset.type === 'assets' &&
          (asset as Asset).attributes.slot >= 0
        );
      })
      .sort((a, b) => a.attributes.slot - b.attributes.slot);
  }

  /**
   * Fetch inventory data for a character
   */
  async getInventory(tenant: Tenant, characterId: string, options?: ServiceOptions): Promise<InventoryResponse> {
    // Set tenant for this request
    api.setTenant(tenant);
    
    // Use the API client to fetch inventory
    return api.getOne<InventoryResponse>(`${this.basePath}/${characterId}/inventory`, options);
  }

  /**
   * Delete an asset from inventory
   */
  async deleteAsset(
    tenant: Tenant,
    characterId: string,
    compartmentId: string,
    assetId: string,
    options?: ServiceOptions
  ): Promise<void> {
    // Set tenant for this request
    api.setTenant(tenant);
    
    // Use the API client to delete the asset
    return api.delete(
      `${this.basePath}/${characterId}/inventory/compartments/${compartmentId}/assets/${assetId}`,
      options
    );
  }

  /**
   * Get inventory compartments for a character
   */
  async getCompartments(tenant: Tenant, characterId: string, options?: ServiceOptions): Promise<Compartment[]> {
    // Set tenant for this request
    api.setTenant(tenant);
    
    // Fetch full inventory and extract compartments
    const inventoryResponse = await this.getInventory(tenant, characterId, options);
    
    // Filter included items to get only compartments
    return inventoryResponse.included.filter((item): item is Compartment => 
      item.type === 'compartments'
    );
  }

  /**
   * Get assets for a specific compartment
   */
  async getCompartmentAssets(
    tenant: Tenant,
    characterId: string,
    compartmentId: string,
    options?: ServiceOptions
  ): Promise<Asset[]> {
    // Set tenant for this request
    api.setTenant(tenant);
    
    // Fetch full inventory
    const inventoryResponse = await this.getInventory(tenant, characterId, options);
    
    // Find the specific compartment
    const compartment = inventoryResponse.included.find(
      (item): item is Compartment => 
        item.type === 'compartments' && item.id === compartmentId
    );
    
    if (!compartment) {
      throw new Error(`Compartment ${compartmentId} not found`);
    }
    
    // Get assets for this compartment
    return this.getAssetsForCompartment(compartment, inventoryResponse.included);
  }

  /**
   * Check if an asset exists in inventory
   */
  async hasAsset(
    tenant: Tenant,
    characterId: string,
    assetId: string,
    options?: ServiceOptions
  ): Promise<boolean> {
    try {
      // Set tenant for this request
      api.setTenant(tenant);
      
      // Fetch full inventory
      const inventoryResponse = await this.getInventory(tenant, characterId, options);
      
      // Check if asset exists in included items
      return inventoryResponse.included.some(
        item => item.type === 'assets' && item.id === assetId
      );
    } catch (error) {
      // If inventory fetch fails, assume asset doesn't exist
      return false;
    }
  }

  /**
   * Get inventory summary with compartment counts
   */
  async getInventorySummary(tenant: Tenant, characterId: string, options?: ServiceOptions): Promise<{
    totalCompartments: number;
    totalAssets: number;
    compartmentSummary: Array<{
      type: number;
      name: string;
      assetCount: number;
      capacity: number;
    }>;
  }> {
    // Set tenant for this request
    api.setTenant(tenant);
    
    // Fetch full inventory
    const inventoryResponse = await this.getInventory(tenant, characterId, options);
    
    // Get compartments
    const compartments = inventoryResponse.included.filter((item): item is Compartment => 
      item.type === 'compartments'
    );
    
    // Get assets
    const assets = inventoryResponse.included.filter((item): item is Asset => 
      item.type === 'assets'
    );
    
    // Create compartment summary
    const compartmentSummary = compartments.map(compartment => {
      const compartmentAssets = this.getAssetsForCompartment(compartment, inventoryResponse.included);
      
      return {
        type: compartment.attributes.type,
        name: this.getCompartmentTypeName(compartment.attributes.type),
        assetCount: compartmentAssets.length,
        capacity: compartment.attributes.capacity,
      };
    });
    
    return {
      totalCompartments: compartments.length,
      totalAssets: assets.length,
      compartmentSummary,
    };
  }

  // Legacy methods for backward compatibility
  
  /**
   * Legacy method: fetchInventory
   * @deprecated Use getInventory() instead
   */
  async fetchInventory(tenant: Tenant, characterId: string): Promise<InventoryResponse> {
    return this.getInventory(tenant, characterId);
  }
}

export const inventoryService = new InventoryService();

// Export the service class for potential extension
export { InventoryService };

// Re-export types for convenience - note: these are already exported above as interfaces
// Keeping this comment for clarity but removing duplicate exports