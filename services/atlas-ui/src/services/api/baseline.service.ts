import { tenantHeaders, canonicalHeaders, type CanonicalSelection } from '@/lib/headers';
import type { Tenant } from '@/types/models/tenant';

export interface BaselineRestoreInput {
  region: string;
  majorVersion: number;
  minorVersion: number;
  tenantId: string;
}

export interface Baseline {
  region: string;
  majorVersion: number;
  minorVersion: number;
  sha256: string;
  publishedAt: string; // RFC3339
  sizeBytes: number;
}

interface JsonApiCollection<A> {
  data: Array<{ type: string; id: string; attributes: A }>;
}

// GET /data/baselines needs tenant headers only to clear the shared REST
// middleware; the server ignores their values. A fixed dummy selection keeps
// the call signature tenant-free.
const LIST_HEADER_SELECTION: CanonicalSelection = { region: 'NONE', majorVersion: 0, minorVersion: 0 };

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

  // publish was always a shared-scope operation; the former Tenant argument
  // only fed headers. It now takes the explicit canonical selection.
  async publish(sel: CanonicalSelection): Promise<void> {
    const headers = canonicalHeaders(sel);
    headers.set('Content-Type', 'application/json');
    const r = await fetch('/api/data/baseline/publish', {
      method: 'POST',
      headers,
      body: JSON.stringify({
        data: {
          type: 'baselinePublishes',
          attributes: {
            region: sel.region,
            majorVersion: sel.majorVersion,
            minorVersion: sel.minorVersion,
          },
        },
      }),
    });
    if (!r.ok) {
      const message = await decodeErrorMessage(r, `publish failed: ${r.status}`);
      throw new Error(message);
    }
  }

  async listBaselines(): Promise<Baseline[]> {
    const headers = canonicalHeaders(LIST_HEADER_SELECTION);
    headers.set('Accept', 'application/vnd.api+json');
    const r = await fetch('/api/data/baselines', { method: 'GET', headers });
    if (!r.ok) {
      const message = await decodeErrorMessage(r, `baselines list failed: ${r.status}`);
      throw new Error(message);
    }
    const body = (await r.json()) as JsonApiCollection<Baseline>;
    return body.data.map((d) => ({ ...d.attributes }));
  }
}

export const baselineService = new BaselineService();
