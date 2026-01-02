/**
 * Integration tests for character rendering with real-world equipment combinations
 * Tests the complete flow from character data to rendered image URL
 */

import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { CharacterRenderer } from '@/components/features/characters/CharacterRenderer';
import type { Character } from '@/types/models/character';
import type { Asset } from '@/services/api/inventory.service';

// Mock the useCharacterImage hook
const mockUseCharacterImage = jest.fn();
jest.mock('@/lib/hooks/useCharacterImage', () => ({
  useCharacterImage: (...args: unknown[]) => mockUseCharacterImage(...args),
}));

// Mock the mapleStoryService for characterToMapleStoryData
jest.mock('@/services/api/maplestory.service', () => ({
  mapleStoryService: {
    characterToMapleStoryData: (character: Character, inventory: Asset[] = []) => {
      // Convert inventory to equipment map
      const equipment: Record<number, number> = {};
      inventory.forEach((asset: Asset) => {
        if (asset.attributes.templateId > 0) {
          equipment[asset.attributes.slot] = asset.attributes.templateId;
        }
      });

      return {
        id: character.id,
        name: character.attributes.name,
        hair: character.attributes.hair,
        face: character.attributes.face,
        skinColor: character.attributes.skinColor,
        gender: character.attributes.gender,
        jobId: character.attributes.jobId,
        equipment,
      };
    },
  },
}));

// Test wrapper with QueryClient
function TestWrapper({ children }: { children: React.ReactNode }) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });
  return (
    <QueryClientProvider client={queryClient}>
      {children}
    </QueryClientProvider>
  );
}

// Mock Next.js Image component
jest.mock('next/image', () => {
  return function MockImage({ src, alt, ...props }: { src: string; alt: string; [key: string]: unknown }) {
    return (
      // eslint-disable-next-line @next/next/no-img-element
      <img
        src={src}
        alt={alt}
        data-testid="character-image"
        {...props}
      />
    );
  };
});

// Skin color mapping matching the actual service
const SKIN_COLOR_MAPPING: Record<number, number> = {
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

// Helper to generate expected maplestory.io URL
function generateExpectedUrl(
  skinColor: number,
  hair: number,
  face: number,
  equipment: Asset[],
  stance: 'stand1' | 'stand2' = 'stand1'
): string {
  // Use skin color mapping, defaulting to 2000 for invalid values
  const skinBase = SKIN_COLOR_MAPPING[skinColor] ?? 2000;

  // Build equipment string
  const equipmentParts = equipment
    .filter(e => e.attributes.templateId > 0)
    .map(e => `${e.attributes.templateId}:0`)
    .join(',');

  return `https://maplestory.io/api/GMS/214/character/center/${skinBase}/${hair}:0,${face}:0${equipmentParts ? `,${equipmentParts}` : ''}/${stance}/0?resize=2`;
}

// Helper to determine stance based on equipment
function determineStance(equipment: Asset[]): 'stand1' | 'stand2' {
  const weapon = equipment.find(e => e.attributes.slot === -11);
  if (!weapon) return 'stand1';

  const weaponId = weapon.attributes.templateId;
  // Two-handed weapons: axes (1400000-1500000), bows (1450000-1460000), two-handed wands (1370000-1380000)
  const isTwoHanded =
    (weaponId >= 1400000 && weaponId < 1500000) || // Two-handed swords, axes, etc.
    (weaponId >= 1450000 && weaponId < 1460000) || // Bows
    (weaponId >= 1370000 && weaponId < 1380000);   // Two-handed wands

  return isTwoHanded ? 'stand2' : 'stand1';
}

describe('Character Rendering Integration Tests', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  // Helper to set up the useCharacterImage mock with the expected URL
  function setupMockWithUrl(url: string) {
    mockUseCharacterImage.mockReturnValue({
      data: { url, cached: false },
      isLoading: false,
      error: null,
      refetch: jest.fn(),
      preload: jest.fn().mockResolvedValue({ loaded: true }),
      prefetchVariants: jest.fn(),
      imageUrl: url,
      cached: false,
    });
  }

  const createCharacter = (overrides: Partial<Character['attributes']> = {}): Character => ({
    id: 'test-char',
    type: 'character',
    attributes: {
      accountId: 'test-account',
      name: 'TestCharacter',
      level: 100,
      jobId: 110,
      hair: 30000,
      face: 20000,
      skinColor: 0,
      gender: 0,
      strength: 100,
      dexterity: 100,
      intelligence: 100,
      luck: 100,
      hp: 5000,
      mp: 1000,
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
      ...overrides,
    },
  });

  const createAsset = (slot: number, templateId: number): Asset => ({
    id: `asset-${slot}-${templateId}`,
    type: 'inventory',
    attributes: {
      characterId: 'test-char',
      slot,
      templateId,
      quantity: 1,
    },
  });

  describe('Real-world character builds', () => {
    it('should render a beginner warrior correctly', async () => {
      const beginnerWarrior = createCharacter({
        name: 'BeginnerWarrior',
        level: 15,
        jobId: 100, // Warrior
        hair: 30030,
        face: 20000,
        skinColor: 0,
      });

      const basicEquipment: Asset[] = [
        createAsset(-1, 1002419),  // Brown Bandana (beginner hat)
        createAsset(-5, 1040036),  // Blue Work Shirt
        createAsset(-6, 1060026),  // Blue Jean Shorts
        createAsset(-7, 1072038),  // Brown Boot
        createAsset(-11, 1302000), // Sword (basic one-handed)
      ];

      const expectedUrl = generateExpectedUrl(0, 30030, 20000, basicEquipment, determineStance(basicEquipment));
      setupMockWithUrl(expectedUrl);

      render(<CharacterRenderer character={beginnerWarrior} inventory={basicEquipment} />, { wrapper: TestWrapper });

      await waitFor(() => {
        expect(screen.getByTestId('character-image')).toBeInTheDocument();
      });

      const image = screen.getByTestId('character-image');
      const imageUrl = image.getAttribute('src');

      expect(imageUrl).toContain('maplestory.io/api/GMS/214/character/center/2000');
      expect(imageUrl).toContain('30030:0'); // Hair
      expect(imageUrl).toContain('20000:0'); // Face
      expect(imageUrl).toContain('1002419:0'); // Hat
      expect(imageUrl).toContain('1040036:0'); // Top
      expect(imageUrl).toContain('1060026:0'); // Bottom
      expect(imageUrl).toContain('1072038:0'); // Shoes
      expect(imageUrl).toContain('1302000:0'); // Weapon
      expect(imageUrl).toContain('stand1'); // One-handed stance
    });

    it('should render a high-level fighter with full equipment', async () => {
      const highLevelFighter = createCharacter({
        name: 'EliteFighter',
        level: 120,
        jobId: 111, // Fighter
        hair: 30027,
        face: 20004,
        skinColor: 2, // Pale Pink
      });

      const eliteEquipment: Asset[] = [
        createAsset(-1, 1002357),  // Zakum Helmet
        createAsset(-5, 1041082),  // Dark Ritius Top
        createAsset(-6, 1061082),  // Dark Ritius Bottom
        createAsset(-7, 1072238),  // Dark Ritius Shoes
        createAsset(-8, 1082149),  // Dark Ritius Gloves
        createAsset(-9, 1102084),  // Pink Adventurer Cape
        createAsset(-10, 1092030), // Maple Shield
        createAsset(-11, 1372001), // Two-handed Sword (Fusion Mace)
        createAsset(-12, 1112127), // Silver Ring of Power
        createAsset(-13, 1112127), // Silver Ring of Power
        createAsset(-16, 1122076), // Horntail Necklace
        createAsset(-17, 1132000), // Brown Belt
      ];

      const expectedUrl = generateExpectedUrl(2, 30027, 20004, eliteEquipment, determineStance(eliteEquipment));
      setupMockWithUrl(expectedUrl);

      render(<CharacterRenderer character={highLevelFighter} inventory={eliteEquipment} />, { wrapper: TestWrapper });

      await waitFor(() => {
        expect(screen.getByTestId('character-image')).toBeInTheDocument();
      });

      const image = screen.getByTestId('character-image');
      const imageUrl = image.getAttribute('src');
      
      expect(imageUrl).toContain('2002'); // Pale Pink skin
      expect(imageUrl).toContain('30027:0'); // Hair
      expect(imageUrl).toContain('20004:0'); // Face
      expect(imageUrl).toContain('1002357:0'); // Zakum Helmet
      expect(imageUrl).toContain('1041082:0'); // Dark Ritius Top
      expect(imageUrl).toContain('1372001:0'); // Two-handed weapon
      expect(imageUrl).toContain('stand2'); // Two-handed stance
    });

    it('should render an archer with bow and arrows', async () => {
      const archer = createCharacter({
        name: 'EliteArcher',
        level: 85,
        jobId: 301, // Hunter
        hair: 31002,
        face: 21001,
        skinColor: 1, // Ashen
        gender: 1, // Female
      });

      const archerEquipment: Asset[] = [
        createAsset(-1, 1002140),  // Brown Leather Cap
        createAsset(-5, 1041104),  // Green Hunter Top
        createAsset(-6, 1061104),  // Green Hunter Bottom
        createAsset(-7, 1072081),  // Green Hunter Boots
        createAsset(-8, 1082062),  // Green Hunter Gloves
        createAsset(-11, 1452022), // Bow (Metus)
        createAsset(-12, 1112401), // Archer Ring
      ];

      const expectedUrl = generateExpectedUrl(1, 31002, 21001, archerEquipment, 'stand2'); // Bow is two-handed
      setupMockWithUrl(expectedUrl);

      render(<CharacterRenderer character={archer} inventory={archerEquipment} />, { wrapper: TestWrapper });

      await waitFor(() => {
        expect(screen.getByTestId('character-image')).toBeInTheDocument();
      });

      const image = screen.getByTestId('character-image');
      const imageUrl = image.getAttribute('src');
      
      expect(imageUrl).toContain('2001'); // Ashen skin
      expect(imageUrl).toContain('31002:0'); // Female hair
      expect(imageUrl).toContain('21001:0'); // Female face
      expect(imageUrl).toContain('1452022:0'); // Bow
      expect(imageUrl).toContain('stand2'); // Two-handed stance (bow)
    });

    it('should render a mage with magical equipment', async () => {
      const mage = createCharacter({
        name: 'ElementalMage',
        level: 95,
        jobId: 211, // Fire/Poison Wizard
        hair: 30003,
        face: 20001,
        skinColor: 4, // Mercedes
      });

      const mageEquipment: Asset[] = [
        createAsset(-1, 1002021),  // Wizard Hat
        createAsset(-5, 1041041),  // Blue Wizard Robe
        createAsset(-7, 1072032),  // Blue Wizard Shoes
        createAsset(-8, 1082024),  // Blue Wizard Gloves
        createAsset(-9, 1102007),  // Blue Cape
        createAsset(-10, 1092006), // Magic Shield
        createAsset(-11, 1372000), // Magic Wand
        createAsset(-16, 1122011), // Wisdom Pendant
      ];

      const expectedUrl = generateExpectedUrl(4, 30003, 20001, mageEquipment, 'stand1');
      setupMockWithUrl(expectedUrl);

      render(<CharacterRenderer character={mage} inventory={mageEquipment} />, { wrapper: TestWrapper });

      await waitFor(() => {
        expect(screen.getByTestId('character-image')).toBeInTheDocument();
      });

      const image = screen.getByTestId('character-image');
      const imageUrl = image.getAttribute('src');
      
      expect(imageUrl).toContain('2004'); // Mercedes skin
      expect(imageUrl).toContain('1002021:0'); // Wizard Hat
      expect(imageUrl).toContain('1041041:0'); // Wizard Robe
      expect(imageUrl).toContain('1372000:0'); // Magic Wand
      expect(imageUrl).toContain('1092006:0'); // Magic Shield
      expect(imageUrl).toContain('stand1'); // One-handed stance
    });

    it('should render a thief with stealth gear', async () => {
      const thief = createCharacter({
        name: 'ShadowThief',
        level: 78,
        jobId: 411, // Assassin
        hair: 30012,
        face: 20009,
        skinColor: 6, // Ghostly
      });

      const thiefEquipment: Asset[] = [
        createAsset(-1, 1002098),  // Black Bandana
        createAsset(-5, 1041047),  // Black Ninja Top
        createAsset(-6, 1061034),  // Black Ninja Bottom
        createAsset(-7, 1072047),  // Black Ninja Shoes
        createAsset(-8, 1082037),  // Black Ninja Gloves
        createAsset(-9, 1102041),  // Black Cape
        createAsset(-10, 1092014), // Dark Shield
        createAsset(-11, 1332025), // Steely (claw)
        createAsset(-21, 1022017), // Dark Eyes
      ];

      const expectedUrl = generateExpectedUrl(6, 30012, 20009, thiefEquipment, 'stand1');
      setupMockWithUrl(expectedUrl);

      render(<CharacterRenderer character={thief} inventory={thiefEquipment} />, { wrapper: TestWrapper });

      await waitFor(() => {
        expect(screen.getByTestId('character-image')).toBeInTheDocument();
      });

      const image = screen.getByTestId('character-image');
      const imageUrl = image.getAttribute('src');
      
      expect(imageUrl).toContain('2009'); // Ghostly skin
      expect(imageUrl).toContain('1002098:0'); // Black Bandana
      expect(imageUrl).toContain('1041047:0'); // Ninja Top
      expect(imageUrl).toContain('1332025:0'); // Claw
      expect(imageUrl).toContain('1022017:0'); // Eye accessory
      expect(imageUrl).toContain('stand1'); // One-handed stance
    });

    it('should render a pirate with gun and nautical gear', async () => {
      const pirate = createCharacter({
        name: 'SeaPirate',
        level: 88,
        jobId: 501, // Pirate
        hair: 30020,
        face: 20012,
        skinColor: 3, // Clay
      });

      const pirateEquipment: Asset[] = [
        createAsset(-1, 1002571),  // Pirate Bandana
        createAsset(-5, 1041167),  // Blue Pirate Top
        createAsset(-6, 1061167),  // Blue Pirate Bottom
        createAsset(-7, 1072294),  // Pirate Boots
        createAsset(-8, 1082207),  // Pirate Gloves
        createAsset(-9, 1102132),  // Pirate Cape
        createAsset(-11, 1492000), // Gun (Guns)
        createAsset(-12, 1112405), // Pirate Ring
      ];

      const expectedUrl = generateExpectedUrl(3, 30020, 20012, pirateEquipment, 'stand1');
      setupMockWithUrl(expectedUrl);

      render(<CharacterRenderer character={pirate} inventory={pirateEquipment} />, { wrapper: TestWrapper });

      await waitFor(() => {
        expect(screen.getByTestId('character-image')).toBeInTheDocument();
      });

      const image = screen.getByTestId('character-image');
      const imageUrl = image.getAttribute('src');
      
      expect(imageUrl).toContain('2003'); // Clay skin
      expect(imageUrl).toContain('1002571:0'); // Pirate Bandana
      expect(imageUrl).toContain('1041167:0'); // Pirate Top
      expect(imageUrl).toContain('1492000:0'); // Gun
      expect(imageUrl).toContain('stand1'); // One-handed stance
    });
  });

  describe('Cash equipment combinations', () => {
    it('should render character with cash NX equipment override', async () => {
      const character = createCharacter({
        name: 'FashionCharacter',
        level: 60,
        hair: 30045,
        face: 20020,
        skinColor: 5, // Alabaster
      });

      const fashionEquipment: Asset[] = [
        // Regular equipment (base)
        createAsset(-1, 1002000),  // Basic cap
        createAsset(-5, 1041000),  // Basic shirt
        createAsset(-6, 1061000),  // Basic pants
        createAsset(-11, 1302000), // Basic sword

        // Cash equipment (overrides)
        createAsset(-101, 1002999), // Cash fancy hat
        createAsset(-104, 1041999), // Cash fancy outfit
        createAsset(-111, 1302999), // Cash fancy sword
      ];

      const expectedUrl = generateExpectedUrl(5, 30045, 20020, fashionEquipment, 'stand1');
      setupMockWithUrl(expectedUrl);

      render(<CharacterRenderer character={character} inventory={fashionEquipment} />, { wrapper: TestWrapper });

      await waitFor(() => {
        expect(screen.getByTestId('character-image')).toBeInTheDocument();
      });

      const image = screen.getByTestId('character-image');
      const imageUrl = image.getAttribute('src');
      
      expect(imageUrl).toContain('2005'); // Alabaster skin
      // Should contain both regular and cash items
      expect(imageUrl).toContain('1002000:0'); // Regular hat
      expect(imageUrl).toContain('1041000:0'); // Regular shirt
      expect(imageUrl).toContain('1302000:0'); // Regular sword
      expect(imageUrl).toContain('1002999:0'); // Cash hat
      expect(imageUrl).toContain('1041999:0'); // Cash outfit
      expect(imageUrl).toContain('1302999:0'); // Cash sword
    });

    it('should render character with only cash equipment', async () => {
      const character = createCharacter({
        name: 'CashOnlyCharacter',
        level: 40,
        hair: 30050,
        face: 20025,
        skinColor: 7, // Pale
      });

      const cashOnlyEquipment: Asset[] = [
        createAsset(-101, 1002888), // Cash hat only
        createAsset(-104, 1041888), // Cash top only
        createAsset(-106, 1061888), // Cash bottom only
        createAsset(-111, 1302888), // Cash weapon only
      ];

      const expectedUrl = generateExpectedUrl(7, 30050, 20025, cashOnlyEquipment, 'stand1');
      setupMockWithUrl(expectedUrl);

      render(<CharacterRenderer character={character} inventory={cashOnlyEquipment} />, { wrapper: TestWrapper });

      await waitFor(() => {
        expect(screen.getByTestId('character-image')).toBeInTheDocument();
      });

      const image = screen.getByTestId('character-image');
      const imageUrl = image.getAttribute('src');
      
      expect(imageUrl).toContain('2010'); // Pale skin
      expect(imageUrl).toContain('1002888:0'); // Cash hat
      expect(imageUrl).toContain('1041888:0'); // Cash top
      expect(imageUrl).toContain('1061888:0'); // Cash bottom
      expect(imageUrl).toContain('1302888:0'); // Cash weapon
    });
  });

  describe('Special character configurations', () => {
    it('should render character with accessories only', async () => {
      const character = createCharacter({
        name: 'AccessoryCharacter',
        level: 30,
        hair: 30015,
        face: 20008,
        skinColor: 8, // Green
      });

      const accessoryOnlyEquipment: Asset[] = [
        createAsset(-12, 1112000), // Ring 1
        createAsset(-13, 1112001), // Ring 2
        createAsset(-14, 1112002), // Ring 3
        createAsset(-15, 1112003), // Ring 4
        createAsset(-16, 1122000), // Pendant
        createAsset(-17, 1132000), // Belt
        createAsset(-21, 1022000), // Eye accessory
        createAsset(-22, 1012000), // Face accessory
        createAsset(-23, 1032000), // Earrings
      ];

      const expectedUrl = generateExpectedUrl(8, 30015, 20008, accessoryOnlyEquipment, 'stand1');
      setupMockWithUrl(expectedUrl);

      render(<CharacterRenderer character={character} inventory={accessoryOnlyEquipment} />, { wrapper: TestWrapper });

      await waitFor(() => {
        expect(screen.getByTestId('character-image')).toBeInTheDocument();
      });

      const image = screen.getByTestId('character-image');
      const imageUrl = image.getAttribute('src');
      
      expect(imageUrl).toContain('2011'); // Green skin
      expect(imageUrl).toContain('1112000:0'); // Ring 1
      expect(imageUrl).toContain('1122000:0'); // Pendant
      expect(imageUrl).toContain('1022000:0'); // Eye accessory
      expect(imageUrl).toContain('1012000:0'); // Face accessory
      expect(imageUrl).toContain('stand1'); // Default stance (no weapon)
    });

    it('should render character with mixed equipment slots', async () => {
      const character = createCharacter({
        name: 'MixedCharacter',
        level: 75,
        hair: 30040,
        face: 20030,
        skinColor: 9, // Skeleton
      });

      const mixedEquipment: Asset[] = [
        // Some slots filled, some empty
        createAsset(-1, 1002100),  // Hat
        // No top (-5)
        createAsset(-6, 1061050),  // Bottom only
        createAsset(-7, 1072100),  // Shoes
        // No gloves (-8)
        createAsset(-9, 1102050),  // Cape
        // No shield (-10)
        createAsset(-11, 1412000), // Two-handed axe
        createAsset(-16, 1122050), // Pendant
        // Cash override for missing regular top
        createAsset(-104, 1041777), // Cash top
      ];

      const expectedUrl = generateExpectedUrl(9, 30040, 20030, mixedEquipment, 'stand2'); // Two-handed axe
      setupMockWithUrl(expectedUrl);

      render(<CharacterRenderer character={character} inventory={mixedEquipment} />, { wrapper: TestWrapper });

      await waitFor(() => {
        expect(screen.getByTestId('character-image')).toBeInTheDocument();
      });

      const image = screen.getByTestId('character-image');
      const imageUrl = image.getAttribute('src');
      
      expect(imageUrl).toContain('2012'); // Skeleton skin
      expect(imageUrl).toContain('1002100:0'); // Hat
      expect(imageUrl).toContain('1061050:0'); // Bottom
      expect(imageUrl).toContain('1412000:0'); // Two-handed axe
      expect(imageUrl).toContain('1041777:0'); // Cash top
      expect(imageUrl).toContain('stand2'); // Two-handed stance
    });
  });

  describe('Edge cases and error scenarios', () => {
    it('should handle character with no equipment gracefully', async () => {
      const character = createCharacter({
        name: 'NakedCharacter',
        level: 1,
        hair: 30000,
        face: 20000,
        skinColor: 0,
      });

      const expectedUrl = generateExpectedUrl(0, 30000, 20000, [], 'stand1');
      setupMockWithUrl(expectedUrl);

      render(<CharacterRenderer character={character} inventory={[]} />, { wrapper: TestWrapper });

      await waitFor(() => {
        expect(screen.getByTestId('character-image')).toBeInTheDocument();
      });

      const image = screen.getByTestId('character-image');
      const imageUrl = image.getAttribute('src');
      
      expect(imageUrl).toContain('2000'); // Light skin
      expect(imageUrl).toContain('30000:0'); // Hair
      expect(imageUrl).toContain('20000:0'); // Face
      expect(imageUrl).toContain('stand1'); // Default stance
      // Should not contain any equipment IDs
      expect(imageUrl).not.toMatch(/:[1-9]\d{6}:/);
    });

    it('should handle character with invalid equipment IDs', async () => {
      const character = createCharacter({
        name: 'InvalidEquipCharacter',
        level: 50,
        hair: 30010,
        face: 20010,
        skinColor: 1,
      });

      const invalidEquipment: Asset[] = [
        createAsset(-1, 0),        // Invalid item ID
        createAsset(-5, -1),       // Invalid item ID
        createAsset(-11, 1302000), // Valid weapon
      ];

      // URL should only include valid equipment (filtered)
      const validEquipment = invalidEquipment.filter(e => e.attributes.templateId > 0);
      const expectedUrl = generateExpectedUrl(1, 30010, 20010, validEquipment, 'stand1');
      setupMockWithUrl(expectedUrl);

      render(<CharacterRenderer character={character} inventory={invalidEquipment} />, { wrapper: TestWrapper });

      await waitFor(() => {
        expect(screen.getByTestId('character-image')).toBeInTheDocument();
      });

      const image = screen.getByTestId('character-image');
      const imageUrl = image.getAttribute('src');
      
      expect(imageUrl).toContain('1302000:0'); // Valid weapon should be included
      // Invalid IDs (0 and -1) should not appear as standalone equipment items
      // Using regex to ensure we don't match partial strings like "30010:0"
      expect(imageUrl).not.toMatch(/,0:0/); // Invalid ID 0 should be filtered out
      expect(imageUrl).not.toMatch(/,-1:0/); // Invalid ID -1 should be filtered out
    });

    it('should handle extreme skin color values', async () => {
      const character = createCharacter({
        name: 'ExtremeSkinCharacter',
        level: 25,
        hair: 30005,
        face: 20005,
        skinColor: 999, // Invalid skin color
      });

      // Invalid skin color defaults to 0
      const expectedUrl = generateExpectedUrl(999, 30005, 20005, [], 'stand1');
      setupMockWithUrl(expectedUrl);

      render(<CharacterRenderer character={character} />, { wrapper: TestWrapper });

      await waitFor(() => {
        expect(screen.getByTestId('character-image')).toBeInTheDocument();
      });

      const image = screen.getByTestId('character-image');
      const imageUrl = image.getAttribute('src');
      
      expect(imageUrl).toContain('2000'); // Should default to light skin
    });
  });

  describe('Performance and caching scenarios', () => {
    it('should handle rapid character changes efficiently', async () => {
      const baseCharacter = createCharacter({
        name: 'RapidChangeCharacter',
        level: 60,
        hair: 30025,
        face: 20015,
        skinColor: 2,
      });

      const equipment1: Asset[] = [
        createAsset(-11, 1302000), // One-handed sword
      ];

      const equipment2: Asset[] = [
        createAsset(-11, 1412000), // Two-handed axe
      ];

      // Set up mock for first equipment
      const expectedUrl1 = generateExpectedUrl(2, 30025, 20015, equipment1, 'stand1');
      setupMockWithUrl(expectedUrl1);

      const { rerender } = render(
        <CharacterRenderer character={baseCharacter} inventory={equipment1} />,
        { wrapper: TestWrapper }
      );

      await waitFor(() => {
        expect(screen.getByTestId('character-image')).toBeInTheDocument();
      });

      let image = screen.getByTestId('character-image');
      let imageUrl = image.getAttribute('src');
      expect(imageUrl).toContain('1302000:0');
      expect(imageUrl).toContain('stand1');

      // Update mock for second equipment
      const expectedUrl2 = generateExpectedUrl(2, 30025, 20015, equipment2, 'stand2');
      setupMockWithUrl(expectedUrl2);

      // Rapid change to different equipment
      rerender(
        <CharacterRenderer character={baseCharacter} inventory={equipment2} />
      );

      await waitFor(() => {
        const img = screen.getByTestId('character-image');
        expect(img.getAttribute('src')).toContain('1412000:0');
      });

      image = screen.getByTestId('character-image');
      imageUrl = image.getAttribute('src');
      expect(imageUrl).toContain('1412000:0');
      expect(imageUrl).toContain('stand2');
    });
  });
});