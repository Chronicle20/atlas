/**
 * Services Service
 *
 * Provides service configuration management for:
 * - Login services (atlas-login)
 * - Channel services (atlas-channel)
 * - Drops services (atlas-drops)
 */

import { BaseService, type ServiceOptions, type QueryOptions, type ValidationError } from './base.service';
import type { ApiSingleResponse } from '@/types/api/responses';
import type {
  Service,
  ServiceType,
  ServiceAttributes,
  LoginService,
  ChannelService,
  DropsService,
  LoginServiceAttributes,
  ChannelServiceAttributes,
  DropsServiceAttributes,
  CreateServiceInput,
  UpdateServiceInput,
  TaskConfig,
  LoginTenant,
  ChannelTenant,
  TASK_TYPES_BY_SERVICE,
} from '@/types/models/service';

// JSON:API request format for creating a service
interface CreateServiceRequest {
  data: {
    id?: string;
    type: 'services';
    attributes: ServiceAttributes;
  };
}

// JSON:API request format for updating a service
interface UpdateServiceRequest {
  data: {
    id: string;
    type: 'services';
    attributes: ServiceAttributes;
  };
}

/**
 * Services service class extending BaseService with service-specific functionality
 */
class ServicesService extends BaseService {
  protected basePath = '/api/configurations/services';

  /**
   * Validate service data before API calls
   */
  protected override validate<T>(data: T): ValidationError[] {
    const errors: ValidationError[] = [];

    if (this.isCreateServiceInput(data) || this.isUpdateServiceInput(data)) {
      const input = data as CreateServiceInput | UpdateServiceInput;

      // Validate type
      if (!input.type || !['login-service', 'channel-service', 'drops-service'].includes(input.type)) {
        errors.push({ field: 'type', message: 'Invalid service type' });
      }

      // Validate tasks
      if (input.tasks) {
        input.tasks.forEach((task, index) => {
          if (!task.type || task.type.trim().length === 0) {
            errors.push({ field: `tasks[${index}].type`, message: 'Task type is required' });
          }
          if (task.interval < 0) {
            errors.push({ field: `tasks[${index}].interval`, message: 'Interval must be non-negative' });
          }
          if (task.duration < 0) {
            errors.push({ field: `tasks[${index}].duration`, message: 'Duration must be non-negative' });
          }
        });
      }

      // Validate tenants based on service type
      if (input.type === 'login-service' && input.tenants) {
        (input.tenants as LoginTenant[]).forEach((tenant, index) => {
          if (!tenant.id || tenant.id.trim().length === 0) {
            errors.push({ field: `tenants[${index}].id`, message: 'Tenant ID is required' });
          }
          if (tenant.port < 1 || tenant.port > 65535) {
            errors.push({ field: `tenants[${index}].port`, message: 'Port must be between 1 and 65535' });
          }
        });
      }

      if (input.type === 'channel-service' && input.tenants) {
        (input.tenants as ChannelTenant[]).forEach((tenant, tIndex) => {
          if (!tenant.id || tenant.id.trim().length === 0) {
            errors.push({ field: `tenants[${tIndex}].id`, message: 'Tenant ID is required' });
          }
          if (!tenant.ipAddress || tenant.ipAddress.trim().length === 0) {
            errors.push({ field: `tenants[${tIndex}].ipAddress`, message: 'IP address is required' });
          }

          // Validate worlds
          if (tenant.worlds) {
            const worldIds = new Set<number>();
            tenant.worlds.forEach((world, wIndex) => {
              if (world.id < 0 || world.id > 255) {
                errors.push({ field: `tenants[${tIndex}].worlds[${wIndex}].id`, message: 'World ID must be between 0 and 255' });
              }
              if (worldIds.has(world.id)) {
                errors.push({ field: `tenants[${tIndex}].worlds[${wIndex}].id`, message: 'Duplicate world ID' });
              }
              worldIds.add(world.id);

              // Validate channels
              if (world.channels) {
                const channelIds = new Set<number>();
                world.channels.forEach((channel, cIndex) => {
                  if (channel.id < 0 || channel.id > 255) {
                    errors.push({ field: `tenants[${tIndex}].worlds[${wIndex}].channels[${cIndex}].id`, message: 'Channel ID must be between 0 and 255' });
                  }
                  if (channelIds.has(channel.id)) {
                    errors.push({ field: `tenants[${tIndex}].worlds[${wIndex}].channels[${cIndex}].id`, message: 'Duplicate channel ID within world' });
                  }
                  channelIds.add(channel.id);

                  if (channel.port < 1 || channel.port > 65535) {
                    errors.push({ field: `tenants[${tIndex}].worlds[${wIndex}].channels[${cIndex}].port`, message: 'Port must be between 1 and 65535' });
                  }
                });
              }
            });
          }
        });
      }
    }

    return errors;
  }

  /**
   * Transform API response to normalized Service format
   */
  private transformServiceResponse(data: Record<string, unknown>): Service {
    // The API returns JSON:API format: {id, type: "services", attributes: {...}}
    // We need to extract from attributes
    const id = data.id as string;
    const attributes = data.attributes as Record<string, unknown> | undefined;

    // Handle both flat (legacy) and nested (JSON:API) response formats
    const type = (attributes?.type ?? data.type) as ServiceType;
    const tasks = ((attributes?.tasks ?? data.tasks) as TaskConfig[]) || [];
    const tenants = attributes?.tenants ?? data.tenants;

    if (type === 'login-service') {
      return {
        id,
        attributes: {
          type,
          tasks,
          tenants: (tenants as LoginTenant[]) || [],
        },
      } as LoginService;
    } else if (type === 'channel-service') {
      return {
        id,
        attributes: {
          type,
          tasks,
          tenants: (tenants as ChannelTenant[]) || [],
        },
      } as ChannelService;
    } else {
      return {
        id,
        attributes: {
          type: 'drops-service',
          tasks,
        },
      } as DropsService;
    }
  }

  /**
   * Get all services
   */
  async getAllServices(options?: QueryOptions): Promise<Service[]> {
    const response = await this.getAll<Record<string, unknown>>(options);
    return response.map(item => this.transformServiceResponse(item));
  }

  /**
   * Get service by ID
   */
  async getServiceById(id: string, options?: ServiceOptions): Promise<Service> {
    const response = await this.getById<Record<string, unknown>>(id, options);
    return this.transformServiceResponse(response);
  }

  /**
   * Create a new service
   */
  async createService(input: CreateServiceInput, options?: ServiceOptions): Promise<Service> {
    // Build the attributes based on service type
    let attributes: ServiceAttributes;

    if (input.type === 'login-service') {
      attributes = {
        type: 'login-service',
        tasks: input.tasks || [],
        tenants: (input.tenants as LoginTenant[]) || [],
      };
    } else if (input.type === 'channel-service') {
      attributes = {
        type: 'channel-service',
        tasks: input.tasks || [],
        tenants: (input.tenants as ChannelTenant[]) || [],
      };
    } else {
      attributes = {
        type: 'drops-service',
        tasks: input.tasks || [],
      };
    }

    const request: CreateServiceRequest = {
      data: {
        type: 'services',
        attributes,
      },
    };

    // Add optional ID if provided
    if (input.id) {
      request.data.id = input.id;
    }

    const response = await this.create<ApiSingleResponse<Record<string, unknown>>, CreateServiceRequest>(request, options);
    return this.transformServiceResponse(response.data);
  }

  /**
   * Update an existing service
   */
  async updateService(id: string, input: UpdateServiceInput, options?: ServiceOptions): Promise<Service> {
    // Build the attributes based on service type
    let attributes: ServiceAttributes;

    if (input.type === 'login-service') {
      attributes = {
        type: 'login-service',
        tasks: input.tasks || [],
        tenants: (input.tenants as LoginTenant[]) || [],
      };
    } else if (input.type === 'channel-service') {
      attributes = {
        type: 'channel-service',
        tasks: input.tasks || [],
        tenants: (input.tenants as ChannelTenant[]) || [],
      };
    } else {
      attributes = {
        type: 'drops-service',
        tasks: input.tasks || [],
      };
    }

    const request: UpdateServiceRequest = {
      data: {
        id,
        type: 'services',
        attributes,
      },
    };

    const response = await this.patch<ApiSingleResponse<Record<string, unknown>>, UpdateServiceRequest>(id, request, options);
    return this.transformServiceResponse(response.data);
  }

  /**
   * Delete a service
   */
  async deleteService(id: string, options?: ServiceOptions): Promise<void> {
    return this.delete(id, options);
  }

  // === TYPE GUARDS ===

  private isCreateServiceInput(data: unknown): data is CreateServiceInput {
    return (
      typeof data === 'object' &&
      data !== null &&
      'type' in data &&
      typeof (data as CreateServiceInput).type === 'string'
    );
  }

  private isUpdateServiceInput(data: unknown): data is UpdateServiceInput {
    return (
      typeof data === 'object' &&
      data !== null &&
      'type' in data &&
      typeof (data as UpdateServiceInput).type === 'string'
    );
  }
}

// Create and export a singleton instance
export const servicesService = new ServicesService();

// Export types for use in other files
export type {
  Service,
  ServiceType,
  ServiceAttributes,
  LoginService,
  ChannelService,
  DropsService,
  LoginServiceAttributes,
  ChannelServiceAttributes,
  DropsServiceAttributes,
  CreateServiceInput,
  UpdateServiceInput,
  TaskConfig,
  LoginTenant,
  ChannelTenant,
};

// Re-export helpers
export {
  isLoginService,
  isChannelService,
  isDropsService,
  getServiceTypeDisplayName,
  getServiceTenantCount,
  getServiceTaskCount,
  TASK_TYPES_BY_SERVICE,
} from '@/types/models/service';
