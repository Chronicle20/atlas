// Character domain model types
// Re-exported from lib/characters.tsx to centralize type definitions

export interface Character {
    id: string;
    attributes: CharacterAttributes;
}

export interface CharacterAttributes {
    accountId: number;
    worldId: number;
    name: string;
    level: number;
    experience: number;
    gachaponExperience: number;
    strength: number;
    dexterity: number;
    intelligence: number;
    luck: number;
    hp: number;
    maxHp: number;
    mp: number;
    maxMp: number;
    meso: number;
    hpMpUsed: number;
    jobId: number;
    skinColor: number;
    gender: number;
    fame: number;
    hair: number;
    face: number;
    mapId: number;
    spawnPoint: number;
    gm: number;
    x: number;
    y: number;
    stance: number;
}

// Character update request types
export interface UpdateCharacterData {
    mapId?: number;
}