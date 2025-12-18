/**
 * NPCs Service
 * 
 * Provides comprehensive NPC management functionality including:
 * - NPC discovery and listing (combines shop and conversation data)
 * - Shop management (create, update, delete shops)
 * - Commodity management (add, update, delete items in shops)
 * - Integration with conversations service
 * - Enhanced error handling and validation
 * - Batch operations for bulk commodity management
 */

import { BaseService, type ServiceOptions, type QueryOptions, type ValidationError } from './base.service';
import type { ApiSingleResponse } from '@/types/api/responses';
import type { NPC, Shop, Commodity, CommodityAttributes, ShopResponse } from '@/types/models/npc';
import type { Tenant } from '@/types/models/tenant';

// Input types for JSON:API requests
interface CreateShopInput {
  data: {
    type: 'shops';
    id: string;
    attributes: {
      npcId: number;
      recharger?: boolean;
    };
    relationships: {
      commodities: {
        data: Array<{ type: 'commodities'; id: string }>;
      };
    };
  };
  included: Array<{
    type: 'commodities';
    id: string;
    attributes: Omit<CommodityAttributes, 'id'>;
  }>;
}

interface UpdateShopInput {
  data: {
    type: 'shops';
    id: string;
    attributes: {
      npcId: number;
      recharger?: boolean;
    };
    relationships: {
      commodities: {
        data: Array<{ type: 'commodities'; id: string }>;
      };
    };
  };
  included: Array<{
    type: 'commodities';
    id: string;
    attributes: CommodityAttributes;
  }>;
}

interface CreateCommodityInput {
  data: {
    type: 'commodities';
    attributes: CommodityAttributes;
  };
}

interface UpdateCommodityInput {
  data: {
    type: 'commodities';
    attributes: Partial<CommodityAttributes>;
  };
}

/**
 * NPCs service class extending BaseService with NPC-specific functionality
 */
class NpcsService extends BaseService {
  protected basePath = '/api/npcs';

  /**
   * Validate commodity data before API calls
   */
  protected override validate<T>(data: T): ValidationError[] {
    const errors: ValidationError[] = [];

    if (this.isCommodityAttributes(data)) {
      if (data.templateId <= 0) {
        errors.push({ field: 'templateId', message: 'Template ID must be positive' });
      }
      if (data.mesoPrice < 0) {
        errors.push({ field: 'mesoPrice', message: 'Meso price must be non-negative' });
      }
      if (data.discountRate < 0 || data.discountRate > 100) {
        errors.push({ field: 'discountRate', message: 'Discount rate must be between 0 and 100' });
      }
      if (data.tokenPrice < 0) {
        errors.push({ field: 'tokenPrice', message: 'Token price must be non-negative' });
      }
      if (data.period < 0) {
        errors.push({ field: 'period', message: 'Period must be non-negative' });
      }
      if (data.levelLimit < 0) {
        errors.push({ field: 'levelLimit', message: 'Level limit must be non-negative' });
      }
    }

    return errors;
  }

  /**
   * Transform request data to proper API format
   */
  protected override transformRequest<T>(data: T): T {
    // Handle JSON:API formatted inputs
    if (this.isCreateShopInput(data) || this.isUpdateShopInput(data) || 
        this.isCreateCommodityInput(data) || this.isUpdateCommodityInput(data)) {
      return data;
    }

    return data;
  }

  /**
   * Get all NPCs by combining shop and conversation data
   * This method aggregates NPCs from both shops and conversations APIs
   */
  async getAllNPCs(tenant: Tenant, options?: QueryOptions): Promise<NPC[]> {
    try {
      // Set tenant context for API calls
      const { api } = await import('@/lib/api/client');
      api.setTenant(tenant);

      // Fetch NPCs with shops
      const shops = await api.getList<Shop>('/api/shops', this.processServiceOptions(options));
      
      // Extract NPCs from shops data
      const npcsWithShops = shops.map((shop: Shop) => ({
        id: shop.attributes.npcId,
        hasShop: true,
        hasConversation: false
      }));

      // Fetch NPCs with conversations
      try {
        const { conversationsService } = await import('@/services/api');
        const conversations = await conversationsService.getAll();

        // Extract NPCs from conversations data
        const npcsWithConversations = conversations.map(conversation => ({
          id: conversation.attributes.npcId,
          hasShop: false,
          hasConversation: true
        }));

        // Combine NPCs from both sources, avoiding duplicates
        const npcMap = new Map<number, NPC>();

        // Add NPCs with shops
        npcsWithShops.forEach((npc: NPC) => {
          npcMap.set(npc.id, npc);
        });

        // Add or update NPCs with conversations
        npcsWithConversations.forEach(npc => {
          if (npcMap.has(npc.id)) {
            // NPC already exists (has a shop), update to indicate it also has a conversation
            const existingNpc = npcMap.get(npc.id)!;
            existingNpc.hasConversation = true;
          } else {
            // New NPC (only has conversation)
            npcMap.set(npc.id, npc);
          }
        });

        // Convert map back to array and sort by ID
        return Array.from(npcMap.values()).sort((a, b) => a.id - b.id);
      } catch (conversationError) {
        console.error('Failed to fetch NPCs with conversations:', conversationError);
        // If fetching conversations fails, return just the NPCs with shops
        return npcsWithShops.sort((a, b) => a.id - b.id);
      }
    } catch (error) {
      console.error('Failed to fetch NPCs:', error);
      throw new Error('Unable to retrieve NPC data. Please try again later.');
    }
  }

  /**
   * Get NPC shop details with commodities
   */
  async getNPCShop(npcId: number, tenant: Tenant, options?: ServiceOptions): Promise<ShopResponse> {
    const { api } = await import('@/lib/api/client');
    api.setTenant(tenant);
    
    const processedOptions = this.processServiceOptions(options);
    return api.get<ShopResponse>(`${this.basePath}/${npcId}/shop?include=commodities`, processedOptions);
  }

  /**
   * Create a new shop for an NPC with initial commodities
   */
  async createShop(
    npcId: number,
    commodities: Omit<CommodityAttributes, 'id'>[],
    tenant: Tenant,
    recharger?: boolean,
    options?: ServiceOptions
  ): Promise<Shop> {
    const { api } = await import('@/lib/api/client');
    api.setTenant(tenant);

    // Validate commodities
    if (options?.validate !== false) {
      for (const commodity of commodities) {
        const validationErrors = this.validate(commodity);
        if (validationErrors.length > 0) {
          throw new Error(`Commodity validation failed: ${validationErrors.map(e => e.message).join(', ')}`);
        }
      }
    }

    // Create commodity data for included section
    const includedCommodities = commodities.map((commodity, index) => ({
      type: 'commodities' as const,
      id: `temp-id-${index}`, // Temporary ID, will be replaced by server
      attributes: commodity
    }));

    // Create commodity references for relationships section
    const commodityReferences = includedCommodities.map(commodity => ({
      type: 'commodities' as const,
      id: commodity.id
    }));

    const input: CreateShopInput = {
      data: {
        type: 'shops',
        id: `shop-${npcId}`,
        attributes: {
          npcId: npcId,
          ...(recharger !== undefined && { recharger })
        },
        relationships: {
          commodities: {
            data: commodityReferences
          }
        }
      },
      included: includedCommodities
    };

    const processedOptions = this.processServiceOptions(options);
    const response = await api.post<{ data: Shop }>(
      `${this.basePath}/${npcId}/shop`, 
      input, 
      processedOptions
    );
    
    return response.data;
  }

  /**
   * Update an existing shop with new commodity data
   */
  async updateShop(
    npcId: number,
    commodities: Commodity[],
    tenant: Tenant,
    recharger?: boolean,
    options?: ServiceOptions
  ): Promise<Shop> {
    const { api } = await import('@/lib/api/client');
    api.setTenant(tenant);

    // Validate commodities
    if (options?.validate !== false) {
      for (const commodity of commodities) {
        const validationErrors = this.validate(commodity.attributes);
        if (validationErrors.length > 0) {
          throw new Error(`Commodity validation failed: ${validationErrors.map(e => e.message).join(', ')}`);
        }
      }
    }

    // Create commodity references for relationships section
    const commodityReferences = commodities.map(commodity => ({
      type: 'commodities' as const,
      id: commodity.id
    }));

    // Create included commodities
    const includedCommodities = commodities.map(commodity => ({
      type: 'commodities' as const,
      id: commodity.id,
      attributes: commodity.attributes
    }));

    const input: UpdateShopInput = {
      data: {
        type: 'shops',
        id: `shop-${npcId}`,
        attributes: {
          npcId: npcId,
          ...(recharger !== undefined && { recharger })
        },
        relationships: {
          commodities: {
            data: commodityReferences
          }
        }
      },
      included: includedCommodities
    };

    const processedOptions = this.processServiceOptions(options);
    const response = await api.put<{ data: Shop }>(
      `${this.basePath}/${npcId}/shop`, 
      input, 
      processedOptions
    );
    
    return response.data;
  }

  /**
   * Add a new commodity to an NPC's shop
   */
  async createCommodity(
    npcId: number,
    commodityAttributes: CommodityAttributes,
    tenant: Tenant,
    options?: ServiceOptions
  ): Promise<Commodity> {
    const { api } = await import('@/lib/api/client');
    api.setTenant(tenant);

    const input: CreateCommodityInput = {
      data: {
        type: 'commodities',
        attributes: commodityAttributes
      }
    };

    const processedOptions = this.processServiceOptions(options);
    const response = await api.post<{ data: Commodity }>(
      `${this.basePath}/${npcId}/shop/relationships/commodities`, 
      input, 
      processedOptions
    );
    
    return response.data;
  }

  /**
   * Update an existing commodity in an NPC's shop
   */
  async updateCommodity(
    npcId: number,
    commodityId: string,
    commodityAttributes: Partial<CommodityAttributes>,
    tenant: Tenant,
    options?: ServiceOptions
  ): Promise<Commodity> {
    const { api } = await import('@/lib/api/client');
    api.setTenant(tenant);

    const input: UpdateCommodityInput = {
      data: {
        type: 'commodities',
        attributes: commodityAttributes
      }
    };

    const processedOptions = this.processServiceOptions(options);
    const response = await api.put<{ data: Commodity }>(
      `${this.basePath}/${npcId}/shop/relationships/commodities/${commodityId}`, 
      input, 
      processedOptions
    );
    
    return response.data;
  }

  /**
   * Remove a commodity from an NPC's shop
   */
  async deleteCommodity(
    npcId: number,
    commodityId: string,
    tenant: Tenant,
    options?: ServiceOptions
  ): Promise<void> {
    const { api } = await import('@/lib/api/client');
    api.setTenant(tenant);
    
    const processedOptions = this.processServiceOptions(options);
    return api.delete(
      `${this.basePath}/${npcId}/shop/relationships/commodities/${commodityId}`, 
      processedOptions
    );
  }

  /**
   * Delete all commodities for a specific NPC
   */
  async deleteAllCommoditiesForNPC(
    npcId: number,
    tenant: Tenant,
    options?: ServiceOptions
  ): Promise<void> {
    const { api } = await import('@/lib/api/client');
    api.setTenant(tenant);
    
    const processedOptions = this.processServiceOptions(options);
    return api.delete(
      `${this.basePath}/${npcId}/shop/relationships/commodities`, 
      processedOptions
    );
  }

  /**
   * Delete all shops (bulk operation)
   */
  async deleteAllShops(tenant: Tenant, options?: ServiceOptions): Promise<void> {
    const { api } = await import('@/lib/api/client');
    api.setTenant(tenant);
    
    const processedOptions = this.processServiceOptions(options);
    return api.delete('/api/shops', processedOptions);
  }

  /**
   * Batch create commodities for an NPC
   */
  async createCommoditiesBatch(
    npcId: number,
    commodities: CommodityAttributes[],
    tenant: Tenant,
    options?: ServiceOptions
  ): Promise<Commodity[]> {
    const results: Commodity[] = [];
    
    for (const commodity of commodities) {
      try {
        const result = await this.createCommodity(npcId, commodity, tenant, options);
        results.push(result);
      } catch (error) {
        console.error(`Failed to create commodity for NPC ${npcId}:`, error);
        throw error;
      }
    }
    
    return results;
  }

  /**
   * Get NPCs that have shops
   */
  async getNPCsWithShops(tenant: Tenant, options?: QueryOptions): Promise<NPC[]> {
    const allNPCs = await this.getAllNPCs(tenant, options);
    return allNPCs.filter(npc => npc.hasShop);
  }

  /**
   * Get NPCs that have conversations
   */
  async getNPCsWithConversations(tenant: Tenant, options?: QueryOptions): Promise<NPC[]> {
    const allNPCs = await this.getAllNPCs(tenant, options);
    return allNPCs.filter(npc => npc.hasConversation);
  }

  /**
   * Check if an NPC exists and what features it has
   */
  async getNPCById(npcId: number, tenant: Tenant, options?: ServiceOptions): Promise<NPC | null> {
    const allNPCs = await this.getAllNPCs(tenant, options);
    return allNPCs.find(npc => npc.id === npcId) || null;
  }

  // === HELPER METHODS ===

  /**
   * Process service options and merge with defaults
   */
  private processServiceOptions(options?: ServiceOptions): any {
    const processedOptions: any = { ...options };
    
    // Configure caching (NPCs data changes frequently, use shorter cache)
    if (options?.useCache !== false) {
      processedOptions.cacheConfig = options?.cacheConfig || {
        ttl: 2 * 60 * 1000, // 2 minutes
        staleWhileRevalidate: true,
        maxStaleTime: 30 * 1000, // 30 seconds
      };
    } else {
      processedOptions.cacheConfig = false;
    }
    
    return processedOptions;
  }

  // === TYPE GUARDS ===

  private isCommodityAttributes(data: unknown): data is CommodityAttributes {
    return (
      typeof data === 'object' &&
      data !== null &&
      'templateId' in data &&
      'mesoPrice' in data &&
      'discountRate' in data &&
      'tokenTemplateId' in data &&
      'tokenPrice' in data &&
      'period' in data &&
      'levelLimit' in data
    );
  }

  private isCreateShopInput(data: unknown): data is CreateShopInput {
    return (
      typeof data === 'object' &&
      data !== null &&
      'data' in data &&
      'included' in data &&
      typeof (data as any).data === 'object' &&
      (data as any).data.type === 'shops'
    );
  }

  private isUpdateShopInput(data: unknown): data is UpdateShopInput {
    return (
      typeof data === 'object' &&
      data !== null &&
      'data' in data &&
      'included' in data &&
      typeof (data as any).data === 'object' &&
      (data as any).data.type === 'shops'
    );
  }

  private isCreateCommodityInput(data: unknown): data is CreateCommodityInput {
    return (
      typeof data === 'object' &&
      data !== null &&
      'data' in data &&
      typeof (data as any).data === 'object' &&
      (data as any).data.type === 'commodities'
    );
  }

  private isUpdateCommodityInput(data: unknown): data is UpdateCommodityInput {
    return (
      typeof data === 'object' &&
      data !== null &&
      'data' in data &&
      typeof (data as any).data === 'object' &&
      (data as any).data.type === 'commodities'
    );
  }
}

// Create and export a singleton instance
export const npcsService = new NpcsService();

// Export types for use in other files  
export type { 
  NPC, 
  Shop, 
  Commodity, 
  CommodityAttributes, 
  ShopResponse 
};