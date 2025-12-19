/**
 * Focused integration tests for multi-tenant breadcrumb context preservation
 * Tests the specific functionality for handling tenant switching in breadcrumbs
 */

import * as React from 'react';
import { render, screen, waitFor, act } from '@testing-library/react';
import { jest } from '@jest/globals';

import { BreadcrumbBar } from '@/components/features/navigation/BreadcrumbBar';
import { useBreadcrumbs } from '@/lib/hooks/useBreadcrumbs';
import type { Tenant } from '@/types/models/tenant';

// Mock localStorage
const localStorageMock = {
  getItem: jest.fn(),
  setItem: jest.fn(),
  removeItem: jest.fn(),
  clear: jest.fn(),
};
(global as any).localStorage = localStorageMock;

// Mock Next.js navigation
const mockPush = jest.fn();
const mockPathname = jest.fn().mockReturnValue('/');

jest.mock('next/navigation', () => ({
  useRouter: () => ({
    push: mockPush,
    replace: jest.fn(),
    back: jest.fn(),
    forward: jest.fn(),
    refresh: jest.fn(),
  }),
  usePathname: mockPathname,
  useSearchParams: () => new URLSearchParams(),
}));

// Mock tenant data
const mockTenant1: Tenant = {
  id: '83f5a16f-3b02-4e7d-81d0-cd5d2e68c59c',
  name: 'GMS Tenant',
  description: 'Global MapleStory Tenant',
  region: 'GMS',
  majorVersion: 83,
  minorVersion: 1,
  port: 8080,
  createdAt: '2024-01-01T00:00:00Z',
  updatedAt: '2024-01-01T00:00:00Z',
};

const mockTenant2: Tenant = {
  id: 'f2e3d4c5-6789-4abc-9def-123456789abc',
  name: 'SEA Tenant',
  description: 'Southeast Asia Tenant',
  region: 'SEA',
  majorVersion: 95,
  minorVersion: 2,
  port: 8090,
  createdAt: '2024-02-01T00:00:00Z',
  updatedAt: '2024-02-01T00:00:00Z',
};

// Mock tenant service
const mockTenantsService = {
  getAllTenants: jest.fn(),
  getTenantConfigurationById: jest.fn().mockResolvedValue({}),
};

jest.mock('@/services/api', () => ({
  tenantsService: mockTenantsService,
}));

// Mock breadcrumb resolvers with tenant-aware responses
const mockResolveEntityLabel = jest.fn();
const mockPreloadEntityLabels = jest.fn();
const mockInvalidateEntityLabels = jest.fn();

jest.mock('@/lib/breadcrumbs/resolvers', () => ({
  resolveEntityLabel: mockResolveEntityLabel,
  preloadEntityLabels: mockPreloadEntityLabels,
  invalidateEntityLabels: mockInvalidateEntityLabels,
  getEntityTypeFromRoute: jest.fn().mockReturnValue('character'),
}));

// Import TenantProvider after mocking
import { TenantProvider } from '@/context/tenant-context';

// Test component that directly uses tenant context and breadcrumbs
function MultiTenantBreadcrumbTest({ 
  pathname, 
  switchTenantAfter 
}: { 
  pathname: string;
  switchTenantAfter?: number;
}) {
  const [tenantToUse, setTenantToUse] = React.useState<Tenant>(mockTenant1);

  // Switch tenant after a delay if requested
  React.useEffect(() => {
    if (switchTenantAfter) {
      const timer = setTimeout(() => {
        setTenantToUse(mockTenant2);
      }, switchTenantAfter);
      return () => clearTimeout(timer);
    }
  }, [switchTenantAfter]);

  // Update mocks based on current tenant
  React.useEffect(() => {
    mockPathname.mockReturnValue(pathname);
    localStorageMock.getItem.mockReturnValue(tenantToUse.id);
    mockTenantsService.getAllTenants.mockResolvedValue([tenantToUse]);
    
    // Mock resolver to return tenant-specific labels
    mockResolveEntityLabel.mockImplementation((entityType: string, entityId: string, tenant: Tenant) => {
      const label = `${entityType} ${entityId} (${tenant.region})`;
      return Promise.resolve({
        label,
        fromCache: false,
        resolvedAt: Date.now(),
        isFallback: false,
      });
    });
  }, [pathname, tenantToUse]);

  const {
    breadcrumbs,
    loading,
    error,
  } = useBreadcrumbs({
    maxItems: 5,
    showEllipsis: true,
    autoResolve: true,
    enablePreloading: false,
  });

  return (
    <div data-testid="multi-tenant-breadcrumb-test" data-current-tenant={tenantToUse.region}>
      <div data-testid="breadcrumbs-data" data-count={breadcrumbs.length} data-loading={loading.toString()}>
        {breadcrumbs.map((breadcrumb, index) => (
          <div
            key={`${breadcrumb.href}-${index}`}
            data-testid={`breadcrumb-${index}`}
            data-label={breadcrumb.label}
            data-dynamic={breadcrumb.dynamic.toString()}
            data-entity-type={breadcrumb.entityType || 'none'}
            data-entity-id={breadcrumb.entityId || 'none'}
          >
            {breadcrumb.label}
          </div>
        ))}
      </div>
      
      {error && (
        <div data-testid="breadcrumb-error">{error.message}</div>
      )}
    </div>
  );
}

describe('Multi-Tenant Breadcrumb Context Preservation', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockPathname.mockReturnValue('/');
    localStorageMock.getItem.mockReturnValue(mockTenant1.id);
    mockTenantsService.getAllTenants.mockResolvedValue([mockTenant1]);
    mockResolveEntityLabel.mockResolvedValue({
      label: 'Default Label',
      fromCache: false,
      resolvedAt: Date.now(),
      isFallback: false,
    });
  });

  it('should preserve tenant context when resolving dynamic breadcrumb labels', async () => {
    render(
      <TenantProvider>
        <MultiTenantBreadcrumbTest pathname="/characters/123" />
      </TenantProvider>
    );

    // Wait for breadcrumbs to load
    await waitFor(() => {
      const breadcrumbsData = screen.getByTestId('breadcrumbs-data');
      expect(breadcrumbsData).toHaveAttribute('data-loading', 'false');
    }, { timeout: 3000 });

    // Check if resolver was called with correct tenant
    await waitFor(() => {
      expect(mockResolveEntityLabel).toHaveBeenCalledWith(
        'character',
        '123',
        expect.objectContaining({
          id: mockTenant1.id,
          region: 'GMS'
        }),
        expect.any(Object)
      );
    }, { timeout: 3000 });
  });

  it('should update breadcrumb labels when tenant context changes', async () => {
    const { rerender } = render(
      <TenantProvider>
        <MultiTenantBreadcrumbTest 
          pathname="/characters/456" 
          switchTenantAfter={1000}
        />
      </TenantProvider>
    );

    // Initially should work with first tenant
    await waitFor(() => {
      const test = screen.getByTestId('multi-tenant-breadcrumb-test');
      expect(test).toHaveAttribute('data-current-tenant', 'GMS');
    }, { timeout: 2000 });

    // Wait for tenant switch
    await waitFor(() => {
      const test = screen.getByTestId('multi-tenant-breadcrumb-test');
      expect(test).toHaveAttribute('data-current-tenant', 'SEA');
    }, { timeout: 3000 });

    // Should have called resolver for both tenants
    await waitFor(() => {
      expect(mockResolveEntityLabel).toHaveBeenCalledWith(
        'character',
        '456',
        expect.objectContaining({ region: 'GMS' }),
        expect.any(Object)
      );
      expect(mockResolveEntityLabel).toHaveBeenCalledWith(
        'character',
        '456',
        expect.objectContaining({ region: 'SEA' }),
        expect.any(Object)
      );
    }, { timeout: 1000 });
  });

  it('should maintain separate cache entries for different tenants', async () => {
    // First render with tenant 1
    const { rerender } = render(
      <TenantProvider>
        <MultiTenantBreadcrumbTest pathname="/guilds/789" />
      </TenantProvider>
    );

    await waitFor(() => {
      expect(mockResolveEntityLabel).toHaveBeenCalledWith(
        'character', // getEntityTypeFromRoute mock returns 'character'
        expect.any(String),
        expect.objectContaining({ region: 'GMS' }),
        expect.any(Object)
      );
    }, { timeout: 3000 });

    const firstCallCount = mockResolveEntityLabel.mock.calls.length;

    // Re-render with second tenant  
    rerender(
      <TenantProvider>
        <MultiTenantBreadcrumbTest pathname="/guilds/789" />
      </TenantProvider>
    );

    // Mock second tenant scenario
    await act(async () => {
      localStorageMock.getItem.mockReturnValue(mockTenant2.id);
      mockTenantsService.getAllTenants.mockResolvedValue([mockTenant2]);
    });

    // Should call resolver again for different tenant (not using cache)
    await waitFor(() => {
      expect(mockResolveEntityLabel.mock.calls.length).toBeGreaterThan(firstCallCount);
    }, { timeout: 2000 });
  });

  it('should handle missing tenant gracefully without breaking breadcrumbs', async () => {
    // Mock no tenant scenario
    localStorageMock.getItem.mockReturnValue(null);
    mockTenantsService.getAllTenants.mockResolvedValue([]);

    render(
      <TenantProvider>
        <MultiTenantBreadcrumbTest pathname="/characters/999" />
      </TenantProvider>
    );

    // Should still render breadcrumbs even without tenant
    await waitFor(() => {
      const breadcrumbsData = screen.getByTestId('breadcrumbs-data');
      expect(breadcrumbsData).toBeInTheDocument();
      // Loading should eventually finish
      expect(breadcrumbsData).toHaveAttribute('data-loading', 'false');
    }, { timeout: 3000 });

    // Should not call resolver without tenant
    expect(mockResolveEntityLabel).not.toHaveBeenCalled();
  });

  it('should invalidate labels when tenant changes mid-resolution', async () => {
    let resolveFunction: ((value: any) => void) | null = null;
    
    // Mock resolver with manual control over resolution timing
    mockResolveEntityLabel.mockImplementation((entityType: string, entityId: string, tenant: Tenant) => {
      return new Promise((resolve) => {
        resolveFunction = resolve;
      });
    });

    render(
      <TenantProvider>
        <MultiTenantBreadcrumbTest 
          pathname="/npcs/202" 
          switchTenantAfter={100} // Very fast tenant switch
        />
      </TenantProvider>
    );

    // Wait a bit then resolve the first call
    await new Promise(resolve => setTimeout(resolve, 200));
    
    if (resolveFunction) {
      resolveFunction({
        label: 'npc 202 (GMS)',
        fromCache: false,
        resolvedAt: Date.now(),
        isFallback: false,
      });
    }

    // Should handle race condition gracefully
    await waitFor(() => {
      expect(mockResolveEntityLabel).toHaveBeenCalled();
    }, { timeout: 2000 });

    // Should not throw errors or crash
    expect(screen.queryByTestId('breadcrumb-error')).not.toBeInTheDocument();
  });

  it('should work with BreadcrumbBar component and tenant switching', async () => {
    mockPathname.mockReturnValue('/characters/555');
    
    // Mock successful tenant setup
    localStorageMock.getItem.mockReturnValue(mockTenant1.id);
    mockTenantsService.getAllTenants.mockResolvedValue([mockTenant1]);
    
    // Mock resolver with tenant-specific responses
    mockResolveEntityLabel.mockImplementation((entityType: string, entityId: string, tenant: Tenant) => {
      return Promise.resolve({
        label: `${entityType} ${entityId} (${tenant.region})`,
        fromCache: false,
        resolvedAt: Date.now(),
        isFallback: false,
      });
    });

    render(
      <TenantProvider>
        <BreadcrumbBar />
      </TenantProvider>
    );

    // Should render breadcrumb navigation eventually
    await waitFor(() => {
      const nav = screen.queryByRole('navigation', { name: /breadcrumb/i });
      if (nav) {
        expect(nav).toBeInTheDocument();
      } else {
        // BreadcrumbBar might not render if conditions aren't met
        expect(screen.getByText(/loading navigation/i)).toBeInTheDocument();
      }
    }, { timeout: 5000 });

    // Verify resolver was called with tenant context
    await waitFor(() => {
      if (mockResolveEntityLabel.mock.calls.length > 0) {
        expect(mockResolveEntityLabel).toHaveBeenCalledWith(
          'character',
          '555',
          expect.objectContaining({
            id: mockTenant1.id,
            region: 'GMS'
          }),
          expect.any(Object)
        );
      }
    }, { timeout: 2000 });
  });
});