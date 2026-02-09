import { api } from '@/lib/api/client';
import { tenantHeaders } from '@/lib/headers';
import type { Tenant } from '@/types/models/tenant';

export interface SeedResult {
  deletedCount?: number;
  createdCount?: number;
  failedCount?: number;
  errors?: string[];
}

class SeedService {
  async seedDrops(tenant: Tenant): Promise<void> {
    api.setTenant(tenant);
    await api.post('/api/drops/seed', {});
  }

  async seedGachapons(tenant: Tenant): Promise<void> {
    api.setTenant(tenant);
    await api.post('/api/gachapons/seed', {});
  }

  async seedNpcConversations(tenant: Tenant): Promise<SeedResult> {
    api.setTenant(tenant);
    return api.post<SeedResult>('/api/npcs/conversations/seed', {});
  }

  async seedQuestConversations(tenant: Tenant): Promise<SeedResult> {
    api.setTenant(tenant);
    return api.post<SeedResult>('/api/quests/conversations/seed', {});
  }

  async seedNpcShops(tenant: Tenant): Promise<SeedResult> {
    api.setTenant(tenant);
    return api.post<SeedResult>('/api/shops/seed', {});
  }

  async seedPortalScripts(tenant: Tenant): Promise<SeedResult> {
    api.setTenant(tenant);
    return api.post<SeedResult>('/api/portals/scripts/seed', {});
  }

  async seedReactorScripts(tenant: Tenant): Promise<SeedResult> {
    api.setTenant(tenant);
    return api.post<SeedResult>('/api/reactors/actions/seed', {});
  }

  async uploadGameData(tenant: Tenant, file: File): Promise<void> {
    const formData = new FormData();
    formData.append('zip_file', file);

    const headers = tenantHeaders(tenant);

    const response = await fetch('/api/data', {
      method: 'PATCH',
      headers,
      body: formData,
    });

    if (!response.ok) {
      throw new Error(`Upload failed: ${response.status} ${response.statusText}`);
    }
  }
}

export const seedService = new SeedService();
