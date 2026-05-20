import { tenantHeaders } from '@/lib/headers';
import type { Tenant } from '@/types/models/tenant';

export interface BaselineRestoreInput {
  region: string;
  majorVersion: number;
  minorVersion: number;
  tenantId: string;
}

export class BaselineService {
  async restore(tenant: Tenant, body: BaselineRestoreInput): Promise<void> {
    const headers = tenantHeaders(tenant);
    headers.set('Content-Type', 'application/json');
    const r = await fetch('/api/data/baseline/restore', {
      method: 'POST',
      headers,
      body: JSON.stringify(body),
    });
    if (!r.ok) throw new Error(`restore failed: ${r.status}`);
  }

  async publish(
    tenant: Tenant,
    region: string,
    majorVersion: number,
    minorVersion: number,
  ): Promise<void> {
    const headers = tenantHeaders(tenant);
    headers.set('Content-Type', 'application/json');
    headers.set('X-Atlas-Operator', '1');
    const r = await fetch('/api/data/baseline/publish', {
      method: 'POST',
      headers,
      body: JSON.stringify({ region, majorVersion, minorVersion }),
    });
    if (!r.ok) throw new Error(`publish failed: ${r.status}`);
  }
}

export const baselineService = new BaselineService();
