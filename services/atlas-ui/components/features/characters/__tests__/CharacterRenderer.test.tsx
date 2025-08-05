/**
 * Comprehensive tests for CharacterRenderer component with various equipment combinations
 */

import React from 'react';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { CharacterRenderer } from '../CharacterRenderer';
import { mapleStoryService } from '@/services/api/maplestory.service';
import type { Character } from '@/types/models/character';
import type { Asset } from '@/services/api/inventory.service';

// Mock the MapleStory service
jest.mock('@/services/api/maplestory.service', () => ({
  mapleStoryService: {
    characterToMapleStoryData: jest.fn(),
    generateCharacterImage: jest.fn(),
  },
}));

// Mock Next.js Image component
jest.mock('next/image', () => {
  return function MockImage({ src, alt, onError, ...props }: {
    src: string;
    alt: string;
    onError?: () => void;
    [key: string]: unknown;
  }) {
    return (
      // eslint-disable-next-line @next/next/no-img-element
      <img
        src={src}
        alt={alt}
        onError={onError}
        data-testid="character-image"
        {...props}
      />
    );
  };
});

// Mock the character image hook
jest.mock('@/lib/hooks/useCharacterImage', () => ({
  useCharacterImage: jest.fn(),
}));

// Mock the intersection observer hook
jest.mock('@/lib/hooks/useIntersectionObserver', () => ({
  useLazyLoad: jest.fn(() => ({
    shouldLoad: true,
    ref: { current: null },
  })),
}));

const mockMapleStoryService = mapleStoryService as jest.Mocked<typeof mapleStoryService>;

// Test wrapper with QueryClient
function TestWrapper({ children }: { children: React.ReactNode }) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

  return (
    <QueryClientProvider client={queryClient}>
      {children}
    </QueryClientProvider>
  );
}

describe('CharacterRenderer', () => {
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const mockUseCharacterImage = require('@/lib/hooks/useCharacterImage').useCharacterImage as jest.MockedFunction<typeof import('@/lib/hooks/useCharacterImage').useCharacterImage>;

  beforeEach(() => {
    jest.clearAllMocks();
    
    // Default successful mock for useCharacterImage
    mockUseCharacterImage.mockReturnValue({
      data: {
        url: 'https://maplestory.io/api/character/test.png',
        character: {},
        options: {},
        cached: false,
      },
      isLoading: false,
      error: null,
      refetch: jest.fn(),
      preload: jest.fn(),
      prefetchVariants: jest.fn(),
      imageUrl: 'https://maplestory.io/api/character/test.png',
      cached: false,
    });

    // Mock the service methods
    mockMapleStoryService.characterToMapleStoryData.mockReturnValue({
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

  const mockCharacter: Character = {
    id: 'char1',
    type: 'character',
    attributes: {
      accountId: 'acc1',
      name: 'TestWarrior',
      level: 85,
      jobId: 110,
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

  beforeEach(() => {
    jest.clearAllMocks();
    
    // Default mock implementations
    mockMapleStoryService.characterToMapleStoryData.mockReturnValue({
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
    
    mockMapleStoryService.generateCharacterImage.mockResolvedValue({
      url: 'https://maplestory.io/api/GMS/214/character/center/2002/30000:0,20000:0/stand1/0?resize=2',
      character: {
        id: 'char1',
        name: 'TestWarrior',
        level: 85,
        jobId: 110,
        hair: 30000,
        face: 20000,
        skinColor: 2,
        gender: 0,
        equipment: {},
      },
      options: {
        hair: 30000,
        face: 20000,
        skin: 2002,
        equipment: {},
        stance: 'stand1',
        resize: 2,
      },
      cached: false,
    });
  });

  describe('Basic rendering', () => {
    it('should render character with no equipment', async () => {
      render(<CharacterRenderer character={mockCharacter} />, { wrapper: TestWrapper });

      await waitFor(() => {
        expect(screen.getByTestId('character-image')).toBeInTheDocument();
      });

      expect(mockMapleStoryService.characterToMapleStoryData).toHaveBeenCalledWith(
        mockCharacter,
        []
      );
      expect(mockMapleStoryService.generateCharacterImage).toHaveBeenCalledWith(
        expect.objectContaining({
          id: 'char1',
          name: 'TestWarrior',
          equipment: {},
        }),
        { resize: 2 }
      );
    });

    it('should render character with basic equipment', async () => {
      const basicEquipment: Asset[] = [
        createMockAsset(-1, 1001000),  // Hat
        createMockAsset(-5, 1041000),  // Top
        createMockAsset(-11, 1202000), // One-handed sword
      ];

      mockMapleStoryService.characterToMapleStoryData.mockReturnValue({
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
          '-11': 1202000,
        },
      });

      render(<CharacterRenderer character={mockCharacter} inventory={basicEquipment} />, { wrapper: TestWrapper });

      await waitFor(() => {
        expect(screen.getByTestId('character-image')).toBeInTheDocument();
      });

      expect(mockMapleStoryService.characterToMapleStoryData).toHaveBeenCalledWith(
        mockCharacter,
        basicEquipment
      );
    });
  });

  describe('Equipment combinations', () => {
    it('should render character with full armor set', async () => {
      const armorSet: Asset[] = [
        createMockAsset(-1, 1001000),  // Hat
        createMockAsset(-5, 1041000),  // Top
        createMockAsset(-6, 1061000),  // Bottom
        createMockAsset(-7, 1071000),  // Shoes
        createMockAsset(-8, 1081000),  // Gloves
        createMockAsset(-9, 1102000),  // Cape
      ];

      mockMapleStoryService.characterToMapleStoryData.mockReturnValue({
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
          '-6': 1061000,
          '-7': 1071000,
          '-8': 1081000,
          '-9': 1102000,
        },
      });

      render(<CharacterRenderer character={mockCharacter} inventory={armorSet} />, { wrapper: TestWrapper });

      await waitFor(() => {
        expect(screen.getByTestId('character-image')).toBeInTheDocument();
      });

      expect(mockMapleStoryService.characterToMapleStoryData).toHaveBeenCalledWith(
        mockCharacter,
        armorSet
      );
    });

    it('should render character with two-handed weapon and shield', async () => {
      const weaponSet: Asset[] = [
        createMockAsset(-10, 1092000), // Shield
        createMockAsset(-11, 1302000), // Two-handed sword
      ];

      mockMapleStoryService.characterToMapleStoryData.mockReturnValue({
        id: 'char1',
        name: 'TestWarrior',
        level: 85,
        jobId: 110,
        hair: 30000,
        face: 20000,
        skinColor: 2,
        gender: 0,
        equipment: {
          '-10': 1092000,
          '-11': 1302000,
        },
      });

      mockMapleStoryService.generateCharacterImage.mockResolvedValue({
        url: 'https://maplestory.io/api/GMS/214/character/center/2002/30000:0,20000:0,1092000:0,1302000:0/stand2/0?resize=2',
        character: {
          id: 'char1',
          name: 'TestWarrior',
          level: 85,
          jobId: 110,
          hair: 30000,
          face: 20000,
          skinColor: 2,
          gender: 0,
          equipment: {
            '-10': 1092000,
            '-11': 1302000,
          },
        },
        options: {
          hair: 30000,
          face: 20000,
          skin: 2002,
          equipment: {
            '-10': 1092000,
            '-11': 1302000,
          },
          stance: 'stand2', // Two-handed weapon
          resize: 2,
        },
        cached: false,
      });

      render(<CharacterRenderer character={mockCharacter} inventory={weaponSet} />, { wrapper: TestWrapper });

      await waitFor(() => {
        expect(screen.getByTestId('character-image')).toBeInTheDocument();
      });

      const image = screen.getByTestId('character-image');
      expect(image).toHaveAttribute('src', expect.stringContaining('stand2'));
    });

    it('should render character with accessories', async () => {
      const accessorySet: Asset[] = [
        createMockAsset(-12, 1112000), // Ring 1
        createMockAsset(-13, 1112001), // Ring 2
        createMockAsset(-16, 1122000), // Pendant
        createMockAsset(-17, 1132000), // Belt
        createMockAsset(-21, 1022000), // Eye accessory
        createMockAsset(-22, 1012000), // Face accessory
        createMockAsset(-23, 1032000), // Earrings
      ];

      mockMapleStoryService.characterToMapleStoryData.mockReturnValue({
        id: 'char1',
        name: 'TestWarrior',
        level: 85,
        jobId: 110,
        hair: 30000,
        face: 20000,
        skinColor: 2,
        gender: 0,
        equipment: {
          '-12': 1112000,
          '-13': 1112001,
          '-16': 1122000,
          '-17': 1132000,
          '-21': 1022000,
          '-22': 1012000,
          '-23': 1032000,
        },
      });

      render(<CharacterRenderer character={mockCharacter} inventory={accessorySet} />, { wrapper: TestWrapper });

      await waitFor(() => {
        expect(screen.getByTestId('character-image')).toBeInTheDocument();
      });

      expect(mockMapleStoryService.characterToMapleStoryData).toHaveBeenCalledWith(
        mockCharacter,
        accessorySet
      );
    });

    it('should render character with cash equipment', async () => {
      const cashSet: Asset[] = [
        createMockAsset(-101, 1001001), // Cash Hat
        createMockAsset(-104, 1041001), // Cash Top
        createMockAsset(-111, 1302001), // Cash Weapon
      ];

      mockMapleStoryService.characterToMapleStoryData.mockReturnValue({
        id: 'char1',
        name: 'TestWarrior',
        level: 85,
        jobId: 110,
        hair: 30000,
        face: 20000,
        skinColor: 2,
        gender: 0,
        equipment: {
          '-101': 1001001,
          '-104': 1041001,
          '-111': 1302001,
        },
      });

      render(<CharacterRenderer character={mockCharacter} inventory={cashSet} />, { wrapper: TestWrapper });

      await waitFor(() => {
        expect(screen.getByTestId('character-image')).toBeInTheDocument();
      });

      expect(mockMapleStoryService.characterToMapleStoryData).toHaveBeenCalledWith(
        mockCharacter,
        cashSet
      );
    });

    it('should render character with mixed regular and cash equipment', async () => {
      const mixedSet: Asset[] = [
        createMockAsset(-1, 1001000),   // Regular Hat
        createMockAsset(-5, 1041000),   // Regular Top
        createMockAsset(-101, 1001001), // Cash Hat
        createMockAsset(-111, 1302001), // Cash Weapon
      ];

      mockMapleStoryService.characterToMapleStoryData.mockReturnValue({
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
          '-101': 1001001,
          '-111': 1302001,
        },
      });

      render(<CharacterRenderer character={mockCharacter} inventory={mixedSet} />, { wrapper: TestWrapper });

      await waitFor(() => {
        expect(screen.getByTestId('character-image')).toBeInTheDocument();
      });

      expect(mockMapleStoryService.characterToMapleStoryData).toHaveBeenCalledWith(
        mockCharacter,
        mixedSet
      );
    });

    it('should render character with full equipment loadout', async () => {
      const fullSet: Asset[] = [
        // Regular equipment
        createMockAsset(-1, 1001000),  // Hat
        createMockAsset(-5, 1041000),  // Top
        createMockAsset(-6, 1061000),  // Bottom
        createMockAsset(-7, 1071000),  // Shoes
        createMockAsset(-8, 1081000),  // Gloves
        createMockAsset(-9, 1102000),  // Cape
        createMockAsset(-10, 1092000), // Shield
        createMockAsset(-11, 1202000), // One-handed sword
        createMockAsset(-12, 1112000), // Ring 1
        createMockAsset(-13, 1112001), // Ring 2
        createMockAsset(-16, 1122000), // Pendant
        createMockAsset(-17, 1132000), // Belt
        createMockAsset(-21, 1022000), // Eye accessory
        createMockAsset(-22, 1012000), // Face accessory
        createMockAsset(-23, 1032000), // Earrings
        // Cash equipment
        createMockAsset(-101, 1001001), // Cash Hat
        createMockAsset(-111, 1302001), // Cash Weapon
        // Regular inventory (should be ignored)
        createMockAsset(1, 2000000),   // Potion
        createMockAsset(2, 2001000),   // Equipment in inventory
      ];

      mockMapleStoryService.characterToMapleStoryData.mockReturnValue({
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
          '-6': 1061000,
          '-7': 1071000,
          '-8': 1081000,
          '-9': 1102000,
          '-10': 1092000,
          '-11': 1202000,
          '-12': 1112000,
          '-13': 1112001,
          '-16': 1122000,
          '-17': 1132000,
          '-21': 1022000,
          '-22': 1012000,
          '-23': 1032000,
          '-101': 1001001,
          '-111': 1302001,
        },
      });

      render(<CharacterRenderer character={mockCharacter} inventory={fullSet} />, { wrapper: TestWrapper });

      await waitFor(() => {
        expect(screen.getByTestId('character-image')).toBeInTheDocument();
      });

      expect(mockMapleStoryService.characterToMapleStoryData).toHaveBeenCalledWith(
        mockCharacter,
        fullSet
      );
    });
  });

  describe('Component options', () => {
    it('should render with different sizes', async () => {
      const { rerender } = render(
        <CharacterRenderer character={mockCharacter} size="small" />
      );

      await waitFor(() => {
        expect(screen.getByTestId('character-image')).toBeInTheDocument();
      });

      let container = screen.getByTestId('character-image').parentElement;
      expect(container).toHaveClass('w-32', 'h-32');

      rerender(<CharacterRenderer character={mockCharacter} size="large" />);

      await waitFor(() => {
        expect(screen.getByTestId('character-image')).toBeInTheDocument();
      });

      container = screen.getByTestId('character-image').parentElement;
      expect(container).toHaveClass('w-64', 'h-64');
    });

    it('should use custom scale factor', async () => {
      render(<CharacterRenderer character={mockCharacter} scale={4} />, { wrapper: TestWrapper });

      await waitFor(() => {
        expect(mockMapleStoryService.generateCharacterImage).toHaveBeenCalledWith(
          expect.any(Object),
          { resize: 4 }
        );
      });
    });

    it('should handle custom className', async () => {
      render(
        <CharacterRenderer 
          character={mockCharacter} 
          className="custom-class border-2" 
        />
      );

      await waitFor(() => {
        expect(screen.getByTestId('character-image')).toBeInTheDocument();
      });

      const container = screen.getByTestId('character-image').parentElement;
      expect(container).toHaveClass('custom-class', 'border-2');
    });
  });

  describe('Loading states', () => {
    it('should show loading skeleton by default', () => {
      // Don't resolve the promise immediately
      mockMapleStoryService.generateCharacterImage.mockImplementation(
        () => new Promise(() => {}) // Never resolves
      );

      render(<CharacterRenderer character={mockCharacter} />, { wrapper: TestWrapper });

      // Should show skeleton loading
      expect(screen.queryByTestId('character-image')).not.toBeInTheDocument();
    });

    it('should not show loading skeleton when showLoading is false', () => {
      mockMapleStoryService.generateCharacterImage.mockImplementation(
        () => new Promise(() => {}) // Never resolves
      );

      render(<CharacterRenderer character={mockCharacter} showLoading={false} />, { wrapper: TestWrapper });

      // Should not show loading state
      expect(screen.queryByTestId('character-image')).not.toBeInTheDocument();
    });
  });

  describe('Error handling', () => {
    it('should handle API errors gracefully', async () => {
      mockMapleStoryService.generateCharacterImage.mockRejectedValue(
        new Error('API service is temporarily unavailable')
      );

      render(<CharacterRenderer character={mockCharacter} />, { wrapper: TestWrapper });

      await waitFor(() => {
        expect(screen.getByTestId('character-image')).toBeInTheDocument();
      });

      // Should show fallback image
      const image = screen.getByTestId('character-image');
      expect(image).toHaveAttribute('src', '/default-character-avatar.svg');
      expect(image).toHaveAttribute('alt', 'TestWarrior (fallback)');
    });

    it('should handle image load errors', async () => {
      render(<CharacterRenderer character={mockCharacter} />, { wrapper: TestWrapper });

      await waitFor(() => {
        expect(screen.getByTestId('character-image')).toBeInTheDocument();
      });

      // Simulate image load error
      const image = screen.getByTestId('character-image');
      fireEvent.error(image);

      await waitFor(() => {
        // Should still show the image (error handling is internal)
        expect(screen.getByTestId('character-image')).toBeInTheDocument();
      });
    });

    it('should show retry button on retryable errors', async () => {
      mockMapleStoryService.generateCharacterImage.mockRejectedValue(
        new Error('Network connection failed')
      );

      render(<CharacterRenderer character={mockCharacter} maxRetries={2} />, { wrapper: TestWrapper });

      await waitFor(() => {
        expect(screen.getByText(/Network connection failed/)).toBeInTheDocument();
      });

      // Should show retry button
      const retryButton = screen.getByText(/Retry \(1\/2\)/);
      expect(retryButton).toBeInTheDocument();

      // Click retry
      fireEvent.click(retryButton);

      await waitFor(() => {
        expect(mockMapleStoryService.generateCharacterImage).toHaveBeenCalledTimes(2);
      });
    });

    it('should not show retry button when showRetryButton is false', async () => {
      mockMapleStoryService.generateCharacterImage.mockRejectedValue(
        new Error('Network connection failed')
      );

      render(
        <CharacterRenderer 
          character={mockCharacter} 
          showRetryButton={false}
        />
      );

      await waitFor(() => {
        expect(screen.getByText(/Network connection failed/)).toBeInTheDocument();
      });

      // Should not show retry button
      expect(screen.queryByText(/Retry/)).not.toBeInTheDocument();
    });

    it('should use custom fallback avatar', async () => {
      mockMapleStoryService.generateCharacterImage.mockRejectedValue(
        new Error('Failed to generate character image')
      );

      render(
        <CharacterRenderer 
          character={mockCharacter} 
          fallbackAvatar="/custom-avatar.png"
        />
      );

      await waitFor(() => {
        expect(screen.getByTestId('character-image')).toBeInTheDocument();
      });

      const image = screen.getByTestId('character-image');
      expect(image).toHaveAttribute('src', '/custom-avatar.png');
    });

    it('should handle fallback image errors with inline SVG', async () => {
      mockMapleStoryService.generateCharacterImage.mockRejectedValue(
        new Error('Character image unavailable')
      );

      render(<CharacterRenderer character={mockCharacter} />, { wrapper: TestWrapper });

      await waitFor(() => {
        expect(screen.getByTestId('character-image')).toBeInTheDocument();
      });

      // Simulate fallback image error
      const image = screen.getByTestId('character-image');
      fireEvent.error(image);

      await waitFor(() => {
        expect(screen.getByText('Character image unavailable')).toBeInTheDocument();
        expect(screen.getByRole('img', { hidden: true })).toBeInTheDocument(); // SVG
      });
    });
  });

  describe('Callback functions', () => {
    it('should call onImageLoad when image loads successfully', async () => {
      const onImageLoad = jest.fn();

      render(
        <CharacterRenderer 
          character={mockCharacter} 
          onImageLoad={onImageLoad}
        />
      );

      await waitFor(() => {
        expect(screen.getByTestId('character-image')).toBeInTheDocument();
      });

      // Simulate successful image load
      const image = screen.getByTestId('character-image');
      fireEvent.load(image);

      expect(onImageLoad).toHaveBeenCalled();
    });

    it('should call onImageError when image fails to load', async () => {
      const onImageError = jest.fn();
      mockMapleStoryService.generateCharacterImage.mockRejectedValue(
        new Error('Failed to load character image')
      );

      render(
        <CharacterRenderer 
          character={mockCharacter} 
          onImageError={onImageError}
        />
      );

      await waitFor(() => {
        expect(onImageError).toHaveBeenCalledWith(
          expect.objectContaining({
            message: expect.stringContaining('Failed to load character image')
          })
        );
      });
    });
  });

  describe('Different character types', () => {
    it('should render different job characters correctly', async () => {
      const archerCharacter: Character = {
        ...mockCharacter,
        attributes: {
          ...mockCharacter.attributes,
          name: 'TestArcher',
          jobId: 300, // Archer
          hair: 31000,
          face: 21000,
          skinColor: 1,
        },
      };

      const archerEquipment: Asset[] = [
        createMockAsset(-11, 1452000), // Bow
        createMockAsset(-1, 1001001),  // Archer hat
      ];

      mockMapleStoryService.characterToMapleStoryData.mockReturnValue({
        id: 'char1',
        name: 'TestArcher',
        level: 85,
        jobId: 300,
        hair: 31000,
        face: 21000,
        skinColor: 1,
        gender: 0,
        equipment: {
          '-11': 1452000,
          '-1': 1001001,
        },
      });

      render(
        <CharacterRenderer 
          character={archerCharacter} 
          inventory={archerEquipment}
        />
      );

      await waitFor(() => {
        expect(screen.getByTestId('character-image')).toBeInTheDocument();
      });

      expect(mockMapleStoryService.characterToMapleStoryData).toHaveBeenCalledWith(
        archerCharacter,
        archerEquipment
      );
    });

    it('should render female characters correctly', async () => {
      const femaleCharacter: Character = {
        ...mockCharacter,
        attributes: {
          ...mockCharacter.attributes,
          name: 'TestMage',
          gender: 1, // Female
          hair: 32000,
          face: 22000,
        },
      };

      mockMapleStoryService.characterToMapleStoryData.mockReturnValue({
        id: 'char1',
        name: 'TestMage',
        level: 85,
        jobId: 110,
        hair: 32000,
        face: 22000,
        skinColor: 2,
        gender: 1,
        equipment: {},
      });

      render(<CharacterRenderer character={femaleCharacter} />, { wrapper: TestWrapper });

      await waitFor(() => {
        expect(screen.getByTestId('character-image')).toBeInTheDocument();
      });

      expect(mockMapleStoryService.characterToMapleStoryData).toHaveBeenCalledWith(
        femaleCharacter,
        []
      );
    });
  });
});