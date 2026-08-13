// Template domain model types
// Re-exported from lib/templates.tsx to centralize type definitions

import type { SocketConfig } from "@/types/models/socket";

export interface CharacterTemplate {
  jobIndex: number;
  subJobIndex: number;
  gender: number;
  mapId: number;
  faces: number[];
  hairs: number[];
  hairColors: number[];
  skinColors: number[];
  tops: number[];
  bottoms: number[];
  shoes: number[];
  weapons: number[];
  items: number[];
  skills: number[];
}

export interface CharacterPresetStatBlock {
  str: number;
  dex: number;
  int: number;
  luk: number;
  hp: number;
  mp: number;
}

export interface CharacterPresetEquipmentEntry {
  templateId: number;
  useAverageStats: boolean;
}

export interface CharacterPresetInventoryEntry {
  templateId: number;
  quantity: number;
}

export interface CharacterPresetSkillEntry {
  skillId: number;
  level: number;
}

export interface CharacterPresetAttributes {
  name: string;
  description: string;
  tags: string[];
  jobId: number;
  gender: 0 | 1;
  face: number;
  hair: number;
  hairColor: number;
  skinColor: number;
  mapId: number;
  level: number;
  meso: number;
  gm: number;
  stats: CharacterPresetStatBlock;
  defaultName: string;
  equipment: CharacterPresetEquipmentEntry[];
  inventory: CharacterPresetInventoryEntry[];
  skills: CharacterPresetSkillEntry[];
}

export interface CharacterPreset {
  id?: string;
  attributes: CharacterPresetAttributes;
}

export interface TemplateAttributes {
  region: string;
  majorVersion: number;
  minorVersion: number;
  /**
   * SHA-256 of the seed file baked into the RUNNING image for this
   * region/version. Empty string when no such file ships. Computed
   * server-side; ignored on write.
   */
  shippedRevision?: string;
  /** SHA-256 of the persisted template content. Computed server-side. */
  storedRevision?: string;
  /**
   * True when shippedRevision is non-empty and differs from storedRevision.
   * Advisory and image-relative (NFR-4) - during a rolling update two replicas
   * may briefly disagree - so this is never an error state.
   */
  seedDrift?: boolean;
  usesPin: boolean;
  characters: {
    templates: CharacterTemplate[];
    presets: CharacterPreset[];
  };
  npcs: {
    npcId: number;
    impl: string;
  }[];
  socket: SocketConfig;
  worlds: {
    name: string;
    flag: string;
    serverMessage: string;
    eventMessage: string;
    whyAmIRecommended: string;
    expRate?: number;
    mesoRate?: number;
    itemDropRate?: number;
    questExpRate?: number;
  }[];
  cashShop?: {
    commodities: {
      hourlyExpirations?: {
        templateId: number;
        hours: number;
      }[];
    };
    /** Cash item template ids that open as a Cash Shop Surprise box. */
    surprise?: {
      boxTemplateIds?: number[];
    };
  };
}

export interface Template {
  id: string;
  attributes: TemplateAttributes;
}
