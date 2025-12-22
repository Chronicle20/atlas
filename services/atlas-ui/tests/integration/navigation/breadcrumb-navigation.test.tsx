/**
 * Integration tests for breadcrumb navigation
 * Tests breadcrumb structure, hierarchy, and navigation across application routes
 *
 * Note: Dynamic label resolution tests are excluded as they require complex
 * async tenant context setup that's difficult to mock reliably in unit tests.
 * The dynamic resolution functionality is tested via e2e tests.
 */

import * as React from 'react';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
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

// Use the global configurable mock from jest.setup.js
function setMockPathname(pathname: string) {
  global.__mockNextNavigation.pathname = pathname;
}

const mockNavigationState = global.__mockNextNavigation;

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
  return Promise.resolve({
    label: `${entityType} ${entityId}`,
    fromCache: false,
    resolvedAt: Date.now(),
    isFallback: false,
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

// Test wrapper component
interface TestWrapperProps {
  children: React.ReactNode;
}

function TestWrapper({ children }: TestWrapperProps) {
  React.useEffect(() => {
    localStorageMock.getItem.mockReturnValue(mockTenant.id);
    mockTenantsService.getAllTenants.mockResolvedValue([mockTenant]);
  }, []);

  return (
    <TenantProvider>
      <div data-testid="test-wrapper">
        {children}
      </div>
    </TenantProvider>
  );
}
TestWrapper.displayName = 'TestWrapper';

// Test component that uses breadcrumbs hook
function BreadcrumbTestComponent() {
  const {
    breadcrumbs,
    loading,
    error,
    navigation,
    utils
  } = useBreadcrumbs({
    maxItems: 5,
    showEllipsis: true,
    autoResolve: false, // Disable auto-resolve to test structure only
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
    setMockPathname('/');
    localStorageMock.getItem.mockReturnValue(mockTenant.id);
    mockTenantsService.getAllTenants.mockResolvedValue([mockTenant]);
  });

  describe('Route Recognition and Hierarchy', () => {
    it('should generate correct breadcrumbs for home route', async () => {
      setMockPathname('/');

      render(
        <TestWrapper>
          <BreadcrumbTestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('breadcrumbs-count')).toHaveTextContent('1');
      });

      expect(screen.getByTestId('label-0')).toHaveTextContent('Home');
      expect(screen.getByTestId('is-valid-route')).toHaveTextContent('true');
      expect(screen.getByTestId('breadcrumb-0')).toHaveAttribute('data-is-current', 'true');
    });

    it('should generate correct breadcrumbs for characters list route', async () => {
      setMockPathname('/characters');

      render(
        <TestWrapper>
          <BreadcrumbTestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('breadcrumbs-count')).toHaveTextContent('2');
      });

      expect(screen.getByTestId('label-0')).toHaveTextContent('Home');
      expect(screen.getByTestId('label-1')).toHaveTextContent('Characters');
      expect(screen.getByTestId('breadcrumb-0')).toHaveAttribute('data-is-current', 'false');
      expect(screen.getByTestId('breadcrumb-1')).toHaveAttribute('data-is-current', 'true');
    });

    it('should generate correct breadcrumb structure for character detail route', async () => {
      setMockPathname('/characters/123');

      render(
        <TestWrapper>
          <BreadcrumbTestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('breadcrumbs-count')).toHaveTextContent('3');
      });

      expect(screen.getByTestId('label-0')).toHaveTextContent('Home');
      expect(screen.getByTestId('label-1')).toHaveTextContent('Characters');
      // Dynamic segment uses static label when autoResolve is false
      expect(screen.getByTestId('label-2')).toHaveTextContent('Character Details');
      expect(screen.getByTestId('breadcrumb-2')).toHaveAttribute('data-dynamic', 'true');
      expect(screen.getByTestId('breadcrumb-2')).toHaveAttribute('data-is-current', 'true');
    });
  });

  describe('Navigation Functionality', () => {
    it('should navigate when clicking non-current breadcrumb links', async () => {
      const user = userEvent.setup();
      setMockPathname('/characters/123');

      render(
        <TestWrapper>
          <BreadcrumbTestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('breadcrumbs-count')).toHaveTextContent('3');
      });

      const charactersLink = screen.getByTestId('link-1');
      expect(charactersLink).not.toBeDisabled();

      await user.click(charactersLink);

      expect(mockNavigationState.push).toHaveBeenCalledWith('/characters');
    });

    it('should not navigate when clicking current page breadcrumb', async () => {
      const user = userEvent.setup();
      setMockPathname('/characters/123');

      render(
        <TestWrapper>
          <BreadcrumbTestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('breadcrumbs-count')).toHaveTextContent('3');
      });

      const currentPageLink = screen.getByTestId('link-2');
      expect(currentPageLink).toBeDisabled();

      await user.click(currentPageLink);

      expect(mockNavigationState.push).not.toHaveBeenCalled();
    });

    it('should handle parent navigation correctly', async () => {
      const user = userEvent.setup();
      setMockPathname('/characters/123');

      render(
        <TestWrapper>
          <BreadcrumbTestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('breadcrumbs-count')).toHaveTextContent('3');
      });

      const parentButton = screen.getByTestId('parent-navigation');
      await user.click(parentButton);

      expect(mockNavigationState.push).toHaveBeenCalledWith('/characters');
    });
  });

  describe('Error Handling', () => {
    it('should handle invalid routes gracefully', async () => {
      setMockPathname('/invalid/route/path');

      render(
        <TestWrapper>
          <BreadcrumbTestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('is-valid-route')).toHaveTextContent('false');
        expect(screen.getByTestId('error-state')).toHaveTextContent('none');
      });

      // Should still generate basic breadcrumbs from path segments
      expect(screen.getByTestId('breadcrumbs-count')).not.toHaveTextContent('0');
    });
  });

  describe('All Static Route Patterns', () => {
    const staticRoutes = [
      { path: '/', expectedLabels: ['Home'] },
      { path: '/accounts', expectedLabels: ['Home', 'Accounts'] },
      { path: '/characters', expectedLabels: ['Home', 'Characters'] },
      { path: '/guilds', expectedLabels: ['Home', 'Guilds'] },
      { path: '/npcs', expectedLabels: ['Home', 'NPCs'] },
      { path: '/templates', expectedLabels: ['Home', 'Templates'] },
      { path: '/tenants', expectedLabels: ['Home', 'Tenants'] },
    ];

    staticRoutes.forEach(({ path, expectedLabels }) => {
      it(`should generate correct breadcrumbs for route: ${path}`, async () => {
        setMockPathname(path);

        render(
          <TestWrapper>
            <BreadcrumbTestComponent />
          </TestWrapper>
        );

        await waitFor(() => {
          expect(screen.getByTestId('breadcrumbs-count')).toHaveTextContent(expectedLabels.length.toString());
        });

        for (let i = 0; i < expectedLabels.length; i++) {
          expect(screen.getByTestId(`label-${i}`)).toHaveTextContent(expectedLabels[i]);
        }

        const lastIndex = expectedLabels.length - 1;
        expect(screen.getByTestId(`breadcrumb-${lastIndex}`)).toHaveAttribute('data-is-current', 'true');
      });
    });
  });

  describe('Dynamic Route Structure', () => {
    const dynamicRoutes = [
      { path: '/accounts/123', expectedCount: 3, lastLabel: 'Account Details' },
      { path: '/characters/456', expectedCount: 3, lastLabel: 'Character Details' },
      { path: '/guilds/789', expectedCount: 3, lastLabel: 'Guild Details' },
      { path: '/npcs/101', expectedCount: 3, lastLabel: 'NPC Details' },
      { path: '/templates/202', expectedCount: 3, lastLabel: 'Template Details' },
      { path: '/tenants/303', expectedCount: 3, lastLabel: 'Tenant Details' },
    ];

    dynamicRoutes.forEach(({ path, expectedCount, lastLabel }) => {
      it(`should generate correct breadcrumb structure for route: ${path}`, async () => {
        setMockPathname(path);

        render(
          <TestWrapper>
            <BreadcrumbTestComponent />
          </TestWrapper>
        );

        await waitFor(() => {
          expect(screen.getByTestId('breadcrumbs-count')).toHaveTextContent(expectedCount.toString());
        });

        // Check static labels
        expect(screen.getByTestId('label-0')).toHaveTextContent('Home');

        // Check dynamic segment has correct structure
        const lastIndex = expectedCount - 1;
        expect(screen.getByTestId(`label-${lastIndex}`)).toHaveTextContent(lastLabel);
        expect(screen.getByTestId(`breadcrumb-${lastIndex}`)).toHaveAttribute('data-dynamic', 'true');
        expect(screen.getByTestId(`breadcrumb-${lastIndex}`)).toHaveAttribute('data-is-current', 'true');
      });
    });
  });

  describe('Accessibility', () => {
    it('should have proper ARIA attributes on BreadcrumbBar', async () => {
      setMockPathname('/characters');

      render(
        <TestWrapper>
          <BreadcrumbBar />
        </TestWrapper>
      );

      await waitFor(() => {
        const nav = screen.getByRole('navigation', { name: /breadcrumb/i });
        expect(nav).toBeInTheDocument();
        expect(nav).toHaveAttribute('aria-label', 'breadcrumb');
      });
    });

  });
});
