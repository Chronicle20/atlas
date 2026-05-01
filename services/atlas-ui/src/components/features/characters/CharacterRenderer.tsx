import { useState, useEffect, useRef, useCallback, useMemo } from 'react';
import { Button } from '@/components/ui/button';
import { CharacterRendererDetailSkeleton } from '@/components/common/skeletons/CharacterDetailSkeleton';
import { useCharacterImage } from '@/lib/hooks/useCharacterImage';
import { useLazyLoad } from '@/lib/hooks/useIntersectionObserver';
import { characterToLoadout } from '@/services/api/characterRender.service';
import { useTenant } from '@/context/tenant-context';
import type { Asset } from '@/services/api/inventory.service';
import type { CharacterRendererProps, MapleStoryCharacterData } from '@/types/models/maplestory';
import { cn } from '@/lib/utils';
import { getImageLoadingStrategy } from '@/lib/utils/image';

interface CharacterRendererComponentProps extends Omit<CharacterRendererProps, 'equipment'> {
  inventory?: Asset[];
  size?: 'small' | 'medium' | 'large';
  maxRetries?: number;
  showRetryButton?: boolean;
  priority?: boolean;
  lazy?: boolean;
  enablePreload?: boolean;
  prefetchVariants?: boolean;
  region?: string;
  majorVersion?: number;
  /**
   * Layout mode controlling how the rendered character fits its container.
   * - `tight` (default): self-sized via `size`, `object-contain`, character centered. Best for hero/detail views.
   * - `platform`: fills parent (`w-full h-full`), `object-cover` with feet anchored to the parent's bottom
   *   edge via a pixel-scan that locates the bottom-most non-transparent row. Best for tile grids where
   *   uniform foot alignment matters more than preserving headgear extents.
   */
  frameMode?: 'tight' | 'platform';
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
  large: 'w-32 h-32'
};

/**
 * Pixel-scan the loaded image, find the bottom-most non-transparent row, and
 * compute an `objectPosition` that anchors that row to the container's bottom
 * edge under `object-fit: cover`. Falls back silently on CORS-tainted canvas.
 */
function anchorPlatformFeet(
  img: HTMLImageElement,
  setObjectPosition: (p: string) => void,
): void {
  const W = img.naturalWidth;
  const H = img.naturalHeight;
  if (!W || !H) return;
  const canvas = document.createElement('canvas');
  canvas.width = W;
  canvas.height = H;
  const ctx = canvas.getContext('2d');
  if (!ctx) return;
  try {
    ctx.drawImage(img, 0, 0);
    const data = ctx.getImageData(0, 0, W, H).data;
    let footRow = H - 1;
    outer: for (let y = H - 1; y >= 0; y--) {
      for (let x = 0; x < W; x++) {
        if (data[(y * W + x) * 4 + 3]! > 0) {
          footRow = y;
          break outer;
        }
      }
    }
    const bottomMarginSrc = (H - 1) - footRow;
    if (bottomMarginSrc <= 0) return;
    const parent = img.parentElement;
    if (!parent) return;
    const cw = parent.clientWidth;
    const ch = parent.clientHeight;
    if (cw <= 0 || ch <= 0) return;
    const coverScale = Math.max(cw / W, ch / H);
    const shiftPx = bottomMarginSrc * coverScale;
    setObjectPosition(`center calc(100% + ${shiftPx}px)`);
  } catch {
    // CORS-tainted canvas — leave default `object-position: bottom`.
  }
}

const sizeDimensions = {
  small: { width: 128, height: 128 },
  medium: { width: 192, height: 192 },
  large: { width: 128, height: 128 }
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
  region: _region,
  majorVersion: _majorVersion,
  frameMode = 'tight',
}: CharacterRendererComponentProps) {
  const [fallbackImageError, setFallbackImageError] = useState(false);
  const [imageLoaded, setImageLoaded] = useState(false);
  const [manualRetryCount, setManualRetryCount] = useState(0);
  const [platformObjectPosition, setPlatformObjectPosition] = useState<string | undefined>(undefined);
  const mountedRef = useRef(true);
  const imgElRef = useRef<HTMLImageElement | null>(null);

  const { activeTenant } = useTenant();

  // Lazy loading support with intersection observer
  const { shouldLoad, ref: lazyRef } = useLazyLoad<HTMLDivElement>({
    enabled: lazy && !priority,
    rootMargin: '200px', // Start loading 200px before entering viewport
  });

  // Convert Character model to MapleStoryCharacterData, including tenant context
  const mapleStoryData = useMemo((): MapleStoryCharacterData => {
    const loadout = characterToLoadout(character, inventory);
    return {
      id: character.id,
      name: character.attributes.name,
      level: character.attributes.level,
      jobId: character.attributes.jobId,
      hair: loadout.hair,
      face: loadout.face,
      skinColor: loadout.skin,
      gender: character.attributes.gender,
      equipment: loadout.equipment,
      tenant: activeTenant?.id ?? '',
      region: activeTenant?.attributes.region ?? '',
      majorVersion: activeTenant?.attributes.majorVersion ?? 0,
      minorVersion: activeTenant?.attributes.minorVersion ?? 0,
    };
  }, [character, inventory, activeTenant]);

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
      // Disable the query until a tenant is available
      enabled: !!activeTenant && (priority || !lazy || shouldLoad),
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
  
  // Skeleton while tenant context is still loading
  if (!activeTenant) {
    return (
      <div ref={lazy && !priority ? lazyRef : undefined}>
        <CharacterRendererDetailSkeleton
          size={size}
          className={className}
        />
      </div>
    );
  }

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
          <img
            src={fallbackAvatar}
            alt={`${character.attributes.name} (fallback)`}
            width={sizeDimensions[size].width}
            height={sizeDimensions[size].height}
            className={cn('object-contain rounded-lg')}
            onError={handleFallbackError}
            data-testid="character-image"
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
  const isPlatform = frameMode === 'platform';
  return (
    <div
      ref={lazy && !priority ? lazyRef : undefined}
      className={cn(
        isPlatform ? 'w-full h-full' : sizeClasses[size],
        'flex',
        isPlatform ? 'items-end justify-center' : 'items-center justify-center',
        className,
      )}
    >
      <img
        ref={imgElRef}
        src={imageUrl}
        alt={character.attributes.name}
        width={sizeDimensions[size].width}
        height={sizeDimensions[size].height}
        loading={lazy && !priority ? getImageLoadingStrategy() : 'eager'}
        crossOrigin={isPlatform ? 'anonymous' : undefined}
        className={cn(
          isPlatform
            ? 'w-full h-full object-cover'
            : 'object-contain',
          'rounded-lg transition-opacity duration-300 opacity-100',
        )}
        style={
          isPlatform && platformObjectPosition
            ? { objectPosition: platformObjectPosition }
            : undefined
        }
        onLoad={(e) => {
          if (mountedRef.current) {
            setImageLoaded(true);
            onImageLoad?.();
            if (isPlatform) {
              anchorPlatformFeet(e.currentTarget, setPlatformObjectPosition);
            }
          }
        }}
        onError={() => {
          // Handle image load error by falling back to error state
          if (mountedRef.current) {
            const errorState = classifyError(new Error('Character image failed to load'));
            onImageError?.(new Error(errorState.message));
          }
        }}
        data-testid="character-image"
      />
      {/* Show cache indicator in development */}
      {import.meta.env.DEV && cached && (
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