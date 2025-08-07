"use client";

import React, { useEffect, useMemo } from 'react';
import { InventoryCard, InventoryCardSkeleton } from './InventoryCard';
import { cn } from '@/lib/utils';
import { useItemDataCache } from '@/lib/hooks/useItemData';
import type { Asset, Compartment } from '@/services/api/inventory.service';

/**
 * Props for the InventoryGrid component.
 * 
 * @interface InventoryGridProps
 * @example
 * ```typescript
 * const gridProps: InventoryGridProps = {
 *   compartment: {
 *     id: "comp-1",
 *     attributes: { 
 *       compartmentType: 1, // EQUIPABLES
 *       capacity: 24 
 *     }
 *   },
 *   assets: inventoryAssets,
 *   onDeleteAsset: handleDeleteAsset,
 *   region: "GMS",
 *   majorVersion: 83
 * };
 * ```
 */
interface InventoryGridProps {
  /** The inventory compartment containing metadata like capacity and type. */
  compartment: Compartment;
  
  /** Array of inventory assets/items to display. Assets are mapped to grid slots based on their slot property. */
  assets: Asset[];
  
  /** Optional callback function called when an item's delete button is clicked. */
  onDeleteAsset?: (assetId: string) => void;
  
  /** ID of the asset currently being deleted. Used to show loading state on the specific item. */
  deletingAssetId?: string | null;
  
  /** MapleStory region identifier used for API calls. */
  region?: string | undefined;
  
  /** MapleStory major version number used for API calls. */
  majorVersion?: number | undefined;
  
  /** Whether the grid is in a loading state. Shows skeleton components when true. */
  isLoading?: boolean;
  
  /** Additional CSS classes to apply to the grid container. */
  className?: string;
  
  /** Optional callback for when empty slots are clicked. Useful for future drag-and-drop functionality. */
  onSlotClick?: (slotIndex: number) => void;
  
  /** Whether drag-and-drop interactions are enabled. Changes cursor and hover states. */
  isDragEnabled?: boolean;
}

/**
 * Represents a single slot in the inventory grid.
 * 
 * @interface GridSlot
 */
interface GridSlot {
  /** The index/position of this slot (0-based). */
  slotIndex: number;
  
  /** The asset in this slot, or null if the slot is empty. */
  asset: Asset | null;
}

/**
 * A responsive grid component for displaying inventory items with empty slots and interactive features.
 * 
 * ## Features
 * - **Responsive Layout**: Automatically adjusts grid columns based on compartment capacity and type
 * - **Empty Slot Visualization**: Shows empty slots for better inventory management UX
 * - **Cache Warming**: Preloads metadata for all visible items to improve performance
 * - **Image Preloading**: Automatically preloads images for the first 12 items
 * - **Drag-and-Drop Ready**: Prepared for future drag-and-drop functionality
 * - **Compartment-Specific Layouts**: Optimized grid layouts for different inventory types
 * 
 * ## Usage Examples
 * 
 * ### Basic Usage
 * ```tsx
 * <InventoryGrid
 *   compartment={{
 *     id: "equipables",
 *     attributes: { compartmentType: 1, capacity: 24 }
 *   }}
 *   assets={equipmentAssets}
 * />
 * ```
 * 
 * ### With Delete Functionality
 * ```tsx
 * <InventoryGrid
 *   compartment={compartment}
 *   assets={assets}
 *   onDeleteAsset={(assetId) => handleDelete(assetId)}
 *   deletingAssetId={currentlyDeletingId}
 *   region="GMS"
 *   majorVersion={83}
 * />
 * ```
 * 
 * ### With Future Drag-and-Drop Support
 * ```tsx
 * <InventoryGrid
 *   compartment={compartment}
 *   assets={assets}
 *   onSlotClick={(slotIndex) => handleSlotClick(slotIndex)}
 *   isDragEnabled={true}
 * />
 * ```
 * 
 * ## Grid Layout Logic
 * - **EQUIPABLES (Type 1)**: 4-6 columns, focused layout for equipment
 * - **CONSUMABLES/SETUP (Types 2-3)**: 6-8 columns, medium density
 * - **ETC/CASH (Types 4-5)**: 8-12 columns, high density for bulk items
 * - **Adaptive**: Automatically scales based on compartment capacity
 * 
 * ## Performance Features
 * - Uses cache warming to batch-fetch item metadata
 * - Preloads first 12 item images for faster perceived loading
 * - Deferred rendering for large inventories
 * - Optimized grid calculations based on compartment type
 * 
 * @param props - The component props
 * @returns A responsive grid displaying inventory items and empty slots
 */
export function InventoryGrid({
  compartment,
  assets,
  onDeleteAsset,
  deletingAssetId,
  region,
  majorVersion,
  isLoading = false,
  className,
  onSlotClick,
  isDragEnabled = false
}: InventoryGridProps) {
  // Create a grid of all possible slots
  const createGridSlots = (): GridSlot[] => {
    const slots: GridSlot[] = [];
    const { capacity } = compartment.attributes;
    
    // Create a map of slot index to asset for quick lookup
    const assetBySlot = new Map<number, Asset>();
    assets.forEach(asset => {
      assetBySlot.set(asset.attributes.slot, asset);
    });
    
    // Fill all slots (0 to capacity-1)
    for (let i = 0; i < capacity; i++) {
      slots.push({
        slotIndex: i,
        asset: assetBySlot.get(i) || null
      });
    }
    
    return slots;
  };

  const gridSlots = createGridSlots();
  
  // Extract unique item IDs for preloading
  const itemIds = useMemo(() => {
    return Array.from(new Set(assets.map(asset => asset.attributes.templateId)));
  }, [assets]);
  
  // Use cache warming for image preloading
  const { warmCache } = useItemDataCache();
  
  // Preload item data for all visible items when component mounts or assets change
  useEffect(() => {
    if (itemIds.length > 0 && !isLoading) {
      // Warm the cache for all item IDs in this grid
      warmCache(itemIds, region, majorVersion?.toString())
        .then((results) => {
          const successful = results.filter(result => result.status === 'fulfilled').length;
          const failed = results.filter(result => result.status === 'rejected').length;
          
          if (successful > 0) {
            console.log(`[InventoryGrid] Preloaded metadata for ${successful} items`);
          }
          
          if (failed > 0) {
            console.warn(`[InventoryGrid] Failed to preload metadata for ${failed} items`);
          }
        })
        .catch((error) => {
          console.warn('[InventoryGrid] Cache warming failed:', error);
        });
    }
  }, [itemIds, warmCache, region, majorVersion, isLoading]);

  // Calculate grid columns based on capacity
  const getGridColumns = (capacity: number): string => {
    // For common inventory sizes, use optimized layouts
    if (capacity <= 8) return 'grid-cols-4';
    if (capacity <= 16) return 'grid-cols-4';
    if (capacity <= 24) return 'grid-cols-6';
    if (capacity <= 32) return 'grid-cols-8';
    if (capacity <= 48) return 'grid-cols-8';
    if (capacity <= 64) return 'grid-cols-8';
    if (capacity <= 96) return 'grid-cols-12';
    
    // For larger inventories, use 12 columns
    return 'grid-cols-12';
  };

  const gridColumns = getGridColumns(compartment.attributes.capacity);

  const handleSlotClick = (slotIndex: number) => {
    if (onSlotClick && !isLoading) {
      onSlotClick(slotIndex);
    }
  };

  if (isLoading) {
    return (
      <div className={cn("grid gap-3", gridColumns, className)}>
        {Array.from({ length: Math.min(compartment.attributes.capacity, 16) }).map((_, index) => (
          <InventoryCardSkeleton key={index} />
        ))}
      </div>
    );
  }

  return (
    <div className={cn("grid gap-3", gridColumns, className)}>
      {gridSlots.map(({ slotIndex, asset }) => (
        <div
          key={slotIndex}
          className={cn(
            "flex items-center justify-center",
            isDragEnabled && "cursor-pointer",
            !asset && onSlotClick && "hover:bg-muted/30 rounded-lg"
          )}
          onClick={() => handleSlotClick(slotIndex)}
          data-slot={slotIndex}
          data-testid={`inventory-slot-${slotIndex}`}
        >
          {asset ? (
            <InventoryCard
              asset={asset}
              {...(onDeleteAsset && { onDelete: () => onDeleteAsset(asset.id) })}
              isDeleting={deletingAssetId === asset.id}
              region={region}
              majorVersion={majorVersion}
              shouldPreload={slotIndex < 12} // Preload first 12 items for faster loading
            />
          ) : (
            <EmptySlot slotIndex={slotIndex} />
          )}
        </div>
      ))}
    </div>
  );
}

/**
 * Component representing an empty inventory slot.
 * 
 * Provides visual consistency and shows slot numbers to help users understand
 * inventory layout and available space.
 * 
 * @param slotIndex - The slot number to display
 * @returns An empty slot component with dashed border styling
 */
function EmptySlot({ slotIndex }: { slotIndex: number }) {
  return (
    <div className="w-[100px] h-[120px] border-2 border-dashed border-muted-foreground/30 rounded-lg flex flex-col items-center justify-center bg-muted/10 hover:bg-muted/20 transition-colors">
      <div className="text-xs font-medium text-muted-foreground/60 mb-2">
        {slotIndex}
      </div>
      <div className="text-xs text-muted-foreground/50">
        Empty
      </div>
    </div>
  );
}

/**
 * Calculates the optimal grid layout for different compartment types and capacities.
 * 
 * This function provides compartment-specific grid layouts optimized for different
 * types of inventory items. Equipment items use denser layouts, while bulk items
 * like ETC and Cash use wider layouts for better visibility.
 * 
 * ## Compartment Type Mappings
 * - **Type 1 (EQUIPABLES)**: Equipment items - focused, dense layout
 * - **Type 2 (CONSUMABLES)**: Potions, consumables - medium density
 * - **Type 3 (SETUP)**: Setup items - medium density
 * - **Type 4 (ETC)**: Miscellaneous items - wide layout for bulk
 * - **Type 5 (CASH)**: Cash shop items - wide layout for variety
 * 
 * @param compartmentType - The type of inventory compartment (1-5)
 * @param capacity - The maximum number of items the compartment can hold
 * @returns Tailwind CSS grid class string (e.g., 'grid-cols-4', 'grid-cols-8')
 * 
 * @example
 * ```typescript
 * // Equipment compartment with 24 slots
 * const layout = getOptimalGridLayout(1, 24); // Returns 'grid-cols-6'
 * 
 * // ETC compartment with 96 slots  
 * const layout = getOptimalGridLayout(4, 96); // Returns 'grid-cols-12'
 * ```
 */
export function getOptimalGridLayout(compartmentType: number, capacity: number) {
  switch (compartmentType) {
    case 1: // EQUIPABLES - typically smaller, more focused layout
      return capacity <= 16 ? 'grid-cols-4' : 'grid-cols-6';
    case 2: // CONSUMABLES - medium layout
    case 3: // SETUP - medium layout
      return capacity <= 24 ? 'grid-cols-6' : 'grid-cols-8';
    case 4: // ETC - larger layout for bulk items
    case 5: // CASH - larger layout for various items
      return capacity <= 32 ? 'grid-cols-8' : 'grid-cols-12';
    default:
      return capacity <= 24 ? 'grid-cols-6' : 'grid-cols-8';
  }
}

/**
 * Export the GridSlot type for external usage in other components or utilities.
 * 
 * @example
 * ```typescript
 * import { GridSlot } from './InventoryGrid';
 * 
 * // Use in custom inventory processing
 * function processGridSlots(slots: GridSlot[]) {
 *   return slots.filter(slot => slot.asset !== null);
 * }
 * ```
 */
export type { GridSlot };