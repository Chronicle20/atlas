/**
 * MapleStory.io API service for character rendering
 * Provides functionality to generate character images via MapleStory.io API
 */

import {
  type CharacterRenderOptions,
  type EquipmentData,
  type MapleStoryCharacterData,
  type CharacterImageResult,
  type EquipmentExtractionResult,
  type EquipmentExtractionOptions,
  type CharacterRenderingConfig,
  type SkinColorMapping,
  type EquipmentSlotMapping,
  WeaponType,
} from '@/types/models/maplestory';
import { type Character } from '@/types/models/character';
import { type Asset } from '@/services/api/inventory.service';

/**
 * Configuration for the MapleStory API service
 */
const DEFAULT_CONFIG: CharacterRenderingConfig = {
  apiBaseUrl: 'https://maplestory.io/api',
  apiVersion: '214',
  cacheEnabled: true,
  cacheTTL: 60 * 60 * 1000, // 1 hour
  defaultStance: 'stand1',
  defaultResize: 2,
  enableErrorLogging: true,
  defaultRegion: 'GMS',
};

/**
 * Skin color mapping from internal values to MapleStory.io API values
 */
const SKIN_COLOR_MAPPING: SkinColorMapping = {
  0: 2000,  // Light
  1: 2001,  // Ashen
  2: 2002,  // Pale Pink
  3: 2003,  // Clay
  4: 2004,  // Mercedes
  5: 2005,  // Alabaster
  6: 2009,  // Ghostly
  7: 2010,  // Pale
  8: 2011,  // Green
  9: 2012,  // Skeleton
  10: 2013, // Blue
};

/**
 * Equipment slot name mapping for display purposes
 */
const EQUIPMENT_SLOT_MAPPING: EquipmentSlotMapping = {
  '-1': 'Hat',
  '-5': 'Top/Overall',
  '-6': 'Bottom',
  '-7': 'Shoes',
  '-8': 'Gloves',
  '-9': 'Cape',
  '-10': 'Shield',
  '-11': 'Weapon',
  '-12': 'Ring',
  '-13': 'Ring',
  '-14': 'Ring',
  '-15': 'Ring',
  '-16': 'Pendant',
  '-17': 'Belt',
  '-18': 'Medal',
  '-19': 'Shoulder',
  '-20': 'Pocket Item',
  '-21': 'Eye Accessory',
  '-22': 'Face Accessory',
  '-23': 'Earrings',
  '-24': 'Emblem',
  '-25': 'Badge',
  '-101': 'Cash Hat',
  '-102': 'Cash Face',
  '-103': 'Cash Eye',
  '-104': 'Cash Top',
  '-105': 'Cash Overall',
  '-106': 'Cash Bottom',
  '-107': 'Cash Shoes',
  '-108': 'Cash Gloves',
  '-109': 'Cash Cape',
  '-110': 'Cash Shield',
  '-111': 'Cash Weapon',
  '-112': 'Cash Ring',
  '-113': 'Cash Pendant',
  '-114': 'Cash Belt',
};

/**
 * Get weapon type from item ID using MapleStory classification algorithm
 */
function getWeaponType(itemId: number): WeaponType {
  const cat = Math.floor((itemId / 10000) % 100);

  if (cat < 30 || cat > 49) {
    return WeaponType.None;
  }

  switch (cat - 30) {
    case 0: return WeaponType.OneHandedSword;
    case 1: return WeaponType.OneHandedAxe;
    case 2: return WeaponType.OneHandedMace;
    case 3: return WeaponType.Dagger;
    case 7: return WeaponType.Wand;
    case 8: return WeaponType.Staff;
    case 10: return WeaponType.TwoHandedSword;
    case 11: return WeaponType.TwoHandedAxe;
    case 12: return WeaponType.TwoHandedMace;
    case 13: return WeaponType.Spear;
    case 14: return WeaponType.Polearm;
    case 15: return WeaponType.Bow;
    case 16: return WeaponType.Crossbow;
    case 17: return WeaponType.Claw;
    case 18: return WeaponType.Knuckle;
    case 19: return WeaponType.Gun;
    default: return WeaponType.None;
  }
}

/**
 * Two-handed weapon types for stance determination
 */
const TWO_HANDED_WEAPON_TYPES = new Set([
  WeaponType.TwoHandedSword,
  WeaponType.TwoHandedAxe,
  WeaponType.TwoHandedMace,
  WeaponType.Spear,
  WeaponType.Polearm,
  WeaponType.Bow,
  WeaponType.Crossbow,
  WeaponType.Knuckle,
  WeaponType.Gun,
]);

/**
 * Equipment rendering order for layering
 */
const EQUIPMENT_RENDER_ORDER = [
  '-1',   // Hat
  '-9',   // Cape
  '-5',   // Top/Overall
  '-6',   // Bottom
  '-7',   // Shoes
  '-8',   // Gloves
  '-10',  // Shield
  '-11',  // Weapon
];

/**
 * MapleStory API service class - character rendering only
 */
export class MapleStoryService {
  private static instance: MapleStoryService;
  private config: CharacterRenderingConfig;
  private imageCache = new Map<string, string>();
  private timestampCache = new Map<string, number>();

  constructor(config: Partial<CharacterRenderingConfig> = {}) {
    this.config = { ...DEFAULT_CONFIG, ...config };
  }

  static getInstance(config?: Partial<CharacterRenderingConfig>): MapleStoryService {
    if (!MapleStoryService.instance) {
      MapleStoryService.instance = new MapleStoryService(config);
    }
    return MapleStoryService.instance;
  }

  generateCharacterUrl(options: CharacterRenderOptions, region?: string, majorVersion?: number): string {
    const items: string[] = [];
    const stance = options.stance || this.determineStance(options.equipment);

    items.push(`${options.hair}:0`);
    items.push(`${options.face}:0`);

    for (const slot of EQUIPMENT_RENDER_ORDER) {
      const itemId = options.equipment[slot];
      if (itemId) {
        items.push(`${itemId}:0`);
      }
    }

    const cashSlots = Object.keys(options.equipment)
      .filter(slot => parseInt(slot) < -100)
      .sort((a, b) => parseInt(a) - parseInt(b));

    for (const slot of cashSlots) {
      const itemId = options.equipment[slot];
      if (itemId) {
        items.push(`${itemId}:0`);
      }
    }

    const params = new URLSearchParams();
    if (options.resize) params.append('resize', options.resize.toString());
    if (options.renderMode) params.append('renderMode', options.renderMode);
    if (options.flipX) params.append('flipX', 'true');

    const itemString = items.join(',');
    const queryString = params.toString();
    const apiRegion = region || 'GMS';
    const apiVersion = majorVersion?.toString() || this.config.apiVersion;

    return `${this.config.apiBaseUrl}/${apiRegion}/${apiVersion}/character/center/${options.skin}/${itemString}/${stance}/0${queryString ? '?' + queryString : ''}`;
  }

  async generateCharacterImage(
    character: MapleStoryCharacterData,
    options: Partial<CharacterRenderOptions> = {},
    region?: string,
    majorVersion?: number
  ): Promise<CharacterImageResult> {
    const renderOptions: CharacterRenderOptions = {
      hair: character.hair,
      face: character.face,
      skin: this.mapSkinColor(character.skinColor),
      equipment: character.equipment,
      stance: options.stance || this.determineStance(character.equipment),
      resize: options.resize || this.config.defaultResize,
      renderMode: options.renderMode || 'default',
      frame: options.frame || 0,
      flipX: options.flipX || false,
    };

    const cacheKey = this.getCacheKey(renderOptions, region, majorVersion);
    const url = this.generateCharacterUrl(renderOptions, region, majorVersion);

    let cached = false;
    if (this.config.cacheEnabled) {
      const cachedUrl = this.getCachedUrl(cacheKey);
      if (cachedUrl) {
        cached = true;
      } else {
        this.setCachedUrl(cacheKey, url);
      }
    }

    return { url, character, options: renderOptions, cached };
  }

  extractEquipmentFromInventory(
    inventory: Asset[],
    options: EquipmentExtractionOptions = {}
  ): EquipmentExtractionResult {
    const {
      includeNegativeSlots = true,
      includeCashEquipment = true,
      filterBySlotRange,
    } = options;

    const equipment: EquipmentData = {};
    let equippedCount = 0;
    const totalSlots = inventory.length;

    for (const asset of inventory) {
      const slot = asset.attributes.slot.toString();
      const slotNumber = parseInt(slot);

      if (!includeNegativeSlots && slotNumber >= 0) continue;
      if (!includeCashEquipment && slotNumber < -100) continue;
      if (filterBySlotRange) {
        if (slotNumber < filterBySlotRange.min || slotNumber > filterBySlotRange.max) continue;
      }

      if (slotNumber < 0) {
        equipment[slot] = asset.attributes.templateId;
        equippedCount++;
      }
    }

    return { equipment, equippedCount, totalSlots };
  }

  characterToMapleStoryData(character: Character, inventory: Asset[]): MapleStoryCharacterData {
    const { equipment } = this.extractEquipmentFromInventory(inventory);

    return {
      id: character.id,
      name: character.attributes.name,
      level: character.attributes.level,
      jobId: character.attributes.jobId,
      hair: character.attributes.hair,
      face: character.attributes.face,
      skinColor: character.attributes.skinColor,
      gender: character.attributes.gender,
      equipment,
    };
  }

  private determineStance(equipment: EquipmentData): 'stand1' | 'stand2' {
    const weaponId = equipment['-11'] || equipment['-111'];
    if (!weaponId) return this.config.defaultStance;

    const weaponType = getWeaponType(weaponId);
    const isTwoHanded = TWO_HANDED_WEAPON_TYPES.has(weaponType);

    return isTwoHanded ? 'stand2' : 'stand1';
  }

  private mapSkinColor(skincolor: number): number {
    return SKIN_COLOR_MAPPING[skincolor] || SKIN_COLOR_MAPPING[0] || 2000;
  }

  private getCacheKey(options: CharacterRenderOptions, region?: string, majorVersion?: number): string {
    const keyParts = [
      region || 'GMS',
      majorVersion?.toString() || this.config.apiVersion,
      options.hair,
      options.face,
      options.skin,
      Object.entries(options.equipment)
        .sort(([a], [b]) => parseInt(a) - parseInt(b))
        .map(([slot, itemId]) => `${slot}:${itemId || 0}`)
        .join(','),
      options.stance || this.config.defaultStance,
      options.resize || this.config.defaultResize,
      options.renderMode || 'default',
      options.frame || 0,
      options.flipX || false,
    ];

    return btoa(keyParts.join('|'));
  }

  private getCachedUrl(cacheKey: string): string | null {
    if (!this.config.cacheEnabled) return null;

    const url = this.imageCache.get(cacheKey);
    const timestamp = this.timestampCache.get(cacheKey);

    if (!url || !timestamp) return null;

    if (Date.now() - timestamp > this.config.cacheTTL) {
      this.imageCache.delete(cacheKey);
      this.timestampCache.delete(cacheKey);
      return null;
    }

    return url;
  }

  private setCachedUrl(cacheKey: string, url: string): void {
    if (!this.config.cacheEnabled) return;

    this.imageCache.set(cacheKey, url);
    this.timestampCache.set(cacheKey, Date.now());
  }

  clearCache(): void {
    this.imageCache.clear();
    this.timestampCache.clear();
  }

  getCacheStats() {
    return {
      images: {
        size: this.imageCache.size,
        urls: Array.from(this.imageCache.keys()),
      },
      enabled: this.config.cacheEnabled,
      ttl: this.config.cacheTTL,
    };
  }

  getEquipmentSlotName(slot: string): string {
    return EQUIPMENT_SLOT_MAPPING[slot] || 'Unknown';
  }

  getEquipmentSlotMappings(): EquipmentSlotMapping {
    return { ...EQUIPMENT_SLOT_MAPPING };
  }

  isTwoHandedWeapon(weaponId: number): boolean {
    const weaponType = getWeaponType(weaponId);
    return TWO_HANDED_WEAPON_TYPES.has(weaponType);
  }

  getWeaponCategory(weaponId: number): string {
    const weaponType = getWeaponType(weaponId);

    switch (weaponType) {
      case WeaponType.OneHandedSword: return 'One-handed Sword';
      case WeaponType.OneHandedAxe: return 'One-handed Axe';
      case WeaponType.OneHandedMace: return 'One-handed Mace';
      case WeaponType.Dagger: return 'Dagger';
      case WeaponType.Wand: return 'Wand';
      case WeaponType.Staff: return 'Staff';
      case WeaponType.TwoHandedSword: return 'Two-handed Sword';
      case WeaponType.TwoHandedAxe: return 'Two-handed Axe';
      case WeaponType.TwoHandedMace: return 'Two-handed Mace';
      case WeaponType.Spear: return 'Spear';
      case WeaponType.Polearm: return 'Polearm';
      case WeaponType.Bow: return 'Bow';
      case WeaponType.Crossbow: return 'Crossbow';
      case WeaponType.Claw: return 'Claw';
      case WeaponType.Knuckle: return 'Knuckle';
      case WeaponType.Gun: return 'Gun';
      default: return 'Unknown';
    }
  }

  getWeaponType(weaponId: number): WeaponType {
    return getWeaponType(weaponId);
  }
}

export const mapleStoryService = MapleStoryService.getInstance();

export const mapSkinColor = (skincolor: number): number => {
  return SKIN_COLOR_MAPPING[skincolor] || SKIN_COLOR_MAPPING[0] || 2000;
};

export const getEquipmentSlotName = (slot: string): string => {
  return EQUIPMENT_SLOT_MAPPING[slot] || 'Unknown';
};

export const isTwoHandedWeapon = (weaponId: number): boolean => {
  const weaponType = getWeaponType(weaponId);
  return TWO_HANDED_WEAPON_TYPES.has(weaponType);
};

export const getWeaponTypeFromId = (weaponId: number): WeaponType => {
  return getWeaponType(weaponId);
};
