export interface CharacterLocationAttributes {
  worldId: number;
  channelId: number;
  mapId: number;
  instance: string;
}

export interface CharacterLocation {
  id: string;
  type: "character-locations";
  attributes: CharacterLocationAttributes;
}

export interface ChangeMapData {
  mapId: number;
}
