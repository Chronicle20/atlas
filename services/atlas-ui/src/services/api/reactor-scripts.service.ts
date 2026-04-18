import { api } from '@/lib/api/client';
import type { Tenant } from '@/types/models/tenant';

export interface ReactorScriptData {
  id: string;
  type: string;
  attributes: {
    reactorId: string;
    description: string;
    hitRules: unknown[];
    actRules: unknown[];
  };
}

class ReactorScriptsService {
  async getScriptsByReactor(reactorId: string, tenant: Tenant): Promise<ReactorScriptData | null> {
    try {
      api.setTenant(tenant);
      const results = await api.getList<ReactorScriptData>(`/api/reactors/${reactorId}/actions`);
      return results[0] ?? null;
    } catch {
      return null;
    }
  }
}

export const reactorScriptsService = new ReactorScriptsService();
