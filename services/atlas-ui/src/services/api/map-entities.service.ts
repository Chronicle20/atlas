import { api } from "@/lib/api/client";

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

export interface MapObjectData {
  id: string;
  type: string;
  attributes: {
    kind: string;
    name: string;
    objectSource: string;
    l0: string;
    l1: string;
    l2: string;
    x: number;
    y: number;
    z: number;
    layer: number;
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

// The structural minimum a map-pin overlay needs to place and label a
// monster marker (see MapImageOverlay's MonsterMarker/computeMarkers).
// `MapMonsterData` satisfies this shape already; `FieldDetailPage` adapts
// live monsters (`LiveMonsterData`, whose template lives at
// `attributes.monsterId`) to it without fabricating any other field.
export interface PositionedMonster {
  id: string;
  attributes: {
    template: number;
    x: number;
    y: number;
  };
}

// The structural minimum the same overlay needs to place and label a
// character marker (see MapImageOverlay's CharacterMarker/computeMarkers).
// `FieldDetailPage` adapts the batched character-detail results
// (`useFieldCharacterDetails`) to it, dropping characters still pending or
// errored enrichment rather than fabricating a position for them.
export interface PositionedCharacter {
  id: string;
  attributes: {
    name: string;
    x: number;
    y: number;
  };
}

class MapEntitiesService {
  async getPortals(mapId: string): Promise<MapPortalData[]> {
    return api.getList<MapPortalData>(`/api/data/maps/${mapId}/portals`);
  }

  async getPortal(mapId: string, portalId: string): Promise<MapPortalData> {
    return api.getOne<MapPortalData>(
      `/api/data/maps/${mapId}/portals/${portalId}`,
    );
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

  async getObjects(mapId: string): Promise<MapObjectData[]> {
    return api.getList<MapObjectData>(`/api/data/maps/${mapId}/objects`);
  }
}

export const mapEntitiesService = new MapEntitiesService();
