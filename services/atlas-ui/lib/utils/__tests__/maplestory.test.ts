/**
 * Tests for MapleStory utility functions
 */

import {
  mapSkinColor,
  getEquipmentSlotName,
  isTwoHandedWeapon,
  getWeaponCategory,
  determineStance,
  extractEquipmentFromInventory,
  extractEquippedItems,
  filterEquipmentBySlotRange,
  getCashEquipment,
  getRegularEquipment,
  characterToMapleStoryData,
  validateEquipmentData,
  countEquippedItems,
  categorizeEquipment,
  hasEquippedItems,
  mergeEquipment,
} from '../maplestory';
import type { EquipmentData } from '@/types/models/maplestory';
import type { Character } from '@/types/models/character';
import type { Asset } from '@/services/api/inventory.service';

describe('MapleStory Utilities', () => {
  describe('mapSkinColor', () => {
    it('should map valid skin color IDs correctly', () => {
      expect(mapSkinColor(0)).toBe(2000); // Light
      expect(mapSkinColor(1)).toBe(2001); // Ashen
      expect(mapSkinColor(10)).toBe(2013); // Blue
    });

    it('should return default skin color for invalid IDs', () => {
      expect(mapSkinColor(999)).toBe(2000);
      expect(mapSkinColor(-1)).toBe(2000);
    });
  });

  describe('getEquipmentSlotName', () => {
    it('should return correct slot names', () => {
      expect(getEquipmentSlotName('-1')).toBe('Hat');
      expect(getEquipmentSlotName('-11')).toBe('Weapon');
      expect(getEquipmentSlotName('-101')).toBe('Cash Hat');
    });

    it('should return "Unknown" for invalid slots', () => {
      expect(getEquipmentSlotName('-999')).toBe('Unknown');
      expect(getEquipmentSlotName('invalid')).toBe('Unknown');
    });
  });

  describe('isTwoHandedWeapon', () => {
    it('should identify two-handed weapons correctly', () => {
      expect(isTwoHandedWeapon(1300001)).toBe(true); // Two-handed sword
      expect(isTwoHandedWeapon(1450001)).toBe(true); // Polearm
      expect(isTwoHandedWeapon(1590001)).toBe(true); // Gun/Cannon
    });

    it('should identify one-handed weapons correctly', () => {
      expect(isTwoHandedWeapon(1200001)).toBe(false); // One-handed sword
      expect(isTwoHandedWeapon(1600001)).toBe(false); // Not in ranges
    });
  });

  describe('getWeaponCategory', () => {
    it('should return correct weapon categories', () => {
      expect(getWeaponCategory(1300001)).toBe('Two-handed Swords');
      expect(getWeaponCategory(1450001)).toBe('Polearms');
      expect(getWeaponCategory(1200001)).toBe('One-handed');
    });
  });

  describe('determineStance', () => {
    it('should return stand2 for two-handed weapons', () => {
      const equipment: EquipmentData = { '-11': 1300001 }; // Two-handed sword
      expect(determineStance(equipment)).toBe('stand2');
    });

    it('should return stand1 for one-handed weapons', () => {
      const equipment: EquipmentData = { '-11': 1200001 }; // One-handed sword
      expect(determineStance(equipment)).toBe('stand1');
    });

    it('should return stand1 when no weapon equipped', () => {
      const equipment: EquipmentData = {};
      expect(determineStance(equipment)).toBe('stand1');
    });

    it('should check cash weapon slot if regular weapon not found', () => {
      const equipment: EquipmentData = { '-111': 1300001 }; // Cash two-handed sword
      expect(determineStance(equipment)).toBe('stand2');
    });
  });

  describe('extractEquipmentFromInventory', () => {
    const mockInventory: Asset[] = [
      {
        id: '1',
        type: 'inventory',
        attributes: {
          characterId: 'char1',
          slot: -1, // Hat (equipped)
          templateId: 1001,
          quantity: 1,
        },
      },
      {
        id: '2',
        type: 'inventory',
        attributes: {
          characterId: 'char1',
          slot: -11, // Weapon (equipped)
          templateId: 1300001, // Two-handed sword
          quantity: 1,
        },
      },
      {
        id: '3',
        type: 'inventory',
        attributes: {
          characterId: 'char1',
          slot: 1, // Regular inventory slot (not equipped)
          templateId: 2001,
          quantity: 1,
        },
      },
      {
        id: '4',
        type: 'inventory',
        attributes: {
          characterId: 'char1',
          slot: -101, // Cash hat (equipped)
          templateId: 1002,
          quantity: 1,
        },
      },
    ];

    it('should extract equipped items correctly', () => {
      const result = extractEquipmentFromInventory(mockInventory);
      
      expect(result.equipment).toEqual({
        '-1': 1001,
        '-11': 1300001,
        '-101': 1002,
      });
      expect(result.equippedCount).toBe(3);
      expect(result.totalSlots).toBe(4);
    });

    it('should exclude cash equipment when option is set', () => {
      const result = extractEquipmentFromInventory(mockInventory, {
        includeCashEquipment: false,
      });
      
      expect(result.equipment).toEqual({
        '-1': 1001,
        '-11': 1300001,
      });
      expect(result.equippedCount).toBe(2);
    });

    it('should filter by slot range', () => {
      const result = extractEquipmentFromInventory(mockInventory, {
        filterBySlotRange: { min: -11, max: -1 },
      });
      
      expect(result.equipment).toEqual({
        '-1': 1001,
        '-11': 1300001,
      });
      expect(result.equippedCount).toBe(2);
    });
  });

  describe('extractEquippedItems', () => {
    const mockInventory: Asset[] = [
      {
        id: '1',
        type: 'inventory',
        attributes: {
          characterId: 'char1',
          slot: -1,
          templateId: 1001,
          quantity: 1,
        },
      },
      {
        id: '2',
        type: 'inventory',
        attributes: {
          characterId: 'char1',
          slot: 1, // Not equipped
          templateId: 2001,
          quantity: 1,
        },
      },
    ];

    it('should extract only equipped items', () => {
      const equipment = extractEquippedItems(mockInventory);
      expect(equipment).toEqual({ '-1': 1001 });
    });
  });

  describe('filterEquipmentBySlotRange', () => {
    const equipment: EquipmentData = {
      '-1': 1001,
      '-11': 1200001,
      '-101': 1002,
      '-111': 1300001,
    };

    it('should filter equipment by slot range', () => {
      const filtered = filterEquipmentBySlotRange(equipment, -11, -1);
      expect(filtered).toEqual({
        '-1': 1001,
        '-11': 1200001,
      });
    });
  });

  describe('getCashEquipment and getRegularEquipment', () => {
    const equipment: EquipmentData = {
      '-1': 1001,
      '-11': 1200001,
      '-101': 1002,
      '-111': 1300001,
    };

    it('should get cash equipment only', () => {
      const cash = getCashEquipment(equipment);
      expect(cash).toEqual({
        '-101': 1002,
        '-111': 1300001,
      });
    });

    it('should get regular equipment only', () => {
      const regular = getRegularEquipment(equipment);
      expect(regular).toEqual({
        '-1': 1001,
        '-11': 1200001,
      });
    });
  });

  describe('characterToMapleStoryData', () => {
    const mockCharacter: Character = {
      id: 'char1',
      type: 'character',
      attributes: {
        accountId: 'acc1',
        name: 'TestChar',
        level: 50,
        jobId: 100,
        hair: 30000,
        face: 20000,
        skinColor: 0,
        gender: 0,
        strength: 100,
        dexterity: 100,
        intelligence: 100,
        luck: 100,
        hp: 1000,
        mp: 500,
        ap: 0,
        sp: '0,0,0',
        experience: 0,
        fame: 0,
        gachaponExperience: 0,
        mapId: 100000000,
        spawnPoint: 0,
        gm: 0,
        partyId: null,
        guildId: null,
        guildRank: 5,
        messengerGroupId: null,
        messengerPosition: 0,
        mounts: '0,0,0,0,0,0,0,0,0',
        buddyCapacity: 20,
        createdDate: '2024-01-01T00:00:00Z',
        lastLoginDate: '2024-01-01T00:00:00Z',
      },
    };

    const mockInventory: Asset[] = [
      {
        id: '1',
        type: 'inventory',
        attributes: {
          characterId: 'char1',
          slot: -1,
          templateId: 1001,
          quantity: 1,
        },
      },
    ];

    it('should convert character data correctly', () => {
      const result = characterToMapleStoryData(mockCharacter, mockInventory);
      
      expect(result).toEqual({
        id: 'char1',
        name: 'TestChar',
        level: 50,
        jobId: 100,
        hair: 30000,
        face: 20000,
        skinColor: 0,
        gender: 0,
        equipment: { '-1': 1001 },
      });
    });
  });

  describe('validateEquipmentData', () => {
    it('should remove invalid equipment entries', () => {
      const equipment: EquipmentData = {
        '-1': 1001,
        '-2': 0, // Invalid
        '-3': -1, // Invalid
        '-4': undefined, // Invalid
      };

      const validated = validateEquipmentData(equipment);
      expect(validated).toEqual({ '-1': 1001 });
    });
  });

  describe('countEquippedItems', () => {
    it('should count equipped items correctly', () => {
      const equipment: EquipmentData = {
        '-1': 1001,
        '-2': 1002,
        '-3': undefined,
        '-4': 0,
      };

      expect(countEquippedItems(equipment)).toBe(2);
    });
  });

  describe('categorizeEquipment', () => {
    const equipment: EquipmentData = {
      '-1': 1001,   // Armor (Hat)
      '-11': 1200001, // Weapon
      '-12': 1003,  // Accessory (Ring)
      '-101': 1004, // Cash
    };

    it('should categorize equipment correctly', () => {
      const categorized = categorizeEquipment(equipment);
      
      expect(categorized.armor).toEqual({ '-1': 1001 });
      expect(categorized.weapons).toEqual({ '-11': 1200001 });
      expect(categorized.accessories).toEqual({ '-12': 1003 });
      expect(categorized.cash).toEqual({ '-101': 1004 });
    });
  });

  describe('hasEquippedItems', () => {
    it('should return true when equipment has items', () => {
      const equipment: EquipmentData = { '-1': 1001 };
      expect(hasEquippedItems(equipment)).toBe(true);
    });

    it('should return false when equipment is empty', () => {
      const equipment: EquipmentData = {};
      expect(hasEquippedItems(equipment)).toBe(false);
    });

    it('should return false when equipment has only invalid items', () => {
      const equipment: EquipmentData = { '-1': 0, '-2': undefined };
      expect(hasEquippedItems(equipment)).toBe(false);
    });
  });

  describe('mergeEquipment', () => {
    it('should merge multiple equipment sets', () => {
      const equipment1: EquipmentData = { '-1': 1001, '-2': 1002 };
      const equipment2: EquipmentData = { '-2': 2002, '-3': 2003 };
      const equipment3: EquipmentData = { '-3': 3003 };

      const merged = mergeEquipment(equipment1, equipment2, equipment3);
      
      expect(merged).toEqual({
        '-1': 1001,
        '-2': 2002,
        '-3': 3003,
      });
    });
  });
});