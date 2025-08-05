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
 * Two-handed weapon ID ranges for stance determination
 */
const TWO_HANDED_WEAPON_RANGES: WeaponRange[] = [
  { min: 1300000, max: 1399999, category: 'Two-handed Swords' },
  { min: 1400000, max: 1419999, category: 'Two-handed Axes' },
  { min: 1420000, max: 1439999, category: 'Two-handed Blunt Weapons' },
  { min: 1440000, max: 1449999, category: 'Spears' },
  { min: 1450000, max: 1459999, category: 'Polearms' },
  { min: 1520000, max: 1529999, category: 'Knuckles' },
  { min: 1590000, max: 1599999, category: 'Guns/Cannons' },
];

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
    
    // Check if weapon is two-handed
    const isTwoHanded = TWO_HANDED_WEAPON_RANGES.some(range => 
      weaponId >= range.min && weaponId <= range.max
    );
    
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
   * Clear all cached URLs
   */
  clearCache(): void {
    this.imageCache.clear();
    this.timestampCache.clear();
  }

  /**
   * Get cache statistics
   */
  getCacheStats() {
    return {
      size: this.imageCache.size,
      urls: Array.from(this.imageCache.keys()),
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
    return TWO_HANDED_WEAPON_RANGES.some(range => 
      weaponId >= range.min && weaponId <= range.max
    );
  }

  /**
   * Get weapon category by ID
   */
  getWeaponCategory(weaponId: number): string {
    const range = TWO_HANDED_WEAPON_RANGES.find(range => 
      weaponId >= range.min && weaponId <= range.max
    );
    
    return range?.category || 'One-handed';
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
  return TWO_HANDED_WEAPON_RANGES.some(range => 
    weaponId >= range.min && weaponId <= range.max
  );
};