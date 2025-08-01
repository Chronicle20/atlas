/**
 * Tenants Service
 * 
 * Provides comprehensive tenant management functionality including:
 * - Basic tenant CRUD operations (new API)
 * - Tenant configuration management (existing API)  
 * - Enhanced error handling and caching
 * - Batch operations and validation
 */

import { BaseService, type ServiceOptions, type QueryOptions, type ValidationError } from './base.service';
import type { ApiSingleResponse } from '@/types/api/responses';

// Lightweight tenant attributes for the new API
interface TenantBasicAttributes {
  name: string;
  region: string;
  majorVersion: number;
  minorVersion: number;
}

// Lightweight tenant type for the new API
interface TenantBasic {
  id: string;
  attributes: TenantBasicAttributes;
}

// Full tenant configuration attributes for the existing API
interface TenantConfigAttributes {
  region: string;
  majorVersion: number;
  minorVersion: number;
  usesPin: boolean;
  characters: {
    templates: {
      jobIndex: number;
      subJobIndex: number;
      gender: number;
      mapId: number;
      faces: number[];
      hairs: number[];
      hairColors: number[];
      skinColors: number[];
      tops: number[];
      bottoms: number[];
      shoes: number[];
      weapons: number[];
      items: number[];
      skills: number[];
    }[];
  };
  npcs: {
    npcId: number;
    impl: string;
  }[];
  socket: {
    handlers: {
      opCode: string;
      validator: string;
      handler: string;
      options: unknown;
    }[];
    writers: {
      opCode: string;
      writer: string;
      options: unknown;
    }[];
  };
  worlds: {
    name: string;
    flag: string;
    serverMessage: string;
    eventMessage: string;
    whyAmIRecommended: string;
  }[];
}

// Full tenant configuration type for the existing API
interface TenantConfig {
  id: string;
  attributes: TenantConfigAttributes;
}

// For backward compatibility
export type TenantAttributes = TenantConfigAttributes;
export type Tenant = TenantBasic;

// Create tenant input types
interface CreateTenantInput {
  data: {
    type: 'tenants';
    attributes: TenantBasicAttributes;
  };
}

interface UpdateTenantInput {
  data: {
    id: string;
    type: 'tenants';
    attributes: Partial<TenantBasicAttributes>;
  };
}

interface CreateTenantConfigInput {
  data: {
    type: 'tenants';
    attributes: TenantConfigAttributes;
  };
}

interface UpdateTenantConfigInput {
  data: {
    id: string;
    type: 'tenants';
    attributes: Partial<TenantConfigAttributes>;
  };
}

/**
 * Tenants service class extending BaseService with tenant-specific functionality
 */
class TenantsService extends BaseService {
  protected basePath = '/api/tenants';
  private configBasePath = '/api/configurations/tenants';

  /**
   * Validate tenant data before API calls
   */
  protected override validate<T>(data: T): ValidationError[] {
    const errors: ValidationError[] = [];

    if (this.isTenantBasicAttributes(data)) {
      if (!data.name || data.name.trim().length === 0) {
        errors.push({ field: 'name', message: 'Tenant name is required' });
      }
      if (!data.region || data.region.trim().length === 0) {
        errors.push({ field: 'region', message: 'Region is required' });
      }
      if (data.majorVersion < 0) {
        errors.push({ field: 'majorVersion', message: 'Major version must be non-negative' });
      }
      if (data.minorVersion < 0) {
        errors.push({ field: 'minorVersion', message: 'Minor version must be non-negative' });
      }
    }

    return errors;
  }

  /**
   * Transform request data to proper API format
   */
  protected override transformRequest<T>(data: T): T {
    // For create/update operations, ensure proper JSON:API structure
    if (this.isCreateTenantInput(data) || this.isUpdateTenantInput(data)) {
      return data;
    }
    
    // For raw attributes, wrap in JSON:API structure
    if (this.isTenantBasicAttributes(data)) {
      return {
        data: {
          type: 'tenants',
          attributes: data,
        },
      } as T;
    }

    return data;
  }

  /**
   * Transform response data (sort socket handlers/writers)
   */
  protected override transformResponse<T>(data: T): T {
    if (this.isTenantConfig(data)) {
      const transformed = { ...data };
      if (transformed.attributes.socket) {
        // Sort handlers and writers by opCode
        transformed.attributes.socket.handlers = [...transformed.attributes.socket.handlers].sort(
          (a, b) => parseInt(a.opCode, 16) - parseInt(b.opCode, 16)
        );
        transformed.attributes.socket.writers = [...transformed.attributes.socket.writers].sort(
          (a, b) => parseInt(a.opCode, 16) - parseInt(b.opCode, 16)
        );
      }
      return transformed as T;
    }
    return data;
  }

  /**
   * Sort tenants by region, major version, minor version
   */
  private sortTenants<T extends TenantBasic | TenantConfig>(tenants: T[]): T[] {
    return tenants.sort((a, b) => {
      if (a.attributes.region !== b.attributes.region) {
        return a.attributes.region.localeCompare(b.attributes.region);
      }
      if (a.attributes.majorVersion !== b.attributes.majorVersion) {
        return a.attributes.majorVersion - b.attributes.majorVersion;
      }
      return a.attributes.minorVersion - b.attributes.minorVersion;
    });
  }

  /**
   * Get all basic tenants with sorting
   */
  async getAllTenants(options?: QueryOptions): Promise<TenantBasic[]> {
    const tenants = await this.getAll<TenantBasic>(options);
    return this.sortTenants(tenants);
  }

  /**
   * Get tenant by ID
   */
  async getTenantById(id: string, options?: ServiceOptions): Promise<TenantBasic> {
    return this.getById<TenantBasic>(id, options);
  }

  /**
   * Create a new tenant
   */
  async createTenant(attributes: TenantBasicAttributes, options?: ServiceOptions): Promise<TenantBasic> {
    const input: CreateTenantInput = {
      data: {
        type: 'tenants',
        attributes,
      },
    };

    const response = await this.create<ApiSingleResponse<TenantBasic>, CreateTenantInput>(input, options);
    return response.data;
  }

  /**
   * Update an existing tenant
   */
  async updateTenant(
    tenant: TenantBasic,
    updatedAttributes: Partial<TenantBasicAttributes>,
    options?: ServiceOptions
  ): Promise<TenantBasic> {
    const input: UpdateTenantInput = {
      data: {
        id: tenant.id,
        type: 'tenants',
        attributes: {
          ...tenant.attributes,
          ...updatedAttributes,
        },
      },
    };

    await this.patch<void, UpdateTenantInput>(tenant.id, input, options);
    
    // Return updated tenant object
    return {
      ...tenant,
      attributes: { ...tenant.attributes, ...updatedAttributes },
    };
  }

  /**
   * Delete a tenant
   */
  async deleteTenant(tenantId: string, options?: ServiceOptions): Promise<void> {
    return this.delete(tenantId, options);
  }

  // === TENANT CONFIGURATION METHODS ===

  /**
   * Get all tenant configurations with sorting
   */
  async getAllTenantConfigurations(options?: QueryOptions): Promise<TenantConfig[]> {
    const processedOptions = options ? { ...options } : {};
    
    // Override basePath temporarily for configurations
    const originalBasePath = this.basePath;
    (this as any).basePath = this.configBasePath;
    
    try {
      const configs = await this.getAll<TenantConfig>(processedOptions);
      const sortedConfigs = this.sortTenants(configs);
      
      // Transform each config to sort socket handlers/writers
      return sortedConfigs.map(config => this.transformResponse(config));
    } finally {
      // Restore original basePath
      (this as any).basePath = originalBasePath;
    }
  }

  /**
   * Get tenant configuration by ID
   */
  async getTenantConfigurationById(id: string, options?: ServiceOptions): Promise<TenantConfig> {
    const processedOptions = options ? { ...options } : {};
    
    // Override basePath temporarily for configurations
    const originalBasePath = this.basePath;
    (this as any).basePath = this.configBasePath;
    
    try {
      const config = await this.getById<TenantConfig>(id, processedOptions);
      return this.transformResponse(config);
    } finally {
      // Restore original basePath
      (this as any).basePath = originalBasePath;
    }
  }

  /**
   * Create a new tenant configuration
   */
  async createTenantConfiguration(
    attributes: TenantConfigAttributes,
    options?: ServiceOptions
  ): Promise<TenantConfig> {
    const input: CreateTenantConfigInput = {
      data: {
        type: 'tenants',
        attributes,
      },
    };

    // Override basePath temporarily for configurations
    const originalBasePath = this.basePath;
    (this as any).basePath = this.configBasePath;
    
    try {
      const response = await this.create<ApiSingleResponse<TenantConfig>, CreateTenantConfigInput>(input, options);
      return response.data;
    } finally {
      // Restore original basePath
      (this as any).basePath = originalBasePath;
    }
  }

  /**
   * Update an existing tenant configuration
   */
  async updateTenantConfiguration(
    tenant: TenantConfig,
    updatedAttributes: Partial<TenantConfigAttributes>,
    options?: ServiceOptions
  ): Promise<TenantConfig> {
    const input: UpdateTenantConfigInput = {
      data: {
        id: tenant.id,
        type: 'tenants',
        attributes: {
          ...tenant.attributes,
          ...updatedAttributes,
        },
      },
    };

    // Override basePath temporarily for configurations
    const originalBasePath = this.basePath;
    (this as any).basePath = this.configBasePath;
    
    try {
      await this.patch<void, UpdateTenantConfigInput>(tenant.id, input, options);
      
      // Return updated tenant object
      return {
        ...tenant,
        attributes: { ...tenant.attributes, ...updatedAttributes },
      };
    } finally {
      // Restore original basePath
      (this as any).basePath = originalBasePath;
    }
  }

  /**
   * Create tenant configuration from template
   */
  createTenantFromTemplate(template: { attributes: TenantConfigAttributes }): TenantConfigAttributes {
    // Create a deep copy of the template attributes
    return JSON.parse(JSON.stringify(template.attributes));
  }

  // === TYPE GUARDS ===

  private isTenantBasicAttributes(data: unknown): data is TenantBasicAttributes {
    return (
      typeof data === 'object' &&
      data !== null &&
      'name' in data &&
      'region' in data &&
      'majorVersion' in data &&
      'minorVersion' in data
    );
  }

  private isTenantConfig(data: unknown): data is TenantConfig {
    return (
      typeof data === 'object' &&
      data !== null &&
      'id' in data &&
      'attributes' in data &&
      typeof (data as any).attributes === 'object' &&
      'socket' in (data as any).attributes
    );
  }

  private isCreateTenantInput(data: unknown): data is CreateTenantInput {
    return (
      typeof data === 'object' &&
      data !== null &&
      'data' in data &&
      typeof (data as any).data === 'object' &&
      'type' in (data as any).data &&
      (data as any).data.type === 'tenants' &&
      'attributes' in (data as any).data
    );
  }

  private isUpdateTenantInput(data: unknown): data is UpdateTenantInput {
    return (
      typeof data === 'object' &&
      data !== null &&
      'data' in data &&
      typeof (data as any).data === 'object' &&
      'id' in (data as any).data &&
      'type' in (data as any).data &&
      (data as any).data.type === 'tenants' &&
      'attributes' in (data as any).data
    );
  }
}

// Create and export a singleton instance
export const tenantsService = new TenantsService();

// Export types for use in other files
export type { TenantBasic, TenantBasicAttributes, TenantConfig, TenantConfigAttributes };