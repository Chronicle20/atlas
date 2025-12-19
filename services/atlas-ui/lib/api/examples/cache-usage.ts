/**
 * Examples of using the new response caching features
 * 
 * This file demonstrates how to use the caching functionality
 * that was added to the API client.
 */

import { api, cache } from '../client';

// Example 1: Basic caching with default options (5 minutes)
export async function fetchUserProfile(userId: string) {
  return api.get(`/users/${userId}`, {
    cacheConfig: cache.defaultOptions()
  });
}

// Example 2: Short-lived cache for frequently changing data (1 minute)
export async function fetchUnreadNotificationCount() {
  return api.get('/notifications/unread-count', {
    cacheConfig: cache.shortLived()
  });
}

// Example 3: Long-lived cache for static data (30 minutes with stale-while-revalidate)
export async function fetchSystemConfiguration() {
  return api.get('/system/config', {
    cacheConfig: cache.longLived()
  });
}

// Example 4: Custom TTL for specific use cases
export async function fetchTenantSettings(tenantId: string) {
  return api.get(`/tenants/${tenantId}/settings`, {
    cacheConfig: cache.withTTL(15) // 15 minutes
  });
}

// Example 5: Stale-while-revalidate for better UX
export async function fetchRegionsList() {
  return api.get('/regions', {
    cacheConfig: cache.staleWhileRevalidate(10 * 60 * 1000, 2 * 60 * 1000) // 10min TTL, 2min stale
  });
}

// Example 6: Custom cache key prefix for namespacing
export async function fetchGameMaps() {
  return api.get('/maps', {
    cacheConfig: cache.withPrefix('game-data', 5 * 60 * 1000)
  });
}

// Example 7: Disable caching for sensitive or real-time data
export async function fetchSecurityTokens() {
  return api.get('/auth/tokens', {
    cacheConfig: cache.disable()
  });
}

// Example 8: Complex caching configuration
export async function fetchCharacterData(characterId: string) {
  return api.get(`/characters/${characterId}`, {
    cacheConfig: {
      ttl: 3 * 60 * 1000,        // 3 minutes
      keyPrefix: 'characters',    // namespace cache keys
      staleWhileRevalidate: true, // serve stale data while revalidating
      maxStaleTime: 30 * 1000     // max 30 seconds stale
    }
  });
}

// Example 9: Cache management utilities
export class ApiCacheManager {
  /**
   * Clear all cached API responses
   */
  static clearAll(): void {
    api.clearCache();
  }

  /**
   * Clear cache for specific feature
   */
  static clearCharacterCache(): void {
    api.clearCacheByPattern('characters');
  }

  /**
   * Get cache usage statistics
   */
  static getStats() {
    return api.getCacheStats();
  }

  /**
   * Check if cache is healthy (not too many stale entries)
   */
  static isCacheHealthy(): boolean {
    const stats = api.getCacheStats();
    const staleCount = stats.entries.filter(entry => entry.isStale).length;
    const stalePercentage = stats.size > 0 ? (staleCount / stats.size) * 100 : 0;
    
    // Consider cache healthy if less than 50% of entries are stale
    return stalePercentage < 50;
  }
}

// Example 10: Usage in React components with cleanup
export class ComponentCacheManager {
  /**
   * Cleanup cache when component unmounts
   */
  static cleanup(): void {
    // Clear any component-specific cache
    api.clearCacheByPattern('component-*');
  }

  /**
   * Setup cache for component data
   */
  static async fetchComponentData(componentId: string) {
    return api.get(`/components/${componentId}/data`, {
      cacheConfig: cache.withPrefix(`component-${componentId}`, 2 * 60 * 1000)
    });
  }
}

// Usage examples:

/*
// In a React component:
useEffect(() => {
  fetchUserProfile('123').then(setUser);
  
  return () => {
    // Cleanup when component unmounts
    ComponentCacheManager.cleanup();
  };
}, []);

// In a service:
class UserService {
  async getProfile(userId: string) {
    // Uses 5-minute cache
    return fetchUserProfile(userId);
  }

  async getSettings(tenantId: string) {
    // Uses 15-minute cache
    return fetchTenantSettings(tenantId);
  }
}

// For debugging:
console.log('Cache stats:', ApiCacheManager.getStats());
console.log('Cache healthy:', ApiCacheManager.isCacheHealthy());
*/