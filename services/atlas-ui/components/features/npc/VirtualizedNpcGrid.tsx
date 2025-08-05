"use client"

import { useMemo, useState, useEffect, useCallback, useRef } from "react";
import { NpcCard, DropdownAction } from "./NpcCard";
import { NpcCardSkeleton } from "./NpcCardSkeleton";
import { User, Upload } from "lucide-react";
import { NPC } from "@/types/models/npc";

interface VirtualizedNpcGridProps {
  npcs: NPC[];
  isLoading?: boolean;
  containerHeight?: number;
  onBulkUpdateShop?: (npcId: number) => void;
  enableVirtualization?: boolean; // Allow disabling virtualization for small lists
  itemHeight?: number; // Height of each grid item
  itemsPerRow?: number; // Number of items per row, auto-calculated if not provided
  overscan?: number; // Number of extra items to render outside visible area
}

// Default configuration for virtualization
const VIRTUALIZATION_CONFIG = {
  defaultItemHeight: 200, // Approximate height of NPC card + padding
  defaultOverscan: 3, // Render 3 extra rows above and below
  minItemsForVirtualization: 50, // Only virtualize if more than 50 items
  throttleMs: 16, // Throttle scroll events to ~60fps
};

export function VirtualizedNpcGrid({ 
  npcs, 
  isLoading = false, 
  containerHeight = 600,
  onBulkUpdateShop,
  enableVirtualization = true,
  itemHeight = VIRTUALIZATION_CONFIG.defaultItemHeight,
  itemsPerRow,
  overscan = VIRTUALIZATION_CONFIG.defaultOverscan,
}: VirtualizedNpcGridProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [scrollTop, setScrollTop] = useState(0);
  const [, setContainerWidth] = useState(0);
  const [actualItemsPerRow, setActualItemsPerRow] = useState(1);
  
  // Calculate items per row based on container width and responsive breakpoints
  const calculateItemsPerRow = useCallback((width: number): number => {
    if (itemsPerRow) return itemsPerRow;
    
    // Mimic Tailwind CSS responsive grid classes
    if (width >= 1536) return 5; // 2xl:grid-cols-5
    if (width >= 1280) return 4; // xl:grid-cols-4
    if (width >= 1024) return 3; // lg:grid-cols-3
    if (width >= 768) return 2;  // md:grid-cols-2
    return 1; // grid-cols-1
  }, [itemsPerRow]);

  // Update container dimensions and items per row
  useEffect(() => {
    const updateDimensions = () => {
      if (containerRef.current) {
        const { width } = containerRef.current.getBoundingClientRect();
        setContainerWidth(width);
        setActualItemsPerRow(calculateItemsPerRow(width));
      }
    };

    updateDimensions();
    window.addEventListener('resize', updateDimensions);
    return () => window.removeEventListener('resize', updateDimensions);
  }, [calculateItemsPerRow]);

  // Throttled scroll handler to improve performance
  const handleScroll = useCallback(() => {
    if (containerRef.current) {
      setScrollTop(containerRef.current.scrollTop);
    }
  }, []);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    let ticking = false;
    const throttledScrollHandler = () => {
      if (!ticking) {
        requestAnimationFrame(() => {
          handleScroll();
          ticking = false;
        });
        ticking = true;
      }
    };

    container.addEventListener('scroll', throttledScrollHandler, { passive: true });
    return () => container.removeEventListener('scroll', throttledScrollHandler);
  }, [handleScroll]);

  // Determine if we should use virtualization
  const shouldVirtualize = useMemo(() => {
    return enableVirtualization && 
           npcs.length >= VIRTUALIZATION_CONFIG.minItemsForVirtualization &&
           actualItemsPerRow > 0;
  }, [enableVirtualization, npcs.length, actualItemsPerRow]);

  // Calculate virtual scrolling parameters
  const virtualItems = useMemo(() => {
    if (!shouldVirtualize) return null;

    const totalRows = Math.ceil(npcs.length / actualItemsPerRow);
    const visibleRows = Math.ceil(containerHeight / itemHeight);
    const startRow = Math.floor(scrollTop / itemHeight);
    const endRow = Math.min(startRow + visibleRows + overscan * 2, totalRows);
    const visibleStartRow = Math.max(0, startRow - overscan);

    const startIndex = visibleStartRow * actualItemsPerRow;
    const endIndex = Math.min(endRow * actualItemsPerRow, npcs.length);
    
    return {
      totalRows,
      totalHeight: totalRows * itemHeight,
      visibleStartRow,
      startIndex,
      endIndex,
      offsetY: visibleStartRow * itemHeight,
    };
  }, [shouldVirtualize, npcs.length, actualItemsPerRow, containerHeight, itemHeight, scrollTop, overscan]);

  // Memoize the NPC rendering logic
  const renderNpcCard = useCallback((npc: NPC, index: number) => {
    const dropdownActions: DropdownAction[] = npc.hasShop ? [{
      label: "Bulk Update Shop",
      icon: <Upload className="h-4 w-4 mr-2" />,
      onClick: () => onBulkUpdateShop?.(npc.id)
    }] : [];

    return (
      <div key={`npc-${npc.id}-${index}`} className="p-2">
        <NpcCard 
          npc={npc}
          dropdownActions={dropdownActions}
        />
      </div>
    );
  }, [onBulkUpdateShop]);

  // Render visible items
  const renderedItems = useMemo(() => {
    if (isLoading) {
      // Show skeleton cards during loading
      const skeletonCount = shouldVirtualize ? 
        Math.min(Math.ceil(containerHeight / itemHeight) * actualItemsPerRow, 20) : 
        12;
      
      return Array.from({ length: skeletonCount }).map((_, index) => (
        <div key={`skeleton-${index}`} className="p-2">
          <NpcCardSkeleton />
        </div>
      ));
    }

    if (shouldVirtualize && virtualItems) {
      // Render only visible items for large lists
      const visibleNpcs = npcs.slice(virtualItems.startIndex, virtualItems.endIndex);
      return visibleNpcs.map((npc, index) => 
        renderNpcCard(npc, virtualItems.startIndex + index)
      );
    } else {
      // Render all items for small lists
      return npcs.map((npc, index) => renderNpcCard(npc, index));
    }
  }, [
    isLoading, 
    shouldVirtualize, 
    virtualItems, 
    npcs, 
    renderNpcCard, 
    containerHeight, 
    itemHeight, 
    actualItemsPerRow
  ]);
  
  // Handle empty state
  if (!isLoading && npcs.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-20 space-y-4">
        <div className="text-muted-foreground">
          <User className="w-12 h-12 mx-auto mb-2 opacity-50" />
          <p className="text-lg font-medium">No NPCs found</p>
          <p className="text-sm">Try refreshing the page or check your connection.</p>
        </div>
      </div>
    );
  }
  
  if (shouldVirtualize && virtualItems) {
    // Virtual scrolling implementation
    return (
      <div 
        ref={containerRef}
        className="overflow-auto"
        style={{ height: containerHeight }}
      >
        <div style={{ height: virtualItems.totalHeight, position: 'relative' }}>
          <div 
            className="grid gap-4"
            style={{ 
              transform: `translateY(${virtualItems.offsetY}px)`,
              gridTemplateColumns: `repeat(${actualItemsPerRow}, minmax(0, 1fr))`,
            }}
          >
            {renderedItems}
          </div>
        </div>
      </div>
    );
  } else {
    // Regular grid for small lists or when virtualization is disabled
    return (
      <div 
        ref={containerRef}
        className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5 gap-4 overflow-auto"
        style={{ maxHeight: containerHeight }}
      >
        {renderedItems}
      </div>
    );
  }
}

// Export with display name for better debugging
VirtualizedNpcGrid.displayName = 'VirtualizedNpcGrid';