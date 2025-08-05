"use client";

import { useState, useEffect, useRef, useCallback, useMemo } from 'react';
import Image from 'next/image';
import { Button } from '@/components/ui/button';
import { CharacterRendererDetailSkeleton } from '@/components/common/skeletons/CharacterDetailSkeleton';
import { useCharacterImage } from '@/lib/hooks/useCharacterImage';
import { useLazyLoad } from '@/lib/hooks/useIntersectionObserver';
import { mapleStoryService } from '@/services/api/maplestory.service';
import type { Asset } from '@/services/api/inventory.service';
import type { CharacterRendererProps, MapleStoryCharacterData } from '@/types/models/maplestory';
import { cn } from '@/lib/utils';

interface CharacterRendererComponentProps extends Omit<CharacterRendererProps, 'equipment'> {
  inventory?: Asset[];
  size?: 'small' | 'medium' | 'large';
  maxRetries?: number;
  showRetryButton?: boolean;
  priority?: boolean;
  lazy?: boolean;
  enablePreload?: boolean;
  prefetchVariants?: boolean;
}

type ErrorType = 'api_error' | 'image_load_error' | 'network_error' | 'fallback_error' | 'unknown_error';

interface ErrorState {
  type: ErrorType;
  message: string;
  isRetryable: boolean;
}

const sizeClasses = {
  small: 'w-32 h-32',
  medium: 'w-48 h-48', 
  large: 'w-64 h-64'
};

const sizeDimensions = {
  small: { width: 128, height: 128 },
  medium: { width: 192, height: 192 },
  large: { width: 256, height: 256 }
};

export function CharacterRenderer({
  character,
  inventory = [],
  scale = 2,
  size = 'medium',
  showLoading = true,
  fallbackAvatar = '/default-character-avatar.svg',
  className,
  onImageLoad,
  onImageError,
  maxRetries = 3,
  showRetryButton = true,
  priority = false,
  lazy = true,
  enablePreload = true,
  prefetchVariants = false,
}: CharacterRendererComponentProps) {
  const [fallbackImageError, setFallbackImageError] = useState(false);
  const [imageLoaded, setImageLoaded] = useState(false);
  const [manualRetryCount, setManualRetryCount] = useState(0);
  const mountedRef = useRef(true);

  // Lazy loading support with intersection observer
  const { shouldLoad, ref: lazyRef } = useLazyLoad<HTMLDivElement>({
    enabled: lazy && !priority,
    rootMargin: '200px', // Start loading 200px before entering viewport
  });

  // Convert Character model to MapleStoryCharacterData
  const mapleStoryData = useMemo((): MapleStoryCharacterData => 
    mapleStoryService.characterToMapleStoryData(character, inventory),
    [character, inventory]
  );

  // Use optimized character image hook
  const {
    data: imageResult,
    isLoading,
    error: queryError,
    refetch,
    preload,
    prefetchVariants: prefetchVariantsFn,
    imageUrl,
    cached,
  } = useCharacterImage(
    mapleStoryData,
    { resize: scale },
    {
      priority,
      lazy,
      retry: maxRetries,
      enabled: priority || !lazy || shouldLoad, // Load immediately if priority or not lazy, otherwise wait for intersection
      onSuccess: () => {
        setImageLoaded(false); // Reset for new image
        onImageLoad?.();
      },
      onError: (error) => {
        onImageError?.(error);
      },
    }
  );

  // Prefetch variants if enabled
  useEffect(() => {
    if (prefetchVariants && imageResult && !isLoading && !queryError) {
      const variants = [
        { resize: scale * 0.5 }, // Smaller version
        { resize: scale * 1.5 }, // Larger version
      ];
      
      // Don't await - fire and forget
      prefetchVariantsFn(variants);
    }
  }, [prefetchVariants, prefetchVariantsFn, imageResult, isLoading, queryError, scale]);

  // Preload image if enabled and priority
  useEffect(() => {
    if (enablePreload && priority && imageUrl && !imageLoaded) {
      preload().then((result) => {
        if (result?.loaded && mountedRef.current) {
          setImageLoaded(true);
        }
      });
    }
  }, [enablePreload, priority, imageUrl, imageLoaded, preload]);
  
  // Utility function to classify errors
  const classifyError = useCallback((err: unknown): ErrorState => {
    if (err instanceof Error) {
      const message = err.message.toLowerCase();
      
      if (message.includes('network') || message.includes('fetch')) {
        return {
          type: 'network_error',
          message: 'Network connection failed. Please check your internet connection.',
          isRetryable: true
        };
      }
      
      if (message.includes('api') || message.includes('service')) {
        return {
          type: 'api_error', 
          message: 'Character rendering service is temporarily unavailable.',
          isRetryable: true
        };
      }
      
      if (message.includes('load') || message.includes('image')) {
        return {
          type: 'image_load_error',
          message: 'Failed to load character image.',
          isRetryable: true
        };
      }
      
      return {
        type: 'unknown_error',
        message: err.message || 'An unexpected error occurred.',
        isRetryable: true
      };
    }
    
    return {
      type: 'unknown_error',
      message: 'An unexpected error occurred while rendering character.',
      isRetryable: true
    };
  }, []);

  // Convert query error to our error state format
  const error = useMemo((): ErrorState | null => {
    if (!queryError) return null;
    return classifyError(queryError);
  }, [queryError, classifyError]);

  // Retry mechanism
  const handleRetry = useCallback(() => {
    if (manualRetryCount < maxRetries) {
      setManualRetryCount(prev => prev + 1);
      setFallbackImageError(false);
      setImageLoaded(false);
      refetch();
    }
  }, [manualRetryCount, maxRetries, refetch]);
  
  // Cleanup on unmount
  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);
  
  // Loading state
  if (isLoading && showLoading) {
    return (
      <div ref={lazy && !priority ? lazyRef : undefined}>
        <CharacterRendererDetailSkeleton 
          size={size} 
          className={className}
        />
      </div>
    );
  }
  
  // Error state - show fallback avatar with error handling and retry option
  if (error || !imageUrl) {
    // Handle fallback image error - create ultimate fallback
    const handleFallbackError = () => {
      setFallbackImageError(true);
    };

    // If both main image and fallback failed, show inline SVG
    if (fallbackImageError) {
      return (
        <div 
          ref={lazy && !priority ? lazyRef : undefined}
          className={cn(sizeClasses[size], 'flex flex-col items-center justify-center bg-gray-100 border-2 border-dashed border-gray-300 rounded-lg', className)}
        >
          {/* Inline SVG fallback */}
          <svg width="48" height="48" viewBox="0 0 48 48" className="text-gray-400 mb-2">
            <circle cx="24" cy="18" r="6" fill="currentColor" opacity="0.3"/>
            <path d="M12 36c0-6.627 5.373-12 12-12s12 5.373 12 12" stroke="currentColor" strokeWidth="2" fill="none" opacity="0.3"/>
          </svg>
          <div className="text-xs text-gray-500 text-center px-2 mb-2">
            {error?.message || 'Character image unavailable'}
          </div>
          {error?.isRetryable && showRetryButton && manualRetryCount < maxRetries && (
            <Button
              size="sm"
              variant="outline"
              onClick={handleRetry}
              className="text-xs px-2 py-1"
            >
              Retry ({manualRetryCount + 1}/{maxRetries})
            </Button>
          )}
        </div>
      );
    }

    // Show fallback avatar with potential retry option
    return (
      <div 
        ref={lazy && !priority ? lazyRef : undefined}
        className={cn(sizeClasses[size], 'flex flex-col items-center justify-center', className)}
      >
        <div className="relative">
          <Image
            src={fallbackAvatar}
            alt={`${character.attributes.name} (fallback)`}
            width={sizeDimensions[size].width}
            height={sizeDimensions[size].height}
            className={cn('object-contain rounded-lg')}
            onError={handleFallbackError}
          />
        </div>
        {error && (
          <div className="mt-2">
            <div className="text-xs text-gray-500 text-center px-2 mb-1">
              {error.message}
            </div>
            {error.isRetryable && showRetryButton && manualRetryCount < maxRetries && (
              <Button
                size="sm"
                variant="outline"
                onClick={handleRetry}
                className="text-xs px-2 py-1"
              >
                Retry ({manualRetryCount + 1}/{maxRetries})
              </Button>
            )}
          </div>
        )}
      </div>
    );
  }
  
  // Success state - show character image
  return (
    <div 
      ref={lazy && !priority ? lazyRef : undefined}
      className={cn(sizeClasses[size], 'flex items-center justify-center', className)}
    >
      <Image
        src={imageUrl}
        alt={character.attributes.name}
        width={sizeDimensions[size].width}
        height={sizeDimensions[size].height}
{...priority && { priority: true }}
        loading={lazy ? 'lazy' : 'eager'}
        placeholder="blur"
        blurDataURL="data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjAwIiBoZWlnaHQ9IjIwMCIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj48ZGVmcz48bGluZWFyR3JhZGllbnQgaWQ9ImciIHgxPSIwJSIgeTE9IjAlIiB4Mj0iMTAwJSIgeTI9IjEwMCUiPjxzdG9wIG9mZnNldD0iMCUiIHN0b3AtY29sb3I9IiNmNmY2ZjYiLz48c3RvcCBvZmZzZXQ9IjEwMCUiIHN0b3AtY29sb3I9IiNlNGU0ZTQiLz48L2xpbmVhckdyYWRpZW50PjwvZGVmcz48cmVjdCB3aWR0aD0iMTAwJSIgaGVpZ2h0PSIxMDAlIiBmaWxsPSJ1cmwoI2cpIi8+PC9zdmc+"
        className={cn(
          'object-contain rounded-lg transition-opacity duration-300',
          imageLoaded || !enablePreload ? 'opacity-100' : 'opacity-0'
        )}
        onLoad={() => {
          if (mountedRef.current) {
            setImageLoaded(true);
            onImageLoad?.();
          }
        }}
        onError={() => {
          // Handle image load error by falling back to error state
          if (mountedRef.current) {
            const errorState = classifyError(new Error('Character image failed to load'));
            onImageError?.(new Error(errorState.message));
          }
        }}
      />
      {/* Show cache indicator in development */}
      {process.env.NODE_ENV === 'development' && cached && (
        <div className="absolute top-1 right-1 bg-green-500 text-white text-xs px-1 py-0.5 rounded">
          Cached
        </div>
      )}
    </div>
  );
}

// Export skeleton component for external use
export function CharacterRendererSkeleton({ 
  size = 'medium', 
  className 
}: { 
  size?: 'small' | 'medium' | 'large';
  className?: string;
}) {
  return (
    <CharacterRendererDetailSkeleton 
      size={size} 
      className={className}
    />
  );
}