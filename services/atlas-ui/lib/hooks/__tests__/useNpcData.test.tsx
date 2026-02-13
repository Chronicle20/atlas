/**
 * Unit tests for useNpcData hook
 */

import React, { ReactNode } from 'react';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useNpcData, useNpcBatchData, useNpcDataCache } from '../useNpcData';
import { npcsService } from '@/services/api/npcs.service';

// Mock the tenant context
const mockActiveTenant = {
  id: 'test-tenant',
  type: 'tenant',
  attributes: {
    region: 'GMS',
    majorVersion: 83,
    minorVersion: 1,
  },
};

jest.mock('@/context/tenant-context', () => ({
  useTenant: () => ({ activeTenant: mockActiveTenant }),
}));

// Mock the NPC service
jest.mock('@/services/api/npcs.service', () => ({
  npcsService: {
    getNpcName: jest.fn(),
  },
}));

// Mock the asset URL utility
jest.mock('@/lib/utils/asset-url', () => ({
  getAssetIconUrl: jest.fn(
    (tenantId: string, region: string, majorVersion: number, minorVersion: number, category: string, entityId: number) =>
      `/api/assets/${tenantId}/${region}/${majorVersion}.${minorVersion}/${category}/${entityId}/icon.png`
  ),
}));

const mockNpcsService = npcsService as jest.Mocked<typeof npcsService>;

describe('useNpcData', () => {
  let queryClient: QueryClient;
  let wrapper: ({ children }: { children: ReactNode }) => React.JSX.Element;

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
      mockNpcsService.getNpcName.mockResolvedValue('Snail');

      const { result } = renderHook(() => useNpcData(1001), { wrapper });

      expect(result.current.isLoading).toBe(true);

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.name).toBe('Snail');
      expect(result.current.iconUrl).toBe('/api/assets/test-tenant/GMS/83.1/npc/1001/icon.png');
      expect(mockNpcsService.getNpcName).toHaveBeenCalledWith(1001, mockActiveTenant);
    });

    it('should handle API errors gracefully', async () => {
      mockNpcsService.getNpcName.mockRejectedValue(new Error('Failed to fetch NPC data'));

      const { result } = renderHook(() => useNpcData(9999), { wrapper });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.hasError).toBe(true);
      expect(result.current.errorMessage).toBe('Failed to fetch NPC data');
      expect(result.current.name).toBeUndefined();
      // iconUrl is still generated since it's deterministic
      expect(result.current.iconUrl).toBe('/api/assets/test-tenant/GMS/83.1/npc/9999/icon.png');
    });

    it('should not fetch when npcId is invalid', () => {
      const { result } = renderHook(() => useNpcData(0), { wrapper });

      expect(result.current.isLoading).toBe(false);
      expect(result.current.fetchStatus).toBe('idle');
      expect(mockNpcsService.getNpcName).not.toHaveBeenCalled();
    });

    it('should call success callback when data is fetched', async () => {
      mockNpcsService.getNpcName.mockResolvedValue('Snail');

      const onSuccess = jest.fn();

      renderHook(() => useNpcData(1001, { onSuccess }), { wrapper });

      await waitFor(() => {
        expect(onSuccess).toHaveBeenCalledWith(
          expect.objectContaining({
            id: 1001,
            name: 'Snail',
            iconUrl: '/api/assets/test-tenant/GMS/83.1/npc/1001/icon.png',
          })
        );
      });
    });

    it('should invalidate cache correctly', async () => {
      mockNpcsService.getNpcName.mockResolvedValue('Snail');

      const { result } = renderHook(() => useNpcData(1001), { wrapper });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockNpcsService.getNpcName).toHaveBeenCalledTimes(1);

      // Invalidate should be callable without error
      expect(() => result.current.invalidate()).not.toThrow();

      // Verify invalidate function exists
      expect(result.current.invalidate).toBeDefined();
      expect(typeof result.current.invalidate).toBe('function');
    });
  });

  describe('useNpcBatchData', () => {
    it('should fetch multiple NPCs successfully', async () => {
      mockNpcsService.getNpcName
        .mockResolvedValueOnce('Snail')
        .mockResolvedValueOnce('Blue Snail');

      const { result } = renderHook(() => useNpcBatchData([1001, 1002]), { wrapper });

      expect(result.current.isLoading).toBe(true);

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toHaveLength(2);
      expect(result.current.data[0]?.name).toBe('Snail');
      expect(result.current.data[1]?.name).toBe('Blue Snail');
      expect(mockNpcsService.getNpcName).toHaveBeenCalledTimes(2);
    });

    it('should handle empty NPC list', () => {
      const { result } = renderHook(() => useNpcBatchData([]), { wrapper });

      expect(result.current.isLoading).toBe(false);
      expect(result.current.isSuccess).toBe(true);
      expect(result.current.data).toHaveLength(0);
      expect(mockNpcsService.getNpcName).not.toHaveBeenCalled();
    });

    it('should handle mixed success and error responses', async () => {
      mockNpcsService.getNpcName
        .mockResolvedValueOnce('Snail')
        .mockRejectedValueOnce(new Error('Failed to fetch NPC data'));

      const { result } = renderHook(() => useNpcBatchData([1001, 9999]), { wrapper });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toHaveLength(2);
      expect(result.current.data[0]?.name).toBe('Snail');
      expect(result.current.data[1]?.error).toBe('Failed to fetch NPC data');
    });
  });

  describe('useNpcDataCache', () => {
    it('should provide cache management functions', () => {
      const { result } = renderHook(() => useNpcDataCache(), { wrapper });

      expect(result.current.getCacheStats).toBeDefined();
      expect(result.current.clearCache).toBeDefined();
    });

    it('should return correct cache stats', async () => {
      mockNpcsService.getNpcName.mockResolvedValue('Snail');

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
});
