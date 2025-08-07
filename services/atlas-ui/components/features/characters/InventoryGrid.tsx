"use client";

import React from 'react';
import { InventoryCard, InventoryCardSkeleton } from './InventoryCard';
import { cn } from '@/lib/utils';
import type { Asset, Compartment } from '@/services/api/inventory.service';

interface InventoryGridProps {
  compartment: Compartment;
  assets: Asset[];
  onDeleteAsset?: (assetId: string) => void;
  deletingAssetId?: string | null;
  region?: string | undefined;
  majorVersion?: number | undefined;
  isLoading?: boolean;
  className?: string;
  // Future drag-and-drop preparation
  onSlotClick?: (slotIndex: number) => void;
  isDragEnabled?: boolean;
}

interface GridSlot {
  slotIndex: number;
  asset: Asset | null;
}

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
            />
          ) : (
            <EmptySlot slotIndex={slotIndex} />
          )}
        </div>
      ))}
    </div>
  );
}

// Empty slot component for visual consistency
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

// Grid layout helper for different compartment types
export function getOptimalGridLayout(compartmentType: number, capacity: number) {
  // Compartment type specific layouts
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

// Export helper function for external usage
export type { GridSlot };