import { api } from '@/lib/api/client';
import { tenantHeaders } from '@/lib/headers';
import type { Tenant } from '@/types/models/tenant';

export interface SeedResult {
  deletedCount?: number;
  createdCount?: number;
  failedCount?: number;
  errors?: string[];
}

export interface WzInputStatus {
  fileCount: number;
  totalBytes: number;
  updatedAt: string | null;
}

export interface WzExtractionStatus {
  fileCount: number;
  totalBytes: number;
  updatedAt: string | null;
}

export interface DataStatus {
  documentCount: number;
  updatedAt: string | null;
}

interface JsonApiEnvelope<A> {
  data: {
    type: string;
    id: string;
    attributes: A;
  };
}

async function fetchJsonApi<A>(url: string, tenant: Tenant): Promise<A> {
  const headers = tenantHeaders(tenant);
  headers.set('Accept', 'application/vnd.api+json');
  const response = await fetch(url, { method: 'GET', headers });
  if (!response.ok) {
    throw new Error(`GET ${url} failed: ${response.status} ${response.statusText}`);
  }
  const body = (await response.json()) as JsonApiEnvelope<A>;
  return body.data.attributes;
}

class SeedService {
  async seedDrops(): Promise<void> {
    await api.post('/api/drops/seed', {});
  }

  async seedGachapons(): Promise<void> {
    await api.post('/api/gachapons/seed', {});
  }

  async seedNpcConversations(): Promise<SeedResult> {
    return api.post<SeedResult>('/api/npcs/conversations/seed', {});
  }

  async seedQuestConversations(): Promise<SeedResult> {
    return api.post<SeedResult>('/api/quests/conversations/seed', {});
  }

  async seedNpcShops(): Promise<SeedResult> {
    return api.post<SeedResult>('/api/shops/seed', {});
  }

  async seedPortalScripts(): Promise<SeedResult> {
    return api.post<SeedResult>('/api/portals/scripts/seed', {});
  }

  async seedReactorScripts(): Promise<SeedResult> {
    return api.post<SeedResult>('/api/reactors/actions/seed', {});
  }

  async uploadWzFiles(tenant: Tenant, file: File): Promise<void> {
    const formData = new FormData();
    formData.append('zip_file', file);

    const headers = tenantHeaders(tenant);

    const response = await fetch('/api/wz/input', {
      method: 'PATCH',
      headers,
      body: formData,
    });

    if (!response.ok) {
      let message = `Upload failed: ${response.status} ${response.statusText}`;
      try {
        const body = (await response.json()) as { error?: string };
        if (body.error) {
          message = body.error;
        }
      } catch {
        // non-JSON error body; fall back to status text
      }
      const err = new Error(message) as Error & { status?: number };
      err.status = response.status;
      throw err;
    }
  }

  async runWzExtraction(tenant: Tenant): Promise<void> {
    const headers = tenantHeaders(tenant);
    const response = await fetch('/api/wz/extractions', { method: 'POST', headers });
    if (!response.ok) {
      throw new Error(`Extraction failed: ${response.status} ${response.statusText}`);
    }
  }

  async runDataProcessing(tenant: Tenant): Promise<void> {
    const headers = tenantHeaders(tenant);
    const response = await fetch('/api/data/process', { method: 'POST', headers });
    if (!response.ok) {
      throw new Error(`Data processing failed: ${response.status} ${response.statusText}`);
    }
  }

  async getWzInputStatus(tenant: Tenant): Promise<WzInputStatus> {
    return fetchJsonApi<WzInputStatus>('/api/wz/input', tenant);
  }

  async getExtractionStatus(tenant: Tenant): Promise<WzExtractionStatus> {
    return fetchJsonApi<WzExtractionStatus>('/api/wz/extractions', tenant);
  }

  async getDataStatus(tenant: Tenant): Promise<DataStatus> {
    return fetchJsonApi<DataStatus>('/api/data/status', tenant);
  }
}

export const seedService = new SeedService();
