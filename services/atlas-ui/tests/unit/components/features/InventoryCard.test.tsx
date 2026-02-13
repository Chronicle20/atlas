/**
 * Unit tests for InventoryCard component
 * Tests loading states, error handling, icon display, and user interactions
 */

import React, { ReactNode } from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { InventoryCard, InventoryCardSkeleton } from '@/components/features/characters/InventoryCard';
import type { Asset } from '@/services/api/inventory.service';

// Mock the tenant context
jest.mock('@/context/tenant-context', () => ({
  useTenant: () => ({
    activeTenant: {
      id: 'test-tenant',
      type: 'tenant',
      attributes: {
        region: 'GMS',
        majorVersion: 83,
        minorVersion: 1,
      },
    },
  }),
}));

// Mock the useItemData hook
const mockUseItemData = jest.fn();
jest.mock('@/lib/hooks/useItemData', () => ({
  useItemData: (...args: any[]) => mockUseItemData(...args),
}));

// Mock the intersection observer hook
jest.mock('@/lib/hooks/useIntersectionObserver', () => ({
  useLazyLoad: jest.fn(() => ({
    shouldLoad: true,
    ref: jest.fn(),
  })),
}));

// Mock Next.js Image component
jest.mock('next/image', () => {
  return function MockImage(props: any) {
    const { priority, unoptimized, ...otherProps } = props;
    // eslint-disable-next-line @next/next/no-img-element, jsx-a11y/alt-text
    return <img {...otherProps} data-testid="next-image" data-priority={priority} data-unoptimized={unoptimized} />;
  };
});

// Import the mocked intersection observer hook
import { useLazyLoad } from '@/lib/hooks/useIntersectionObserver';
const mockUseLazyLoad = useLazyLoad as jest.MockedFunction<typeof useLazyLoad>;

// Mock asset data - cast partial data since tests only use id, slot, templateId
const mockAsset = {
  id: 'test-asset-123',
  type: 'assets',
  attributes: {
    templateId: 1001,
    slot: 1,
  },
} as unknown as Asset;

const mockAssetWithHighSlot = {
  id: 'test-asset-456',
  type: 'assets',
  attributes: {
    templateId: 2001,
    slot: 99,
  },
} as unknown as Asset;

describe('InventoryCard', () => {
  let queryClient: QueryClient;
  let wrapper: ({ children }: { children: ReactNode }) => React.JSX.Element;
  const mockOnDelete = jest.fn();

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

    // Default mock return for useItemData
    mockUseItemData.mockReturnValue({
      itemData: undefined,
      isLoading: true,
      hasError: false,
      errorMessage: undefined,
      name: undefined,
      iconUrl: undefined,
      cached: false,
      isError: false,
      error: null,
      invalidate: jest.fn(),
      prefetchItem: jest.fn(),
    });
  });

  afterEach(() => {
    queryClient.clear();
  });

  describe('Basic Rendering', () => {
    it('should render with basic asset information', async () => {
      mockUseItemData.mockReturnValue({
        itemData: { id: 1001, name: 'Blue Potion', iconUrl: '/api/assets/test-tenant/GMS/83.1/item/1001/icon.png', cached: false },
        isLoading: false,
        hasError: false,
        errorMessage: undefined,
        name: 'Blue Potion',
        iconUrl: '/api/assets/test-tenant/GMS/83.1/item/1001/icon.png',
        cached: false,
        isError: false,
        error: null,
        invalidate: jest.fn(),
        prefetchItem: jest.fn(),
      });

      render(
        <InventoryCard asset={mockAsset} />,
        { wrapper }
      );

      // Verify slot number is displayed
      expect(screen.getByText('1')).toBeInTheDocument();

      // Wait for item data to load
      await waitFor(() => {
        expect(screen.getByText('Blue Potion')).toBeInTheDocument();
      });

      // Verify image is rendered
      const image = screen.getByTestId('next-image');
      expect(image).toBeInTheDocument();
      expect(image).toHaveAttribute('src', '/api/assets/test-tenant/GMS/83.1/item/1001/icon.png');
      expect(image).toHaveAttribute('alt', 'Blue Potion');
    });

    it('should render slot numbers correctly for various slots', () => {
      render(
        <InventoryCard asset={mockAssetWithHighSlot} />,
        { wrapper }
      );

      expect(screen.getByText('99')).toBeInTheDocument();
    });

    it('should apply custom className', () => {
      const customClass = 'my-custom-inventory-card';

      render(
        <InventoryCard asset={mockAsset} className={customClass} />,
        { wrapper }
      );

      const card = document.querySelector(`.${customClass}`);
      expect(card).toBeInTheDocument();
      expect(card).toHaveClass(customClass);
    });
  });

  describe('Loading States', () => {
    it('should show loading skeleton while data is fetching', () => {
      mockUseItemData.mockReturnValue({
        itemData: undefined,
        isLoading: true,
        hasError: false,
        errorMessage: undefined,
        name: undefined,
        iconUrl: undefined,
        cached: false,
        isError: false,
        error: null,
        invalidate: jest.fn(),
        prefetchItem: jest.fn(),
      });

      render(
        <InventoryCard asset={mockAsset} />,
        { wrapper }
      );

      // Should show skeleton loading state
      expect(document.querySelector('.animate-pulse')).toBeInTheDocument();
    });

    it('should hide loading state after data loads successfully', async () => {
      mockUseItemData.mockReturnValue({
        itemData: { id: 1001, name: 'Red Potion', iconUrl: '/api/assets/test-tenant/GMS/83.1/item/1001/icon.png', cached: false },
        isLoading: false,
        hasError: false,
        errorMessage: undefined,
        name: 'Red Potion',
        iconUrl: '/api/assets/test-tenant/GMS/83.1/item/1001/icon.png',
        cached: false,
        isError: false,
        error: null,
        invalidate: jest.fn(),
        prefetchItem: jest.fn(),
      });

      render(
        <InventoryCard asset={mockAsset} />,
        { wrapper }
      );

      await waitFor(() => {
        expect(screen.getByText('Red Potion')).toBeInTheDocument();
      });

      // After data loads, the main loading skeletons should be gone
      const mainLoadingSkeletons = document.querySelectorAll('.animate-pulse:not(.absolute)');
      expect(mainLoadingSkeletons.length).toBe(0);
    });
  });

  describe('Error Handling', () => {
    it('should display fallback when item data fetch fails', async () => {
      mockUseItemData.mockReturnValue({
        itemData: { id: 1001, cached: false, error: 'Item not found' },
        isLoading: false,
        hasError: true,
        errorMessage: 'Item not found',
        name: undefined,
        iconUrl: undefined,
        cached: false,
        isError: false,
        error: null,
        invalidate: jest.fn(),
        prefetchItem: jest.fn(),
      });

      render(
        <InventoryCard asset={mockAsset} />,
        { wrapper }
      );

      await waitFor(() => {
        // Should show Package icon and template ID as fallback
        expect(screen.getByText('1001')).toBeInTheDocument();
        // Look for the lucide package icon by class
        expect(document.querySelector('.lucide-package')).toBeInTheDocument();
      });
    });

    it('should handle image load errors gracefully', async () => {
      mockUseItemData.mockReturnValue({
        itemData: { id: 1001, name: 'Test Item', iconUrl: 'https://invalid-url.com/image.png', cached: false },
        isLoading: false,
        hasError: false,
        errorMessage: undefined,
        name: 'Test Item',
        iconUrl: 'https://invalid-url.com/image.png',
        cached: false,
        isError: false,
        error: null,
        invalidate: jest.fn(),
        prefetchItem: jest.fn(),
      });

      render(
        <InventoryCard asset={mockAsset} />,
        { wrapper }
      );

      await waitFor(() => {
        expect(screen.getByText('Test Item')).toBeInTheDocument();
      });

      // Simulate image error
      const image = screen.getByTestId('next-image');
      fireEvent.error(image);

      await waitFor(() => {
        // Should still display the item name since that loaded successfully
        expect(screen.getByText('Test Item')).toBeInTheDocument();
        expect(document.querySelector('.lucide-package')).toBeInTheDocument();
      });
    });

    it('should show template ID when no name or icon is available', async () => {
      mockUseItemData.mockReturnValue({
        itemData: { id: 1001, cached: false },
        isLoading: false,
        hasError: false,
        errorMessage: undefined,
        name: undefined,
        iconUrl: undefined,
        cached: false,
        isError: false,
        error: null,
        invalidate: jest.fn(),
        prefetchItem: jest.fn(),
      });

      render(
        <InventoryCard asset={mockAsset} />,
        { wrapper }
      );

      await waitFor(() => {
        expect(screen.getByText('1001')).toBeInTheDocument();
        expect(document.querySelector('.lucide-package')).toBeInTheDocument();
      });
    });
  });

  describe('Delete Functionality', () => {
    it('should render delete button when onDelete is provided', () => {
      render(
        <InventoryCard asset={mockAsset} onDelete={mockOnDelete} />,
        { wrapper }
      );

      const deleteButton = screen.getByRole('button');
      expect(deleteButton).toBeInTheDocument();
      expect(deleteButton.className).toContain('absolute');
      expect(deleteButton.className).toContain('top-0');
      expect(deleteButton.className).toContain('right-0');
    });

    it('should not render delete button when onDelete is not provided', () => {
      render(
        <InventoryCard asset={mockAsset} />,
        { wrapper }
      );

      const deleteButton = screen.queryByRole('button');
      expect(deleteButton).not.toBeInTheDocument();
    });

    it('should call onDelete with asset ID when delete button is clicked', () => {
      render(
        <InventoryCard asset={mockAsset} onDelete={mockOnDelete} />,
        { wrapper }
      );

      const deleteButton = screen.getByRole('button');
      fireEvent.click(deleteButton);

      expect(mockOnDelete).toHaveBeenCalledWith(mockAsset.id);
    });

    it('should disable delete button when isDeleting is true', () => {
      render(
        <InventoryCard
          asset={mockAsset}
          onDelete={mockOnDelete}
          isDeleting={true}
        />,
        { wrapper }
      );

      const deleteButton = screen.getByRole('button');
      expect(deleteButton).toBeDisabled();

      fireEvent.click(deleteButton);
      expect(mockOnDelete).not.toHaveBeenCalled();
    });

    it('should show hover effects on delete button', () => {
      render(
        <InventoryCard asset={mockAsset} onDelete={mockOnDelete} />,
        { wrapper }
      );

      const deleteButton = screen.getByRole('button');
      expect(deleteButton.className).toContain('hover:bg-red-100');
      expect(deleteButton.className).toContain('hover:text-red-600');
    });
  });

  describe('Image Preloading', () => {
    it('should preload images when shouldPreload is true', async () => {
      mockUseItemData.mockReturnValue({
        itemData: { id: 1001, name: 'Preload Item', iconUrl: '/api/assets/test-tenant/GMS/83.1/item/1001/icon.png', cached: false },
        isLoading: false,
        hasError: false,
        errorMessage: undefined,
        name: 'Preload Item',
        iconUrl: '/api/assets/test-tenant/GMS/83.1/item/1001/icon.png',
        cached: false,
        isError: false,
        error: null,
        invalidate: jest.fn(),
        prefetchItem: jest.fn(),
      });

      render(
        <InventoryCard asset={mockAsset} shouldPreload={true} />,
        { wrapper }
      );

      await waitFor(() => {
        expect(screen.getByText('Preload Item')).toBeInTheDocument();
      });

      // Verify priority attribute is set on Next.js Image
      const image = screen.getByTestId('next-image');
      expect(image).toHaveAttribute('data-priority', 'true');
    });

    it('should not set priority when shouldPreload is false and not in view', async () => {
      // Mock shouldLoad to be false for this test
      mockUseLazyLoad.mockReturnValueOnce({
        shouldLoad: false,
        ref: { current: null } as unknown as React.RefObject<Element>,
      });

      render(
        <InventoryCard asset={mockAsset} shouldPreload={false} />,
        { wrapper }
      );

      // Since shouldLoad is false and shouldPreload is false, no image should render
      const image = screen.queryByTestId('next-image');
      expect(image).not.toBeInTheDocument();
    });
  });

  describe('Image Loading States', () => {
    it('should show image loading skeleton until image loads', async () => {
      mockUseItemData.mockReturnValue({
        itemData: { id: 1001, name: 'Loading Item', iconUrl: '/api/assets/test-tenant/GMS/83.1/item/1001/icon.png', cached: false },
        isLoading: false,
        hasError: false,
        errorMessage: undefined,
        name: 'Loading Item',
        iconUrl: '/api/assets/test-tenant/GMS/83.1/item/1001/icon.png',
        cached: false,
        isError: false,
        error: null,
        invalidate: jest.fn(),
        prefetchItem: jest.fn(),
      });

      render(
        <InventoryCard asset={mockAsset} />,
        { wrapper }
      );

      await waitFor(() => {
        expect(screen.getByText('Loading Item')).toBeInTheDocument();
      });

      // Image should initially be transparent with skeleton
      const image = screen.getByTestId('next-image');
      expect(image).toHaveClass('opacity-0');

      // Simulate image load
      fireEvent.load(image);

      expect(image).toHaveClass('opacity-100');
    });
  });

  describe('Accessibility', () => {
    it('should have proper alt text for images', async () => {
      mockUseItemData.mockReturnValue({
        itemData: { id: 1001, name: 'Accessibility Item', iconUrl: '/api/assets/test-tenant/GMS/83.1/item/1001/icon.png', cached: false },
        isLoading: false,
        hasError: false,
        errorMessage: undefined,
        name: 'Accessibility Item',
        iconUrl: '/api/assets/test-tenant/GMS/83.1/item/1001/icon.png',
        cached: false,
        isError: false,
        error: null,
        invalidate: jest.fn(),
        prefetchItem: jest.fn(),
      });

      render(
        <InventoryCard asset={mockAsset} />,
        { wrapper }
      );

      await waitFor(() => {
        const image = screen.getByTestId('next-image');
        expect(image).toHaveAttribute('alt', 'Accessibility Item');
      });
    });

    it('should provide fallback alt text when item name is not available', async () => {
      mockUseItemData.mockReturnValue({
        itemData: { id: 1001, iconUrl: '/api/assets/test-tenant/GMS/83.1/item/1001/icon.png', cached: false },
        isLoading: false,
        hasError: false,
        errorMessage: undefined,
        name: undefined,
        iconUrl: '/api/assets/test-tenant/GMS/83.1/item/1001/icon.png',
        cached: false,
        isError: false,
        error: null,
        invalidate: jest.fn(),
        prefetchItem: jest.fn(),
      });

      render(
        <InventoryCard asset={mockAsset} />,
        { wrapper }
      );

      await waitFor(() => {
        const image = screen.getByTestId('next-image');
        expect(image).toHaveAttribute('alt', 'Item 1001');
      });
    });

    it('should have proper button styling for delete action', () => {
      render(
        <InventoryCard asset={mockAsset} onDelete={mockOnDelete} />,
        { wrapper }
      );

      const deleteButton = screen.getByRole('button');
      expect(deleteButton.className).toContain('absolute');
      expect(deleteButton.className).toContain('top-0');
      expect(deleteButton.className).toContain('right-0');
      expect(deleteButton.className).toContain('z-10');
    });
  });
});

describe('InventoryCardSkeleton', () => {
  it('should render skeleton with proper structure', () => {
    render(<InventoryCardSkeleton />);

    // Should have consistent card dimensions
    const skeletonCards = document.querySelectorAll('.animate-pulse');
    expect(skeletonCards.length).toBeGreaterThan(0);
  });

  it('should apply custom className to skeleton', () => {
    const customClass = 'my-skeleton-class';
    render(<InventoryCardSkeleton className={customClass} />);

    const skeleton = document.querySelector(`.${customClass}`);
    expect(skeleton).toBeInTheDocument();
  });

  it('should maintain consistent dimensions with actual card', () => {
    render(<InventoryCardSkeleton />);

    const skeletonCard = document.querySelector('.w-\\[100px\\].h-\\[120px\\]');
    expect(skeletonCard).toBeInTheDocument();
  });
});
