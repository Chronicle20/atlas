/**
 * Service configuration types for atlas-login, atlas-channel, and atlas-drops services
 */

// Service type discriminator
export type ServiceType = 'login-service' | 'channel-service' | 'drops-service';

// Task configuration
export interface TaskConfig {
  type: string;
  interval: number;  // milliseconds
  duration: number;  // milliseconds
}

// Predefined task types by service (must match backend expectations)
export const TASK_TYPES_BY_SERVICE: Record<ServiceType, string[]> = {
  'login-service': ['timeout'],
  'channel-service': ['timeout', 'heartbeat'],
  'drops-service': ['drop_expiration_task'],
};

// Login service tenant configuration
export interface LoginTenant {
  id: string;
  port: number;
}

// Channel service nested types
export interface ChannelChannel {
  id: number;  // byte 0-255
  port: number;
}

export interface ChannelWorld {
  id: number;  // byte 0-255
  channels: ChannelChannel[];
}

export interface ChannelTenant {
  id: string;
  ipAddress: string;
  worlds: ChannelWorld[];
}

// Service attributes (what comes inside the JSON:API attributes object)
export interface LoginServiceAttributes {
  type: 'login-service';
  tasks: TaskConfig[];
  tenants: LoginTenant[];
}

export interface ChannelServiceAttributes {
  type: 'channel-service';
  tasks: TaskConfig[];
  tenants: ChannelTenant[];
}

export interface DropsServiceAttributes {
  type: 'drops-service';
  tasks: TaskConfig[];
}

export type ServiceAttributes = LoginServiceAttributes | ChannelServiceAttributes | DropsServiceAttributes;

// Full service models (with id from JSON:API)
export interface LoginService {
  id: string;
  attributes: LoginServiceAttributes;
}

export interface ChannelService {
  id: string;
  attributes: ChannelServiceAttributes;
}

export interface DropsService {
  id: string;
  attributes: DropsServiceAttributes;
}

// Union type for any service
export type Service = LoginService | ChannelService | DropsService;

// Type guards
export function isLoginService(service: Service): service is LoginService {
  return service.attributes.type === 'login-service';
}

export function isChannelService(service: Service): service is ChannelService {
  return service.attributes.type === 'channel-service';
}

export function isDropsService(service: Service): service is DropsService {
  return service.attributes.type === 'drops-service';
}

// Input types for create/update operations
export interface CreateServiceInput {
  id?: string;  // Optional UUID, will be generated if not provided
  type: ServiceType;
  tasks: TaskConfig[];
  tenants?: LoginTenant[] | ChannelTenant[];
}

export interface UpdateServiceInput {
  type: ServiceType;
  tasks: TaskConfig[];
  tenants?: LoginTenant[] | ChannelTenant[];
}

// Helper to get display name for service type
export function getServiceTypeDisplayName(type: ServiceType): string {
  switch (type) {
    case 'login-service':
      return 'Login Service';
    case 'channel-service':
      return 'Channel Service';
    case 'drops-service':
      return 'Drops Service';
    default:
      return type;
  }
}

// Helper to get tenant count for any service
export function getServiceTenantCount(service: Service): number {
  if (isDropsService(service)) {
    return 0;
  }
  return service.attributes.tenants?.length ?? 0;
}

// Helper to get task count
export function getServiceTaskCount(service: Service): number {
  return service.attributes.tasks?.length ?? 0;
}
