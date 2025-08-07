/**
 * React hook for breadcrumb navigation management
 * 
 * Provides comprehensive breadcrumb functionality including:
 * - Route parsing and hierarchy generation
 * - Dynamic label resolution with caching
 * - Tenant context integration
 * - Loading states for async label resolution
 * - Error handling and fallbacks
 * - Performance optimizations
 */

import { useEffect, useMemo, useState, useRef } from 'react';
import { usePathname, useRouter } from 'next/navigation';
import { useTenant } from '@/context/tenant-context';
import { 
  parsePathname, 
  buildBreadcrumbPath, 
  getParentBreadcrumb, 
  getBreadcrumbKey,
  filterVisibleBreadcrumbs,
  type BreadcrumbSegment 
} from '@/lib/breadcrumbs/utils';
import { 
  resolveEntityLabel, 
  preloadEntityLabels,
  invalidateEntityLabels,
  getEntityTypeFromRoute,
  type EntityType,
  type ResolvedLabel,
  type ResolverOptions 
} from '@/lib/breadcrumbs/resolvers';
import { 
  findRouteConfig, 
  getBreadcrumbsFromRoute,
  type RouteConfig 
} from '@/lib/breadcrumbs/routes';

// Hook configuration options
export interface UseBreadcrumbsOptions {
  /** Maximum number of breadcrumb items to display */
  maxItems?: number;
  /** Whether to show ellipsis when truncating */
  showEllipsis?: boolean;
  /** Routes to hide from breadcrumbs */
  hiddenRoutes?: string[];
  /** Custom resolver options for entity labels */
  resolverOptions?: ResolverOptions;
  /** Whether to automatically resolve dynamic labels */
  autoResolve?: boolean;
  /** Whether to preload labels for better performance */
  enablePreloading?: boolean;
}

// State for individual breadcrumb resolution
interface BreadcrumbResolutionState {
  /** Whether this breadcrumb is currently being resolved */
  loading: boolean;
  /** Any error that occurred during resolution */
  error: Error | null;
  /** The resolved label information */
  resolved: ResolvedLabel | null;
}

// Combined hook result interface
export interface UseBreadcrumbsResult {
  /** Current breadcrumb segments */
  breadcrumbs: BreadcrumbSegment[];
  /** Whether any breadcrumb labels are being resolved */
  loading: boolean;
  /** Any errors from breadcrumb resolution */
  error: Error | null;
  /** Current route configuration */
  routeConfig: RouteConfig | null;
  /** Navigation utilities */
  navigation: {
    /** Navigate to parent route if available */
    goToParent: () => void;
    /** Navigate to specific breadcrumb */
    navigateTo: (breadcrumb: BreadcrumbSegment) => void;
    /** Get parent breadcrumb for navigation */
    getParent: () => BreadcrumbSegment | null;
  };
  /** Label resolution utilities */
  resolution: {
    /** Manually resolve a specific entity label */
    resolveLabel: (entityType: EntityType, entityId: string) => Promise<ResolvedLabel>;
    /** Invalidate cached labels for specific entities */
    invalidateLabels: (entityType: EntityType, entityIds: string[]) => void;
    /** Preload labels for better performance */
    preloadLabels: (entityType: EntityType, entityIds: string[]) => Promise<void>;
    /** Get resolution state for all breadcrumbs */
    resolutionStates: Map<string, BreadcrumbResolutionState>;
  };
  /** Breadcrumb utilities */
  utils: {
    /** Get breadcrumb cache key */
    getCacheKey: () => string;
    /** Check if current route is valid */
    isValidRoute: boolean;
    /** Get filtered breadcrumbs based on options */
    getFilteredBreadcrumbs: () => BreadcrumbSegment[];
  };
}

// Default options
const DEFAULT_OPTIONS: Required<UseBreadcrumbsOptions> = {
  maxItems: 5,
  showEllipsis: true,
  hiddenRoutes: [],
  resolverOptions: {
    fallback: 'Unknown',
    timeout: 5000,
    useCache: true,
  },
  autoResolve: true,
  enablePreloading: false,
};

/**
 * Main breadcrumbs hook
 */
export function useBreadcrumbs(options: UseBreadcrumbsOptions = {}): UseBreadcrumbsResult {
  const opts = { ...DEFAULT_OPTIONS, ...options };
  const pathname = usePathname();
  const router = useRouter();
  const { activeTenant } = useTenant();
  
  // Destructure specific options to avoid dependency issues
  const { autoResolve, resolverOptions, enablePreloading } = opts;

  // State management
  const [resolutionStates, setResolutionStates] = useState<Map<string, BreadcrumbResolutionState>>(new Map());
  const [globalError, setGlobalError] = useState<Error | null>(null);

  // Memoized route analysis
  const routeConfig = useMemo(() => findRouteConfig(pathname), [pathname]);
  const isValidRoute = useMemo(() => !!routeConfig, [routeConfig]);

  // Parse pathname into initial breadcrumb segments
  const initialBreadcrumbs = useMemo(() => {
    try {
      setGlobalError(null);
      
      // Try route-based parsing first for better accuracy
      const routeBreadcrumbs = getBreadcrumbsFromRoute(pathname);
      if (routeBreadcrumbs.length > 0) {
        return routeBreadcrumbs.map(partial => ({
          segment: partial.segment || '',
          label: partial.label || 'Unknown',
          href: partial.href || '',
          dynamic: partial.dynamic || false,
          isCurrentPage: partial.isCurrentPage || false,
          ...(partial.entityId && { entityId: partial.entityId }),
          ...(partial.entityType && { entityType: partial.entityType }),
        })) as BreadcrumbSegment[];
      }

      // Fallback to pathname parsing
      return parsePathname(pathname);
    } catch (error) {
      setGlobalError(error instanceof Error ? error : new Error('Failed to parse breadcrumbs'));
      return [];
    }
  }, [pathname]);

  // Process breadcrumbs with dynamic label resolution
  const [processedBreadcrumbs, setProcessedBreadcrumbs] = useState<BreadcrumbSegment[]>(initialBreadcrumbs);
  
  // Create a stable key based on breadcrumbs structure to prevent infinite loops
  const breadcrumbsKey = useMemo(() => 
    initialBreadcrumbs.map(b => `${b.segment}:${b.entityId || 'static'}`).join('|'),
    [initialBreadcrumbs]
  );
  
  const prevBreadcrumbsKeyRef = useRef(breadcrumbsKey);

  // Effect to resolve dynamic labels when breadcrumbs structure changes
  useEffect(() => {
    // Only run if breadcrumbs structure actually changed
    if (prevBreadcrumbsKeyRef.current === breadcrumbsKey) {
      return;
    }
    
    prevBreadcrumbsKeyRef.current = breadcrumbsKey;

    if (!activeTenant || !autoResolve) {
      setProcessedBreadcrumbs(initialBreadcrumbs);
      return;
    }

    const dynamicBreadcrumbs = initialBreadcrumbs.filter(b => b.dynamic && b.entityId && b.entityType);
    
    if (dynamicBreadcrumbs.length === 0) {
      setProcessedBreadcrumbs(initialBreadcrumbs);
      return;
    }

    const resolveDynamicLabels = async () => {
      // Batch all resolutions first
      const resolutionResults = new Map<string, BreadcrumbResolutionState>();
      const updatedBreadcrumbs = [...initialBreadcrumbs];

      // Set all items to loading state
      dynamicBreadcrumbs.forEach(breadcrumb => {
        const key = `${breadcrumb.entityType}:${breadcrumb.entityId}`;
        resolutionResults.set(key, { loading: true, error: null, resolved: null });
      });

      // Update resolution states once for loading
      setResolutionStates(new Map(resolutionResults));

      // Process all resolutions in parallel
      const resolutionPromises = dynamicBreadcrumbs.map(async (breadcrumb) => {
        const key = `${breadcrumb.entityType}:${breadcrumb.entityId}`;
        
        try {
          const resolved = await resolveEntityLabel(
            breadcrumb.entityType as EntityType,
            breadcrumb.entityId!,
            activeTenant,
            resolverOptions
          );
          
          // Store the result for batch update
          resolutionResults.set(key, { loading: false, error: null, resolved });
          
          // Update the breadcrumb in our local array
          const index = updatedBreadcrumbs.findIndex(b => 
            b.entityId === breadcrumb.entityId && b.entityType === breadcrumb.entityType
          );
          if (index !== -1) {
            updatedBreadcrumbs[index] = { ...updatedBreadcrumbs[index], label: resolved.label };
          }
          
        } catch (error) {
          const err = error instanceof Error ? error : new Error('Resolution failed');
          resolutionResults.set(key, { loading: false, error: err, resolved: null });
          console.warn(`Failed to resolve ${breadcrumb.entityType}:${breadcrumb.entityId}`, error);
        }
      });

      // Wait for all resolutions to complete
      await Promise.all(resolutionPromises);

      // Update both states once after all resolutions are done
      setResolutionStates(new Map(resolutionResults));
      setProcessedBreadcrumbs(updatedBreadcrumbs);
    };

    resolveDynamicLabels();
  }, [breadcrumbsKey, activeTenant, autoResolve, resolverOptions, initialBreadcrumbs]);

  // Effect to preload labels
  useEffect(() => {
    const preloadLabelsForRoute = async () => {
      if (!activeTenant || !enablePreloading) return;

      const entityType = getEntityTypeFromRoute(pathname);
      if (!entityType) return;

      const dynamicBreadcrumbs = initialBreadcrumbs.filter(b => 
        b.dynamic && b.entityId && b.entityType === entityType
      );

      if (dynamicBreadcrumbs.length === 0) return;

      const entityIds = dynamicBreadcrumbs.map(b => b.entityId!);
      
      try {
        await preloadEntityLabels(entityType, entityIds, activeTenant, resolverOptions);
      } catch (error) {
        console.warn(`Failed to preload labels for ${entityType}:`, error);
      }
    };

    preloadLabelsForRoute();
  }, [pathname, initialBreadcrumbs, activeTenant, enablePreloading, resolverOptions]);

  // Build final breadcrumbs with filtering and truncation
  const finalBreadcrumbs = useMemo(() => {
    const filtered = filterVisibleBreadcrumbs(processedBreadcrumbs, {
      hiddenRoutes: opts.hiddenRoutes,
    });

    return buildBreadcrumbPath(filtered, {
      maxItems: opts.maxItems,
      showEllipsis: opts.showEllipsis,
    });
  }, [processedBreadcrumbs, opts.hiddenRoutes, opts.maxItems, opts.showEllipsis]);

  // Calculate loading state
  const loading = useMemo(() => {
    return Array.from(resolutionStates.values()).some(state => state.loading);
  }, [resolutionStates]);

  // Navigation utilities
  const navigation = useMemo(() => ({
    goToParent: () => {
      const parent = getParentBreadcrumb(finalBreadcrumbs);
      if (parent && parent.href) {
        router.push(parent.href);
      }
    },
    navigateTo: (breadcrumb: BreadcrumbSegment) => {
      if (breadcrumb.href && !breadcrumb.isCurrentPage) {
        router.push(breadcrumb.href);
      }
    },
    getParent: () => getParentBreadcrumb(finalBreadcrumbs),
  }), [finalBreadcrumbs, router]);

  // Label resolution utilities
  const resolution = useMemo(() => ({
    resolveLabel: async (entityType: EntityType, entityId: string) => {
      if (!activeTenant) throw new Error('No active tenant');
      return resolveEntityLabel(entityType, entityId, activeTenant, resolverOptions);
    },
    invalidateLabels: (entityType: EntityType, entityIds: string[]) => {
      if (!activeTenant) return;
      invalidateEntityLabels(entityType, entityIds, activeTenant);
      
      // Clear resolution states for invalidated entities
      entityIds.forEach(entityId => {
        const key = `${entityType}:${entityId}`;
        setResolutionStates(prev => {
          const newMap = new Map(prev);
          newMap.delete(key);
          return newMap;
        });
      });
      
      // Reset to allow re-resolution by updating the key reference
      prevBreadcrumbsKeyRef.current = '';
      setProcessedBreadcrumbs(initialBreadcrumbs);
    },
    preloadLabels: async (entityType: EntityType, entityIds: string[]) => {
      if (!activeTenant) throw new Error('No active tenant');
      await preloadEntityLabels(entityType, entityIds, activeTenant, resolverOptions);
    },
    resolutionStates,
  }), [activeTenant, resolverOptions, resolutionStates, initialBreadcrumbs]);

  // Utility functions
  const utils = useMemo(() => ({
    getCacheKey: () => getBreadcrumbKey(finalBreadcrumbs),
    isValidRoute,
    getFilteredBreadcrumbs: () => filterVisibleBreadcrumbs(processedBreadcrumbs, {
      hiddenRoutes: opts.hiddenRoutes,
    }),
  }), [finalBreadcrumbs, isValidRoute, processedBreadcrumbs, opts.hiddenRoutes]);

  return {
    breadcrumbs: finalBreadcrumbs,
    loading,
    error: globalError,
    routeConfig,
    navigation,
    resolution,
    utils,
  };
}

// Simplified hook for basic breadcrumb needs
export function useSimpleBreadcrumbs(): BreadcrumbSegment[] {
  const { breadcrumbs } = useBreadcrumbs({
    maxItems: 3,
    autoResolve: false,
    enablePreloading: false,
  });
  
  return breadcrumbs;
}

// Hook for breadcrumb navigation only
export function useBreadcrumbNavigation() {
  const { navigation, breadcrumbs } = useBreadcrumbs({
    autoResolve: false,
    enablePreloading: false,
  });
  
  return {
    ...navigation,
    breadcrumbs: breadcrumbs.map(b => ({
      label: b.label,
      href: b.href,
      isCurrentPage: b.isCurrentPage,
    })),
  };
}

// Types are already exported inline above