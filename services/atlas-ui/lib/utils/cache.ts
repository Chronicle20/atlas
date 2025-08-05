/**
 * Client-side caching utilities for performance optimization
 */

/**
 * Cache configuration for different resource types
 */
export const CACHE_STRATEGIES = {
  // MapleStory.io API responses - cache for longer periods
  MAPLESTORY_API: {
    cacheName: 'maplestory-api-v1',
    maxAge: 24 * 60 * 60 * 1000, // 24 hours
    strategy: 'cache-first',
  },
  
  // NPC images - cache aggressively
  NPC_IMAGES: {
    cacheName: 'npc-images-v1',
    maxAge: 7 * 24 * 60 * 60 * 1000, // 7 days
    strategy: 'cache-first',
  },
  
  // Static assets
  STATIC_ASSETS: {
    cacheName: 'static-assets-v1',
    maxAge: 30 * 24 * 60 * 60 * 1000, // 30 days
    strategy: 'cache-first',
  },
} as const;

/**
 * Enhanced fetch with intelligent caching
 */
export async function cachedFetch(
  url: string,
  options: RequestInit & { 
    cacheStrategy?: keyof typeof CACHE_STRATEGIES;
    bypassCache?: boolean;
  } = {}
): Promise<Response> {
  const { cacheStrategy = 'MAPLESTORY_API', bypassCache = false, ...fetchOptions } = options;
  
  // If Service Worker isn't supported or bypass is requested, use regular fetch
  if (!('serviceWorker' in navigator) || bypassCache) {
    return fetch(url, fetchOptions);
  }
  
  const strategy = CACHE_STRATEGIES[cacheStrategy];
  
  // Add cache-related headers
  const enhancedHeaders = {
    ...fetchOptions.headers,
    'Cache-Control': bypassCache 
      ? 'no-cache, no-store, must-revalidate'
      : `public, max-age=${Math.floor(strategy.maxAge / 1000)}`,
  };
  
  return fetch(url, {
    ...fetchOptions,
    headers: enhancedHeaders,
  });
}

/**
 * Clear specific cache by name
 */
export async function clearCache(cacheName?: string): Promise<void> {
  if (!('caches' in window)) return;
  
  if (cacheName) {
    await caches.delete(cacheName);
  } else {
    // Clear all atlas-related caches
    const cacheNames = await caches.keys();
    const atlasCaches = cacheNames.filter(name => 
      name.includes('maplestory') || name.includes('npc') || name.includes('atlas')
    );
    
    await Promise.all(atlasCaches.map(name => caches.delete(name)));
  }
}

/**
 * Get cache storage usage information
 */
export async function getCacheStorageInfo(): Promise<{
  usage: number;
  quota: number;
  percentage: number;
  caches: Array<{ name: string; size: number }>;
}> {
  if (!('navigator' in window) || !('storage' in navigator) || !('estimate' in navigator.storage)) {
    return {
      usage: 0,
      quota: 0,
      percentage: 0,
      caches: [],
    };
  }
  
  const estimate = await navigator.storage.estimate();
  const usage = estimate.usage || 0;
  const quota = estimate.quota || 0;
  const percentage = quota > 0 ? (usage / quota) * 100 : 0;
  
  const cacheList: Array<{ name: string; size: number }> = [];
  
  if ('caches' in window) {
    const cacheNames = await window.caches.keys();
    
    for (const cacheName of cacheNames) {
      try {
        const cache = await window.caches.open(cacheName);
        const keys = await cache.keys();
        cacheList.push({
          name: cacheName,
          size: keys.length,
        });
      } catch (error) {
        console.warn(`Failed to get cache info for ${cacheName}:`, error);
      }
    }
  }
  
  return {
    usage,
    quota,
    percentage,
    caches: cacheList,
  };
}

/**
 * Preload critical resources
 */
export async function preloadCriticalResources(resources: Array<{
  url: string;
  type: 'image' | 'data' | 'script';
  priority?: 'high' | 'medium' | 'low';
}>): Promise<void> {
  const preloadPromises = resources.map(async ({ url, type, priority = 'medium' }) => {
    try {
      // Create appropriate request based on resource type
      const requestInit: RequestInit = {
        mode: 'cors',
        credentials: 'same-origin',
      };
      
      // Add type-specific headers
      switch (type) {
        case 'image':
          requestInit.headers = {
            'Accept': 'image/*',
            'Priority': priority,
          };
          break;
        case 'data':
          requestInit.headers = {
            'Accept': 'application/json',
            'Priority': priority,
          };
          break;
        case 'script':
          requestInit.headers = {
            'Accept': 'application/javascript, text/javascript',
            'Priority': priority,
          };
          break;
      }
      
      const response = await cachedFetch(url, requestInit);
      
      // For images, we can also add them to the browser's cache
      if (type === 'image' && response.ok) {
        const blob = await response.blob();
        const imageUrl = URL.createObjectURL(blob);
        
        // Preload the image
        return new Promise<void>((resolve, reject) => {
          const img = new Image();
          img.onload = () => {
            URL.revokeObjectURL(imageUrl);
            resolve();
          };
          img.onerror = () => {
            URL.revokeObjectURL(imageUrl);
            reject(new Error(`Failed to preload image: ${url}`));
          };
          img.src = imageUrl;
        });
      }
      
      return response;
    } catch (error) {
      console.warn(`Failed to preload resource ${url}:`, error);
    }
  });
  
  await Promise.allSettled(preloadPromises);
}

/**
 * React hook for cache management
 */
import { useCallback } from 'react';

export function useCacheManager() {
  const clearSpecificCache = useCallback(async (cacheName: string) => {
    await clearCache(cacheName);
  }, []);
  
  const clearAllCaches = useCallback(async () => {
    await clearCache();
  }, []);
  
  const getCacheInfo = useCallback(async () => {
    return getCacheStorageInfo();
  }, []);
  
  const preloadResources = useCallback(async (resources: Parameters<typeof preloadCriticalResources>[0]) => {
    await preloadCriticalResources(resources);
  }, []);
  
  return {
    clearSpecificCache,
    clearAllCaches,
    getCacheInfo,
    preloadResources,
  };
}