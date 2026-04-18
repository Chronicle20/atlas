import { api } from '@/lib/api/client';

export interface MapPortalData {
  id: string;
  type: string;
  attributes: {
    name: string;
    target: string;
    type: number;
    x: number;
    y: number;
    targetMapId: number;
    scriptName: string;
  };
}

export interface MapNpcData {
  id: string;
  type: string;
  attributes: {
    template: number;
    name: string;
    cy: number;
    x: number;
    y: number;
    f: number;
    fh: number;
    rx0: number;
    rx1: number;
    hide: boolean;
  };
}

export interface MapReactorData {
  id: string;
  type: string;
  attributes: {
    classification: number;
    name: string;
    x: number;
    y: number;
    delay: number;
    direction: number;
  };
}

export interface MapMonsterData {
  id: string;
  type: string;
  attributes: {
    template: number;
    mobTime: number;
    team: number;
    cy: number;
    x: number;
    y: number;
    f: number;
    fh: number;
    rx0: number;
    rx1: number;
    hide: boolean;
  };
}

class MapEntitiesService {
  async getPortals(mapId: string): Promise<MapPortalData[]> {
    return api.getList<MapPortalData>(`/api/data/maps/${mapId}/portals`);
  }

  async getPortal(mapId: string, portalId: string): Promise<MapPortalData> {
    return api.getOne<MapPortalData>(`/api/data/maps/${mapId}/portals/${portalId}`);
  }

  async getNpcs(mapId: string): Promise<MapNpcData[]> {
    return api.getList<MapNpcData>(`/api/data/maps/${mapId}/npcs`);
  }

  async getReactors(mapId: string): Promise<MapReactorData[]> {
    return api.getList<MapReactorData>(`/api/data/maps/${mapId}/reactors`);
  }

  async getMonsters(mapId: string): Promise<MapMonsterData[]> {
    return api.getList<MapMonsterData>(`/api/data/maps/${mapId}/monsters`);
  }
}

export const mapEntitiesService = new MapEntitiesService();
