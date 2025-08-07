"use client";

import { useState, useEffect } from 'react';
import Image from 'next/image';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { X, Package } from 'lucide-react';
import { cn } from '@/lib/utils';
import { mapleStoryService } from '@/services/api/maplestory.service';
import type { Asset } from '@/services/api/inventory.service';
import type { ItemDataResult } from '@/types/models/maplestory';

interface InventoryCardProps {
  asset: Asset;
  onDelete?: (assetId: string) => void;
  isDeleting?: boolean;
  region?: string | undefined;
  majorVersion?: number | undefined;
  className?: string;
}

export function InventoryCard({ 
  asset, 
  onDelete, 
  isDeleting = false,
  region,
  majorVersion,
  className 
}: InventoryCardProps) {
  const [itemData, setItemData] = useState<ItemDataResult | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [imageError, setImageError] = useState(false);
  const [imageLoaded, setImageLoaded] = useState(false);

  // Fetch item data on mount
  useEffect(() => {
    let isMounted = true;

    const fetchItemData = async () => {
      try {
        setIsLoading(true);
        const data = await mapleStoryService.getItemDataWithCache(
          asset.attributes.templateId,
          region,
          majorVersion?.toString()
        );
        
        if (isMounted) {
          setItemData(data);
        }
      } catch (error) {
        console.warn('Failed to fetch item data:', error);
        if (isMounted) {
          setItemData({
            id: asset.attributes.templateId,
            cached: false,
            error: 'Failed to load item data'
          });
        }
      } finally {
        if (isMounted) {
          setIsLoading(false);
        }
      }
    };

    fetchItemData();

    return () => {
      isMounted = false;
    };
  }, [asset.attributes.templateId, region, majorVersion]);

  const handleImageLoad = () => {
    setImageLoaded(true);
    setImageError(false);
  };

  const handleImageError = () => {
    setImageError(true);
    setImageLoaded(true);
  };

  const handleDelete = () => {
    if (onDelete && !isDeleting) {
      onDelete(asset.id);
    }
  };

  // Determine what to display
  const hasIcon = itemData?.iconUrl && !imageError;
  const hasName = itemData?.name;
  const hasError = itemData?.error || (!isLoading && !hasIcon && !hasName);

  return (
    <Card className={cn("overflow-hidden relative py-0 w-[100px] h-[120px]", className)}>
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
        {isLoading ? (
          // Loading State
          <div className="flex flex-col items-center space-y-2">
            <Skeleton className="h-8 w-8 rounded" />
            <Skeleton className="h-3 w-12" />
          </div>
        ) : hasError ? (
          // Error State - Text Fallback
          <div className="flex flex-col items-center text-center space-y-1">
            <Package className="h-6 w-6 text-muted-foreground" />
            <div className="text-xs font-medium text-foreground break-all">
              {asset.attributes.templateId}
            </div>
          </div>
        ) : (
          // Success State
          <div className="flex flex-col items-center space-y-1">
            {/* Item Icon */}
            {hasIcon && itemData?.iconUrl ? (
              <div className="relative h-8 w-8 flex-shrink-0">
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
                />
                {!imageLoaded && (
                  <Skeleton className="absolute inset-0 h-8 w-8 rounded" />
                )}
              </div>
            ) : (
              <Package className="h-6 w-6 text-muted-foreground flex-shrink-0" />
            )}

            {/* Item Name or Template ID */}
            <div className="text-xs text-center break-all min-h-[16px] flex items-center">
              <span className="font-medium">
                {hasName && itemData?.name ? itemData.name : asset.attributes.templateId}
              </span>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

// Loading skeleton component for consistent loading states
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