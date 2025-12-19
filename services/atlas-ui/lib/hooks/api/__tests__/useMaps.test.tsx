/**
 * Tests for useMaps React Query hooks
 */

import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ReactNode } from 'react';

import {
  useMaps,
  useMap,
  useMapsByName,
  useMapsByStreetName,
  useCreateMap,
  useUpdateMap,
  useDeleteMap,
  useInvalidateMaps,
  mapKeys,
} from '../useMaps';
import { mapsService, type MapData, type MapAttributes } from '@/services/api/maps.service';

// Mock the maps service
jest.mock('@/services/api/maps.service', () => ({
  mapsService: {
    getAllMaps: jest.fn(),
    getMapById: jest.fn(),
    searchMapsByName: jest.fn(),
    getMapsByStreetName: jest.fn(),
    createMap: jest.fn(),
    updateMap: jest.fn(),
    deleteMap: jest.fn(),
  },
}));

const mockMapsService = jest.mocked(mapsService);

// Test data
const mockMapData: MapData = {
  id: '1',
  attributes: {
    name: 'Test Map',
    streetName: 'Test Street',
  },
};

const mockMapAttributes: MapAttributes = {
  name: 'New Map',
  streetName: 'New Street',
};

// Test wrapper with QueryClient
function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
      },
      mutations: {
        retry: false,
      },
    },
  });

  const Wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  Wrapper.displayName = 'QueryClientWrapper';
  
  return Wrapper;
}

describe('useMaps hooks', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  describe('Query hooks', () => {
    it('should fetch all maps', async () => {
      const mockMaps = [mockMapData];
      mockMapsService.getAllMaps.mockResolvedValueOnce(mockMaps);

      const { result } = renderHook(() => useMaps(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockMapsService.getAllMaps).toHaveBeenCalledTimes(1);
      expect(result.current.data).toEqual(mockMaps);
    });

    it('should fetch a specific map by ID', async () => {
      mockMapsService.getMapById.mockResolvedValueOnce(mockMapData);

      const { result } = renderHook(() => useMap('1'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockMapsService.getMapById).toHaveBeenCalledWith('1', undefined);
      expect(result.current.data).toEqual(mockMapData);
    });

    it('should not fetch map when ID is empty', () => {
      const { result } = renderHook(() => useMap(''), {
        wrapper: createWrapper(),
      });

      expect(result.current.isFetching).toBe(false);
      expect(mockMapsService.getMapById).not.toHaveBeenCalled();
    });

    it('should search maps by name', async () => {
      const mockMaps = [mockMapData];
      mockMapsService.searchMapsByName.mockResolvedValueOnce(mockMaps);

      const { result } = renderHook(() => useMapsByName('Test Map'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockMapsService.searchMapsByName).toHaveBeenCalledWith('Test Map', undefined);
      expect(result.current.data).toEqual(mockMaps);
    });

    it('should not search when name is empty', () => {
      const { result } = renderHook(() => useMapsByName(''), {
        wrapper: createWrapper(),
      });

      expect(result.current.isFetching).toBe(false);
      expect(mockMapsService.searchMapsByName).not.toHaveBeenCalled();
    });

    it('should fetch maps by street name', async () => {
      const mockMaps = [mockMapData];
      mockMapsService.getMapsByStreetName.mockResolvedValueOnce(mockMaps);

      const { result } = renderHook(() => useMapsByStreetName('Test Street'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockMapsService.getMapsByStreetName).toHaveBeenCalledWith('Test Street', undefined);
      expect(result.current.data).toEqual(mockMaps);
    });
  });

  describe('Mutation hooks', () => {
    it('should create a new map', async () => {
      mockMapsService.createMap.mockResolvedValueOnce(mockMapData);

      const { result } = renderHook(() => useCreateMap(), {
        wrapper: createWrapper(),
      });

      result.current.mutate(mockMapAttributes);

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockMapsService.createMap).toHaveBeenCalledWith(mockMapAttributes);
      expect(result.current.data).toEqual(mockMapData);
    });

    it('should update an existing map', async () => {
      const updatedMap = { ...mockMapData, attributes: { ...mockMapData.attributes, name: 'Updated Map' } };
      mockMapsService.updateMap.mockResolvedValueOnce(updatedMap);

      const { result } = renderHook(() => useUpdateMap(), {
        wrapper: createWrapper(),
      });

      const updateData = {
        map: mockMapData,
        updates: { name: 'Updated Map' },
      };

      result.current.mutate(updateData);

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockMapsService.updateMap).toHaveBeenCalledWith(mockMapData, { name: 'Updated Map' });
      expect(result.current.data).toEqual(updatedMap);
    });

    it('should delete a map', async () => {
      mockMapsService.deleteMap.mockResolvedValueOnce(undefined);

      const { result } = renderHook(() => useDeleteMap(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ id: '1' });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockMapsService.deleteMap).toHaveBeenCalledWith('1');
    });
  });

  describe('Utility hooks', () => {
    it('should provide invalidation functions', () => {
      const queryClient = new QueryClient();
      const invalidateAllSpy = jest.spyOn(queryClient, 'invalidateQueries');

      const { result } = renderHook(() => useInvalidateMaps(), {
        wrapper: ({ children }) => <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>,
      });

      // Test invalidateAll
      result.current.invalidateAll();
      expect(invalidateAllSpy).toHaveBeenCalledWith({ queryKey: mapKeys.all });

      // Test invalidateLists
      result.current.invalidateLists();
      expect(invalidateAllSpy).toHaveBeenCalledWith({ queryKey: mapKeys.lists() });

      // Test invalidateMap
      result.current.invalidateMap('1');
      expect(invalidateAllSpy).toHaveBeenCalledWith({ queryKey: mapKeys.detail('1') });
    });
  });

  describe('Query keys', () => {
    it('should generate correct query keys', () => {
      expect(mapKeys.all).toEqual(['maps']);
      expect(mapKeys.lists()).toEqual(['maps', 'list']);
      expect(mapKeys.list({ search: 'test' })).toEqual(['maps', 'list', { search: 'test' }]);
      expect(mapKeys.details()).toEqual(['maps', 'detail']);
      expect(mapKeys.detail('1')).toEqual(['maps', 'detail', '1']);
      expect(mapKeys.search()).toEqual(['maps', 'search']);
      expect(mapKeys.searchByName('test')).toEqual(['maps', 'search', 'name', 'test']);
      expect(mapKeys.searchByStreet('test street')).toEqual(['maps', 'search', 'street', 'test street']);
    });
  });
});