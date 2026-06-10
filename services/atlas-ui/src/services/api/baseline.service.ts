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

// atlas-data registers these endpoints with RegisterInputHandler, which decodes
// the body via api2go's JSON:API unmarshaller. Bodies MUST be wrapped in a
// { data: { type, attributes } } envelope whose `type` matches the Go model's
// GetName() ("baselineRestores" / "baselinePublishes"); a bare attributes object
// is rejected with 400 "Source JSON is empty and has no attributes payload object".
export class BaselineService {
  async restore(tenant: Tenant, body: BaselineRestoreInput): Promise<void> {
    const headers = tenantHeaders(tenant);
    headers.set('Content-Type', 'application/json');
    const r = await fetch('/api/data/baseline/restore', {
      method: 'POST',
      headers,
      body: JSON.stringify({
        data: {
          type: 'baselineRestores',
          attributes: body,
        },
      }),
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
      body: JSON.stringify({
        data: {
          type: 'baselinePublishes',
          attributes: { region, majorVersion, minorVersion },
        },
      }),
    });
    if (!r.ok) {
      const message = await decodeErrorMessage(r, `publish failed: ${r.status}`);
      throw new Error(message);
    }
  }
}

export const baselineService = new BaselineService();
