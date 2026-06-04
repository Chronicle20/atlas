/**
 * React Query hook for character image loading with advanced caching and optimization
 */

import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useCallback, useRef, useEffect, useMemo } from 'react';
import {
  generateCharacterUrl,
  filterEquipment,
  canonicalLoadoutString,
  loadoutHash,
  resolveGender,
  type RenderOptions,
  type Stance,
} from '@/services/api/characterRender.service';

/**
 * Build a RenderOptions object omitting any keys that are undefined,
 * satisfying exactOptionalPropertyTypes.
 */
function compactRenderOptions(o: { stance?: Stance | undefined; frame?: number | undefined; resize?: number | undefined }): RenderOptions {
  const out: RenderOptions = {};
  if (o.stance !== undefined) out.stance = o.stance;
  if (o.frame !== undefined) out.frame = o.frame;
  if (o.resize !== undefined) out.resize = o.resize;
  return out;
}
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
  gcTime?: number;
  retry?: number;
  region?: string;
  majorVersion?: number;
  onSuccess?: (data: CharacterImageResult) => void;
  onError?: (error: Error) => void;
}

interface ImagePreloadResult {
  loaded: boolean;
  error: boolean;
  dimensions?: { width: number; height: number };
}

const DEFAULT_OPTIONS: Required<Omit<UseCharacterImageOptions, 'onSuccess' | 'onError' | 'region' | 'majorVersion'>> = {
  enabled: true,
  priority: false,
  lazy: true,
  staleTime: 60 * 60 * 1000, // 1 hour
  gcTime: 24 * 60 * 60 * 1000, // 24 hours
  retry: 3,
};

/**
 * Generate a stable query key for character image using a loadout hash.
 * The hash is derived from the canonical loadout string so different loadouts
 * cache separately while the key remains short and stable.
 */
export function generateQueryKey(character: MapleStoryCharacterData, options?: Partial<CharacterRenderOptions>): string[] {
  const stance = (options?.stance ?? 'stand1') as RenderOptions['stance'];
  const frame = options?.frame ?? 0;
  const resize = options?.resize ?? 2;
  const filtered = filterEquipment(
    Object.fromEntries(
      Object.entries(character.equipment).map(([k, v]) => [k, v as number])
    )
  );
  const items = Object.values(filtered) as number[];
  const gender = resolveGender(character.gender, character.face);
  const canonical = canonicalLoadoutString(
    character.tenant,
    character.region,
    character.majorVersion,
    character.minorVersion,
    character.skinColor,
    character.hair,
    character.face,
    stance ?? 'stand1',
    frame,
    resize,
    items,
    gender,
  );
  return ['character-image', loadoutHash(canonical)];
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
  const options = useMemo(() => ({ ...DEFAULT_OPTIONS, ...hookOptions }), [hookOptions]);
  const queryClient = useQueryClient();
  const preloadPromiseRef = useRef<Promise<ImagePreloadResult> | null>(null);
  
  const queryKey = generateQueryKey(character, renderOptions);

  // Main query for character image generation
  const query = useQuery({
    queryKey,
    queryFn: async (): Promise<CharacterImageResult> => {
      const url = generateCharacterUrl(
        character.tenant,
        character.region,
        character.majorVersion,
        character.minorVersion,
        {
          skin: character.skinColor,
          hair: character.hair,
          face: character.face,
          equipment: Object.fromEntries(
            Object.entries(character.equipment).map(([k, v]) => [k, v as number])
          ),
          gender: character.gender,
        },
        compactRenderOptions({
          stance: renderOptions?.stance,
          frame: renderOptions?.frame,
          resize: renderOptions?.resize,
        }),
      );

      const mergedOptions: CharacterRenderOptions = {
        hair: character.hair,
        face: character.face,
        skin: character.skinColor,
        equipment: character.equipment,
      };
      if (renderOptions?.stance !== undefined) mergedOptions.stance = renderOptions.stance;
      if (renderOptions?.frame !== undefined) mergedOptions.frame = renderOptions.frame;
      if (renderOptions?.resize !== undefined) mergedOptions.resize = renderOptions.resize;
      if (renderOptions?.renderMode !== undefined) mergedOptions.renderMode = renderOptions.renderMode;
      if (renderOptions?.flipX !== undefined) mergedOptions.flipX = renderOptions.flipX;

      const result: CharacterImageResult = {
        url,
        character,
        options: mergedOptions,
        cached: false,
      };

      // If this is a priority image, preload it immediately
      if (options.priority) {
        preloadPromiseRef.current = preloadImage(result.url);
      }

      return result;
    },
    enabled: options.enabled,
    staleTime: options.staleTime,
    gcTime: options.gcTime,
    retry: options.retry,
    // Keep failed queries in cache briefly for retry optimization
    retryOnMount: false,
    refetchOnWindowFocus: false,
  });

  // Handle success/error callbacks with useEffect
  useEffect(() => {
    if (query.isSuccess && query.data && options.onSuccess) {
      options.onSuccess(query.data);
    }
  }, [query.isSuccess, query.data, options]);

  useEffect(() => {
    if (query.isError && query.error && options.onError) {
      options.onError(query.error);
    }
  }, [query.isError, query.error, options]);

  // Preload management
  const preload = useCallback(async (): Promise<ImagePreloadResult | null> => {
    if (!query.data?.url) return null;

    // Reuse existing preload promise if available
    if (preloadPromiseRef.current) {
      return preloadPromiseRef.current;
    }

    preloadPromiseRef.current = preloadImage(query.data.url);
    return preloadPromiseRef.current;
  }, [query.data]);

  // Prefetch related images (e.g., different stances or scales)
  const prefetchVariants = useCallback((variants: Array<Partial<CharacterRenderOptions>>) => {
    variants.forEach((variant) => {
      const merged = { ...renderOptions, ...variant };
      const variantKey = generateQueryKey(character, merged);
      queryClient.prefetchQuery({
        queryKey: variantKey,
        queryFn: () => {
          const url = generateCharacterUrl(
            character.tenant,
            character.region,
            character.majorVersion,
            character.minorVersion,
            {
              skin: character.skinColor,
              hair: character.hair,
              face: character.face,
              equipment: Object.fromEntries(
                Object.entries(character.equipment).map(([k, v]) => [k, v as number])
              ),
              gender: character.gender,
            },
            compactRenderOptions({
              stance: merged.stance,
              frame: merged.frame,
              resize: merged.resize,
            }),
          );
          const result: CharacterImageResult = {
            url,
            character,
            options: {
              hair: character.hair,
              face: character.face,
              skin: character.skinColor,
              equipment: character.equipment,
              ...merged,
            },
            cached: false,
          };
          return Promise.resolve(result);
        },
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
        queryFn: () => {
          const url = generateCharacterUrl(
            character.tenant,
            character.region,
            character.majorVersion,
            character.minorVersion,
            {
              skin: character.skinColor,
              hair: character.hair,
              face: character.face,
              equipment: Object.fromEntries(
                Object.entries(character.equipment).map(([k, v]) => [k, v as number])
              ),
              gender: character.gender,
            },
            compactRenderOptions({
              stance: options?.stance,
              frame: options?.frame,
              resize: options?.resize,
            }),
          );
          const preloadOpts: CharacterRenderOptions = {
            hair: character.hair,
            face: character.face,
            skin: character.skinColor,
            equipment: character.equipment,
          };
          if (options?.stance !== undefined) preloadOpts.stance = options.stance;
          if (options?.frame !== undefined) preloadOpts.frame = options.frame;
          if (options?.resize !== undefined) preloadOpts.resize = options.resize;
          if (options?.renderMode !== undefined) preloadOpts.renderMode = options.renderMode;
          if (options?.flipX !== undefined) preloadOpts.flipX = options.flipX;
          const result: CharacterImageResult = {
            url,
            character,
            options: preloadOpts,
            cached: false,
          };
          return Promise.resolve(result);
        },
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
      loadingQueries: characterImageQueries.filter(q => q.state.status === 'pending').length,
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
    options?: Partial<CharacterRenderOptions>,
  ) => {
    const warmupPromises = characters.map(character => {
      const queryKey = generateQueryKey(character, options);

      return queryClient.prefetchQuery({
        queryKey,
        queryFn: () => {
          const url = generateCharacterUrl(
            character.tenant,
            character.region,
            character.majorVersion,
            character.minorVersion,
            {
              skin: character.skinColor,
              hair: character.hair,
              face: character.face,
              equipment: Object.fromEntries(
                Object.entries(character.equipment).map(([k, v]) => [k, v as number])
              ),
              gender: character.gender,
            },
            compactRenderOptions({
              stance: options?.stance,
              frame: options?.frame,
              resize: options?.resize,
            }),
          );
          const warmOpts: CharacterRenderOptions = {
            hair: character.hair,
            face: character.face,
            skin: character.skinColor,
            equipment: character.equipment,
          };
          if (options?.stance !== undefined) warmOpts.stance = options.stance;
          if (options?.frame !== undefined) warmOpts.frame = options.frame;
          if (options?.resize !== undefined) warmOpts.resize = options.resize;
          if (options?.renderMode !== undefined) warmOpts.renderMode = options.renderMode;
          if (options?.flipX !== undefined) warmOpts.flipX = options.flipX;
          const result: CharacterImageResult = {
            url,
            character,
            options: warmOpts,
            cached: false,
          };
          return Promise.resolve(result);
        },
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