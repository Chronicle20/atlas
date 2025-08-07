/**
 * Dynamic label resolver service with caching for breadcrumb navigation
 * 
 * Provides efficient resolution of entity IDs to human-readable names for breadcrumbs.
 * Uses React Query for caching and supports all Atlas entity types with tenant context.
 */

import type { Tenant } from '@/types/models/tenant';

// Types for resolver functions
export type EntityResolver<T = string> = (
  tenant: Tenant,
  entityId: string,
  options?: ResolverOptions
) => Promise<T>;

export interface ResolverOptions {
  /** Fallback value if resolution fails */
  fallback?: string;
  /** Maximum time to wait for resolution (ms) */
  timeout?: number;
  /** Whether to use cached values */
  useCache?: boolean;
}

export interface ResolvedLabel {
  /** The resolved human-readable label */
  label: string;
  /** Whether this came from cache or fresh fetch */
  fromCache: boolean;
  /** Timestamp of when this was resolved */
  resolvedAt: number;
  /** Whether resolution failed and fallback was used */
  isFallback: boolean;
}

// Entity type mapping for resolver selection
export enum EntityType {
  ACCOUNT = 'account',
  CHARACTER = 'character',
  GUILD = 'guild',
  NPC = 'npc',
  TEMPLATE = 'template',
  TENANT = 'tenant',
}

// Default resolver options
const DEFAULT_OPTIONS: Required<ResolverOptions> = {
  fallback: 'Unknown',
  timeout: 5000, // 5 seconds
  useCache: true,
};

// Cache configuration
const CACHE_CONFIG = {
  // Cache TTL in milliseconds
  TTL: {
    [EntityType.ACCOUNT]: 10 * 60 * 1000,      // 10 minutes
    [EntityType.CHARACTER]: 5 * 60 * 1000,     // 5 minutes (changes more frequently)  
    [EntityType.GUILD]: 15 * 60 * 1000,        // 15 minutes
    [EntityType.NPC]: 30 * 60 * 1000,          // 30 minutes (rarely changes)
    [EntityType.TEMPLATE]: 30 * 60 * 1000,     // 30 minutes (rarely changes)
    [EntityType.TENANT]: 60 * 60 * 1000,       // 1 hour (very stable)
  },
  // Maximum cache size per entity type
  MAX_SIZE: 1000,
};

// In-memory cache for quick lookups
interface CacheEntry {
  label: string;
  timestamp: number;
  tenantId: string;
}

class ResolverCache {
  private cache = new Map<string, Map<string, CacheEntry>>();

  private getCacheKey(entityType: EntityType, entityId: string, tenantId: string): string {
    return `${tenantId}:${entityType}:${entityId}`;
  }

  get(entityType: EntityType, entityId: string, tenantId: string): CacheEntry | null {
    const typeCache = this.cache.get(entityType);
    if (!typeCache) return null;

    const key = this.getCacheKey(entityType, entityId, tenantId);
    const entry = typeCache.get(key);
    
    if (!entry) return null;

    // Check if entry is expired
    const ttl = CACHE_CONFIG.TTL[entityType];
    if (Date.now() - entry.timestamp > ttl) {
      typeCache.delete(key);
      return null;
    }

    return entry;
  }

  set(entityType: EntityType, entityId: string, tenantId: string, label: string): void {
    if (!this.cache.has(entityType)) {
      this.cache.set(entityType, new Map());
    }

    const typeCache = this.cache.get(entityType)!;
    const key = this.getCacheKey(entityType, entityId, tenantId);

    // Implement LRU-style cache size management
    if (typeCache.size >= CACHE_CONFIG.MAX_SIZE) {
      // Remove oldest entries
      const entries = Array.from(typeCache.entries());
      entries.sort(([, a], [, b]) => a.timestamp - b.timestamp);
      
      // Remove oldest 10% of entries
      const toRemove = Math.floor(CACHE_CONFIG.MAX_SIZE * 0.1);
      for (let i = 0; i < toRemove && i < entries.length; i++) {
        const entry = entries[i];
        if (entry) {
          typeCache.delete(entry[0]);
        }
      }
    }

    typeCache.set(key, {
      label,
      timestamp: Date.now(),
      tenantId,
    });
  }

  clear(entityType?: EntityType): void {
    if (entityType) {
      this.cache.delete(entityType);
    } else {
      this.cache.clear();
    }
  }

  invalidate(entityType: EntityType, entityId: string, tenantId: string): void {
    const typeCache = this.cache.get(entityType);
    if (!typeCache) return;

    const key = this.getCacheKey(entityType, entityId, tenantId);
    typeCache.delete(key);
  }

  getStats(): Record<string, { size: number; oldestEntry?: number }> {
    const stats: Record<string, { size: number; oldestEntry?: number }> = {};

    this.cache.forEach((typeCache, entityType) => {
      const entries = Array.from(typeCache.values());
      const oldestEntry = entries.length > 0 
        ? Math.min(...entries.map(e => e.timestamp))
        : undefined;

      stats[entityType] = {
        size: typeCache.size,
        ...(oldestEntry !== undefined && { oldestEntry }),
      };
    });

    return stats;
  }
}

// Global cache instance
const resolverCache = new ResolverCache();

// Special error type for distinguishing between service errors and fallback scenarios
class ResolverError extends Error {
  constructor(message: string, public isServiceError: boolean = false) {
    super(message);
    this.name = 'ResolverError';
  }
}

// Entity resolver implementations
const resolvers: Record<EntityType, EntityResolver> = {
  [EntityType.ACCOUNT]: async (tenant, entityId, options = {}) => {
    const { accountsService } = await import('@/services/api');
    try {
      const account = await accountsService.getAccountById(tenant, entityId, options);
      return account.attributes?.name || `Account ${entityId}`;
    } catch (error) {
      console.warn(`Failed to resolve account name for ID ${entityId}:`, error);
      throw new ResolverError(`Failed to resolve account: ${error}`, true);
    }
  },

  [EntityType.CHARACTER]: async (tenant, entityId, options = {}) => {
    const { charactersService } = await import('@/services/api');
    try {
      const character = await charactersService.getById(tenant, entityId, options);
      return character.attributes?.name || `Character ${entityId}`;
    } catch (error) {
      console.warn(`Failed to resolve character name for ID ${entityId}:`, error);
      throw new ResolverError(`Failed to resolve character: ${error}`, true);
    }
  },

  [EntityType.GUILD]: async (tenant, entityId, options = {}) => {
    const { guildsService } = await import('@/services/api');
    try {
      const guild = await guildsService.getById(tenant, entityId, options);
      return guild.attributes?.name || `Guild ${entityId}`;
    } catch (error) {
      console.warn(`Failed to resolve guild name for ID ${entityId}:`, error);
      throw new ResolverError(`Failed to resolve guild: ${error}`, true);
    }
  },

  [EntityType.NPC]: async (tenant, entityId, options = {}) => {
    const { npcsService } = await import('@/services/api');
    try {
      const npc = await npcsService.getNPCById(parseInt(entityId), tenant, options);
      return npc?.name || `NPC ${entityId}`;
    } catch (error) {
      console.warn(`Failed to resolve NPC name for ID ${entityId}:`, error);
      throw new ResolverError(`Failed to resolve NPC: ${error}`, true);
    }
  },

  [EntityType.TEMPLATE]: async (tenant, entityId, options = {}) => {
    const { templatesService } = await import('@/services/api');
    try {
      const template = await templatesService.getById(entityId, options);
      // Templates don't have a name field, use ID for now
      return `Template ${template.id}`;
    } catch (error) {
      console.warn(`Failed to resolve template name for ID ${entityId}:`, error);
      throw new ResolverError(`Failed to resolve template: ${error}`, true);
    }
  },

  [EntityType.TENANT]: async (tenant, entityId, options = {}) => {
    const { tenantsService } = await import('@/services/api');
    try {
      const targetTenant = await tenantsService.getTenantById(entityId, options);
      return targetTenant.attributes?.name || `Tenant ${entityId}`;
    } catch (error) {
      console.warn(`Failed to resolve tenant name for ID ${entityId}:`, error);
      throw new ResolverError(`Failed to resolve tenant: ${error}`, true);
    }
  },
};

/**
 * Main resolver function that handles caching and fallbacks
 */
export async function resolveEntityLabel(
  entityType: EntityType,
  entityId: string,
  tenant: Tenant,
  options: ResolverOptions = {}
): Promise<ResolvedLabel> {
  const opts = { ...DEFAULT_OPTIONS, ...options };

  // Check cache first
  if (opts.useCache) {
    const cached = resolverCache.get(entityType, entityId, tenant.id);
    if (cached) {
      return {
        label: cached.label,
        fromCache: true,
        resolvedAt: cached.timestamp,
        isFallback: false,
      };
    }
  }

  // Get resolver function
  const resolver = resolvers[entityType];
  if (!resolver) {
    return {
      label: opts.fallback,
      fromCache: false,
      resolvedAt: Date.now(),
      isFallback: true,
    };
  }

  try {
    // Create timeout promise
    const timeoutPromise = new Promise<never>((_, reject) => {
      setTimeout(() => reject(new Error(`Resolver timeout for ${entityType}:${entityId}`)), opts.timeout);
    });

    // Race resolver against timeout
    const label = await Promise.race([
      resolver(tenant, entityId, opts),
      timeoutPromise,
    ]);

    // Cache the result
    if (opts.useCache) {
      resolverCache.set(entityType, entityId, tenant.id, label);
    }

    return {
      label,
      fromCache: false,
      resolvedAt: Date.now(),
      isFallback: false,
    };

  } catch (error) {
    console.warn(`Entity resolution failed for ${entityType}:${entityId}`, error);
    
    return {
      label: opts.fallback,
      fromCache: false,
      resolvedAt: Date.now(),
      isFallback: true,
    };
  }
}

/**
 * Batch resolver for multiple entities of the same type
 */
export async function resolveEntityLabels(
  entityType: EntityType,
  entityIds: string[],
  tenant: Tenant,
  options: ResolverOptions = {}
): Promise<Record<string, ResolvedLabel>> {
  const results: Record<string, ResolvedLabel> = {};

  // Process in parallel with limited concurrency
  const BATCH_SIZE = 10;
  const batches = [];
  
  for (let i = 0; i < entityIds.length; i += BATCH_SIZE) {
    batches.push(entityIds.slice(i, i + BATCH_SIZE));
  }

  for (const batch of batches) {
    const promises = batch.map(async (entityId) => {
      const result = await resolveEntityLabel(entityType, entityId, tenant, options);
      return { entityId, result };
    });

    const batchResults = await Promise.all(promises);
    batchResults.forEach(({ entityId, result }) => {
      results[entityId] = result;
    });
  }

  return results;
}

/**
 * Preload entity labels for better performance
 */
export async function preloadEntityLabels(
  entityType: EntityType,
  entityIds: string[],
  tenant: Tenant,
  options: ResolverOptions = {}
): Promise<void> {
  // Only preload entities not in cache
  const uncachedIds = entityIds.filter(id => {
    if (!options.useCache) return true;
    return !resolverCache.get(entityType, id, tenant.id);
  });

  if (uncachedIds.length === 0) return;

  // Resolve in background without blocking
  resolveEntityLabels(entityType, uncachedIds, tenant, options).catch(error => {
    console.warn(`Failed to preload labels for ${entityType}:`, error);
  });
}

/**
 * Invalidate cached labels for specific entities
 */
export function invalidateEntityLabels(
  entityType: EntityType,
  entityIds: string[],
  tenant: Tenant
): void {
  entityIds.forEach(entityId => {
    resolverCache.invalidate(entityType, entityId, tenant.id);
  });
}

/**
 * Clear all cached labels
 */
export function clearResolverCache(entityType?: EntityType): void {
  resolverCache.clear(entityType);
}

/**
 * Get cache statistics
 */
export function getResolverCacheStats(): Record<string, { size: number; oldestEntry?: number }> {
  return resolverCache.getStats();
}

/**
 * Helper function to determine entity type from route pattern
 */
export function getEntityTypeFromRoute(pathname: string): EntityType | null {
  if (pathname.includes('/accounts/')) return EntityType.ACCOUNT;
  if (pathname.includes('/characters/')) return EntityType.CHARACTER;
  if (pathname.includes('/guilds/')) return EntityType.GUILD;
  if (pathname.includes('/npcs/')) return EntityType.NPC;
  if (pathname.includes('/templates/')) return EntityType.TEMPLATE;
  if (pathname.includes('/tenants/')) return EntityType.TENANT;
  
  return null;
}

/**
 * Type-safe entity type checker
 */
export function isValidEntityType(type: string): type is EntityType {
  return Object.values(EntityType).includes(type as EntityType);
}

// Export cache instance for advanced usage
export { resolverCache };