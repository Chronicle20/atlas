/**
 * MapleStory utility functions for character rendering and equipment processing
 * Provides standalone utility functions for working with MapleStory character data
 */

import type {
  EquipmentData,
  EquipmentExtractionResult,
  EquipmentExtractionOptions,
  SkinColorMapping,
  EquipmentSlotMapping,
  WeaponRange,
  MapleStoryCharacterData,
} from '@/types/models/maplestory';
import type { Character } from '@/types/models/character';
import type { Asset } from '@/services/api/inventory.service';

/**
 * Skin color mapping from internal values to MapleStory.io API values
 */
export const SKIN_COLOR_MAPPING: SkinColorMapping = {
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
export const EQUIPMENT_SLOT_MAPPING: EquipmentSlotMapping = {
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
export const TWO_HANDED_WEAPON_RANGES: WeaponRange[] = [
  { min: 1300000, max: 1399999, category: 'Two-handed Swords' },
  { min: 1400000, max: 1419999, category: 'Two-handed Axes' },
  { min: 1420000, max: 1439999, category: 'Two-handed Blunt Weapons' },
  { min: 1440000, max: 1449999, category: 'Spears' },
  { min: 1450000, max: 1459999, category: 'Polearms' },
  { min: 1520000, max: 1529999, category: 'Knuckles' },
  { min: 1590000, max: 1599999, category: 'Guns/Cannons' },
];

/**
 * Equipment rendering order for proper layering
 */
export const EQUIPMENT_RENDER_ORDER = [
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
 * Map internal skin color value to MapleStory.io API value
 * @param skincolor - Internal skin color ID (0-10)
 * @returns MapleStory.io API skin color ID
 */
export const mapSkinColor = (skincolor: number): number => {
  return SKIN_COLOR_MAPPING[skincolor] ?? SKIN_COLOR_MAPPING[0] ?? 2000;
};

/**
 * Get equipment slot display name by slot ID
 * @param slot - Equipment slot ID (e.g., '-1', '-11', '-101')
 * @returns Human-readable slot name
 */
export const getEquipmentSlotName = (slot: string): string => {
  return EQUIPMENT_SLOT_MAPPING[slot] ?? 'Unknown';
};

/**
 * Check if a weapon ID represents a two-handed weapon
 * @param weaponId - Weapon item ID
 * @returns true if weapon is two-handed, false otherwise
 */
export const isTwoHandedWeapon = (weaponId: number): boolean => {
  return TWO_HANDED_WEAPON_RANGES.some(range => 
    weaponId >= range.min && weaponId <= range.max
  );
};

/**
 * Get weapon category by weapon ID
 * @param weaponId - Weapon item ID
 * @returns Weapon category name
 */
export const getWeaponCategory = (weaponId: number): string => {
  const range = TWO_HANDED_WEAPON_RANGES.find(range => 
    weaponId >= range.min && weaponId <= range.max
  );
  
  return range?.category ?? 'One-handed';
};

/**
 * Determine character stance based on equipped weapon
 * @param equipment - Character equipment data
 * @returns 'stand1' for one-handed weapons, 'stand2' for two-handed weapons
 */
export const determineStance = (equipment: EquipmentData): 'stand1' | 'stand2' => {
  const weaponId = equipment['-11'] || equipment['-111']; // Regular or cash weapon
  if (!weaponId) return 'stand1';
  
  return isTwoHandedWeapon(weaponId) ? 'stand2' : 'stand1';
};

/**
 * Extract equipment data from character inventory assets
 * Filters inventory items to find equipped items (negative slot indexes)
 * @param inventory - Array of inventory assets
 * @param options - Extraction options for filtering
 * @returns Equipment extraction result with equipment data and metadata
 */
export const extractEquipmentFromInventory = (
  inventory: Asset[],
  options: EquipmentExtractionOptions = {}
): EquipmentExtractionResult => {
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
};

/**
 * Extract only equipped items from inventory (items with negative slot indexes)
 * This is a simplified version of extractEquipmentFromInventory that only returns the equipment data
 * @param inventory - Array of inventory assets
 * @returns Equipment data object mapping slot IDs to item template IDs
 */
export const extractEquippedItems = (inventory: Asset[]): EquipmentData => {
  const equipment: EquipmentData = {};

  for (const asset of inventory) {
    const slot = asset.attributes.slot;
    
    // Only include equipped items (negative slots)
    if (slot < 0) {
      equipment[slot.toString()] = asset.attributes.templateId;
    }
  }

  return equipment;
};

/**
 * Filter equipment data to include only specific slot ranges
 * @param equipment - Equipment data to filter
 * @param minSlot - Minimum slot number (inclusive)
 * @param maxSlot - Maximum slot number (inclusive)
 * @returns Filtered equipment data
 */
export const filterEquipmentBySlotRange = (
  equipment: EquipmentData,
  minSlot: number,
  maxSlot: number
): EquipmentData => {
  const filtered: EquipmentData = {};

  for (const [slot, itemId] of Object.entries(equipment)) {
    const slotNumber = parseInt(slot);
    if (slotNumber >= minSlot && slotNumber <= maxSlot && itemId !== undefined) {
      filtered[slot] = itemId;
    }
  }

  return filtered;
};

/**
 * Get all cash equipment items from equipment data
 * @param equipment - Equipment data to filter
 * @returns Equipment data containing only cash items (slots < -100)
 */
export const getCashEquipment = (equipment: EquipmentData): EquipmentData => {
  return filterEquipmentBySlotRange(equipment, -200, -100);
};

/**
 * Get all regular equipment items from equipment data
 * @param equipment - Equipment data to filter
 * @returns Equipment data containing only regular items (slots -1 to -99)
 */
export const getRegularEquipment = (equipment: EquipmentData): EquipmentData => {
  return filterEquipmentBySlotRange(equipment, -99, -1);
};

/**
 * Convert Character model to MapleStoryCharacterData
 * Combines character attributes with extracted equipment data
 * @param character - Character model from API
 * @param inventory - Character's inventory assets
 * @returns MapleStory character data ready for rendering
 */
export const characterToMapleStoryData = (
  character: Character, 
  inventory: Asset[]
): MapleStoryCharacterData => {
  const equipment = extractEquippedItems(inventory);
  
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
};

/**
 * Validate equipment data to ensure all values are valid item IDs
 * @param equipment - Equipment data to validate
 * @returns Validated equipment data with invalid entries removed
 */
export const validateEquipmentData = (equipment: EquipmentData): EquipmentData => {
  const validated: EquipmentData = {};

  for (const [slot, itemId] of Object.entries(equipment)) {
    // Check if itemId is a valid positive number
    if (itemId && typeof itemId === 'number' && itemId > 0) {
      validated[slot] = itemId;
    }
  }

  return validated;
};

/**
 * Count equipped items in equipment data
 * @param equipment - Equipment data to count
 * @returns Number of equipped items
 */
export const countEquippedItems = (equipment: EquipmentData): number => {
  return Object.values(equipment).filter(itemId => itemId && itemId > 0).length;
};

/**
 * Get equipped items grouped by category
 * @param equipment - Equipment data to categorize
 * @returns Object with equipment categorized by type
 */
export const categorizeEquipment = (equipment: EquipmentData) => {
  const categories = {
    weapons: {} as EquipmentData,
    armor: {} as EquipmentData,
    accessories: {} as EquipmentData,
    cash: {} as EquipmentData,
  };

  for (const [slot, itemId] of Object.entries(equipment)) {
    if (!itemId) continue;
    
    const slotNumber = parseInt(slot);
    
    if (slotNumber < -100) {
      categories.cash[slot] = itemId;
    } else if (slot === '-11' || slot === '-10') {
      categories.weapons[slot] = itemId;
    } else if (['-1', '-5', '-6', '-7', '-8', '-9'].includes(slot)) {
      categories.armor[slot] = itemId;
    } else {
      categories.accessories[slot] = itemId;
    }
  }

  return categories;
};

/**
 * Check if equipment data contains any items
 * @param equipment - Equipment data to check
 * @returns true if equipment contains at least one item, false otherwise
 */
export const hasEquippedItems = (equipment: EquipmentData): boolean => {
  return Object.values(equipment).some(itemId => itemId && itemId > 0);
};

/**
 * Merge multiple equipment data objects
 * Later objects in the array take precedence for conflicting slots
 * @param equipmentSets - Array of equipment data to merge
 * @returns Merged equipment data
 */
export const mergeEquipment = (...equipmentSets: EquipmentData[]): EquipmentData => {
  return equipmentSets.reduce((merged, equipment) => {
    return { ...merged, ...equipment };
  }, {});
};