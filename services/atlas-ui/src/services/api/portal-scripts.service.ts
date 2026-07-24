import { fetchPaged } from "@/services/api/pagination";

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
      const result = await fetchPaged<PortalScriptData>(
        `/api/portals/${portalId}/scripts`,
        { number: 1, size: 1 },
      );
      return result.data[0] ?? null;
    } catch {
      return null;
    }
  }
}

export const portalScriptsService = new PortalScriptsService();
