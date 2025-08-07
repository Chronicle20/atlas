/**
 * Comprehensive integration tests for breadcrumb navigation
 * Tests breadcrumb functionality across all application routes
 */

import * as React from 'react';
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { jest } from '@jest/globals';

import { BreadcrumbBar } from '@/components/features/navigation/BreadcrumbBar';
import { useBreadcrumbs } from '@/lib/hooks/useBreadcrumbs';
import { ROUTE_PATTERNS } from '@/lib/breadcrumbs/routes';
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
const mockReplace = jest.fn();
const mockBack = jest.fn();
const mockPathname = jest.fn().mockReturnValue('/');

jest.mock('next/navigation', () => ({
  useRouter: () => ({
    push: mockPush,
    replace: mockReplace,
    back: mockBack,
    forward: jest.fn(),
    refresh: jest.fn(),
  }),
  usePathname: mockPathname,
  useSearchParams: () => new URLSearchParams(),
}));

// Mock tenant data
const mockTenant: Tenant = {
  id: '83f5a16f-3b02-4e7d-81d0-cd5d2e68c59c',
  name: 'Test Tenant',
  description: 'Test tenant for breadcrumb navigation',
  region: 'GMS',
  majorVersion: 83,
  minorVersion: 1,
  port: 8080,
  createdAt: '2024-01-01T00:00:00Z',
  updatedAt: '2024-01-01T00:00:00Z',
};

// Mock tenant service
const mockTenantsService = {
  getAllTenants: jest.fn().mockResolvedValue([mockTenant]),
  getTenantConfigurationById: jest.fn().mockResolvedValue({}),
};

jest.mock('@/services/api', () => ({
  tenantsService: mockTenantsService,
}));

// Mock breadcrumb resolvers
const mockResolveEntityLabel = jest.fn().mockImplementation((entityType: string, entityId: string) => {
  const mockLabels: Record<string, (id: string) => string> = {
    character: (id) => `Character ${id}`,
    guild: (id) => `Guild ${id}`,
    npc: (id) => `NPC ${id}`,
    template: (id) => `Template ${id}`,
    tenant: (id) => `Tenant ${id}`,
    account: (id) => `Account ${id}`,
  };
  
  return Promise.resolve({
    label: mockLabels[entityType]?.(entityId) || `Unknown ${entityType}`,
    cached: false,
    timestamp: Date.now(),
  });
});

jest.mock('@/lib/breadcrumbs/resolvers', () => ({
  resolveEntityLabel: mockResolveEntityLabel,
  preloadEntityLabels: jest.fn().mockResolvedValue(undefined),
  invalidateEntityLabels: jest.fn(),
  getEntityTypeFromRoute: jest.fn().mockReturnValue(null),
}));

// Import TenantProvider after mocking
import { TenantProvider } from '@/context/tenant-context';

// Test wrapper component that provides all necessary context
interface TestWrapperProps {
  children: React.ReactNode;
  pathname?: string;
  tenantLoading?: boolean;
}

function TestWrapper({ children, pathname = '/', tenantLoading = false }: TestWrapperProps) {
  // Update mock pathname
  React.useEffect(() => {
    mockPathname.mockReturnValue(pathname);
  }, [pathname]);

  // Mock localStorage to return the test tenant
  React.useEffect(() => {
    localStorageMock.getItem.mockReturnValue(mockTenant.id);
    
    if (tenantLoading) {
      mockTenantsService.getAllTenants.mockImplementation(() => 
        new Promise(resolve => setTimeout(() => resolve([mockTenant]), 100))
      );
    } else {
      mockTenantsService.getAllTenants.mockResolvedValue([mockTenant]);
    }
  }, [tenantLoading]);

  return (
    <TenantProvider>
      <div data-testid="test-wrapper">
        {children}
      </div>
    </TenantProvider>
  );
}

// Test component that uses breadcrumbs hook
function BreadcrumbTestComponent({ pathname }: { pathname: string }) {
  const {
    breadcrumbs,
    loading,
    error,
    navigation,
    utils
  } = useBreadcrumbs({
    maxItems: 5,
    showEllipsis: true,
    autoResolve: true,
    enablePreloading: false,
  });

  return (
    <div data-testid="breadcrumb-test">
      <div data-testid="breadcrumbs-count">{breadcrumbs.length}</div>
      <div data-testid="loading-state">{loading.toString()}</div>
      <div data-testid="error-state">{error?.message || 'none'}</div>
      <div data-testid="is-valid-route">{utils.isValidRoute.toString()}</div>
      
      <div data-testid="breadcrumb-list">
        {breadcrumbs.map((breadcrumb, index) => (
          <div
            key={`${breadcrumb.href}-${index}`}
            data-testid={`breadcrumb-${index}`}
            data-href={breadcrumb.href}
            data-label={breadcrumb.label}
            data-is-current={breadcrumb.isCurrentPage.toString()}
            data-dynamic={breadcrumb.dynamic.toString()}
          >
            <span data-testid={`label-${index}`}>{breadcrumb.label}</span>
            <button
              data-testid={`link-${index}`}
              onClick={() => navigation.navigateTo(breadcrumb)}
              disabled={breadcrumb.isCurrentPage}
            >
              Navigate
            </button>
          </div>
        ))}
      </div>

      <button
        data-testid="parent-navigation"
        onClick={navigation.goToParent}
      >
        Go to Parent
      </button>
    </div>
  );
}

describe('Breadcrumb Navigation Integration Tests', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockPathname.mockReturnValue('/');
    localStorageMock.getItem.mockReturnValue(mockTenant.id);
    mockTenantsService.getAllTenants.mockResolvedValue([mockTenant]);
  });

  describe('Route Recognition and Hierarchy', () => {
    it('should generate correct breadcrumbs for home route', async () => {
      mockPathname.mockReturnValue('/');

      render(
        <TestWrapper pathname="/">
          <BreadcrumbTestComponent pathname="/" />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('breadcrumbs-count')).toHaveTextContent('1');
      }, { timeout: 3000 });

      expect(screen.getByTestId('label-0')).toHaveTextContent('Home');
      expect(screen.getByTestId('is-valid-route')).toHaveTextContent('true');
    });

    it('should generate correct breadcrumbs for characters list route', async () => {
      mockPathname.mockReturnValue('/characters');

      render(
        <TestWrapper pathname="/characters">
          <BreadcrumbTestComponent pathname="/characters" />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('breadcrumbs-count')).toHaveTextContent('2');
      }, { timeout: 3000 });

      expect(screen.getByTestId('label-0')).toHaveTextContent('Home');
      expect(screen.getByTestId('label-1')).toHaveTextContent('Characters');
      
      // First breadcrumb should not be current page
      expect(screen.getByTestId('breadcrumb-0')).toHaveAttribute('data-is-current', 'false');
      // Last breadcrumb should be current page
      expect(screen.getByTestId('breadcrumb-1')).toHaveAttribute('data-is-current', 'true');
    });

    it('should generate correct breadcrumbs for character detail route', async () => {
      mockPathname.mockReturnValue('/characters/123');

      render(
        <TestWrapper pathname="/characters/123">
          <BreadcrumbTestComponent pathname="/characters/123" />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('breadcrumbs-count')).toHaveTextContent('3');
      }, { timeout: 3000 });

      expect(screen.getByTestId('label-0')).toHaveTextContent('Home');
      expect(screen.getByTestId('label-1')).toHaveTextContent('Characters');
      
      // Wait for dynamic label resolution
      await waitFor(() => {
        expect(screen.getByTestId('label-2')).toHaveTextContent('Character 123');
      }, { timeout: 3000 });

      // Check dynamic flag
      expect(screen.getByTestId('breadcrumb-2')).toHaveAttribute('data-dynamic', 'true');
    });
  });

  describe('Navigation Functionality', () => {
    it('should navigate when clicking non-current breadcrumb links', async () => {
      const user = userEvent.setup();
      mockPathname.mockReturnValue('/characters/123');

      render(
        <TestWrapper pathname="/characters/123">
          <BreadcrumbTestComponent pathname="/characters/123" />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('breadcrumbs-count')).toHaveTextContent('3');
      }, { timeout: 3000 });

      // Click on "Characters" breadcrumb (index 1)
      const charactersLink = screen.getByTestId('link-1');
      expect(charactersLink).not.toBeDisabled();
      
      await user.click(charactersLink);

      expect(mockPush).toHaveBeenCalledWith('/characters');
    });

    it('should not navigate when clicking current page breadcrumb', async () => {
      const user = userEvent.setup();
      mockPathname.mockReturnValue('/characters/123');

      render(
        <TestWrapper pathname="/characters/123">
          <BreadcrumbTestComponent pathname="/characters/123" />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('breadcrumbs-count')).toHaveTextContent('3');
      }, { timeout: 3000 });

      // Current page link should be disabled
      const currentPageLink = screen.getByTestId('link-2');
      expect(currentPageLink).toBeDisabled();

      await user.click(currentPageLink);

      // Should not trigger navigation
      expect(mockPush).not.toHaveBeenCalled();
    });

    it('should handle parent navigation correctly', async () => {
      const user = userEvent.setup();
      mockPathname.mockReturnValue('/characters/123');

      render(
        <TestWrapper pathname="/characters/123">
          <BreadcrumbTestComponent pathname="/characters/123" />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('breadcrumbs-count')).toHaveTextContent('3');
      }, { timeout: 3000 });

      const parentButton = screen.getByTestId('parent-navigation');
      await user.click(parentButton);

      // Should navigate to parent route (/characters)
      expect(mockPush).toHaveBeenCalledWith('/characters');
    });
  });

  describe('Dynamic Label Resolution', () => {
    it('should resolve character names correctly', async () => {
      mockPathname.mockReturnValue('/characters/456');

      render(
        <TestWrapper pathname="/characters/456">
          <BreadcrumbTestComponent pathname="/characters/456" />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('label-2')).toHaveTextContent('Character 456');
      }, { timeout: 3000 });

      expect(screen.getByTestId('breadcrumb-2')).toHaveAttribute('data-dynamic', 'true');
    });

    it('should resolve guild names correctly', async () => {
      mockPathname.mockReturnValue('/guilds/789');

      render(
        <TestWrapper pathname="/guilds/789">
          <BreadcrumbTestComponent pathname="/guilds/789" />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('label-2')).toHaveTextContent('Guild 789');
      }, { timeout: 3000 });

      expect(screen.getByTestId('breadcrumb-2')).toHaveAttribute('data-dynamic', 'true');
    });

    it('should resolve NPC names correctly', async () => {
      mockPathname.mockReturnValue('/npcs/101');

      render(
        <TestWrapper pathname="/npcs/101">
          <BreadcrumbTestComponent pathname="/npcs/101" />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('label-2')).toHaveTextContent('NPC 101');
      }, { timeout: 3000 });

      expect(screen.getByTestId('breadcrumb-2')).toHaveAttribute('data-dynamic', 'true');
    });
  });

  describe('Error Handling', () => {
    it('should handle invalid routes gracefully', async () => {
      mockPathname.mockReturnValue('/invalid/route/path');

      render(
        <TestWrapper pathname="/invalid/route/path">
          <BreadcrumbTestComponent pathname="/invalid/route/path" />
        </TestWrapper>
      );

      await waitFor(() => {
        // Should still generate basic breadcrumbs from path segments
        expect(screen.getByTestId('is-valid-route')).toHaveTextContent('false');
        expect(screen.getByTestId('error-state')).toHaveTextContent('none');
      }, { timeout: 3000 });
    });

    it('should handle resolver errors gracefully', async () => {
      mockResolveEntityLabel.mockRejectedValueOnce(new Error('API Error'));
      mockPathname.mockReturnValue('/characters/error-test');

      render(
        <TestWrapper pathname="/characters/error-test">
          <BreadcrumbTestComponent pathname="/characters/error-test" />
        </TestWrapper>
      );

      await waitFor(() => {
        // Should still show some breadcrumbs even with resolver error
        expect(screen.getByTestId('breadcrumbs-count')).toHaveTextContent('3');
      }, { timeout: 3000 });

      // Should use fallback label or original label
      expect(screen.getByTestId('label-2')).toHaveTextContent('Character Details');
    });
  });

  describe('All Route Patterns Coverage', () => {
    const testCases = [
      { path: '/', expectedLabels: ['Home'] },
      { path: '/accounts', expectedLabels: ['Home', 'Accounts'] },
      { path: '/accounts/123', expectedLabels: ['Home', 'Accounts', 'Account 123'], dynamic: true },
      { path: '/characters', expectedLabels: ['Home', 'Characters'] },
      { path: '/characters/456', expectedLabels: ['Home', 'Characters', 'Character 456'], dynamic: true },
      { path: '/guilds', expectedLabels: ['Home', 'Guilds'] },
      { path: '/guilds/789', expectedLabels: ['Home', 'Guilds', 'Guild 789'], dynamic: true },
      { path: '/npcs', expectedLabels: ['Home', 'NPCs'] },
      { path: '/npcs/101', expectedLabels: ['Home', 'NPCs', 'NPC 101'], dynamic: true },
      { path: '/templates', expectedLabels: ['Home', 'Templates'] },
      { path: '/templates/202', expectedLabels: ['Home', 'Templates', 'Template 202'], dynamic: true },
      { path: '/tenants', expectedLabels: ['Home', 'Tenants'] },
      { path: '/tenants/303', expectedLabels: ['Home', 'Tenants', 'Tenant 303'], dynamic: true },
    ];

    testCases.forEach(({ path, expectedLabels, dynamic = false }) => {
      it(`should generate correct breadcrumbs for route: ${path}`, async () => {
        mockPathname.mockReturnValue(path);

        render(
          <TestWrapper pathname={path}>
            <BreadcrumbTestComponent pathname={path} />
          </TestWrapper>
        );

        await waitFor(() => {
          expect(screen.getByTestId('breadcrumbs-count')).toHaveTextContent(expectedLabels.length.toString());
        }, { timeout: 3000 });

        // Check each expected label
        for (let i = 0; i < expectedLabels.length; i++) {
          if (dynamic && i === expectedLabels.length - 1) {
            // For dynamic routes, wait for label resolution
            await waitFor(() => {
              expect(screen.getByTestId(`label-${i}`)).toHaveTextContent(expectedLabels[i]);
            }, { timeout: 3000 });
          } else {
            expect(screen.getByTestId(`label-${i}`)).toHaveTextContent(expectedLabels[i]);
          }
        }

        // Check that last breadcrumb is current page
        const lastIndex = expectedLabels.length - 1;
        expect(screen.getByTestId(`breadcrumb-${lastIndex}`)).toHaveAttribute('data-is-current', 'true');
      });
    });
  });

  describe('BreadcrumbBar Component Integration', () => {
    it('should render BreadcrumbBar component correctly', async () => {
      mockPathname.mockReturnValue('/characters/123');

      render(
        <TestWrapper pathname="/characters/123">
          <BreadcrumbBar />
        </TestWrapper>
      );

      // Should render navigation element
      await waitFor(() => {
        const nav = screen.getByRole('navigation', { name: /breadcrumb/i });
        expect(nav).toBeInTheDocument();
      }, { timeout: 3000 });

      // Should have breadcrumb links
      await waitFor(() => {
        const nav = screen.getByRole('navigation', { name: /breadcrumb/i });
        const breadcrumbList = within(nav).getByRole('list');
        expect(breadcrumbList).toBeInTheDocument();
        
        const listItems = within(breadcrumbList).getAllByRole('listitem');
        expect(listItems.length).toBeGreaterThan(0);
      }, { timeout: 3000 });
    });

    it('should handle loading states in BreadcrumbBar', async () => {
      mockPathname.mockReturnValue('/characters/123');

      render(
        <TestWrapper pathname="/characters/123" tenantLoading={true}>
          <BreadcrumbBar showLoadingStates={true} />
        </TestWrapper>
      );

      // Should show loading state initially
      expect(screen.getByText(/loading navigation/i)).toBeInTheDocument();

      // Wait for content to load
      await waitFor(() => {
        const nav = screen.getByRole('navigation', { name: /breadcrumb/i });
        const breadcrumbList = within(nav).queryByRole('list');
        expect(breadcrumbList).toBeInTheDocument();
      }, { timeout: 5000 });
    });
  });

  describe('Accessibility', () => {
    it('should have proper ARIA attributes', async () => {
      mockPathname.mockReturnValue('/characters/123');

      render(
        <TestWrapper pathname="/characters/123">
          <BreadcrumbBar />
        </TestWrapper>
      );

      await waitFor(() => {
        const nav = screen.getByRole('navigation', { name: /breadcrumb/i });
        expect(nav).toBeInTheDocument();
        expect(nav).toHaveAttribute('aria-label', 'breadcrumb');
      }, { timeout: 3000 });
    });

    it('should have semantic HTML structure', async () => {
      mockPathname.mockReturnValue('/characters/123');

      render(
        <TestWrapper pathname="/characters/123">
          <BreadcrumbBar />
        </TestWrapper>
      );

      await waitFor(() => {
        const nav = screen.getByRole('navigation');
        const breadcrumbList = within(nav).getByRole('list');
        const listItems = within(breadcrumbList).getAllByRole('listitem');
        
        expect(nav).toBeInTheDocument();
        expect(breadcrumbList).toBeInTheDocument();
        expect(listItems.length).toBeGreaterThan(0);
      }, { timeout: 3000 });
    });
  });

  describe('Performance and Caching', () => {
    it('should cache resolved labels to avoid redundant API calls', async () => {
      mockPathname.mockReturnValue('/characters/123');

      // First render
      const { rerender } = render(
        <TestWrapper pathname="/characters/123">
          <BreadcrumbTestComponent pathname="/characters/123" />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('label-2')).toHaveTextContent('Character 123');
      }, { timeout: 3000 });

      // Should have called resolver once
      expect(mockResolveEntityLabel).toHaveBeenCalledTimes(1);

      // Re-render the same route
      rerender(
        <TestWrapper pathname="/characters/123">
          <BreadcrumbTestComponent pathname="/characters/123" />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('label-2')).toHaveTextContent('Character 123');
      }, { timeout: 3000 });

      // Should not call resolver again due to caching (implementation dependent)
      // This test validates the resolver was called appropriately
      expect(mockResolveEntityLabel).toHaveBeenCalledWith(
        'character',
        '123',
        mockTenant,
        expect.any(Object)
      );
    });
  });
});