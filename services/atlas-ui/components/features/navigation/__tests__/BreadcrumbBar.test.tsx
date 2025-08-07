/**
 * @jest-environment jsdom
 */

import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { useRouter } from 'next/navigation';
import { BreadcrumbBar, SimpleBreadcrumbBar } from '../BreadcrumbBar';
import { useTenant } from '@/context/tenant-context';
import { useBreadcrumbs } from '@/lib/hooks/useBreadcrumbs';

// Mock dependencies
jest.mock('next/navigation', () => ({
  useRouter: jest.fn(),
  usePathname: jest.fn(() => '/characters/123'),
}));

jest.mock('@/context/tenant-context', () => ({
  useTenant: jest.fn(),
}));

jest.mock('@/lib/hooks/useBreadcrumbs', () => ({
  useBreadcrumbs: jest.fn(),
}));

// Mock Link component
jest.mock('next/link', () => {
  return ({ children, href, ...props }: any) => (
    <a href={href} {...props}>
      {children}
    </a>
  );
});

const mockUseTenant = useTenant as jest.MockedFunction<typeof useTenant>;
const mockUseBreadcrumbs = useBreadcrumbs as jest.MockedFunction<typeof useBreadcrumbs>;
const mockUseRouter = useRouter as jest.MockedFunction<typeof useRouter>;

// Test data
const mockTenant = {
  id: 'tenant-123',
  type: 'tenant',
  attributes: {
    name: 'Test Tenant',
    version: { major: 83, minor: 1 },
    region: 'GMS',
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
    segment: '123',
    label: 'Character Details',
    href: '/characters/123',
    dynamic: true,
    isCurrentPage: true,
    entityId: '123',
    entityType: 'character',
  },
];

const mockBreadcrumbsResult = {
  breadcrumbs: mockBreadcrumbs,
  loading: false,
  error: null,
  routeConfig: {
    pattern: '/characters/[id]',
    label: 'Character Details',
    parent: '/characters',
    entityType: 'character',
  },
  navigation: {
    goToParent: jest.fn(),
    navigateTo: jest.fn(),
    getParent: jest.fn(() => mockBreadcrumbs[1]),
  },
  resolution: {
    resolveLabel: jest.fn(),
    invalidateLabels: jest.fn(),
    preloadLabels: jest.fn(),
    resolutionStates: new Map(),
  },
  utils: {
    getCacheKey: jest.fn(() => 'cache-key'),
    isValidRoute: true,
    getFilteredBreadcrumbs: jest.fn(() => mockBreadcrumbs),
  },
};

describe('BreadcrumbBar', () => {
  beforeEach(() => {
    // Setup default mocks
    mockUseTenant.mockReturnValue({
      tenants: [mockTenant],
      activeTenant: mockTenant,
      loading: false,
      setActiveTenant: jest.fn(),
      refreshTenants: jest.fn(),
      fetchTenantConfiguration: jest.fn(),
    });

    mockUseBreadcrumbs.mockReturnValue(mockBreadcrumbsResult);

    mockUseRouter.mockReturnValue({
      push: jest.fn(),
      replace: jest.fn(),
      back: jest.fn(),
      forward: jest.fn(),
      refresh: jest.fn(),
      prefetch: jest.fn(),
    });
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  describe('Basic Rendering', () => {
    it('should render breadcrumbs correctly', async () => {
      render(<BreadcrumbBar />);

      // Wait for client-side rendering
      await waitFor(() => {
        expect(screen.getByRole('navigation')).toBeInTheDocument();
      });

      expect(screen.getByText('Home')).toBeInTheDocument();
      expect(screen.getByText('Characters')).toBeInTheDocument();
      expect(screen.getByText('Character Details')).toBeInTheDocument();
    });

    it('should render loading state when tenant is loading', () => {
      mockUseTenant.mockReturnValue({
        tenants: [],
        activeTenant: null,
        loading: true,
        setActiveTenant: jest.fn(),
        refreshTenants: jest.fn(),
        fetchTenantConfiguration: jest.fn(),
      });

      render(<BreadcrumbBar />);

      expect(screen.getByText('Loading navigation...')).toBeInTheDocument();
    });

    it('should not render when route is invalid', async () => {
      mockUseBreadcrumbs.mockReturnValue({
        ...mockBreadcrumbsResult,
        utils: {
          ...mockBreadcrumbsResult.utils,
          isValidRoute: false,
        },
      });

      const { container } = render(<BreadcrumbBar />);

      await waitFor(() => {
        expect(container.firstChild).toBeNull();
      });
    });

    it('should handle error state gracefully', async () => {
      const consoleSpy = jest.spyOn(console, 'warn').mockImplementation();
      
      mockUseBreadcrumbs.mockReturnValue({
        ...mockBreadcrumbsResult,
        error: new Error('Test error'),
      });

      render(<BreadcrumbBar />);

      await waitFor(() => {
        expect(screen.getByText('Navigation unavailable')).toBeInTheDocument();
      });

      expect(consoleSpy).toHaveBeenCalledWith('Breadcrumb error:', expect.any(Error));
      
      consoleSpy.mockRestore();
    });
  });

  describe('Navigation Links', () => {
    it('should render clickable links for non-current pages', async () => {
      render(<BreadcrumbBar />);

      await waitFor(() => {
        const homeLink = screen.getByRole('link', { name: 'Home' });
        const charactersLink = screen.getByRole('link', { name: 'Characters' });
        
        expect(homeLink).toHaveAttribute('href', '/');
        expect(charactersLink).toHaveAttribute('href', '/characters');
      });
    });

    it('should render current page as span, not link', async () => {
      render(<BreadcrumbBar />);

      await waitFor(() => {
        const currentPage = screen.getByText('Character Details');
        // Check that it's not a link
        expect(currentPage.closest('a')).toBeNull();
        // Check that the breadcrumb page has the correct attributes
        const breadcrumbPage = currentPage.closest('span[aria-current="page"]');
        expect(breadcrumbPage).toHaveAttribute('aria-current', 'page');
        expect(breadcrumbPage).toHaveAttribute('role', 'link');
      });
    });
  });

  describe('Loading States', () => {
    it('should show loading indicators for dynamic breadcrumbs', async () => {
      mockUseBreadcrumbs.mockReturnValue({
        ...mockBreadcrumbsResult,
        loading: true,
      });

      render(<BreadcrumbBar showLoadingStates={true} />);

      await waitFor(() => {
        // Should show loading spinners for dynamic breadcrumbs
        const spinners = screen.getAllByTestId('loading-spinner');
        expect(spinners.length).toBeGreaterThan(0);
      });
    });

    it('should not show loading indicators when disabled', async () => {
      mockUseBreadcrumbs.mockReturnValue({
        ...mockBreadcrumbsResult,
        loading: true,
      });

      render(<BreadcrumbBar showLoadingStates={false} />);

      await waitFor(() => {
        expect(screen.queryByTestId('loading-spinner')).not.toBeInTheDocument();
      });
    });
  });

  describe('Ellipsis and Truncation', () => {
    it('should show ellipsis when breadcrumbs exceed maxItems', async () => {
      const longBreadcrumbs = [
        { segment: '', label: 'Home', href: '/', dynamic: false, isCurrentPage: false },
        { segment: 'level1', label: 'Level 1', href: '/level1', dynamic: false, isCurrentPage: false },
        { segment: 'level2', label: 'Level 2', href: '/level1/level2', dynamic: false, isCurrentPage: false },
        { segment: 'level3', label: 'Level 3', href: '/level1/level2/level3', dynamic: false, isCurrentPage: false },
        { segment: 'level4', label: 'Level 4', href: '/level1/level2/level3/level4', dynamic: false, isCurrentPage: true },
      ];

      mockUseBreadcrumbs.mockReturnValue({
        ...mockBreadcrumbsResult,
        breadcrumbs: longBreadcrumbs,
      });

      render(<BreadcrumbBar maxItems={3} showEllipsis={true} />);

      await waitFor(() => {
        expect(screen.getByText('Home')).toBeInTheDocument();
        expect(screen.getByText('Level 3')).toBeInTheDocument();
        expect(screen.getByText('Level 4')).toBeInTheDocument();
        
        // Should show ellipsis
        expect(document.querySelector('[aria-hidden="true"]')).toBeInTheDocument();
      });
    });
  });

  describe('Label Overrides', () => {
    it('should apply custom label overrides', async () => {
      const labelOverrides = {
        '/characters': 'Custom Characters',
      };

      render(<BreadcrumbBar labelOverrides={labelOverrides} />);

      await waitFor(() => {
        expect(screen.getByText('Custom Characters')).toBeInTheDocument();
      });
    });
  });

  describe('Tenant Context Integration', () => {
    it('should disable auto-resolve when no active tenant', async () => {
      mockUseTenant.mockReturnValue({
        tenants: [],
        activeTenant: null,
        loading: false,
        setActiveTenant: jest.fn(),
        refreshTenants: jest.fn(),
        fetchTenantConfiguration: jest.fn(),
      });

      render(<BreadcrumbBar />);

      await waitFor(() => {
        expect(mockUseBreadcrumbs).toHaveBeenCalledWith(
          expect.objectContaining({
            autoResolve: false,
          })
        );
      });
    });

    it('should enable auto-resolve when active tenant exists', async () => {
      render(<BreadcrumbBar />);

      await waitFor(() => {
        expect(mockUseBreadcrumbs).toHaveBeenCalledWith(
          expect.objectContaining({
            autoResolve: true,
          })
        );
      });
    });
  });
});

describe('SimpleBreadcrumbBar', () => {
  beforeEach(() => {
    mockUseTenant.mockReturnValue({
      tenants: [mockTenant],
      activeTenant: mockTenant,
      loading: false,
      setActiveTenant: jest.fn(),
      refreshTenants: jest.fn(),
      fetchTenantConfiguration: jest.fn(),
    });

    mockUseBreadcrumbs.mockReturnValue(mockBreadcrumbsResult);
  });

  it('should render with simplified options', async () => {
    render(<SimpleBreadcrumbBar />);

    await waitFor(() => {
      expect(mockUseBreadcrumbs).toHaveBeenCalledWith(
        expect.objectContaining({
          maxItems: 3,
          showEllipsis: false,
        })
      );
    });
  });

  it('should accept className prop', async () => {
    render(<SimpleBreadcrumbBar className="test-class" />);

    await waitFor(() => {
      const nav = screen.getByRole('navigation');
      expect(nav.closest('div')).toHaveClass('test-class');
    });
  });
});