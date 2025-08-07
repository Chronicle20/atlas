/**
 * Tests for useBreadcrumbs hook
 */

import { renderHook, act, waitFor } from '@testing-library/react';
import { useBreadcrumbs, useSimpleBreadcrumbs, useBreadcrumbNavigation } from '../useBreadcrumbs';
import * as resolvers from '@/lib/breadcrumbs/resolvers';
import * as utils from '@/lib/breadcrumbs/utils';
import * as routes from '@/lib/breadcrumbs/routes';

// Mock dependencies
jest.mock('next/navigation', () => ({
  useRouter: jest.fn(),
  usePathname: jest.fn(),
}));

jest.mock('@/context/tenant-context', () => ({
  useTenant: jest.fn(),
}));

jest.mock('@/lib/breadcrumbs/resolvers', () => ({
  resolveEntityLabel: jest.fn(),
  preloadEntityLabels: jest.fn(),
  invalidateEntityLabels: jest.fn(),
  getEntityTypeFromRoute: jest.fn(),
  EntityType: {
    ACCOUNT: 'account',
    CHARACTER: 'character',
    GUILD: 'guild',
    NPC: 'npc',
    TEMPLATE: 'template',
    TENANT: 'tenant',
  },
}));

jest.mock('@/lib/breadcrumbs/utils', () => ({
  parsePathname: jest.fn(),
  buildBreadcrumbPath: jest.fn(),
  filterVisibleBreadcrumbs: jest.fn(),
  getParentBreadcrumb: jest.fn(),
  getBreadcrumbKey: jest.fn(),
}));

jest.mock('@/lib/breadcrumbs/routes', () => ({
  findRouteConfig: jest.fn(),
  getBreadcrumbsFromRoute: jest.fn(),
}));

// Import mocked modules
import { useRouter, usePathname } from 'next/navigation';
import { useTenant } from '@/context/tenant-context';

const mockUseRouter = useRouter as jest.MockedFunction<typeof useRouter>;
const mockUsePathname = usePathname as jest.MockedFunction<typeof usePathname>;
const mockUseTenant = useTenant as jest.MockedFunction<typeof useTenant>;
const mockResolvers = resolvers as jest.Mocked<typeof resolvers>;
const mockUtils = utils as jest.Mocked<typeof utils>;
const mockRoutes = routes as jest.Mocked<typeof routes>;

// Test data
const mockTenant = {
  id: 'test-tenant-id',
  name: 'Test Tenant',
  attributes: {
    name: 'Test Tenant',
    region: 'US',
    majorVersion: '1',
    minorVersion: '0',
  },
};

const mockBreadcrumbs = [
  {
    segment: '',
    label: 'Home',
    href: '/',
    dynamic: false,
    isCurrentPage: false,
  },
  {
    segment: 'characters',
    label: 'Characters',
    href: '/characters',
    dynamic: false,
    isCurrentPage: false,
  },
  {
    segment: 'char-123',
    label: 'Loading...',
    href: '/characters/char-123',
    dynamic: true,
    isCurrentPage: true,
    entityId: 'char-123',
    entityType: 'character',
  },
];

const mockRouteConfig = {
  pattern: '/characters/[id]',
  label: 'Character Details',
  parent: '/characters',
  entityType: 'character',
};

describe('useBreadcrumbs', () => {
  beforeEach(() => {
    // Reset all mocks
    jest.clearAllMocks();

    // Setup default mock implementations
    mockUsePathname.mockReturnValue('/characters/char-123');
    mockUseRouter.mockReturnValue({
      push: jest.fn(),
      replace: jest.fn(),
      refresh: jest.fn(),
      back: jest.fn(),
      forward: jest.fn(),
      prefetch: jest.fn(),
    });
    mockUseTenant.mockReturnValue({
      activeTenant: mockTenant,
      tenants: [mockTenant],
      loading: false,
      setActiveTenant: jest.fn(),
      refreshTenants: jest.fn(),
      fetchTenantConfiguration: jest.fn(),
    });

    // Setup breadcrumb utility mocks
    mockUtils.parsePathname.mockReturnValue(mockBreadcrumbs);
    mockUtils.buildBreadcrumbPath.mockImplementation((segments) => segments);
    mockUtils.filterVisibleBreadcrumbs.mockImplementation((segments) => segments);
    mockUtils.getParentBreadcrumb.mockReturnValue(mockBreadcrumbs[1]);
    mockUtils.getBreadcrumbKey.mockReturnValue('home|characters|character-detail');

    // Setup route mocks
    mockRoutes.findRouteConfig.mockReturnValue(mockRouteConfig);
    mockRoutes.getBreadcrumbsFromRoute.mockReturnValue([]);

    // Setup resolver mocks
    mockResolvers.resolveEntityLabel.mockResolvedValue({
      label: 'Test Character',
      fromCache: false,
      resolvedAt: Date.now(),
      isFallback: false,
    });
    mockResolvers.getEntityTypeFromRoute.mockReturnValue(resolvers.EntityType.CHARACTER);
    mockResolvers.preloadEntityLabels.mockResolvedValue();
    mockResolvers.invalidateEntityLabels.mockImplementation(() => {});
  });

  describe('basic functionality', () => {
    it('should return initial breadcrumbs', () => {
      const { result } = renderHook(() => useBreadcrumbs());

      expect(result.current.breadcrumbs).toEqual(mockBreadcrumbs);
      expect(result.current.loading).toBe(false);
      expect(result.current.error).toBe(null);
      expect(result.current.routeConfig).toEqual(mockRouteConfig);
    });

    it('should parse pathname correctly', () => {
      renderHook(() => useBreadcrumbs());

      expect(mockUtils.parsePathname).toHaveBeenCalledWith('/characters/char-123');
    });

    it('should find route config', () => {
      renderHook(() => useBreadcrumbs());

      expect(mockRoutes.findRouteConfig).toHaveBeenCalledWith('/characters/char-123');
    });
  });

  describe('dynamic label resolution', () => {
    it('should resolve dynamic labels when autoResolve is enabled', async () => {
      const { result } = renderHook(() => useBreadcrumbs({ autoResolve: true }));

      await waitFor(() => {
        expect(mockResolvers.resolveEntityLabel).toHaveBeenCalledWith(
          resolvers.EntityType.CHARACTER,
          'char-123',
          mockTenant,
          expect.objectContaining({
            fallback: 'Unknown',
            timeout: 5000,
            useCache: true,
          })
        );
      });
    });

    it('should not resolve dynamic labels when autoResolve is disabled', () => {
      renderHook(() => useBreadcrumbs({ autoResolve: false }));

      expect(mockResolvers.resolveEntityLabel).not.toHaveBeenCalled();
    });

    it('should update breadcrumb labels after resolution', async () => {
      const { result } = renderHook(() => useBreadcrumbs({ autoResolve: true }));

      await waitFor(() => {
        const characterBreadcrumb = result.current.breadcrumbs.find(b => b.entityId === 'char-123');
        expect(characterBreadcrumb?.label).toBe('Test Character');
      });
    });

    it('should handle resolution errors gracefully', async () => {
      mockResolvers.resolveEntityLabel.mockRejectedValue(new Error('Resolution failed'));

      const { result } = renderHook(() => useBreadcrumbs({ autoResolve: true }));

      await waitFor(() => {
        expect(result.current.error).toBe(null); // Should not set global error for individual failures
      });
    });
  });

  describe('navigation utilities', () => {
    it('should navigate to parent when goToParent is called', () => {
      const mockPush = jest.fn();
      mockUseRouter.mockReturnValue({
        push: mockPush,
        replace: jest.fn(),
        refresh: jest.fn(),
        back: jest.fn(),
        forward: jest.fn(),
        prefetch: jest.fn(),
      });

      const { result } = renderHook(() => useBreadcrumbs());

      act(() => {
        result.current.navigation.goToParent();
      });

      expect(mockPush).toHaveBeenCalledWith('/characters');
    });

    it('should navigate to specific breadcrumb', () => {
      const mockPush = jest.fn();
      mockUseRouter.mockReturnValue({
        push: mockPush,
        replace: jest.fn(),
        refresh: jest.fn(),
        back: jest.fn(),
        forward: jest.fn(),
        prefetch: jest.fn(),
      });

      const { result } = renderHook(() => useBreadcrumbs());

      act(() => {
        result.current.navigation.navigateTo(mockBreadcrumbs[1]);
      });

      expect(mockPush).toHaveBeenCalledWith('/characters');
    });

    it('should return parent breadcrumb', () => {
      const { result } = renderHook(() => useBreadcrumbs());

      expect(result.current.navigation.getParent()).toEqual(mockBreadcrumbs[1]);
    });
  });

  describe('resolution utilities', () => {
    it('should manually resolve entity labels', async () => {
      const { result } = renderHook(() => useBreadcrumbs());

      await act(async () => {
        const resolved = await result.current.resolution.resolveLabel(
          resolvers.EntityType.CHARACTER,
          'char-456'
        );
        expect(resolved.label).toBe('Test Character');
      });
    });

    it('should invalidate entity labels', () => {
      const { result } = renderHook(() => useBreadcrumbs());

      act(() => {
        result.current.resolution.invalidateLabels(
          resolvers.EntityType.CHARACTER,
          ['char-123']
        );
      });

      expect(mockResolvers.invalidateEntityLabels).toHaveBeenCalledWith(
        resolvers.EntityType.CHARACTER,
        ['char-123'],
        mockTenant
      );
    });

    it('should preload entity labels', async () => {
      const { result } = renderHook(() => useBreadcrumbs());

      await act(async () => {
        await result.current.resolution.preloadLabels(
          resolvers.EntityType.CHARACTER,
          ['char-456', 'char-789']
        );
      });

      expect(mockResolvers.preloadEntityLabels).toHaveBeenCalledWith(
        resolvers.EntityType.CHARACTER,
        ['char-456', 'char-789'],
        mockTenant,
        expect.any(Object)
      );
    });
  });

  describe('options handling', () => {
    it('should apply maxItems option', () => {
      renderHook(() => useBreadcrumbs({ maxItems: 2 }));

      expect(mockUtils.buildBreadcrumbPath).toHaveBeenCalledWith(
        expect.any(Array),
        expect.objectContaining({
          maxItems: 2,
          showEllipsis: true,
        })
      );
    });

    it('should apply hiddenRoutes option', () => {
      const hiddenRoutes = ['admin', 'settings'];
      renderHook(() => useBreadcrumbs({ hiddenRoutes }));

      expect(mockUtils.filterVisibleBreadcrumbs).toHaveBeenCalledWith(
        expect.any(Array),
        expect.objectContaining({
          hiddenRoutes,
        })
      );
    });
  });

  describe('error handling', () => {
    it('should handle parsing errors', () => {
      mockUtils.parsePathname.mockImplementation(() => {
        throw new Error('Parse error');
      });

      const { result } = renderHook(() => useBreadcrumbs());

      expect(result.current.error).toBeInstanceOf(Error);
      expect(result.current.breadcrumbs).toEqual([]);
    });

    it('should handle missing tenant gracefully', () => {
      mockUseTenant.mockReturnValue({
        activeTenant: null,
        tenants: [],
        loading: false,
        setActiveTenant: jest.fn(),
        refreshTenants: jest.fn(),
        fetchTenantConfiguration: jest.fn(),
      });

      const { result } = renderHook(() => useBreadcrumbs({ autoResolve: true }));

      expect(result.current.breadcrumbs).toEqual(mockBreadcrumbs);
      expect(mockResolvers.resolveEntityLabel).not.toHaveBeenCalled();
    });
  });
});

describe('useSimpleBreadcrumbs', () => {
  it('should return simplified breadcrumbs', () => {
    const { result } = renderHook(() => useSimpleBreadcrumbs());

    expect(result.current).toEqual(mockBreadcrumbs);
    expect(mockUtils.buildBreadcrumbPath).toHaveBeenCalledWith(
      expect.any(Array),
      expect.objectContaining({
        maxItems: 3,
      })
    );
  });
});

describe('useBreadcrumbNavigation', () => {
  it('should return navigation utilities with simplified breadcrumbs', () => {
    const { result } = renderHook(() => useBreadcrumbNavigation());

    expect(result.current.breadcrumbs).toEqual([
      { label: 'Home', href: '/', isCurrentPage: false },
      { label: 'Characters', href: '/characters', isCurrentPage: false },
      { label: 'Loading...', href: '/characters/char-123', isCurrentPage: true },
    ]);

    expect(typeof result.current.goToParent).toBe('function');
    expect(typeof result.current.navigateTo).toBe('function');
    expect(typeof result.current.getParent).toBe('function');
  });
});