import { api } from '@/lib/api/client';

export interface PortalScriptData {
  id: string;
  type: string;
  attributes: {
    portalId: string;
    mapId: number;
    description: string;
    rules: unknown[];
  };
}

class PortalScriptsService {
  async getScriptsByPortal(portalId: string): Promise<PortalScriptData | null> {
    try {
      const results = await api.getList<PortalScriptData>(`/api/portals/${portalId}/scripts`);
      return results[0] ?? null;
    } catch {
      return null;
    }
  }
}

export const portalScriptsService = new PortalScriptsService();
