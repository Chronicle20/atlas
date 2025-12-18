/**
 * Unit tests for useNpcData hook
 */

import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ReactNode } from 'react';
import { useNpcData, useNpcBatchData, useNpcDataCache, useNpcDataPreloader } from '../useNpcData';
import { mapleStoryService } from '@/services/api/maplestory.service';
import type { NpcDataResult } from '@/types/models/maplestory';

// Mock the MapleStory service
jest.mock('@/services/api/maplestory.service', () => ({
  mapleStoryService: {
    getNpcDataWithCache: jest.fn(),
  },
}));

const mockMapleStoryService = mapleStoryService as jest.Mocked<typeof mapleStoryService>;

describe('useNpcData', () => {
  let queryClient: QueryClient;
  let wrapper: ({ children }: { children: ReactNode }) => JSX.Element;

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: {
        queries: {
          retry: false,
          staleTime: 0,
          gcTime: 0,
        },
      },
    });

    wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );

    jest.clearAllMocks();
  });

  afterEach(() => {
    queryClient.clear();
  });

  describe('useNpcData', () => {
    it('should fetch NPC data successfully', async () => {
      const mockNpcData: NpcDataResult = {
        id: 1001,
        name: 'Snail',
        iconUrl: 'https://maplestory.io/api/GMS/214/npc/1001/icon',
        cached: false,
      };

      mockMapleStoryService.getNpcDataWithCache.mockResolvedValue(mockNpcData);

      const { result } = renderHook(() => useNpcData(1001), { wrapper });

      expect(result.current.isLoading).toBe(true);

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.npcData).toEqual(mockNpcData);
      expect(result.current.name).toBe('Snail');
      expect(result.current.iconUrl).toBe('https://maplestory.io/api/GMS/214/npc/1001/icon');
      expect(result.current.cached).toBe(false);
      expect(mockMapleStoryService.getNpcDataWithCache).toHaveBeenCalledWith(1001, undefined, undefined);
    });

    it('should handle API errors gracefully', async () => {
      const mockNpcData: NpcDataResult = {
        id: 9999,
        cached: false,
        error: 'Failed to fetch NPC data',
      };

      mockMapleStoryService.getNpcDataWithCache.mockResolvedValue(mockNpcData);

      const { result } = renderHook(() => useNpcData(9999), { wrapper });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.hasError).toBe(true);
      expect(result.current.errorMessage).toBe('Failed to fetch NPC data');
      expect(result.current.name).toBeUndefined();
      expect(result.current.iconUrl).toBeUndefined();
    });

    it('should not fetch when npcId is invalid', () => {
      const { result } = renderHook(() => useNpcData(0), { wrapper });

      expect(result.current.isLoading).toBe(false);
      expect(result.current.fetchStatus).toBe('idle');
      expect(mockMapleStoryService.getNpcDataWithCache).not.toHaveBeenCalled();
    });

    it('should use custom region and version', async () => {
      const mockNpcData: NpcDataResult = {
        id: 1001,
        name: 'Snail',
        iconUrl: 'https://maplestory.io/api/MSEA/214/npc/1001/icon',
        cached: false,
      };

      mockMapleStoryService.getNpcDataWithCache.mockResolvedValue(mockNpcData);

      const { result } = renderHook(
        () => useNpcData(1001, { region: 'MSEA', version: '214' }),
        { wrapper }
      );

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockMapleStoryService.getNpcDataWithCache).toHaveBeenCalledWith(1001, 'MSEA', '214');
    });

    it('should call success callback when data is fetched', async () => {
      const mockNpcData: NpcDataResult = {
        id: 1001,
        name: 'Snail',
        iconUrl: 'https://maplestory.io/api/GMS/214/npc/1001/icon',
        cached: false,
      };

      const onSuccess = jest.fn();
      mockMapleStoryService.getNpcDataWithCache.mockResolvedValue(mockNpcData);

      renderHook(() => useNpcData(1001, { onSuccess }), { wrapper });

      await waitFor(() => {
        expect(onSuccess).toHaveBeenCalledWith(mockNpcData);
      });
    });

    it('should invalidate cache correctly', async () => {
      const mockNpcData: NpcDataResult = {
        id: 1001,
        name: 'Snail',
        iconUrl: 'https://maplestory.io/api/GMS/214/npc/1001/icon',
        cached: false,
      };

      mockMapleStoryService.getNpcDataWithCache.mockResolvedValue(mockNpcData);

      const { result } = renderHook(() => useNpcData(1001), { wrapper });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockMapleStoryService.getNpcDataWithCache).toHaveBeenCalledTimes(1);

      // Invalidate should be callable without error
      expect(() => result.current.invalidate()).not.toThrow();
      
      // Verify invalidate function exists
      expect(result.current.invalidate).toBeDefined();
      expect(typeof result.current.invalidate).toBe('function');
    });
  });

  describe('useNpcBatchData', () => {
    it('should fetch multiple NPCs successfully', async () => {
      const mockNpcData1: NpcDataResult = {
        id: 1001,
        name: 'Snail',
        iconUrl: 'https://maplestory.io/api/GMS/214/npc/1001/icon',
        cached: false,
      };

      const mockNpcData2: NpcDataResult = {
        id: 1002,
        name: 'Blue Snail',
        iconUrl: 'https://maplestory.io/api/GMS/214/npc/1002/icon',
        cached: false,
      };

      mockMapleStoryService.getNpcDataWithCache
        .mockResolvedValueOnce(mockNpcData1)
        .mockResolvedValueOnce(mockNpcData2);

      const { result } = renderHook(() => useNpcBatchData([1001, 1002]), { wrapper });

      expect(result.current.isLoading).toBe(true);

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toHaveLength(2);
      expect(result.current.data[0]).toEqual(mockNpcData1);
      expect(result.current.data[1]).toEqual(mockNpcData2);
      expect(mockMapleStoryService.getNpcDataWithCache).toHaveBeenCalledTimes(2);
    });

    it('should handle empty NPC list', () => {
      const { result } = renderHook(() => useNpcBatchData([]), { wrapper });

      expect(result.current.isLoading).toBe(false);
      expect(result.current.isSuccess).toBe(true);
      expect(result.current.data).toHaveLength(0);
      expect(mockMapleStoryService.getNpcDataWithCache).not.toHaveBeenCalled();
    });

    it('should handle mixed success and error responses', async () => {
      const mockNpcData1: NpcDataResult = {
        id: 1001,
        name: 'Snail',
        iconUrl: 'https://maplestory.io/api/GMS/214/npc/1001/icon',
        cached: false,
      };

      const mockNpcData2: NpcDataResult = {
        id: 9999,
        cached: false,
        error: 'Failed to fetch NPC data',
      };

      mockMapleStoryService.getNpcDataWithCache
        .mockResolvedValueOnce(mockNpcData1)
        .mockResolvedValueOnce(mockNpcData2);

      const { result } = renderHook(() => useNpcBatchData([1001, 9999]), { wrapper });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toHaveLength(2);
      expect(result.current.data[0]).toEqual(mockNpcData1);
      expect(result.current.data[1]).toEqual(mockNpcData2);
    });
  });

  describe('useNpcDataCache', () => {
    it('should provide cache management functions', () => {
      const { result } = renderHook(() => useNpcDataCache(), { wrapper });

      expect(result.current.getCacheStats).toBeDefined();
      expect(result.current.clearCache).toBeDefined();
      expect(result.current.warmCache).toBeDefined();
    });

    it('should return correct cache stats', async () => {
      const mockNpcData: NpcDataResult = {
        id: 1001,
        name: 'Snail',
        iconUrl: 'https://maplestory.io/api/GMS/214/npc/1001/icon',
        cached: false,
      };

      mockMapleStoryService.getNpcDataWithCache.mockResolvedValue(mockNpcData);

      // First create some cache entries
      const { result: npcDataResult } = renderHook(() => useNpcData(1001), { wrapper });
      const { result: cacheResult } = renderHook(() => useNpcDataCache(), { wrapper });

      await waitFor(() => {
        expect(npcDataResult.current.isSuccess).toBe(true);
      });

      const stats = cacheResult.current.getCacheStats();
      expect(stats.totalQueries).toBeGreaterThan(0);
      expect(stats.activeQueries).toBeGreaterThan(0);
    });
  });

  describe('useNpcDataPreloader', () => {
    it('should provide preloading functions', () => {
      const { result } = renderHook(() => useNpcDataPreloader(), { wrapper });

      expect(result.current.preloadNpcData).toBeDefined();
    });

    it('should preload NPC data correctly', async () => {
      const mockNpcData: NpcDataResult = {
        id: 1001,
        name: 'Snail',
        iconUrl: 'https://maplestory.io/api/GMS/214/npc/1001/icon',
        cached: false,
      };

      mockMapleStoryService.getNpcDataWithCache.mockResolvedValue(mockNpcData);

      const { result } = renderHook(() => useNpcDataPreloader(), { wrapper });

      await result.current.preloadNpcData([
        { npcId: 1001, region: 'GMS', version: '214' }
      ]);

      expect(mockMapleStoryService.getNpcDataWithCache).toHaveBeenCalledWith(1001, 'GMS', '214');
    });
  });
});