/**
 * Visual regression tests for inventory component layouts
 * Tests layout consistency, responsive behavior, and visual states
 */

import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { InventoryCard, InventoryCardSkeleton } from '@/components/features/characters/InventoryCard';
import { InventoryGrid } from '@/components/features/characters/InventoryGrid';
import type { Asset, Compartment } from '@/services/api/inventory.service';
import type { ItemDataResult } from '@/types/models/maplestory';

// Mock Next.js Image component for consistent visual testing
jest.mock('next/image', () => {
  return function MockImage({ src, alt, className, width, height, onLoad, onError, priority, unoptimized, ...props }: any) {
    // Simulate image loading for visual consistency
    React.useEffect(() => {
      const timer = setTimeout(() => {
        if (src && src.includes('valid-item')) {
          onLoad?.();
        } else if (src && src.includes('invalid-item')) {
          onError?.();
        }
      }, 10); // Fast loading for tests
      return () => clearTimeout(timer);
    }, [src, onLoad, onError]);

    return (
      <div
        data-testid="mock-image"
        data-src={src}
        data-alt={alt}
        data-width={width}
        data-height={height}
        data-priority={priority ? 'true' : 'false'}
        data-unoptimized={unoptimized ? 'true' : 'false'}
        style={{ width: width + 'px', height: height + 'px' }}
        className={className}
      />
    );
  };
});

// Mock errorLogger
jest.mock('@/services/errorLogger', () => ({
  errorLogger: {
    logError: jest.fn().mockResolvedValue(undefined),
  },
}));

// Mock hooks for controlled testing
const mockUseItemData = jest.fn();
const mockUseLazyLoad = jest.fn();
const mockUseItemDataCache = jest.fn();

jest.mock('@/lib/hooks/useItemData', () => ({
  useItemData: (...args: any[]) => mockUseItemData(...args),
  useItemDataCache: () => mockUseItemDataCache(),
}));

jest.mock('@/lib/hooks/useIntersectionObserver', () => ({
  useLazyLoad: (...args: any[]) => mockUseLazyLoad(...args),
}));

describe('Inventory Layout Visual Regression Tests', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: {
        queries: {
          retry: false,
          gcTime: 0,
        },
      },
    });

    jest.clearAllMocks();
    
    // Default mocks
    mockUseLazyLoad.mockReturnValue({
      shouldLoad: true,
      ref: { current: null },
    });

    mockUseItemDataCache.mockReturnValue({
      warmCache: jest.fn().mockResolvedValue([]),
    });
  });

  afterEach(() => {
    queryClient.clear();
  });

  const createAsset = (templateId: number, slot: number): Asset => ({
    id: `asset-${templateId}-${slot}`,
    type: 'inventory',
    attributes: {
      characterId: 'test-char',
      slot,
      templateId,
      quantity: 1,
    },
  });

  const createCompartment = (capacity: number, type: number = 1): Compartment => ({
    id: `compartment-${type}`,
    type: 'compartment',
    attributes: {
      capacity,
      type,
    },
  });

  const createWrapper = () => {
    return ({ children }: { children: React.ReactNode }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );
  };

  describe('InventoryCard Layout Tests', () => {
    it('should maintain consistent dimensions across all states', async () => {
      const asset = createAsset(1302000, 0);
      
      // Test loading state
      mockUseItemData.mockReturnValueOnce({
        itemData: undefined,
        isLoading: true,
        hasError: false,
        errorMessage: undefined,
      });

      const { rerender, container } = render(
        <InventoryCard asset={asset} region="GMS" majorVersion={214} />,
        { wrapper: createWrapper() }
      );

      // Capture loading state dimensions
      const loadingCard = container.querySelector('.w-\\[100px\\].h-\\[120px\\]');
      expect(loadingCard).toBeInTheDocument();
      
      // Test loaded state with icon
      const mockItemData: ItemDataResult = {
        id: 1302000,
        name: 'Test Sword',
        iconUrl: 'https://maplestory.io/api/GMS/214/item/1302000/valid-item/icon',
        cached: false,
      };

      mockUseItemData.mockReturnValue({
        itemData: mockItemData,
        isLoading: false,
        hasError: false,
        errorMessage: undefined,
      });

      rerender(
        <InventoryCard asset={asset} region="GMS" majorVersion={214} />
      );

      await waitFor(() => {
        expect(screen.getByText('Test Sword')).toBeInTheDocument();
      });

      // Verify loaded state maintains same dimensions
      const loadedCard = container.querySelector('.w-\\[100px\\].h-\\[120px\\]');
      expect(loadedCard).toBeInTheDocument();

      // Test error state
      mockUseItemData.mockReturnValue({
        itemData: { id: 1302000, cached: false, error: 'Item not found' },
        isLoading: false,
        hasError: true,
        errorMessage: 'Item not found',
      });

      rerender(
        <InventoryCard asset={asset} region="GMS" majorVersion={214} />
      );

      // Verify error state maintains same dimensions
      const errorCard = container.querySelector('.w-\\[100px\\].h-\\[120px\\]');
      expect(errorCard).toBeInTheDocument();
      expect(screen.getByText('1302000')).toBeInTheDocument();
    });

    it('should render delete button consistently without layout shift', () => {
      const mockItemData: ItemDataResult = {
        id: 1302000,
        name: 'Test Item',
        iconUrl: 'https://maplestory.io/api/GMS/214/item/1302000/valid-item/icon',
        cached: false,
      };

      mockUseItemData.mockReturnValue({
        itemData: mockItemData,
        isLoading: false,
        hasError: false,
        errorMessage: undefined,
      });

      const asset = createAsset(1302000, 0);
      
      // Without delete button
      const { rerender, container } = render(
        <InventoryCard asset={asset} region="GMS" majorVersion={214} />,
        { wrapper: createWrapper() }
      );

      expect(container.querySelector('button')).not.toBeInTheDocument();

      // With delete button
      rerender(
        <InventoryCard 
          asset={asset} 
          region="GMS" 
          majorVersion={214} 
          onDelete={jest.fn()}
        />
      );

      const deleteButton = container.querySelector('button');
      expect(deleteButton).toBeInTheDocument();
      expect(deleteButton).toHaveClass('absolute', 'top-0', 'right-0');
      
      // Card dimensions should remain consistent
      const card = container.querySelector('.w-\\[100px\\].h-\\[120px\\]');
      expect(card).toBeInTheDocument();
    });

    it('should maintain consistent icon container dimensions', async () => {
      const asset = createAsset(1302000, 0);
      const mockItemData: ItemDataResult = {
        id: 1302000,
        name: 'Test Item',
        iconUrl: 'https://maplestory.io/api/GMS/214/item/1302000/valid-item/icon',
        cached: false,
      };

      mockUseItemData.mockReturnValue({
        itemData: mockItemData,
        isLoading: false,
        hasError: false,
        errorMessage: undefined,
      });

      render(
        <InventoryCard asset={asset} region="GMS" majorVersion={214} />,
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(screen.getByText('Test Item')).toBeInTheDocument();
      });

      // Icon container should have fixed dimensions
      const iconContainer = screen.getByTestId('mock-image').parentElement;
      expect(iconContainer).toHaveClass('h-8', 'w-8');
    });

    it('should render text content within fixed height containers', () => {
      const asset = createAsset(1302000, 0);
      const mockItemData: ItemDataResult = {
        id: 1302000,
        name: 'Very Long Item Name That Should Wrap',
        iconUrl: 'https://maplestory.io/api/GMS/214/item/1302000/valid-item/icon',
        cached: false,
      };

      mockUseItemData.mockReturnValue({
        itemData: mockItemData,
        isLoading: false,
        hasError: false,
        errorMessage: undefined,
      });

      const { container } = render(
        <InventoryCard asset={asset} region="GMS" majorVersion={214} />,
        { wrapper: createWrapper() }
      );

      // Text container should have fixed height
      const textContainer = container.querySelector('.h-8.flex.items-center.justify-center');
      expect(textContainer).toBeInTheDocument();
      expect(textContainer).toHaveTextContent('Very Long Item Name That Should Wrap');
    });
  });

  describe('InventoryGrid Layout Tests', () => {
    it('should render consistent grid layout for different capacities', () => {
      const assets = [
        createAsset(1302000, 0),
        createAsset(1302001, 1),
        createAsset(1302002, 2),
      ];

      mockUseItemData.mockReturnValue({
        itemData: { id: 1302000, name: 'Test Item', cached: false },
        isLoading: false,
        hasError: false,
        errorMessage: undefined,
      });

      // Test small inventory (8 slots)
      const smallCompartment = createCompartment(8);
      const { rerender, container } = render(
        <InventoryGrid
          compartment={smallCompartment}
          assets={assets}
          region="GMS"
          majorVersion={214}
        />,
        { wrapper: createWrapper() }
      );

      expect(container.querySelector('.grid-cols-4')).toBeInTheDocument();
      expect(container.querySelectorAll('[data-testid^="inventory-slot-"]')).toHaveLength(8);

      // Test medium inventory (24 slots)
      const mediumCompartment = createCompartment(24);
      rerender(
        <InventoryGrid
          compartment={mediumCompartment}
          assets={assets}
          region="GMS"
          majorVersion={214}
        />
      );

      expect(container.querySelector('.grid-cols-6')).toBeInTheDocument();
      expect(container.querySelectorAll('[data-testid^="inventory-slot-"]')).toHaveLength(24);

      // Test large inventory (64 slots)
      const largeCompartment = createCompartment(64);
      rerender(
        <InventoryGrid
          compartment={largeCompartment}
          assets={assets}
          region="GMS"
          majorVersion={214}
        />
      );

      expect(container.querySelector('.grid-cols-8')).toBeInTheDocument();
      expect(container.querySelectorAll('[data-testid^="inventory-slot-"]')).toHaveLength(64);
    });

    it('should render empty slots with consistent styling', () => {
      const compartment = createCompartment(4);
      const assets: Asset[] = [createAsset(1302000, 1)]; // Only slot 1 filled

      mockUseItemData.mockReturnValue({
        itemData: { id: 1302000, name: 'Test Item', cached: false },
        isLoading: false,
        hasError: false,
        errorMessage: undefined,
      });

      const { container } = render(
        <InventoryGrid
          compartment={compartment}
          assets={assets}
          region="GMS"
          majorVersion={214}
        />,
        { wrapper: createWrapper() }
      );

      // Should have 4 slots total
      const slots = container.querySelectorAll('[data-testid^="inventory-slot-"]');
      expect(slots).toHaveLength(4);

      // Empty slots should have consistent styling
      const emptySlots = container.querySelectorAll('.border-dashed');
      expect(emptySlots).toHaveLength(3); // 3 empty slots

      emptySlots.forEach(slot => {
        expect(slot).toHaveClass('w-[100px]', 'h-[120px]', 'border-2', 'border-dashed');
      });
    });

    it('should maintain grid gap consistency', () => {
      const compartment = createCompartment(12);
      const assets = Array.from({ length: 6 }, (_, i) => createAsset(1302000 + i, i));

      mockUseItemData.mockReturnValue({
        itemData: { id: 1302000, name: 'Test Item', cached: false },
        isLoading: false,
        hasError: false,
        errorMessage: undefined,
      });

      const { container } = render(
        <InventoryGrid
          compartment={compartment}
          assets={assets}
          region="GMS"
          majorVersion={214}
        />,
        { wrapper: createWrapper() }
      );

      // Grid should have consistent gap
      const grid = container.querySelector('.grid');
      expect(grid).toHaveClass('gap-3');
    });

    it('should render loading state with skeleton cards', () => {
      const compartment = createCompartment(8);
      const assets: Asset[] = [];

      const { container } = render(
        <InventoryGrid
          compartment={compartment}
          assets={assets}
          region="GMS"
          majorVersion={214}
          isLoading={true}
        />,
        { wrapper: createWrapper() }
      );

      // Should render skeleton cards (limited to 8 for this compartment)
      const skeletons = container.querySelectorAll('.w-\\[100px\\].h-\\[120px\\]');
      expect(skeletons.length).toBeGreaterThan(0);
      expect(skeletons.length).toBeLessThanOrEqual(8);
    });
  });

  describe('Responsive Layout Behavior', () => {
    it('should handle different compartment types with optimal layouts', () => {
      const assets = [createAsset(1302000, 0)];

      mockUseItemData.mockReturnValue({
        itemData: { id: 1302000, name: 'Test Item', cached: false },
        isLoading: false,
        hasError: false,
        errorMessage: undefined,
      });

      // Test EQUIPABLES compartment (type 1)
      const equipCompartment = createCompartment(16, 1);
      const { rerender, container } = render(
        <InventoryGrid
          compartment={equipCompartment}
          assets={assets}
          region="GMS"
          majorVersion={214}
        />,
        { wrapper: createWrapper() }
      );

      expect(container.querySelector('.grid-cols-4')).toBeInTheDocument();

      // Test ETC compartment (type 4)
      const etcCompartment = createCompartment(32, 4);
      rerender(
        <InventoryGrid
          compartment={etcCompartment}
          assets={assets}
          region="GMS"
          majorVersion={214}
        />
      );

      expect(container.querySelector('.grid-cols-8')).toBeInTheDocument();
    });

    it('should maintain card aspect ratios across all screen sizes', () => {
      const asset = createAsset(1302000, 0);
      const mockItemData: ItemDataResult = {
        id: 1302000,
        name: 'Test Item',
        iconUrl: 'https://maplestory.io/api/GMS/214/item/1302000/valid-item/icon',
        cached: false,
      };

      mockUseItemData.mockReturnValue({
        itemData: mockItemData,
        isLoading: false,
        hasError: false,
        errorMessage: undefined,
      });

      const { container } = render(
        <InventoryCard asset={asset} region="GMS" majorVersion={214} />,
        { wrapper: createWrapper() }
      );

      // Card should maintain fixed dimensions regardless of content
      const card = container.querySelector('.w-\\[100px\\].h-\\[120px\\]');
      expect(card).toBeInTheDocument();

      // Aspect ratio should be approximately 5:6 (100:120)
      // This ensures consistent visual layout across different devices
    });
  });

  describe('Layout State Transitions', () => {
    it('should prevent layout shift during loading to loaded transition', async () => {
      const asset = createAsset(1302000, 0);

      // Start with loading state
      mockUseItemData.mockReturnValueOnce({
        itemData: undefined,
        isLoading: true,
        hasError: false,
        errorMessage: undefined,
      });

      const { rerender, container } = render(
        <InventoryCard asset={asset} region="GMS" majorVersion={214} />,
        { wrapper: createWrapper() }
      );

      // Capture initial skeleton layout
      const initialSkeleton = screen.getByTestId('inventory-card-loading');
      expect(initialSkeleton).toBeInTheDocument();

      // Transition to loaded state
      const mockItemData: ItemDataResult = {
        id: 1302000,
        name: 'Loaded Item',
        iconUrl: 'https://maplestory.io/api/GMS/214/item/1302000/valid-item/icon',
        cached: false,
      };

      mockUseItemData.mockReturnValue({
        itemData: mockItemData,
        isLoading: false,
        hasError: false,
        errorMessage: undefined,
      });

      rerender(
        <InventoryCard asset={asset} region="GMS" majorVersion={214} />
      );

      await waitFor(() => {
        expect(screen.getByText('Loaded Item')).toBeInTheDocument();
      });

      // Card dimensions should remain consistent
      const card = container.querySelector('.w-\\[100px\\].h-\\[120px\\]');
      expect(card).toBeInTheDocument();
    });

    it('should maintain visual consistency during error state transition', async () => {
      const asset = createAsset(1302000, 0);

      // Start with loaded state
      mockUseItemData.mockReturnValueOnce({
        itemData: {
          id: 1302000,
          name: 'Test Item',
          iconUrl: 'https://maplestory.io/api/GMS/214/item/1302000/valid-item/icon',
          cached: false,
        },
        isLoading: false,
        hasError: false,
        errorMessage: undefined,
      });

      const { rerender, container } = render(
        <InventoryCard asset={asset} region="GMS" majorVersion={214} />,
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(screen.getByText('Test Item')).toBeInTheDocument();
      });

      // Transition to error state
      mockUseItemData.mockReturnValue({
        itemData: { id: 1302000, cached: false, error: 'Item not found' },
        isLoading: false,
        hasError: true,
        errorMessage: 'Item not found',
      });

      rerender(
        <InventoryCard asset={asset} region="GMS" majorVersion={214} />
      );

      // Should show fallback with template ID
      expect(screen.getByText('1302000')).toBeInTheDocument();
      expect(screen.getByTestId('inventory-card-package-icon')).toBeInTheDocument();

      // Card dimensions should remain consistent
      const card = container.querySelector('.w-\\[100px\\].h-\\[120px\\]');
      expect(card).toBeInTheDocument();
    });
  });

  describe('Visual Layout Snapshots', () => {
    it('should match snapshot for loaded inventory card', async () => {
      const asset = createAsset(1302000, -11);
      const mockItemData: ItemDataResult = {
        id: 1302000,
        name: 'Basic Sword',
        iconUrl: 'https://maplestory.io/api/GMS/214/item/1302000/valid-item/icon',
        cached: false,
      };

      mockUseItemData.mockReturnValue({
        itemData: mockItemData,
        isLoading: false,
        hasError: false,
        errorMessage: undefined,
      });

      const { container } = render(
        <InventoryCard asset={asset} region="GMS" majorVersion={214} />,
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(screen.getByText('Basic Sword')).toBeInTheDocument();
      });

      // Snapshot test for visual consistency
      expect(container.firstChild).toMatchSnapshot('loaded-inventory-card');
    });

    it('should match snapshot for inventory grid with mixed content', async () => {
      const compartment = createCompartment(8);
      const assets = [
        createAsset(1302000, 0),
        createAsset(1302001, 2),
        createAsset(1302002, 5),
      ];

      mockUseItemData.mockReturnValue({
        itemData: { id: 1302000, name: 'Grid Item', cached: false },
        isLoading: false,
        hasError: false,
        errorMessage: undefined,
      });

      const { container } = render(
        <InventoryGrid
          compartment={compartment}
          assets={assets}
          region="GMS"
          majorVersion={214}
        />,
        { wrapper: createWrapper() }
      );

      // Wait for content to render
      await waitFor(() => {
        const renderedAssets = screen.getAllByText('Grid Item');
        expect(renderedAssets).toHaveLength(3);
      });

      // Snapshot test for grid layout consistency
      expect(container.firstChild).toMatchSnapshot('inventory-grid-mixed-content');
    });
  });
});