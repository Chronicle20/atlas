import { api } from "@/lib/api/client";
import { buildQueryString, type ServiceOptions, type QueryOptions } from "@/lib/api/query-params";
import { fetchAll, fetchPaged } from "@/services/api/pagination";

const BASE_PATH = "/api/data/maps";

export interface MapArea {
  x: number;
  y: number;
  width: number;
  height: number;
}

export interface MapAttributes {
  name: string;
  streetName: string;
  mapArea?: MapArea | null;
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

// MapsPage advertises "Results are limited to 50 entries" — the search
// methods below are bounded single-page lookups, not drained browses
// (task-117).
const SEARCH_RESULT_LIMIT = 50;

/**
 * Drain every page for a genuinely unbounded fetch (task-117).
 */
async function fetchAllMaps(options?: QueryOptions): Promise<MapData[]> {
  const finalOptions = withSparseFields(options);
  const maps = await fetchAll<MapData>(
    `${BASE_PATH}${buildQueryString(finalOptions)}`,
    undefined,
    finalOptions,
  );
  return sortMaps(maps);
}

/**
 * Fetch a single bounded page (task-117) — matches the "Results are limited
 * to 50 entries" copy on MapsPage.
 */
async function fetchSearchPage(options?: QueryOptions): Promise<MapData[]> {
  const finalOptions = withSparseFields(options);
  const result = await fetchPaged<MapData>(
    `${BASE_PATH}${buildQueryString(finalOptions)}`,
    { number: 1, size: SEARCH_RESULT_LIMIT },
    finalOptions,
  );
  return sortMaps(result.data);
}

export const mapsService = {
  async getAllMaps(options?: QueryOptions): Promise<MapData[]> {
    return fetchAllMaps(options);
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
    return fetchSearchPage({ ...options, search: query });
  },

  async searchMapsByName(name: string, options?: ServiceOptions): Promise<MapData[]> {
    return fetchSearchPage({ ...options, search: name, filters: { name } });
  },

  /**
   * Get every map on the given street, draining all pages (task-117).
   */
  async getMapsByStreetName(streetName: string, options?: ServiceOptions): Promise<MapData[]> {
    return fetchAllMaps({ ...options, filters: { streetName } });
  },
};

export type Map = MapData;
