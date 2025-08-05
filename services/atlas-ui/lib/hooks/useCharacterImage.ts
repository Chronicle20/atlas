/**
 * React Query hook for character image loading with advanced caching and optimization
 */

import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useCallback, useRef, useEffect } from 'react';
import { mapleStoryService } from '@/services/api/maplestory.service';
import type { 
  MapleStoryCharacterData, 
  CharacterRenderOptions, 
  CharacterImageResult 
} from '@/types/models/maplestory';

interface UseCharacterImageOptions {
  enabled?: boolean;
  priority?: boolean;
  lazy?: boolean;
  staleTime?: number;
  cacheTime?: number;
  retry?: number;
  onSuccess?: (data: CharacterImageResult) => void;
  onError?: (error: Error) => void;
}

interface ImagePreloadResult {
  loaded: boolean;
  error: boolean;
  dimensions?: { width: number; height: number };
}

const DEFAULT_OPTIONS: Required<Omit<UseCharacterImageOptions, 'onSuccess' | 'onError'>> = {
  enabled: true,
  priority: false,
  lazy: true,
  staleTime: 60 * 60 * 1000, // 1 hour
  cacheTime: 24 * 60 * 60 * 1000, // 24 hours
  retry: 3,
};

/**
 * Generate a stable query key for character image
 */
function generateQueryKey(character: MapleStoryCharacterData, options?: Partial<CharacterRenderOptions>): string[] {
  const baseKey = [
    'character-image',
    character.id.toString(),
    character.hair.toString(),
    character.face.toString(),
    character.skinColor.toString(),
    JSON.stringify(character.equipment),
  ];

  if (options) {
    baseKey.push(
      options.resize?.toString() || '2',
      options.stance || 'auto',
      options.renderMode || 'default',
      options.frame?.toString() || '0',
      options.flipX?.toString() || 'false'
    );
  }

  return baseKey;
}

/**
 * Preload image and get dimensions
 */
function preloadImage(url: string): Promise<ImagePreloadResult> {
  return new Promise((resolve) => {
    const img = new Image();
    
    img.onload = () => {
      resolve({
        loaded: true,
        error: false,
        dimensions: {
          width: img.naturalWidth,
          height: img.naturalHeight,
        },
      });
    };
    
    img.onerror = () => {
      resolve({
        loaded: false,
        error: true,
      });
    };
    
    // Start loading
    img.src = url;
  });
}

/**
 * Hook for character image loading with performance optimizations
 */
export function useCharacterImage(
  character: MapleStoryCharacterData,
  renderOptions?: Partial<CharacterRenderOptions>,
  hookOptions: UseCharacterImageOptions = {}
) {
  const options = { ...DEFAULT_OPTIONS, ...hookOptions };
  const queryClient = useQueryClient();
  const preloadPromiseRef = useRef<Promise<ImagePreloadResult> | null>(null);
  
  const queryKey = generateQueryKey(character, renderOptions);
  
  // Main query for character image generation
  const query = useQuery({
    queryKey,
    queryFn: async (): Promise<CharacterImageResult> => {
      const result = await mapleStoryService.generateCharacterImage(character, renderOptions);
      
      // If this is a priority image, preload it immediately
      if (options.priority) {
        preloadPromiseRef.current = preloadImage(result.url);
      }
      
      return result;
    },
    enabled: options.enabled,
    staleTime: options.staleTime,
    cacheTime: options.cacheTime,
    retry: options.retry,
    onSuccess: options.onSuccess,
    onError: options.onError,
    // Keep failed queries in cache briefly for retry optimization
    retryOnMount: false,
    refetchOnWindowFocus: false,
  });

  // Preload management
  const preload = useCallback(async (): Promise<ImagePreloadResult | null> => {
    if (!query.data?.url) return null;
    
    // Reuse existing preload promise if available
    if (preloadPromiseRef.current) {
      return preloadPromiseRef.current;
    }
    
    preloadPromiseRef.current = preloadImage(query.data.url);
    return preloadPromiseRef.current;
  }, [query.data?.url]);

  // Prefetch related images (e.g., different stances or scales)
  const prefetchVariants = useCallback((variants: Array<Partial<CharacterRenderOptions>>) => {
    variants.forEach((variant) => {
      const variantKey = generateQueryKey(character, { ...renderOptions, ...variant });
      queryClient.prefetchQuery({
        queryKey: variantKey,
        queryFn: () => mapleStoryService.generateCharacterImage(character, { ...renderOptions, ...variant }),
        staleTime: options.staleTime,
      });
    });
  }, [character, renderOptions, queryClient, options.staleTime]);

  // Invalidate cache for this character
  const invalidate = useCallback(() => {
    queryClient.invalidateQueries({
      queryKey: ['character-image', character.id.toString()],
    });
  }, [queryClient, character.id]);

  // Clear preload reference on unmount or data change
  useEffect(() => {
    return () => {
      preloadPromiseRef.current = null;
    };
  }, [query.data?.url]);

  return {
    ...query,
    preload,
    prefetchVariants,
    invalidate,
    imageUrl: query.data?.url,
    cached: query.data?.cached ?? false,
    character: query.data?.character,
    renderOptions: query.data?.options,
  };
}

/**
 * Hook for preloading multiple character images
 */
export function useCharacterImagePreloader() {
  const queryClient = useQueryClient();

  const preloadImages = useCallback(async (
    characters: Array<{
      character: MapleStoryCharacterData;
      options?: Partial<CharacterRenderOptions>;
    }>
  ) => {
    const preloadPromises = characters.map(({ character, options }) => {
      const queryKey = generateQueryKey(character, options);
      
      return queryClient.prefetchQuery({
        queryKey,
        queryFn: () => mapleStoryService.generateCharacterImage(character, options),
        staleTime: DEFAULT_OPTIONS.staleTime,
      });
    });

    return Promise.allSettled(preloadPromises);
  }, [queryClient]);

  const preloadImageUrls = useCallback(async (urls: string[]) => {
    const preloadPromises = urls.map(url => preloadImage(url));
    return Promise.allSettled(preloadPromises);
  }, []);

  return {
    preloadImages,
    preloadImageUrls,
  };
}

/**
 * Hook for managing character image cache
 */
export function useCharacterImageCache() {
  const queryClient = useQueryClient();

  const getCacheStats = useCallback(() => {
    const cache = queryClient.getQueryCache();
    const characterImageQueries = cache.findAll({ queryKey: ['character-image'] });
    
    return {
      totalQueries: characterImageQueries.length,
      activeQueries: characterImageQueries.filter(q => q.state.status === 'success').length,
      errorQueries: characterImageQueries.filter(q => q.state.status === 'error').length,
      loadingQueries: characterImageQueries.filter(q => q.state.status === 'loading').length,
    };
  }, [queryClient]);

  const clearCache = useCallback((characterId?: number) => {
    if (characterId) {
      queryClient.removeQueries({
        queryKey: ['character-image', characterId.toString()],
      });
    } else {
      queryClient.removeQueries({
        queryKey: ['character-image'],
      });
    }
  }, [queryClient]);

  const warmCache = useCallback(async (
    characters: MapleStoryCharacterData[],
    options?: Partial<CharacterRenderOptions>
  ) => {
    const warmupPromises = characters.map(character => {
      const queryKey = generateQueryKey(character, options);
      
      return queryClient.prefetchQuery({
        queryKey,
        queryFn: () => mapleStoryService.generateCharacterImage(character, options),
        staleTime: DEFAULT_OPTIONS.staleTime,
      });
    });

    return Promise.allSettled(warmupPromises);
  }, [queryClient]);

  return {
    getCacheStats,
    clearCache,
    warmCache,
  };
}