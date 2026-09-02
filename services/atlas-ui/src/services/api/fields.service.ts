import { api } from "@/lib/api/client";

export interface FieldData {
  id: string; // "{worldId}:{channelId}:{mapId}:{instanceId}"
  type: string;
  attributes: {
    worldId: number;
    channelId: number;
    mapId: number;
    instanceId: string;
    characterCount: number;
  };
}

export interface FieldFilters {
  worldId?: number;
  channelId?: number;
  mapId?: number;
}

// The backend caps page[size] at 250 (paginate.MaxPageSize), which is also
// the default when unspecified — an unparameterised request silently
// truncates at 250 fields. Request the max explicitly.
const PAGE_SIZE = 250;

class FieldsService {
  async getFields(filters: FieldFilters): Promise<FieldData[]> {
    const params = new URLSearchParams({ "page[size]": String(PAGE_SIZE) });
    if (filters.worldId !== undefined)
      params.set("filter[worldId]", String(filters.worldId));
    if (filters.channelId !== undefined)
      params.set("filter[channelId]", String(filters.channelId));
    if (filters.mapId !== undefined)
      params.set("filter[mapId]", String(filters.mapId));
    return api.getList<FieldData>(`/api/fields?${params.toString()}`);
  }
}

export const fieldsService = new FieldsService();
