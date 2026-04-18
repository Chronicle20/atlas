import { api } from "@/lib/api/client";
import { buildQueryString, type ServiceOptions, type QueryOptions } from "@/lib/api/query-params";

const BASE_PATH = "/api/data/maps";

export interface MapAttributes {
  name: string;
  streetName: string;
}

export interface MapData {
  id: string;
  attributes: MapAttributes;
}

interface CreateMapInput {
  data: {
    type: "maps";
    attributes: MapAttributes;
  };
}

interface UpdateMapInput {
  data: {
    id: string;
    type: "maps";
    attributes: Partial<MapAttributes>;
  };
}

function sortMaps(maps: MapData[]): MapData[] {
  return maps.sort((a, b) => a.attributes.name.localeCompare(b.attributes.name));
}

function withSparseFields(options?: QueryOptions): QueryOptions {
  return {
    ...options,
    fields: { maps: ["name", "streetName"], ...options?.fields },
  };
}

async function fetchAll(options?: QueryOptions): Promise<MapData[]> {
  const finalOptions = withSparseFields(options);
  const maps = await api.getList<MapData>(
    `${BASE_PATH}${buildQueryString(finalOptions)}`,
    finalOptions,
  );
  return sortMaps(maps);
}

export const mapsService = {
  async getAllMaps(options?: QueryOptions): Promise<MapData[]> {
    return fetchAll(options);
  },

  async getMapById(id: string, options?: ServiceOptions): Promise<MapData> {
    return api.getOne<MapData>(`${BASE_PATH}/${id}`, options);
  },

  async createMap(attributes: MapAttributes, options?: ServiceOptions): Promise<MapData> {
    const input: CreateMapInput = { data: { type: "maps", attributes } };
    return api.post<MapData>(BASE_PATH, input, options);
  },

  async updateMap(map: MapData, updatedAttributes: Partial<MapAttributes>, options?: ServiceOptions): Promise<MapData> {
    const input: UpdateMapInput = {
      data: {
        id: map.id,
        type: "maps",
        attributes: { ...map.attributes, ...updatedAttributes },
      },
    };
    await api.patch<void>(`${BASE_PATH}/${map.id}`, input, options);
    return { ...map, attributes: { ...map.attributes, ...updatedAttributes } };
  },

  async deleteMap(mapId: string, options?: ServiceOptions): Promise<void> {
    return api.delete(`${BASE_PATH}/${mapId}`, options);
  },

  async searchMaps(query: string, options?: QueryOptions): Promise<MapData[]> {
    return fetchAll({ ...options, search: query });
  },

  async searchMapsByName(name: string, options?: ServiceOptions): Promise<MapData[]> {
    return fetchAll({ ...options, search: name, filters: { name } });
  },

  async getMapsByStreetName(streetName: string, options?: ServiceOptions): Promise<MapData[]> {
    return fetchAll({ ...options, filters: { streetName } });
  },
};

export type Map = MapData;
