/**
 * Tests for useNpcs React Query hooks
 */

import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ReactNode } from 'react';
import { 
  useNPCs, 
  useNPC, 
  useNPCsWithShops, 
  useNPCsWithConversations, 
  useNPCShop,
  useCreateShop,
  useUpdateShop,
  useCreateCommodity,
  useUpdateCommodity,
  useDeleteCommodity,
  npcKeys 
} from '../useNpcs';
import { npcsService } from '@/services/api/npcs.service';
import type { NPC, Shop, Commodity, CommodityAttributes, ShopResponse } from '@/services/api/npcs.service';
import type { Tenant } from '@/types/models/tenant';

// Mock the npcsService
jest.mock('@/services/api/npcs.service');
const mockNpcsService = npcsService as jest.Mocked<typeof npcsService>;

// Mock data
const mockTenant: Tenant = {
  id: 'tenant-1',
  type: 'tenants',
  attributes: {
    id: 'tenant-1',
    name: 'Test Tenant',
    region: 'GMS',
    majorVersion: 83,
    minorVersion: 1,
  }
};

const mockNPCs: NPC[] = [
  {
    id: 1,
    hasShop: true,
    hasConversation: false
  },
  {
    id: 2,
    hasShop: false,
    hasConversation: true
  },
  {
    id: 3,
    hasShop: true,
    hasConversation: true
  }
];

const mockShop: Shop = {
  id: 'shop-1',
  type: 'shops',
  attributes: {
    npcId: 1,
    recharger: false
  },
  relationships: {
    commodities: {
      data: [
        { type: 'commodities', id: 'commodity-1' }
      ]
    }
  }
};

const mockShopResponse: ShopResponse = {
  data: mockShop,
  included: []
};

const mockCommodity: Commodity = {
  id: 'commodity-1',
  type: 'commodities',
  attributes: {
    id: 'commodity-1',
    templateId: 1000,
    mesoPrice: 100,
    discountRate: 0,
    tokenTemplateId: 0,
    tokenPrice: 0,
    period: 0,
    levelLimit: 0
  }
};

const mockCommodityAttributes: CommodityAttributes = {
  id: 'commodity-new',
  templateId: 2000,
  mesoPrice: 200,
  discountRate: 10,
  tokenTemplateId: 0,
  tokenPrice: 0,
  period: 0,
  levelLimit: 0
};

// Test wrapper component
function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

  const TestWrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      {children}
    </QueryClientProvider>
  );
  TestWrapper.displayName = 'TestWrapper';
  return TestWrapper;
}

describe('useNpcs hooks', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  // ============================================================================
  // QUERY HOOKS TESTS
  // ============================================================================

  describe('useNPCs', () => {
    it('should fetch all NPCs successfully', async () => {
      mockNpcsService.getAllNPCs.mockResolvedValue(mockNPCs);

      const { result } = renderHook(() => useNPCs(mockTenant), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockNPCs);
      expect(mockNpcsService.getAllNPCs).toHaveBeenCalledWith(mockTenant, undefined);
    });

    it('should not fetch when tenant is not provided', () => {
      mockNpcsService.getAllNPCs.mockResolvedValue(mockNPCs);

      const { result } = renderHook(() => useNPCs(null as Tenant | null), {
        wrapper: createWrapper(),
      });

      expect(result.current.fetchStatus).toBe('idle');
      expect(mockNpcsService.getAllNPCs).not.toHaveBeenCalled();
    });

    it('should handle errors gracefully', async () => {
      const error = new Error('Failed to fetch NPCs');
      mockNpcsService.getAllNPCs.mockRejectedValue(error);

      const { result } = renderHook(() => useNPCs(mockTenant), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isError).toBe(true);
      });

      expect(result.current.error).toEqual(error);
    });
  });

  describe('useNPC', () => {
    it('should fetch specific NPC successfully', async () => {
      mockNpcsService.getNPCById.mockResolvedValue(mockNPCs[0]);

      const { result } = renderHook(() => useNPC(mockTenant, 1), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockNPCs[0]);
      expect(mockNpcsService.getNPCById).toHaveBeenCalledWith(1, mockTenant, undefined);
    });

    it('should return null when NPC is not found', async () => {
      mockNpcsService.getNPCById.mockResolvedValue(null);

      const { result } = renderHook(() => useNPC(mockTenant, 999), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toBeNull();
    });
  });

  describe('useNPCsWithShops', () => {
    it('should fetch NPCs with shops successfully', async () => {
      const npcsWithShops = mockNPCs.filter(npc => npc.hasShop);
      mockNpcsService.getNPCsWithShops.mockResolvedValue(npcsWithShops);

      const { result } = renderHook(() => useNPCsWithShops(mockTenant), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(npcsWithShops);
      expect(mockNpcsService.getNPCsWithShops).toHaveBeenCalledWith(mockTenant, undefined);
    });
  });

  describe('useNPCsWithConversations', () => {
    it('should fetch NPCs with conversations successfully', async () => {
      const npcsWithConversations = mockNPCs.filter(npc => npc.hasConversation);
      mockNpcsService.getNPCsWithConversations.mockResolvedValue(npcsWithConversations);

      const { result } = renderHook(() => useNPCsWithConversations(mockTenant), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(npcsWithConversations);
      expect(mockNpcsService.getNPCsWithConversations).toHaveBeenCalledWith(mockTenant, undefined);
    });
  });

  describe('useNPCShop', () => {
    it('should fetch NPC shop successfully', async () => {
      mockNpcsService.getNPCShop.mockResolvedValue(mockShopResponse);

      const { result } = renderHook(() => useNPCShop(mockTenant, 1), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockShopResponse);
      expect(mockNpcsService.getNPCShop).toHaveBeenCalledWith(1, mockTenant, undefined);
    });
  });

  // ============================================================================
  // MUTATION HOOKS TESTS
  // ============================================================================

  describe('useCreateShop', () => {
    it('should create shop successfully', async () => {
      mockNpcsService.createShop.mockResolvedValue(mockShop);

      const { result } = renderHook(() => useCreateShop(), {
        wrapper: createWrapper(),
      });

      const commodities = [mockCommodityAttributes];
      
      result.current.mutate({
        npcId: 1,
        commodities,
        tenant: mockTenant,
        recharger: false
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockShop);
      expect(mockNpcsService.createShop).toHaveBeenCalledWith(1, commodities, mockTenant, false, undefined);
    });

    it('should handle create shop errors', async () => {
      const error = new Error('Failed to create shop');
      mockNpcsService.createShop.mockRejectedValue(error);

      const { result } = renderHook(() => useCreateShop(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({
        npcId: 1,
        commodities: [mockCommodityAttributes],
        tenant: mockTenant
      });

      await waitFor(() => {
        expect(result.current.isError).toBe(true);
      });

      expect(result.current.error).toEqual(error);
    });
  });

  describe('useUpdateShop', () => {
    it('should update shop successfully', async () => {
      mockNpcsService.updateShop.mockResolvedValue(mockShop);

      const { result } = renderHook(() => useUpdateShop(), {
        wrapper: createWrapper(),
      });

      const commodities = [mockCommodity];
      
      result.current.mutate({
        npcId: 1,
        commodities,
        tenant: mockTenant,
        recharger: true
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockShop);
      expect(mockNpcsService.updateShop).toHaveBeenCalledWith(1, commodities, mockTenant, true, undefined);
    });
  });

  describe('useCreateCommodity', () => {
    it('should create commodity successfully', async () => {
      mockNpcsService.createCommodity.mockResolvedValue(mockCommodity);

      const { result } = renderHook(() => useCreateCommodity(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({
        npcId: 1,
        commodityAttributes: mockCommodityAttributes,
        tenant: mockTenant
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockCommodity);
      expect(mockNpcsService.createCommodity).toHaveBeenCalledWith(1, mockCommodityAttributes, mockTenant, undefined);
    });
  });

  describe('useUpdateCommodity', () => {
    it('should update commodity successfully', async () => {
      mockNpcsService.updateCommodity.mockResolvedValue(mockCommodity);

      const { result } = renderHook(() => useUpdateCommodity(), {
        wrapper: createWrapper(),
      });

      const updates = { mesoPrice: 150 };
      
      result.current.mutate({
        npcId: 1,
        commodityId: 'commodity-1',
        commodityAttributes: updates,
        tenant: mockTenant
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockCommodity);
      expect(mockNpcsService.updateCommodity).toHaveBeenCalledWith(1, 'commodity-1', updates, mockTenant, undefined);
    });
  });

  describe('useDeleteCommodity', () => {
    it('should delete commodity successfully', async () => {
      mockNpcsService.deleteCommodity.mockResolvedValue(undefined);

      const { result } = renderHook(() => useDeleteCommodity(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({
        npcId: 1,
        commodityId: 'commodity-1',
        tenant: mockTenant
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockNpcsService.deleteCommodity).toHaveBeenCalledWith(1, 'commodity-1', mockTenant, undefined);
    });
  });

  // ============================================================================
  // QUERY KEY TESTS
  // ============================================================================

  describe('npcKeys', () => {
    it('should generate correct query keys', () => {
      expect(npcKeys.all).toEqual(['npcs']);
      expect(npcKeys.lists()).toEqual(['npcs', 'list']);
      expect(npcKeys.list(mockTenant)).toEqual(['npcs', 'list', 'tenant-1', undefined]);
      expect(npcKeys.details()).toEqual(['npcs', 'detail']);
      expect(npcKeys.detail(mockTenant, 1)).toEqual(['npcs', 'detail', 'tenant-1', 1]);
      expect(npcKeys.shops()).toEqual(['npcs', 'shops']);
      expect(npcKeys.shop(mockTenant, 1)).toEqual(['npcs', 'shops', 'tenant-1', 1]);
      expect(npcKeys.withShops(mockTenant)).toEqual(['npcs', 'withShops', 'tenant-1']);
      expect(npcKeys.withConversations(mockTenant)).toEqual(['npcs', 'withConversations', 'tenant-1']);
    });

    it('should handle null tenant in query keys', () => {
      expect(npcKeys.list(null)).toEqual(['npcs', 'list', 'no-tenant', undefined]);
      expect(npcKeys.detail(null, 1)).toEqual(['npcs', 'detail', 'no-tenant', 1]);
      expect(npcKeys.shop(null, 1)).toEqual(['npcs', 'shops', 'no-tenant', 1]);
    });
  });
});