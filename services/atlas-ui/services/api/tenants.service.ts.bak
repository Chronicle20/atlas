/**
 * Tenant management service
 * Handles all tenant-related API operations
 */
import { BaseService } from './base.service';
import type { Tenant, CreateTenantDto } from '@/types/models/tenant';

class TenantsService extends BaseService {
  protected basePath = '/tenants';

  /**
   * Get all tenants
   */
  async getAll(signal?: AbortSignal): Promise<Tenant[]> {
    return super.getAll<Tenant>(signal);
  }

  /**
   * Get tenant by ID
   */
  async getById(id: string, signal?: AbortSignal): Promise<Tenant> {
    return super.getById<Tenant>(id, signal);
  }

  /**
   * Create new tenant
   */
  async create(data: CreateTenantDto, signal?: AbortSignal): Promise<Tenant> {
    return super.create<Tenant, CreateTenantDto>(data, signal);
  }

  /**
   * Update existing tenant
   */
  async update(id: string, data: Partial<Tenant>, signal?: AbortSignal): Promise<Tenant> {
    return super.update<Tenant, Partial<Tenant>>(id, data, signal);
  }

  /**
   * Delete tenant
   */
  async delete(id: string, signal?: AbortSignal): Promise<void> {
    return super.delete(id, signal);
  }
}

export const tenantsService = new TenantsService();