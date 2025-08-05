/**
 * MapleStory.io API service for character rendering and item data
 * Provides functionality to generate character images and retrieve item information
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
  type WeaponRange,
  type NpcApiData,
  type NpcDataResult,
  WeaponType,
} from '@/types/models/maplestory';
import { type Character } from '@/types/models/character';
import { type Asset } from '@/services/api/inventory.service';

/**
 * Configuration for the MapleStory API service
 */
const DEFAULT_CONFIG: CharacterRenderingConfig = {
  apiBaseUrl: 'https://maplestory.io/api',
  apiVersion: '214', // Fallback version if tenant not provided
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
 * Based on Go backend GetWeaponType function
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
  // Note: Claw is one-handed, Staff is one-handed (despite being long)
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
 * MapleStory API service class
 */
export class MapleStoryService {
  private static instance: MapleStoryService;
  private config: CharacterRenderingConfig;
  private imageCache = new Map<string, string>();
  private timestampCache = new Map<string, number>();
  private npcDataCache = new Map<string, NpcDataResult>();
  private npcTimestampCache = new Map<string, number>();

  constructor(config: Partial<CharacterRenderingConfig> = {}) {
    this.config = { ...DEFAULT_CONFIG, ...config };
  }

  /**
   * Get singleton instance
   */
  static getInstance(config?: Partial<CharacterRenderingConfig>): MapleStoryService {
    if (!MapleStoryService.instance) {
      MapleStoryService.instance = new MapleStoryService(config);
    }
    return MapleStoryService.instance;
  }

  /**
   * Generate character image URL using MapleStory.io API
   */
  generateCharacterUrl(options: CharacterRenderOptions, region?: string, majorVersion?: number): string {
    const items: string[] = [];
    const stance = options.stance || this.determineStance(options.equipment);
    
    // Add base character features
    items.push(`${options.hair}:0`);
    items.push(`${options.face}:0`);
    
    // Process equipment in specific rendering order
    for (const slot of EQUIPMENT_RENDER_ORDER) {
      const itemId = options.equipment[slot];
      if (itemId) {
        items.push(`${itemId}:0`);
      }
    }
    
    // Process cash equipment (slots < -100)
    const cashSlots = Object.keys(options.equipment)
      .filter(slot => parseInt(slot) < -100)
      .sort((a, b) => parseInt(a) - parseInt(b));
    
    for (const slot of cashSlots) {
      const itemId = options.equipment[slot];
      if (itemId) {
        items.push(`${itemId}:0`);
      }
    }
    
    // Build URL parameters
    const params = new URLSearchParams();
    if (options.resize) params.append('resize', options.resize.toString());
    if (options.renderMode) params.append('renderMode', options.renderMode);
    if (options.flipX) params.append('flipX', 'true');
    
    const itemString = items.join(',');
    const queryString = params.toString();
    
    // Use provided region and majorVersion, fallback to defaults
    const apiRegion = region || 'GMS';
    const apiVersion = majorVersion?.toString() || this.config.apiVersion;
    
    return `${this.config.apiBaseUrl}/${apiRegion}/${apiVersion}/character/center/${options.skin}/${itemString}/${stance}/0${queryString ? '?' + queryString : ''}`;
  }

  /**
   * Generate character image with full result metadata
   */
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

    return {
      url,
      character,
      options: renderOptions,
      cached,
    };
  }

  /**
   * Extract equipment data from character inventory
   */
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
      
      // Skip non-negative slots if not including them
      if (!includeNegativeSlots && slotNumber >= 0) {
        continue;
      }
      
      // Skip cash equipment if not including them
      if (!includeCashEquipment && slotNumber < -100) {
        continue;
      }
      
      // Apply slot range filter if provided
      if (filterBySlotRange) {
        if (slotNumber < filterBySlotRange.min || slotNumber > filterBySlotRange.max) {
          continue;
        }
      }
      
      // Only include equipped items (negative slots)
      if (slotNumber < 0) {
        equipment[slot] = asset.attributes.templateId;
        equippedCount++;
      }
    }

    return {
      equipment,
      equippedCount,
      totalSlots,
    };
  }

  /**
   * Convert Character model to MapleStoryCharacterData
   */
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

  /**
   * Determine character stance based on equipped weapon
   */
  private determineStance(equipment: EquipmentData): 'stand1' | 'stand2' {
    const weaponId = equipment['-11'] || equipment['-111']; // Regular or cash weapon
    if (!weaponId) return this.config.defaultStance;
    
    // Get weapon type using proper classification algorithm
    const weaponType = getWeaponType(weaponId);
    const isTwoHanded = TWO_HANDED_WEAPON_TYPES.has(weaponType);
    
    return isTwoHanded ? 'stand2' : 'stand1';
  }

  /**
   * Map internal skin color value to MapleStory.io API value
   */
  private mapSkinColor(skincolor: number): number {
    return SKIN_COLOR_MAPPING[skincolor] || SKIN_COLOR_MAPPING[0] || 2000;
  }

  /**
   * Generate cache key for a character render
   */
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

  /**
   * Get cached URL if still valid
   */
  private getCachedUrl(cacheKey: string): string | null {
    if (!this.config.cacheEnabled) return null;
    
    const url = this.imageCache.get(cacheKey);
    const timestamp = this.timestampCache.get(cacheKey);
    
    if (!url || !timestamp) return null;
    
    const now = Date.now();
    if (now - timestamp > this.config.cacheTTL) {
      // Cache expired
      this.imageCache.delete(cacheKey);
      this.timestampCache.delete(cacheKey);
      return null;
    }
    
    return url;
  }

  /**
   * Cache a URL with timestamp
   */
  private setCachedUrl(cacheKey: string, url: string): void {
    if (!this.config.cacheEnabled) return;
    
    this.imageCache.set(cacheKey, url);
    this.timestampCache.set(cacheKey, Date.now());
  }

  /**
   * Clear all cached URLs and NPC data
   */
  clearCache(): void {
    this.imageCache.clear();
    this.timestampCache.clear();
    this.npcDataCache.clear();
    this.npcTimestampCache.clear();
  }

  /**
   * Get cache statistics
   */
  getCacheStats() {
    return {
      images: {
        size: this.imageCache.size,
        urls: Array.from(this.imageCache.keys()),
      },
      npcs: {
        size: this.npcDataCache.size,
        keys: Array.from(this.npcDataCache.keys()),
      },
      enabled: this.config.cacheEnabled,
      ttl: this.config.cacheTTL,
    };
  }

  /**
   * Get equipment slot name by slot ID
   */
  getEquipmentSlotName(slot: string): string {
    return EQUIPMENT_SLOT_MAPPING[slot] || 'Unknown';
  }

  /**
   * Get all equipment slot mappings
   */
  getEquipmentSlotMappings(): EquipmentSlotMapping {
    return { ...EQUIPMENT_SLOT_MAPPING };
  }

  /**
   * Check if a weapon ID represents a two-handed weapon
   */
  isTwoHandedWeapon(weaponId: number): boolean {
    const weaponType = getWeaponType(weaponId);
    return TWO_HANDED_WEAPON_TYPES.has(weaponType);
  }

  /**
   * Get weapon category by ID
   */
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

  /**
   * Get weapon type enum for a weapon ID
   */
  getWeaponType(weaponId: number): WeaponType {
    return getWeaponType(weaponId);
  }

  /**
   * Get NPC icon URL from MapleStory.io API
   */
  async getNpcIcon(npcId: number, region?: string, version?: string): Promise<string> {
    const apiRegion = region || this.config.defaultRegion || 'GMS';
    const apiVersion = version || this.config.apiVersion;
    
    const iconUrl = `${this.config.apiBaseUrl}/${apiRegion}/${apiVersion}/npc/${npcId}/icon`;
    
    try {
      // Validate that the icon exists by making a HEAD request
      const response = await fetch(iconUrl, { method: 'HEAD' });
      if (!response.ok) {
        throw new Error(`NPC icon not found: ${response.status}`);
      }
      return iconUrl;
    } catch (error) {
      if (this.config.enableErrorLogging) {
        console.warn(`Failed to fetch NPC icon for ID ${npcId}:`, error);
      }
      throw new Error(`Failed to fetch NPC icon for ID ${npcId}`);
    }
  }

  /**
   * Get NPC name from MapleStory.io API
   */
  async getNpcName(npcId: number, region?: string, version?: string): Promise<string> {
    const apiRegion = region || this.config.defaultRegion || 'GMS';
    const apiVersion = version || this.config.apiVersion;
    
    const nameUrl = `${this.config.apiBaseUrl}/${apiRegion}/${apiVersion}/npc/${npcId}/name`;
    
    try {
      const response = await fetch(nameUrl);
      if (!response.ok) {
        throw new Error(`NPC name not found: ${response.status}`);
      }
      const name = await response.text();
      return name.trim();
    } catch (error) {
      if (this.config.enableErrorLogging) {
        console.warn(`Failed to fetch NPC name for ID ${npcId}:`, error);
      }
      throw new Error(`Failed to fetch NPC name for ID ${npcId}`);
    }
  }

  /**
   * Get full NPC data from MapleStory.io API
   */
  async getNpcData(npcId: number, region?: string, version?: string): Promise<NpcApiData> {
    const apiRegion = region || this.config.defaultRegion || 'GMS';
    const apiVersion = version || this.config.apiVersion;
    
    const dataUrl = `${this.config.apiBaseUrl}/${apiRegion}/${apiVersion}/npc/${npcId}`;
    
    try {
      const response = await fetch(dataUrl);
      if (!response.ok) {
        throw new Error(`NPC data not found: ${response.status}`);
      }
      const data = await response.json();
      return {
        id: npcId,
        name: data.name || `NPC ${npcId}`,
        description: data.description,
        scripts: data.scripts,
      };
    } catch (error) {
      if (this.config.enableErrorLogging) {
        console.warn(`Failed to fetch NPC data for ID ${npcId}:`, error);
      }
      throw new Error(`Failed to fetch NPC data for ID ${npcId}`);
    }
  }

  /**
   * Get NPC data with caching and error handling
   */
  async getNpcDataWithCache(npcId: number, region?: string, version?: string): Promise<NpcDataResult> {
    const cacheKey = this.getNpcCacheKey(npcId, region, version);
    
    // Check cache first
    if (this.config.cacheEnabled) {
      const cachedData = this.getCachedNpcData(cacheKey);
      if (cachedData) {
        return { ...cachedData, cached: true };
      }
    }

    const result: NpcDataResult = {
      id: npcId,
      cached: false,
    };

    try {
      // Try to fetch both name and icon in parallel
      const [namePromise, iconPromise] = await Promise.allSettled([
        this.getNpcName(npcId, region, version),
        this.getNpcIcon(npcId, region, version),
      ]);

      if (namePromise.status === 'fulfilled') {
        result.name = namePromise.value;
      }

      if (iconPromise.status === 'fulfilled') {
        result.iconUrl = iconPromise.value;
      }

      // If both failed, set an error
      if (namePromise.status === 'rejected' && iconPromise.status === 'rejected') {
        result.error = 'Failed to fetch NPC data';
      }

      // Cache the result even if partially successful
      if (this.config.cacheEnabled) {
        this.setCachedNpcData(cacheKey, result);
      }

      return result;
    } catch (error) {
      result.error = error instanceof Error ? error.message : 'Unknown error';
      
      // Cache error results for a shorter time to avoid repeated failures
      if (this.config.cacheEnabled) {
        this.setCachedNpcData(cacheKey, result, this.config.cacheTTL / 4); // Cache errors for 1/4 of normal time
      }
      
      return result;
    }
  }

  /**
   * Generate cache key for NPC data
   */
  private getNpcCacheKey(npcId: number, region?: string, version?: string): string {
    const apiRegion = region || this.config.defaultRegion || 'GMS';
    const apiVersion = version || this.config.apiVersion;
    return `npc:${apiRegion}:${apiVersion}:${npcId}`;
  }

  /**
   * Get cached NPC data if still valid
   */
  private getCachedNpcData(cacheKey: string): NpcDataResult | null {
    if (!this.config.cacheEnabled) return null;
    
    const data = this.npcDataCache.get(cacheKey);
    const timestamp = this.npcTimestampCache.get(cacheKey);
    
    if (!data || !timestamp) return null;
    
    const now = Date.now();
    if (now - timestamp > this.config.cacheTTL) {
      // Cache expired
      this.npcDataCache.delete(cacheKey);
      this.npcTimestampCache.delete(cacheKey);
      return null;
    }
    
    return data;
  }

  /**
   * Cache NPC data with timestamp
   */
  private setCachedNpcData(cacheKey: string, data: NpcDataResult, customTTL?: number): void {
    if (!this.config.cacheEnabled) return;
    
    this.npcDataCache.set(cacheKey, data);
    this.npcTimestampCache.set(cacheKey, Date.now());
  }
}

/**
 * Default service instance
 */
export const mapleStoryService = MapleStoryService.getInstance();

/**
 * Utility functions for external use
 */
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