/**
 * @jest-environment jsdom
 */

import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { inventoryService } from '@/services/api/inventory.service';
import { 
  useInventory, 
  useCompartments,
  useCompartmentAssets,
  useInventorySummary,
  useHasAsset,
  useDeleteAsset,
  useInvalidateInventory,
  usePrefetchInventory,
  usePrefetchCompartments,
  usePrefetchCompartmentAssets,
  useCompartmentTypeName,
  useGetAssetsForCompartment,
  inventoryKeys 
} from '../useInventory';
import type { 
  Inventory, 
  InventoryResponse, 
  Compartment, 
  Asset, 
  CompartmentType 
} from '@/services/api/inventory.service';
import type { Tenant } from '@/types/models/tenant';
import { ReactNode } from 'react';

// Mock the inventory service
jest.mock('@/services/api/inventory.service', () => ({
  inventoryService: {
    getInventory: jest.fn(),
    getCompartments: jest.fn(),
    getCompartmentAssets: jest.fn(),
    getInventorySummary: jest.fn(),
    hasAsset: jest.fn(),
    deleteAsset: jest.fn(),
    getCompartmentTypeName: jest.fn(),
    getAssetsForCompartment: jest.fn(),
  },
}));

const mockInventoryService = inventoryService as jest.Mocked<typeof inventoryService>;

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

const mockInventory: Inventory = {
  type: 'inventories',
  id: 'inventory-1',
  attributes: {
    characterId: 123,
  },
  relationships: {
    compartments: {
      links: {
        related: '/api/characters/123/inventory/compartments',
        self: '/api/characters/123/inventory/relationships/compartments',
      },
      data: [
        { type: 'compartments', id: 'comp-1' },
        { type: 'compartments', id: 'comp-2' },
      ],
    },
  },
};

const mockCompartment: Compartment = {
  type: 'compartments',
  id: 'comp-1',
  attributes: {
    type: 1, // CompartmentType.EQUIPABLES
    capacity: 24,
  },
  relationships: {
    assets: {
      links: {
        related: '/api/characters/123/inventory/compartments/comp-1/assets',
        self: '/api/characters/123/inventory/compartments/comp-1/relationships/assets',
      },
      data: [
        { type: 'assets', id: 'asset-1' },
        { type: 'assets', id: 'asset-2' },
      ],
    },
  },
};

const mockAsset: Asset = {
  type: 'assets',
  id: 'asset-1',
  attributes: {
    slot: 0,
    templateId: 1001,
    expiration: '2024-12-31T23:59:59.999Z',
    referenceId: 123,
    referenceType: 'character',
    referenceData: null,
  },
};

const mockInventoryResponse: InventoryResponse = {
  data: mockInventory,
  included: [
    mockCompartment,
    {
      ...mockCompartment,
      id: 'comp-2',
      attributes: { type: 2, capacity: 24 }, // CompartmentType.CONSUMABLES
    },
    mockAsset,
    {
      ...mockAsset,
      id: 'asset-2',
      attributes: { ...mockAsset.attributes, slot: 1 },
    },
  ],
};

const mockCompartments: Compartment[] = [
  mockCompartment,
  {
    ...mockCompartment,
    id: 'comp-2',
    attributes: { type: 2, capacity: 24 },
  },
];

const mockAssets: Asset[] = [
  mockAsset,
  {
    ...mockAsset,
    id: 'asset-2',
    attributes: { ...mockAsset.attributes, slot: 1 },
  },
];

const mockInventorySummary = {
  totalCompartments: 2,
  totalAssets: 2,
  compartmentSummary: [
    {
      type: 1,
      name: 'Equipables',
      assetCount: 2,
      capacity: 24,
    },
    {
      type: 2,
      name: 'Consumables',
      assetCount: 0,
      capacity: 24,
    },
  ],
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

describe('useInventory hooks', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Query Keys', () => {
    it('should generate correct query keys', () => {
      expect(inventoryKeys.all).toEqual(['inventory']);
      expect(inventoryKeys.inventories()).toEqual(['inventory', 'inventory']);
      expect(inventoryKeys.inventory(mockTenant, '123')).toEqual(['inventory', 'inventory', 'tenant-123', '123', undefined]);
      expect(inventoryKeys.compartments()).toEqual(['inventory', 'compartments']);
      expect(inventoryKeys.compartmentsList(mockTenant, '123')).toEqual(['inventory', 'compartments', 'tenant-123', '123', undefined]);
      expect(inventoryKeys.compartmentAssets()).toEqual(['inventory', 'compartmentAssets']);
      expect(inventoryKeys.compartmentAssetsList(mockTenant, '123', 'comp-1')).toEqual(['inventory', 'compartmentAssets', 'tenant-123', '123', 'comp-1', undefined]);
      expect(inventoryKeys.summaries()).toEqual(['inventory', 'summary']);
      expect(inventoryKeys.summary(mockTenant, '123')).toEqual(['inventory', 'summary', 'tenant-123', '123', undefined]);
      expect(inventoryKeys.assets()).toEqual(['inventory', 'assets']);
      expect(inventoryKeys.hasAsset(mockTenant, '123', 'asset-1')).toEqual(['inventory', 'assets', 'has', 'tenant-123', '123', 'asset-1', undefined]);
    });
  });

  describe('useInventory', () => {
    it('should fetch inventory successfully', async () => {
      mockInventoryService.getInventory.mockResolvedValue(mockInventoryResponse);

      const { result } = renderHook(
        () => useInventory(mockTenant, '123'),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockInventoryResponse);
      expect(mockInventoryService.getInventory).toHaveBeenCalledWith(mockTenant, '123', undefined);
    });

    it('should not fetch when tenant is not provided', () => {
      const { result } = renderHook(
        () => useInventory(null as any, '123'),
        { wrapper: createWrapper() }
      );

      expect(result.current.status).toBe('pending');
      expect(mockInventoryService.getInventory).not.toHaveBeenCalled();
    });

    it('should not fetch when characterId is not provided', () => {
      const { result } = renderHook(
        () => useInventory(mockTenant, ''),
        { wrapper: createWrapper() }
      );

      expect(result.current.status).toBe('pending');
      expect(mockInventoryService.getInventory).not.toHaveBeenCalled();
    });

    it('should handle errors gracefully', async () => {
      const error = new Error('Failed to fetch inventory');
      mockInventoryService.getInventory.mockRejectedValue(error);

      const { result } = renderHook(
        () => useInventory(mockTenant, '123'),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.isError).toBe(true);
      });

      expect(result.current.error).toBe(error);
    });
  });

  describe('useCompartments', () => {
    it('should fetch compartments successfully', async () => {
      mockInventoryService.getCompartments.mockResolvedValue(mockCompartments);

      const { result } = renderHook(
        () => useCompartments(mockTenant, '123'),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockCompartments);
      expect(mockInventoryService.getCompartments).toHaveBeenCalledWith(mockTenant, '123', undefined);
    });
  });

  describe('useCompartmentAssets', () => {
    it('should fetch compartment assets successfully', async () => {
      mockInventoryService.getCompartmentAssets.mockResolvedValue(mockAssets);

      const { result } = renderHook(
        () => useCompartmentAssets(mockTenant, '123', 'comp-1'),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockAssets);
      expect(mockInventoryService.getCompartmentAssets).toHaveBeenCalledWith(mockTenant, '123', 'comp-1', undefined);
    });

    it('should not fetch when compartmentId is not provided', () => {
      const { result } = renderHook(
        () => useCompartmentAssets(mockTenant, '123', ''),
        { wrapper: createWrapper() }
      );

      expect(result.current.status).toBe('pending');
      expect(mockInventoryService.getCompartmentAssets).not.toHaveBeenCalled();
    });
  });

  describe('useInventorySummary', () => {
    it('should fetch inventory summary successfully', async () => {
      mockInventoryService.getInventorySummary.mockResolvedValue(mockInventorySummary);

      const { result } = renderHook(
        () => useInventorySummary(mockTenant, '123'),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockInventorySummary);
      expect(mockInventoryService.getInventorySummary).toHaveBeenCalledWith(mockTenant, '123', undefined);
    });
  });

  describe('useHasAsset', () => {
    it('should check asset existence successfully', async () => {
      mockInventoryService.hasAsset.mockResolvedValue(true);

      const { result } = renderHook(
        () => useHasAsset(mockTenant, '123', 'asset-1'),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toBe(true);
      expect(mockInventoryService.hasAsset).toHaveBeenCalledWith(mockTenant, '123', 'asset-1', undefined);
    });

    it('should not fetch when assetId is not provided', () => {
      const { result } = renderHook(
        () => useHasAsset(mockTenant, '123', ''),
        { wrapper: createWrapper() }
      );

      expect(result.current.status).toBe('pending');
      expect(mockInventoryService.hasAsset).not.toHaveBeenCalled();
    });
  });

  describe('useDeleteAsset', () => {
    it('should delete asset successfully', async () => {
      mockInventoryService.deleteAsset.mockResolvedValue(undefined);

      const wrapper = createWrapper();
      const { result } = renderHook(() => useDeleteAsset(), { wrapper });

      result.current.mutate({
        tenant: mockTenant,
        characterId: '123',
        compartmentId: 'comp-1',
        assetId: 'asset-1',
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockInventoryService.deleteAsset).toHaveBeenCalledWith(
        mockTenant,
        '123',
        'comp-1',
        'asset-1',
        undefined
      );
    });

    it('should handle delete errors', async () => {
      const error = new Error('Delete failed');
      mockInventoryService.deleteAsset.mockRejectedValue(error);

      const wrapper = createWrapper();
      const { result } = renderHook(() => useDeleteAsset(), { wrapper });

      result.current.mutate({
        tenant: mockTenant,
        characterId: '123',
        compartmentId: 'comp-1',
        assetId: 'asset-1',
      });

      await waitFor(() => {
        expect(result.current.isError).toBe(true);
      });

      expect(result.current.error).toBe(error);
    });
  });


  describe('Utility hooks', () => {
    describe('useInvalidateInventory', () => {
      it('should provide invalidation functions', () => {
        const wrapper = createWrapper();
        const { result } = renderHook(() => useInvalidateInventory(), { wrapper });

        expect(typeof result.current.invalidateAll).toBe('function');
        expect(typeof result.current.invalidateInventory).toBe('function');
        expect(typeof result.current.invalidateCompartments).toBe('function');
        expect(typeof result.current.invalidateCompartmentAssets).toBe('function');
        expect(typeof result.current.invalidateSummary).toBe('function');
        expect(typeof result.current.invalidateHasAsset).toBe('function');
        expect(typeof result.current.invalidateLegacy).toBe('function');
      });
    });

    describe('usePrefetchInventory', () => {
      it('should provide prefetch function', () => {
        const wrapper = createWrapper();
        const { result } = renderHook(() => usePrefetchInventory(), { wrapper });

        expect(typeof result.current).toBe('function');
      });
    });

    describe('usePrefetchCompartments', () => {
      it('should provide prefetch compartments function', () => {
        const wrapper = createWrapper();
        const { result } = renderHook(() => usePrefetchCompartments(), { wrapper });

        expect(typeof result.current).toBe('function');
      });
    });

    describe('usePrefetchCompartmentAssets', () => {
      it('should provide prefetch compartment assets function', () => {
        const wrapper = createWrapper();
        const { result } = renderHook(() => usePrefetchCompartmentAssets(), { wrapper });

        expect(typeof result.current).toBe('function');
      });
    });
  });

  describe('Helper hooks', () => {
    describe('useCompartmentTypeName', () => {
      it('should return compartment type name function', () => {
        mockInventoryService.getCompartmentTypeName.mockReturnValue('Equipables');

        const wrapper = createWrapper();
        const { result } = renderHook(() => useCompartmentTypeName(), { wrapper });

        const typeName = result.current(1);
        expect(typeName).toBe('Equipables');
        expect(mockInventoryService.getCompartmentTypeName).toHaveBeenCalledWith(1);
      });
    });

    describe('useGetAssetsForCompartment', () => {
      it('should return assets for compartment function', () => {
        mockInventoryService.getAssetsForCompartment.mockReturnValue(mockAssets);

        const wrapper = createWrapper();
        const { result } = renderHook(() => useGetAssetsForCompartment(), { wrapper });

        const assets = result.current(mockCompartment, mockInventoryResponse.included);
        expect(assets).toEqual(mockAssets);
        expect(mockInventoryService.getAssetsForCompartment).toHaveBeenCalledWith(
          mockCompartment, 
          mockInventoryResponse.included
        );
      });
    });
  });
});