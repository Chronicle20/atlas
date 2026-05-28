import { tenantHeaders } from '@/lib/headers';
import type { Tenant } from '@/types/models/tenant';

export interface BaselineRestoreInput {
  region: string;
  majorVersion: number;
  minorVersion: number;
  tenantId: string;
}

async function decodeErrorMessage(response: Response, fallback: string): Promise<string> {
  try {
    const parsed = (await response.json()) as { error?: string };
    if (parsed.error) return parsed.error;
  } catch {
    // non-JSON body; keep the status fallback
  }
  return fallback;
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
    if (!r.ok) {
      const message = await decodeErrorMessage(r, `restore failed: ${r.status}`);
      throw new Error(message);
    }
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
    if (!r.ok) {
      const message = await decodeErrorMessage(r, `publish failed: ${r.status}`);
      throw new Error(message);
    }
  }
}

export const baselineService = new BaselineService();
