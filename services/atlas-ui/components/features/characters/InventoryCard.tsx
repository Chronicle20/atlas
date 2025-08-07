"use client";

import { useState, useEffect } from 'react';
import Image from 'next/image';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { X, Package } from 'lucide-react';
import { cn } from '@/lib/utils';
import { useItemData } from '@/lib/hooks/useItemData';
import { useLazyLoad } from '@/lib/hooks/useIntersectionObserver';
import { errorLogger } from '@/services/errorLogger';
import type { Asset } from '@/services/api/inventory.service';

/**
 * Props for the InventoryCard component.
 * 
 * @interface InventoryCardProps
 * @example
 * ```typescript
 * const cardProps: InventoryCardProps = {
 *   asset: {
 *     id: "asset-123",
 *     attributes: {
 *       templateId: "1002000",
 *       slot: 5
 *     }
 *   },
 *   onDelete: (assetId: string) => console.log(`Deleting ${assetId}`),
 *   region: "GMS",
 *   majorVersion: 83,
 *   shouldPreload: true
 * };
 * ```
 */
interface InventoryCardProps {
  /** The inventory asset/item to display. Must contain templateId and slot information. */
  asset: Asset;
  
  /** Optional callback function called when the delete button is clicked. If not provided, delete button won't be shown. */
  onDelete?: (assetId: string) => void;
  
  /** Whether the item is currently being deleted. Shows loading state on delete button when true. */
  isDeleting?: boolean;
  
  /** MapleStory region identifier (e.g., "GMS", "JMS", "KMS"). Used for API calls to fetch item data. */
  region?: string | undefined;
  
  /** MapleStory major version number (e.g., 83, 251). Used for API calls to fetch version-specific item data. */
  majorVersion?: number | undefined;
  
  /** Additional CSS classes to apply to the card container. */
  className?: string;
  
  /** 
   * Whether to preload the item's icon and metadata even before the card becomes visible.
   * Useful for improving perceived performance of frequently accessed items.
   * Default: false
   */
  shouldPreload?: boolean;
}

/**
 * A card component for displaying inventory items with lazy loading, image preloading, and interactive features.
 * 
 * ## Features
 * - **Lazy Loading**: Only loads item data and icons when the card becomes visible in the viewport
 * - **Image Preloading**: Optional preloading for better perceived performance
 * - **Error Handling**: Graceful fallback to text display when icons fail to load
 * - **Delete Functionality**: Optional delete button with loading states
 * - **MapleStory API Integration**: Fetches item icons and names from MapleStory.io API
 * - **Performance Optimized**: Uses intersection observer and React Query for efficient loading
 * 
 * ## Usage Examples
 * 
 * ### Basic Usage
 * ```tsx
 * <InventoryCard
 *   asset={{
 *     id: "asset-123",
 *     attributes: { templateId: "1002000", slot: 5 }
 *   }}
 * />
 * ```
 * 
 * ### With Delete Functionality
 * ```tsx
 * <InventoryCard
 *   asset={asset}
 *   onDelete={(assetId) => handleDelete(assetId)}
 *   isDeleting={deletingAssetId === asset.id}
 * />
 * ```
 * 
 * ### With Preloading and Custom Styling
 * ```tsx
 * <InventoryCard
 *   asset={asset}
 *   region="GMS"
 *   majorVersion={83}
 *   shouldPreload={true}
 *   className="border-2 border-blue-500"
 * />
 * ```
 * 
 * ## Performance Considerations
 * - Uses intersection observer to only load visible items
 * - Image preloading is optional and should be used sparingly (e.g., first 12 items)
 * - React Query handles caching and deduplication of API requests
 * - Images are optimized using Next.js Image component
 * 
 * ## Error Handling
 * - API failures fall back to displaying templateId
 * - Image loading failures fall back to package icon
 * - All errors are logged for monitoring
 * 
 * @param props - The component props
 * @returns A card component displaying the inventory item
 */
export function InventoryCard({ 
  asset, 
  onDelete, 
  isDeleting = false,
  region,
  majorVersion,
  className,
  shouldPreload = false
}: InventoryCardProps) {
  const [imageError, setImageError] = useState(false);
  const [imageLoaded, setImageLoaded] = useState(false);

  // Use intersection observer for lazy loading
  const { shouldLoad, ref: lazyRef } = useLazyLoad<HTMLDivElement>({
    threshold: 0.1,
    rootMargin: '100px', // Start loading when card is 100px away from viewport
  });

  // Use React Query for data fetching with lazy loading
  const { 
    itemData, 
    isLoading, 
    hasError, 
    errorMessage 
  } = useItemData(asset.attributes.templateId, {
    enabled: shouldLoad || shouldPreload, // Enable if visible or should preload
    ...(region && { region }),
    ...(majorVersion && { version: majorVersion.toString() }),
    staleTime: 30 * 60 * 1000, // 30 minutes
    gcTime: 24 * 60 * 60 * 1000, // 24 hours
    onError: (error: Error) => {
      // Log errors for monitoring with context
      const context: { userId?: string; tenantId?: string; url?: string } = {
        userId: 'character_inventory_user', // Could be enhanced with actual user ID
        tenantId: 'atlas_ui', // Could be enhanced with actual tenant ID
      };
      
      if (typeof window !== 'undefined') {
        context.url = window.location.href;
      }
      
      errorLogger.logError(error, undefined, context).catch((loggingError) => {
        // Fallback logging if errorLogger fails
        console.warn('Failed to log inventory card error:', loggingError);
      });
    },
  });

  // Image preloading effect - preload images when data is available
  useEffect(() => {
    if (typeof window !== 'undefined' && itemData?.iconUrl && (shouldLoad || shouldPreload) && !imageError) {
      // Create a new image element for preloading
      const img = new window.Image();
      img.src = itemData.iconUrl;
      
      // Handle preload success
      img.onload = () => {
        console.log(`[InventoryCard] Preloaded image for item ${asset.attributes.templateId}`);
      };
      
      // Handle preload errors silently - the main image will handle error display
      img.onerror = () => {
        console.warn(`[InventoryCard] Failed to preload image for item ${asset.attributes.templateId}`);
      };
      
      // Clean up
      return () => {
        img.onload = null;
        img.onerror = null;
      };
    }
  }, [itemData?.iconUrl, shouldLoad, shouldPreload, imageError, asset.attributes.templateId]);

  const handleImageLoad = () => {
    setImageLoaded(true);
    setImageError(false);
  };

  const handleImageError = () => {
    setImageError(true);
    setImageLoaded(true);
    
    // Log image loading failures for monitoring
    const imageError = new Error(`Failed to load item icon for item ${asset.attributes.templateId}`);
    const context: { userId?: string; tenantId?: string; url?: string } = {
      userId: 'character_inventory_user', // Could be enhanced with actual user ID
      tenantId: 'atlas_ui', // Could be enhanced with actual tenant ID
    };
    
    if (typeof window !== 'undefined') {
      context.url = window.location.href;
    }
    
    errorLogger.logError(imageError, undefined, context).catch((loggingError) => {
      // Fallback logging if errorLogger fails
      console.warn('Failed to log image error:', loggingError);
    });
  };

  const handleDelete = () => {
    if (onDelete && !isDeleting) {
      onDelete(asset.id);
    }
  };

  // Determine what to display
  const hasIcon = itemData?.iconUrl && !imageError;
  const hasName = itemData?.name;
  const displayError = hasError || errorMessage || (!isLoading && !hasIcon && !hasName);

  return (
    <Card ref={lazyRef} className={cn("overflow-hidden relative py-0 w-[100px] h-[120px]", className)}>
      {/* Delete Button */}
      {onDelete && (
        <Button
          variant="ghost" 
          size="icon" 
          className="absolute top-0 right-0 z-10 h-6 w-6 hover:bg-red-100 hover:text-red-600"
          onClick={handleDelete}
          disabled={isDeleting}
        >
          <X className="h-3 w-3" />
        </Button>
      )}

      {/* Slot Number */}
      <CardHeader className="p-1 pl-2 pb-1 text-left">
        <div className="text-xs font-medium text-muted-foreground">
          {asset.attributes.slot}
        </div>
      </CardHeader>

      {/* Item Content */}
      <CardContent className="p-2 pt-0 flex flex-col items-center justify-center min-h-[70px]">
        {!shouldLoad && !shouldPreload ? (
          // Not loaded yet (lazy loading) with consistent dimensions
          <div className="flex flex-col items-center space-y-1" data-testid="inventory-card-loading">
            <Skeleton className="h-8 w-8 rounded" />
            <Skeleton className="h-8 w-full rounded" />
          </div>
        ) : isLoading ? (
          // Loading State with consistent dimensions
          <div className="flex flex-col items-center space-y-1" data-testid="inventory-card-loading">
            <Skeleton className="h-8 w-8 rounded" />
            <Skeleton className="h-8 w-full rounded" />
          </div>
        ) : displayError ? (
          // Error State - Text Fallback with consistent dimensions
          <div className="flex flex-col items-center text-center space-y-1">
            <div className="relative h-8 w-8 flex-shrink-0 flex items-center justify-center">
              <Package className="h-6 w-6 text-muted-foreground" data-testid="inventory-card-package-icon" />
            </div>
            <div className="text-xs text-center break-all h-8 flex items-center justify-center w-full">
              <span className="font-medium line-clamp-2 leading-tight text-foreground">
                {asset.attributes.templateId}
              </span>
            </div>
          </div>
        ) : (
          // Success State
          <div className="flex flex-col items-center space-y-1">
            {/* Item Icon - Consistent dimensions to prevent layout shift */}
            <div className="relative h-8 w-8 flex-shrink-0">
              {hasIcon && itemData?.iconUrl ? (
                <>
                  <Image
                    src={itemData.iconUrl}
                    alt={hasName && itemData?.name ? itemData.name : `Item ${asset.attributes.templateId}`}
                    width={32}
                    height={32}
                    className={cn(
                      "object-contain transition-opacity duration-200",
                      imageLoaded ? "opacity-100" : "opacity-0"
                    )}
                    onLoad={handleImageLoad}
                    onError={handleImageError}
                    unoptimized
                    priority={shouldLoad || shouldPreload} // Prioritize loading for visible/preloaded items
                  />
                  {!imageLoaded && (
                    <Skeleton className="absolute inset-0 h-8 w-8 rounded" />
                  )}
                </>
              ) : (
                <div className="flex items-center justify-center h-full w-full">
                  <Package className="h-6 w-6 text-muted-foreground flex-shrink-0" data-testid="inventory-card-package-icon" />
                </div>
              )}
            </div>

            {/* Item Name or Template ID - Fixed height to prevent layout shift */}
            <div className="text-xs text-center break-all h-8 flex items-center justify-center w-full">
              <span className="font-medium line-clamp-2 leading-tight">
                {hasName && itemData?.name ? itemData.name : asset.attributes.templateId}
              </span>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

/**
 * A skeleton component that matches the InventoryCard layout for loading states.
 * 
 * Provides a consistent loading experience with the same dimensions and structure
 * as the actual InventoryCard component to prevent layout shift.
 * 
 * @example
 * ```tsx
 * // Display loading skeletons while data loads
 * {isLoading ? (
 *   <div className="grid grid-cols-4 gap-3">
 *     {Array.from({ length: 8 }).map((_, index) => (
 *       <InventoryCardSkeleton key={index} />
 *     ))}
 *   </div>
 * ) : (
 *   // Render actual inventory cards
 * )}
 * ```
 * 
 * @param className - Optional additional CSS classes
 * @returns A skeleton component matching InventoryCard layout
 */
export function InventoryCardSkeleton({ className }: { className?: string }) {
  return (
    <Card className={cn("overflow-hidden relative py-0 w-[100px] h-[120px]", className)}>
      <CardHeader className="p-1 pl-2 pb-1">
        <Skeleton className="h-3 w-4" />
      </CardHeader>
      <CardContent className="p-2 pt-0 flex flex-col items-center justify-center min-h-[70px]">
        <div className="flex flex-col items-center space-y-2">
          <Skeleton className="h-8 w-8 rounded" />
          <Skeleton className="h-3 w-12" />
        </div>
      </CardContent>
    </Card>
  );
}