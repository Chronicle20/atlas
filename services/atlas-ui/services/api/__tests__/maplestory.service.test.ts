/**
 * Comprehensive tests for MapleStory service with various equipment combinations
 */

import { MapleStoryService } from '../maplestory.service';
import type {
  CharacterRenderOptions,
  EquipmentData,
  MapleStoryCharacterData,
  CharacterImageResult,
  EquipmentExtractionResult,
} from '@/types/models/maplestory';
import type { Character } from '@/types/models/character';
import type { Asset } from '@/services/api/inventory.service';

describe('MapleStoryService', () => {
  let service: MapleStoryService;

  beforeEach(() => {
    service = new MapleStoryService({
      cacheEnabled: false, // Disable cache for testing
      enableErrorLogging: false,
    });
  });

  describe('Character Image URL Generation', () => {
    const baseCharacterOptions: CharacterRenderOptions = {
      hair: 30000,
      face: 20000,
      skin: 2000,
      equipment: {},
      stance: 'stand1',
      resize: 2,
    };

    describe('Basic character rendering', () => {
      it('should generate URL for character with no equipment', () => {
        const url = service.generateCharacterUrl(baseCharacterOptions);
        
        expect(url).toContain('maplestory.io/api/GMS/214/character/center');
        expect(url).toContain('2000'); // Skin color
        expect(url).toContain('30000:0,20000:0'); // Hair and face
        expect(url).toContain('stand1');
        expect(url).toContain('resize=2');
      });

      it('should handle different skin colors correctly', () => {
        const darkSkinOptions = { ...baseCharacterOptions, skin: 2013 };
        const url = service.generateCharacterUrl(darkSkinOptions);
        
        expect(url).toContain('2013');
      });

      it('should handle different stances', () => {
        const stand2Options = { ...baseCharacterOptions, stance: 'stand2' };
        const url = service.generateCharacterUrl(stand2Options);
        
        expect(url).toContain('stand2');
      });
    });

    describe('Equipment combinations', () => {
      it('should render character with basic armor set', () => {
        const armorSet: EquipmentData = {
          '-1': 1001000,  // Hat
          '-5': 1041000,  // Top
          '-6': 1061000,  // Bottom
          '-7': 1071000,  // Shoes
          '-8': 1081000,  // Gloves
        };

        const options = { ...baseCharacterOptions, equipment: armorSet };
        const url = service.generateCharacterUrl(options);
        
        expect(url).toContain('1001000:0'); // Hat
        expect(url).toContain('1041000:0'); // Top
        expect(url).toContain('1061000:0'); // Bottom
        expect(url).toContain('1071000:0'); // Shoes
        expect(url).toContain('1081000:0'); // Gloves
      });

      it('should render character with weapon and shield', () => {
        const weaponSet: EquipmentData = {
          '-10': 1092000, // Shield
          '-11': 1302000, // One-handed sword
        };

        const options = { 
          ...baseCharacterOptions, 
          equipment: weaponSet,
          stance: undefined // Let it auto-detect stance
        };
        const url = service.generateCharacterUrl(options);
        
        expect(url).toContain('1092000:0'); // Shield
        expect(url).toContain('1302000:0'); // Weapon
        expect(url).toContain('stand1'); // Should auto-detect one-handed stance
      });

      it('should render character with accessories', () => {
        const accessorySet: EquipmentData = {
          '-9': 1102000,  // Cape
          '-12': 1112000, // Ring 1
          '-13': 1112001, // Ring 2
          '-16': 1122000, // Pendant
          '-17': 1132000, // Belt
          '-21': 1022000, // Eye accessory
          '-22': 1012000, // Face accessory
        };

        const options = { ...baseCharacterOptions, equipment: accessorySet };
        const url = service.generateCharacterUrl(options);
        
        expect(url).toContain('1102000:0'); // Cape
        // Note: Rings and other accessories not in EQUIPMENT_RENDER_ORDER won't appear in URL
        // This is expected behavior - only core visual equipment is rendered
      });

      it('should render character with cash equipment', () => {
        const cashSet: EquipmentData = {
          '-101': 1001001, // Cash Hat
          '-104': 1041001, // Cash Top
          '-106': 1061001, // Cash Bottom
          '-111': 1302001, // Cash Weapon
        };

        const options = { ...baseCharacterOptions, equipment: cashSet };
        const url = service.generateCharacterUrl(options);
        
        expect(url).toContain('1001001:0'); // Cash Hat
        expect(url).toContain('1041001:0'); // Cash Top
        expect(url).toContain('1061001:0'); // Cash Bottom
        expect(url).toContain('1302001:0'); // Cash Weapon
      });

      it('should handle mixed regular and cash equipment', () => {
        const mixedSet: EquipmentData = {
          '-1': 1001000,   // Regular Hat
          '-5': 1041000,   // Regular Top
          '-101': 1001001, // Cash Hat (should override regular)
          '-111': 1302001, // Cash Weapon
        };

        const options = { ...baseCharacterOptions, equipment: mixedSet };
        const url = service.generateCharacterUrl(options);
        
        // Both regular and cash items should be included
        expect(url).toContain('1001000:0'); // Regular Hat
        expect(url).toContain('1041000:0'); // Regular Top
        expect(url).toContain('1001001:0'); // Cash Hat
        expect(url).toContain('1302001:0'); // Cash Weapon
      });

      it('should handle full equipment loadout', () => {
        const fullSet: EquipmentData = {
          // Regular equipment
          '-1': 1001000,  // Hat
          '-5': 1041000,  // Top
          '-6': 1061000,  // Bottom
          '-7': 1071000,  // Shoes
          '-8': 1081000,  // Gloves
          '-9': 1102000,  // Cape
          '-10': 1092000, // Shield
          '-11': 1202000, // One-handed sword
          '-12': 1112000, // Ring 1
          '-13': 1112001, // Ring 2
          '-14': 1112002, // Ring 3
          '-15': 1112003, // Ring 4
          '-16': 1122000, // Pendant
          '-17': 1132000, // Belt
          '-18': 1142000, // Medal
          '-19': 1152000, // Shoulder
          '-20': 1162000, // Pocket
          '-21': 1022000, // Eye accessory
          '-22': 1012000, // Face accessory
          '-23': 1032000, // Earrings
          // Cash equipment
          '-101': 1001001, // Cash Hat
          '-111': 1302001, // Cash Weapon
        };

        const options = { ...baseCharacterOptions, equipment: fullSet };
        const url = service.generateCharacterUrl(options);
        
        // Verify URL contains essential equipment
        expect(url).toContain('1001000:0'); // Hat
        expect(url).toContain('1041000:0'); // Top
        expect(url).toContain('1202000:0'); // Weapon
        expect(url).toContain('1001001:0'); // Cash Hat
        expect(url).toContain('1302001:0'); // Cash Weapon
        expect(url).toContain('stand1'); // One-handed weapon stance
      });
    });

    describe('Weapon stance detection', () => {
      const weaponTestCases = [
        { weaponId: 1302000, expectedStance: 'stand1', category: 'One-handed Sword' },
        { weaponId: 1402000, expectedStance: 'stand2', category: 'Two-handed Sword' },
        { weaponId: 1412000, expectedStance: 'stand2', category: 'Two-handed Axe' },
        { weaponId: 1422000, expectedStance: 'stand2', category: 'Two-handed Mace' },
        { weaponId: 1432000, expectedStance: 'stand2', category: 'Spear' },
        { weaponId: 1442000, expectedStance: 'stand2', category: 'Polearm' },
        { weaponId: 1452000, expectedStance: 'stand2', category: 'Bow' },
        { weaponId: 1462000, expectedStance: 'stand2', category: 'Crossbow' },
        { weaponId: 1472000, expectedStance: 'stand1', category: 'Claw' },
        { weaponId: 1482000, expectedStance: 'stand2', category: 'Knuckle' },
        { weaponId: 1492000, expectedStance: 'stand2', category: 'Gun' },
        { weaponId: 1312000, expectedStance: 'stand1', category: 'One-handed Axe' },
        { weaponId: 1322000, expectedStance: 'stand1', category: 'One-handed Mace' },
        { weaponId: 1332000, expectedStance: 'stand1', category: 'Dagger' },
        { weaponId: 1372000, expectedStance: 'stand1', category: 'Wand' },
        { weaponId: 1382000, expectedStance: 'stand1', category: 'Staff' },
      ];

      weaponTestCases.forEach(({ weaponId, expectedStance, category }) => {
        it(`should use ${expectedStance} stance for ${category} (${weaponId})`, () => {
          const weaponEquipment: EquipmentData = { '-11': weaponId };
          const options = { 
            ...baseCharacterOptions, 
            equipment: weaponEquipment,
            stance: undefined // Let it auto-detect stance 
          };
          const url = service.generateCharacterUrl(options);
          
          expect(url).toContain(expectedStance);
          expect(url).toContain(`${weaponId}:0`);
        });
      });

      it('should check cash weapon slot when regular weapon slot is empty', () => {
        const cashWeaponEquipment: EquipmentData = { '-111': 1402000 }; // Cash two-handed sword
        const options = { 
          ...baseCharacterOptions, 
          equipment: cashWeaponEquipment,
          stance: undefined // Let it auto-detect stance
        };
        const url = service.generateCharacterUrl(options);
        
        expect(url).toContain('stand2');
        expect(url).toContain('1402000:0');
      });

      it('should prioritize regular weapon over cash weapon for stance detection', () => {
        const mixedWeaponEquipment: EquipmentData = {
          '-11': 1302000,  // One-handed sword (regular)
          '-111': 1402000, // Two-handed sword (cash)
        };
        const options = { ...baseCharacterOptions, equipment: mixedWeaponEquipment };
        const url = service.generateCharacterUrl(options);
        
        expect(url).toContain('stand1'); // Should use one-handed stance
        expect(url).toContain('1302000:0'); // Regular weapon
        expect(url).toContain('1402000:0'); // Cash weapon
      });
    });

    describe('URL parameter handling', () => {
      it('should include resize parameter', () => {
        const options = { ...baseCharacterOptions, resize: 3 };
        const url = service.generateCharacterUrl(options);
        
        expect(url).toContain('resize=3');
      });

      it('should include renderMode parameter', () => {
        const options = { ...baseCharacterOptions, renderMode: 'compact' };
        const url = service.generateCharacterUrl(options);
        
        expect(url).toContain('renderMode=compact');
      });

      it('should include flipX parameter when true', () => {
        const options = { ...baseCharacterOptions, flipX: true };
        const url = service.generateCharacterUrl(options);
        
        expect(url).toContain('flipX=true');
      });

      it('should not include flipX parameter when false', () => {
        const options = { ...baseCharacterOptions, flipX: false };
        const url = service.generateCharacterUrl(options);
        
        expect(url).not.toContain('flipX');
      });

      it('should handle multiple parameters correctly', () => {
        const options = {
          ...baseCharacterOptions,
          resize: 4,
          renderMode: 'compact' as const,
          flipX: true,
        };
        const url = service.generateCharacterUrl(options);
        
        expect(url).toContain('resize=4');
        expect(url).toContain('renderMode=compact');
        expect(url).toContain('flipX=true');
      });
    });
  });

  describe('Equipment Extraction from Inventory', () => {
    const createMockAsset = (slot: number, templateId: number): Asset => ({
      id: `asset-${slot}`,
      type: 'inventory',
      attributes: {
        characterId: 'char1',
        slot,
        templateId,
        quantity: 1,
      },
    });

    it('should extract basic equipped items', () => {
      const inventory: Asset[] = [
        createMockAsset(-1, 1001000),  // Hat
        createMockAsset(-11, 1202000), // Weapon
        createMockAsset(1, 2000000),   // Regular inventory (not equipped)
      ];

      const result = service.extractEquipmentFromInventory(inventory);
      
      expect(result.equipment).toEqual({
        '-1': 1001000,
        '-11': 1202000,
      });
      expect(result.equippedCount).toBe(2);
      expect(result.totalSlots).toBe(3);
    });

    it('should extract full armor set', () => {
      const inventory: Asset[] = [
        createMockAsset(-1, 1001000),  // Hat
        createMockAsset(-5, 1041000),  // Top
        createMockAsset(-6, 1061000),  // Bottom
        createMockAsset(-7, 1071000),  // Shoes
        createMockAsset(-8, 1081000),  // Gloves
        createMockAsset(-9, 1102000),  // Cape
      ];

      const result = service.extractEquipmentFromInventory(inventory);
      
      expect(result.equipment).toEqual({
        '-1': 1001000,
        '-5': 1041000,
        '-6': 1061000,
        '-7': 1071000,
        '-8': 1081000,
        '-9': 1102000,
      });
      expect(result.equippedCount).toBe(6);
    });

    it('should extract accessories', () => {
      const inventory: Asset[] = [
        createMockAsset(-12, 1112000), // Ring 1
        createMockAsset(-13, 1112001), // Ring 2
        createMockAsset(-16, 1122000), // Pendant
        createMockAsset(-17, 1132000), // Belt
        createMockAsset(-21, 1022000), // Eye accessory
        createMockAsset(-22, 1012000), // Face accessory
        createMockAsset(-23, 1032000), // Earrings
      ];

      const result = service.extractEquipmentFromInventory(inventory);
      
      expect(result.equipment).toEqual({
        '-12': 1112000,
        '-13': 1112001,
        '-16': 1122000,
        '-17': 1132000,
        '-21': 1022000,
        '-22': 1012000,
        '-23': 1032000,
      });
      expect(result.equippedCount).toBe(7);
    });

    it('should extract cash equipment', () => {
      const inventory: Asset[] = [
        createMockAsset(-101, 1001001), // Cash Hat
        createMockAsset(-104, 1041001), // Cash Top
        createMockAsset(-111, 1302001), // Cash Weapon
      ];

      const result = service.extractEquipmentFromInventory(inventory);
      
      expect(result.equipment).toEqual({
        '-101': 1001001,
        '-104': 1041001,
        '-111': 1302001,
      });
      expect(result.equippedCount).toBe(3);
    });

    it('should exclude cash equipment when option is false', () => {
      const inventory: Asset[] = [
        createMockAsset(-1, 1001000),   // Regular Hat
        createMockAsset(-101, 1001001), // Cash Hat
        createMockAsset(-111, 1302001), // Cash Weapon
      ];

      const result = service.extractEquipmentFromInventory(inventory, {
        includeCashEquipment: false,
      });
      
      expect(result.equipment).toEqual({
        '-1': 1001000,
      });
      expect(result.equippedCount).toBe(1);
    });

    it('should filter by slot range', () => {
      const inventory: Asset[] = [
        createMockAsset(-1, 1001000),   // Hat (in range)
        createMockAsset(-11, 1202000),  // Weapon (in range)
        createMockAsset(-101, 1001001), // Cash Hat (out of range)
      ];

      const result = service.extractEquipmentFromInventory(inventory, {
        filterBySlotRange: { min: -11, max: -1 },
      });
      
      expect(result.equipment).toEqual({
        '-1': 1001000,
        '-11': 1202000,
      });
      expect(result.equippedCount).toBe(2);
    });
  });

  describe('Character Data Conversion', () => {
    const mockCharacter: Character = {
      id: 'char1',
      type: 'character',
      attributes: {
        accountId: 'acc1',
        name: 'TestWarrior',
        level: 85,
        jobId: 110, // Fighter
        hair: 30000,
        face: 20000,
        skinColor: 2,
        gender: 0,
        strength: 150,
        dexterity: 50,
        intelligence: 25,
        luck: 25,
        hp: 2500,
        mp: 300,
        ap: 0,
        sp: '0,15,0',
        experience: 1500000,
        fame: 50,
        gachaponExperience: 0,
        mapId: 100000000,
        spawnPoint: 0,
        gm: 0,
        partyId: null,
        guildId: 'guild1',
        guildRank: 3,
        messengerGroupId: null,
        messengerPosition: 0,
        mounts: '0,0,0,0,0,0,0,0,0',
        buddyCapacity: 30,
        createdDate: '2024-01-01T00:00:00Z',
        lastLoginDate: '2024-01-15T12:00:00Z',
      },
    };

    it('should convert character data with equipment', () => {
      const inventory: Asset[] = [
        createMockAsset(-1, 1001000),  // Hat
        createMockAsset(-5, 1041000),  // Top
        createMockAsset(-11, 1302000), // One-handed sword
      ];

      const result = service.characterToMapleStoryData(mockCharacter, inventory);
      
      expect(result).toEqual({
        id: 'char1',
        name: 'TestWarrior',
        level: 85,
        jobId: 110,
        hair: 30000,
        face: 20000,
        skinColor: 2,
        gender: 0,
        equipment: {
          '-1': 1001000,
          '-5': 1041000,
          '-11': 1302000,
        },
      });
    });

    it('should convert character data without equipment', () => {
      const result = service.characterToMapleStoryData(mockCharacter, []);
      
      expect(result).toEqual({
        id: 'char1',
        name: 'TestWarrior',
        level: 85,
        jobId: 110,
        hair: 30000,
        face: 20000,
        skinColor: 2,
        gender: 0,
        equipment: {},
      });
    });
  });

  describe('Character Image Generation', () => {
    const mockCharacterData: MapleStoryCharacterData = {
      id: 'char1',
      name: 'TestChar',
      level: 50,
      jobId: 100,
      hair: 30000,
      face: 20000,
      skinColor: 0,
      gender: 0,
      equipment: {
        '-1': 1001000,
        '-11': 1302000,
      },
    };

    it('should generate character image result', async () => {
      const result = await service.generateCharacterImage(mockCharacterData);
      
      expect(result.url).toContain('maplestory.io/api');
      expect(result.character).toEqual(mockCharacterData);
      expect(result.options.hair).toBe(30000);
      expect(result.options.face).toBe(20000);
      expect(result.options.skin).toBe(2000); // Mapped from 0
      expect(result.options.stance).toBe('stand1'); // Auto-detected from one-handed weapon (1302000)
      expect(result.cached).toBe(false); // Cache disabled in tests
    });

    it('should generate character image with custom options', async () => {
      const customOptions = {
        resize: 4,
        renderMode: 'compact' as const,
        flipX: true,
      };

      const result = await service.generateCharacterImage(mockCharacterData, customOptions);
      
      expect(result.options.resize).toBe(4);
      expect(result.options.renderMode).toBe('compact');
      expect(result.options.flipX).toBe(true);
      expect(result.url).toContain('resize=4');
      expect(result.url).toContain('renderMode=compact');
      expect(result.url).toContain('flipX=true');
    });

  });

  describe('Utility functions', () => {
    it('should map skin colors correctly', () => {
      expect(service['mapSkinColor'](0)).toBe(2000);
      expect(service['mapSkinColor'](5)).toBe(2005);
      expect(service['mapSkinColor'](10)).toBe(2013);
      expect(service['mapSkinColor'](999)).toBe(2000); // Default
    });

    it('should get equipment slot names', () => {
      expect(service.getEquipmentSlotName('-1')).toBe('Hat');
      expect(service.getEquipmentSlotName('-11')).toBe('Weapon');
      expect(service.getEquipmentSlotName('-101')).toBe('Cash Hat');
      expect(service.getEquipmentSlotName('-999')).toBe('Unknown');
    });

    it('should identify two-handed weapons', () => {
      expect(service.isTwoHandedWeapon(1302000)).toBe(false); // One-handed sword
      expect(service.isTwoHandedWeapon(1402000)).toBe(true);  // Two-handed sword
      expect(service.isTwoHandedWeapon(1452000)).toBe(true);  // Bow
      expect(service.isTwoHandedWeapon(1372000)).toBe(false); // Wand
      expect(service.isTwoHandedWeapon(1382000)).toBe(false); // Staff
      expect(service.isTwoHandedWeapon(1332000)).toBe(false); // Dagger
      expect(service.isTwoHandedWeapon(1472000)).toBe(false); // Claw
      expect(service.isTwoHandedWeapon(1482000)).toBe(true);  // Knuckle
    });

    it('should get weapon categories', () => {
      expect(service.getWeaponCategory(1302000)).toBe('One-handed Sword');
      expect(service.getWeaponCategory(1402000)).toBe('Two-handed Sword');
      expect(service.getWeaponCategory(1452000)).toBe('Bow');
      expect(service.getWeaponCategory(1372000)).toBe('Wand');
      expect(service.getWeaponCategory(1382000)).toBe('Staff');
      expect(service.getWeaponCategory(1332000)).toBe('Dagger');
      expect(service.getWeaponCategory(1472000)).toBe('Claw');
      expect(service.getWeaponCategory(1482000)).toBe('Knuckle');
      expect(service.getWeaponCategory(1999999)).toBe('Unknown'); // Invalid weapon
    });
  });

  describe('Cache functionality', () => {
    let cachedService: MapleStoryService;

    beforeEach(() => {
      cachedService = new MapleStoryService({
        cacheEnabled: true,
        cacheTTL: 1000, // 1 second for testing
      });
    });

    it('should cache generated URLs', async () => {
      const characterData: MapleStoryCharacterData = {
        id: 'char1',
        name: 'TestChar',
        level: 50,
        jobId: 100,
        hair: 30000,
        face: 20000,
        skinColor: 0,
        gender: 0,
        equipment: {},
      };

      const result1 = await cachedService.generateCharacterImage(characterData);
      const result2 = await cachedService.generateCharacterImage(characterData);

      expect(result1.cached).toBe(false); // First call not cached
      expect(result2.cached).toBe(true); // Second call should be cached
      expect(result1.url).toBe(result2.url);
    });

    it('should clear cache', () => {
      cachedService.clearCache();
      const stats = cachedService.getCacheStats();
      
      expect(stats.size).toBe(0);
      expect(stats.urls).toEqual([]);
    });

    it('should provide cache statistics', () => {
      const stats = cachedService.getCacheStats();
      
      expect(typeof stats.size).toBe('number');
      expect(Array.isArray(stats.urls)).toBe(true);
      expect(stats.enabled).toBe(true);
      expect(stats.ttl).toBe(1000);
    });
  });

  function createMockAsset(slot: number, templateId: number): Asset {
    return {
      id: `asset-${slot}`,
      type: 'inventory',
      attributes: {
        characterId: 'char1',
        slot,
        templateId,
        quantity: 1,
      },
    };
  }
});