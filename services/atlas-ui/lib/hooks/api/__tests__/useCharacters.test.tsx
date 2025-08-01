/**
 * @jest-environment jsdom
 */

import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { charactersService } from '@/services/api/characters.service';
import { 
  useCharacters, 
  useCharacter, 
  useUpdateCharacter,
  useInvalidateCharacters,
  usePrefetchCharacters,
  usePrefetchCharacter,
  characterKeys 
} from '../useCharacters';
import type { Character, UpdateCharacterData } from '@/types/models/character';
import type { Tenant } from '@/types/models/tenant';
import { ReactNode } from 'react';

// Mock the characters service
jest.mock('@/services/api/characters.service', () => ({
  charactersService: {
    getAll: jest.fn(),
    getById: jest.fn(),
    update: jest.fn(),
  },
}));

const mockCharactersService = charactersService as jest.Mocked<typeof charactersService>;

// Test data
const mockTenant: Tenant = {
  id: 'tenant-123',
  attributes: {
    name: 'Test Tenant',
    region: 'GMS',
    majorVersion: 83,
    minorVersion: 1,
  },
};

const mockCharacter: Character = {
  id: 'char-123',
  attributes: {
    accountId: 1,
    worldId: 0,
    name: 'TestCharacter',
    level: 50,
    experience: 1000000,
    gachaponExperience: 0,
    strength: 100,
    dexterity: 100,
    intelligence: 100,
    luck: 100,
    hp: 500,
    maxHp: 500,
    mp: 300,
    maxMp: 300,
    meso: 50000,
    hpMpUsed: 0,
    jobId: 100,
    skinColor: 0,
    gender: 0,
    fame: 0,
    hair: 30000,
    face: 20000,
    mapId: 100000000,
    spawnPoint: 0,
    gm: 0,
    x: 0,
    y: 0,
    stance: 0,
  },
};

const mockCharacters: Character[] = [
  mockCharacter,
  {
    ...mockCharacter,
    id: 'char-456',
    attributes: { ...mockCharacter.attributes, name: 'TestCharacter2' },
  },
];

const mockUpdateData: UpdateCharacterData = {
  mapId: 100000001,
};

// Test wrapper with QueryClient
function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

describe('useCharacters hooks', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Query Keys', () => {
    it('should generate correct query keys', () => {
      expect(characterKeys.all).toEqual(['characters']);
      expect(characterKeys.lists()).toEqual(['characters', 'list']);
      expect(characterKeys.list(mockTenant)).toEqual(['characters', 'list', 'tenant-123', undefined]);
      expect(characterKeys.details()).toEqual(['characters', 'detail']);
      expect(characterKeys.detail(mockTenant, 'char-123')).toEqual(['characters', 'detail', 'tenant-123', 'char-123']);
    });
  });

  describe('useCharacters', () => {
    it('should fetch characters successfully', async () => {
      mockCharactersService.getAll.mockResolvedValue(mockCharacters);

      const { result } = renderHook(
        () => useCharacters(mockTenant),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockCharacters);
      expect(mockCharactersService.getAll).toHaveBeenCalledWith(mockTenant, undefined);
    });

    it('should not fetch when tenant is not provided', () => {
      const { result } = renderHook(
        () => useCharacters(null as any),
        { wrapper: createWrapper() }
      );

      expect(result.current.status).toBe('pending');
      expect(mockCharactersService.getAll).not.toHaveBeenCalled();
    });

    it('should handle errors gracefully', async () => {
      const error = new Error('Failed to fetch characters');
      mockCharactersService.getAll.mockRejectedValue(error);

      const { result } = renderHook(
        () => useCharacters(mockTenant),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.isError).toBe(true);
      });

      expect(result.current.error).toBe(error);
    });
  });

  describe('useCharacter', () => {
    it('should fetch a single character successfully', async () => {
      mockCharactersService.getById.mockResolvedValue(mockCharacter);

      const { result } = renderHook(
        () => useCharacter(mockTenant, 'char-123'),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockCharacter);
      expect(mockCharactersService.getById).toHaveBeenCalledWith(mockTenant, 'char-123', undefined);
    });

    it('should not fetch when characterId is not provided', () => {
      const { result } = renderHook(
        () => useCharacter(mockTenant, ''),
        { wrapper: createWrapper() }
      );

      expect(result.current.status).toBe('pending');
      expect(mockCharactersService.getById).not.toHaveBeenCalled();
    });
  });

  describe('useUpdateCharacter', () => {
    it('should update character successfully', async () => {
      mockCharactersService.update.mockResolvedValue(undefined);

      const wrapper = createWrapper();
      const { result } = renderHook(() => useUpdateCharacter(), { wrapper });

      result.current.mutate({
        tenant: mockTenant,
        characterId: 'char-123',
        updates: mockUpdateData,
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockCharactersService.update).toHaveBeenCalledWith(
        mockTenant,
        'char-123',
        mockUpdateData
      );
    });

    it('should handle update errors', async () => {
      const error = new Error('Update failed');
      mockCharactersService.update.mockRejectedValue(error);

      const wrapper = createWrapper();
      const { result } = renderHook(() => useUpdateCharacter(), { wrapper });

      result.current.mutate({
        tenant: mockTenant,
        characterId: 'char-123',
        updates: mockUpdateData,
      });

      await waitFor(() => {
        expect(result.current.isError).toBe(true);
      });

      expect(result.current.error).toBe(error);
    });
  });


  describe('Utility hooks', () => {
    describe('useInvalidateCharacters', () => {
      it('should provide invalidation functions', () => {
        const wrapper = createWrapper();
        const { result } = renderHook(() => useInvalidateCharacters(), { wrapper });

        expect(typeof result.current.invalidateAll).toBe('function');
        expect(typeof result.current.invalidateList).toBe('function');
        expect(typeof result.current.invalidateCharacter).toBe('function');
        expect(typeof result.current.invalidateLegacy).toBe('function');
      });
    });

    describe('usePrefetchCharacters', () => {
      it('should provide prefetch function', () => {
        const wrapper = createWrapper();
        const { result } = renderHook(() => usePrefetchCharacters(), { wrapper });

        expect(typeof result.current).toBe('function');
      });
    });

    describe('usePrefetchCharacter', () => {
      it('should provide prefetch character function', () => {
        const wrapper = createWrapper();
        const { result } = renderHook(() => usePrefetchCharacter(), { wrapper });

        expect(typeof result.current).toBe('function');
      });
    });
  });
});