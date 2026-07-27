import { fetchPaged } from "@/services/api/pagination";

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
  async getScriptsByReactor(
    reactorId: string,
  ): Promise<ReactorScriptData | null> {
    try {
      const result = await fetchPaged<ReactorScriptData>(
        `/api/reactors/${reactorId}/actions`,
        { number: 1, size: 1 },
      );
      return result.data[0] ?? null;
    } catch {
      return null;
    }
  }
}

export const reactorScriptsService = new ReactorScriptsService();
