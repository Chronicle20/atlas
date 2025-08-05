"use client"

import { useMemo } from "react";
import { NpcCard, DropdownAction } from "./NpcCard";
import { NpcCardSkeleton } from "./NpcCardSkeleton";
import { User, Upload } from "lucide-react";
import { NPC } from "@/types/models/npc";

interface VirtualizedNpcGridProps {
  npcs: NPC[];
  isLoading?: boolean;
  containerHeight?: number;
  onBulkUpdateShop?: (npcId: number) => void;
}

export function VirtualizedNpcGrid({ 
  npcs, 
  isLoading = false, 
  containerHeight = 600,
  onBulkUpdateShop,
}: VirtualizedNpcGridProps) {
  // Memoize the NPC list to prevent unnecessary re-renders
  const renderedNpcs = useMemo(() => {
    if (isLoading) {
      // Show skeleton cards during loading
      return Array.from({ length: 12 }).map((_, index) => (
        <div key={`skeleton-${index}`} className="p-2">
          <NpcCardSkeleton />
        </div>
      ));
    }

    return npcs.map((npc) => {
      // Create dropdown actions for NPCs with shops
      const dropdownActions: DropdownAction[] = npc.hasShop ? [{
        label: "Bulk Update Shop",
        icon: <Upload className="h-4 w-4 mr-2" />,
        onClick: () => onBulkUpdateShop?.(npc.id)
      }] : [];

      return (
        <div key={npc.id} className="p-2">
          <NpcCard 
            npc={npc}
            dropdownActions={dropdownActions}
          />
        </div>
      );
    });
  }, [npcs, isLoading, onBulkUpdateShop]);
  
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
  
  return (
    <div 
      className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5 gap-4"
      style={{ maxHeight: containerHeight, overflowY: 'auto' }}
    >
      {renderedNpcs}
    </div>
  );
}

// Export with display name for better debugging
VirtualizedNpcGrid.displayName = 'VirtualizedNpcGrid';