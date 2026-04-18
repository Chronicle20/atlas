/**
 * Character Image Cache Service Worker Utilities
 * Provides utilities for managing the character image cache service worker
 */

interface CacheStats {
  totalEntries: number;
  totalSize: number;
  entries: Array<{
    url: string;
    size: number;
    timestamp: number;
  }>;
}

interface ServiceWorkerMessage {
  type: string;
  urls?: string[];
}

class CharacterCacheManager {
  private swRegistration: ServiceWorkerRegistration | null = null;

  /**
   * Initialize the service worker for character image caching
   */
  async initialize(): Promise<boolean> {
    if (typeof window === 'undefined' || !('serviceWorker' in navigator)) {
      console.warn('Service Workers not supported');
      return false;
    }

    try {
      this.swRegistration = await navigator.serviceWorker.register('/sw-character-cache.js', {
        scope: '/',
      });

      console.log('Character cache service worker registered successfully');

      // Listen for service worker updates
      this.swRegistration.addEventListener('updatefound', () => {
        const newWorker = this.swRegistration?.installing;
        if (newWorker) {
          newWorker.addEventListener('statechange', () => {
            if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
              console.log('New character cache service worker available');
              // Optionally notify user of update
            }
          });
        }
      });

      return true;
    } catch (error) {
      console.error('Failed to register character cache service worker:', error);
      return false;
    }
  }

  /**
   * Send a message to the service worker
   */
  private async sendMessage(message: ServiceWorkerMessage): Promise<CacheStats | void> {
    if (!this.swRegistration?.active) {
      throw new Error('Service worker not available');
    }

    return new Promise((resolve, reject) => {
      const messageChannel = new MessageChannel();
      
      messageChannel.port1.onmessage = (event) => {
        if (event.data.success) {
          resolve(event.data.stats || event.data);
        } else {
          reject(new Error(event.data.error || 'Service worker message failed'));
        }
      };

      this.swRegistration!.active!.postMessage(message, [messageChannel.port2]);
    });
  }

  /**
   * Clear all cached character images
   */
  async clearCache(): Promise<void> {
    try {
      await this.sendMessage({ type: 'CLEAR_CHARACTER_CACHE' });
      console.log('Character image cache cleared');
    } catch (error) {
      console.error('Failed to clear character image cache:', error);
      throw error;
    }
  }

  /**
   * Get cache statistics
   */
  async getCacheStats(): Promise<CacheStats> {
    try {
      const stats = await this.sendMessage({ type: 'GET_CACHE_STATS' });
      return stats as CacheStats;
    } catch (error) {
      console.error('Failed to get cache stats:', error);
      throw error;
    }
  }

  /**
   * Preload character images
   */
  async preloadImages(urls: string[]): Promise<void> {
    if (urls.length === 0) return;

    try {
      await this.sendMessage({ 
        type: 'PRELOAD_IMAGES',
        urls,
      });
      console.log(`Preloaded ${urls.length} character images`);
    } catch (error) {
      console.error('Failed to preload character images:', error);
      throw error;
    }
  }

  /**
   * Check if service worker is available and active
   */
  isAvailable(): boolean {
    return !!(this.swRegistration?.active);
  }

  /**
   * Get formatted cache size
   */
  formatCacheSize(bytes: number): string {
    const units = ['B', 'KB', 'MB', 'GB'];
    let size = bytes;
    let unitIndex = 0;

    while (size >= 1024 && unitIndex < units.length - 1) {
      size /= 1024;
      unitIndex++;
    }

    return `${size.toFixed(1)} ${units[unitIndex]}`;
  }

  /**
   * Update service worker
   */
  async update(): Promise<void> {
    if (this.swRegistration) {
      await this.swRegistration.update();
    }
  }

  /**
   * Unregister service worker
   */
  async unregister(): Promise<void> {
    if (this.swRegistration) {
      await this.swRegistration.unregister();
      this.swRegistration = null;
      console.log('Character cache service worker unregistered');
    }
  }
}

// Singleton instance
export const characterCacheManager = new CharacterCacheManager();

/**
 * React hook for character cache management
 */
import { useState, useEffect, useCallback } from 'react';

interface UseCharacterCacheReturn {
  isAvailable: boolean;
  cacheStats: CacheStats | null;
  isLoading: boolean;
  error: string | null;
  clearCache: () => Promise<void>;
  preloadImages: (urls: string[]) => Promise<void>;
  refreshStats: () => Promise<void>;
}

export function useCharacterCache(): UseCharacterCacheReturn {
  const [isAvailable, setIsAvailable] = useState(false);
  const [cacheStats, setCacheStats] = useState<CacheStats | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Initialize service worker on mount
  useEffect(() => {
    const initializeCache = async () => {
      try {
        setIsLoading(true);
        setError(null);
        
        const available = await characterCacheManager.initialize();
        setIsAvailable(available);
        
        if (available) {
          const stats = await characterCacheManager.getCacheStats();
          setCacheStats(stats);
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to initialize cache');
        console.error('Cache initialization error:', err);
      } finally {
        setIsLoading(false);
      }
    };

    initializeCache();
  }, []);

  const clearCache = useCallback(async () => {
    try {
      setError(null);
      await characterCacheManager.clearCache();
      setCacheStats({ totalEntries: 0, totalSize: 0, entries: [] });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to clear cache');
      throw err;
    }
  }, []);

  const refreshStats = useCallback(async () => {
    if (!isAvailable) return;
    
    try {
      setError(null);
      const stats = await characterCacheManager.getCacheStats();
      setCacheStats(stats);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to refresh stats');
      throw err;
    }
  }, [isAvailable]);

  const preloadImages = useCallback(async (urls: string[]) => {
    try {
      setError(null);
      await characterCacheManager.preloadImages(urls);
      // Refresh stats after preloading
      await refreshStats();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to preload images');
      throw err;
    }
  }, [refreshStats]);

  return {
    isAvailable,
    cacheStats,
    isLoading,
    error,
    clearCache,
    preloadImages,
    refreshStats,
  };
}

/**
 * Utility function to generate character image URLs for preloading
 */
export function generateCharacterImageUrls(
  characters: Array<{
    hair: number;
    face: number;
    skinColor: number;
    equipment: Record<string, number>;
  }>,
  options: {
    scales?: number[];
    stances?: ('stand1' | 'stand2')[];
  } = {}
): string[] {
  const { scales = [1, 2], stances = ['stand1'] } = options;
  const urls: string[] = [];

  for (const character of characters) {
    for (const scale of scales) {
      for (const stance of stances) {
        // This would use the same URL generation logic as the MapleStory service
        // Simplified version for example:
        const equipmentString = Object.entries(character.equipment)
          .map(([, itemId]) => `${itemId}:0`)
          .join(',');
        
        const url = `https://maplestory.io/api/GMS/214/character/center/${character.skinColor}/${character.hair}:0,${character.face}:0,${equipmentString}/${stance}/0?resize=${scale}`;
        urls.push(url);
      }
    }
  }

  return urls;
}

export default characterCacheManager;