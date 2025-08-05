/**
 * Dynamic import utilities for code splitting and performance optimization
 */

import React, { ComponentType, lazy, LazyExoticComponent } from 'react';
import dynamic from 'next/dynamic';

/**
 * Enhanced dynamic import with performance optimizations
 */
export function createOptimizedDynamicImport<T extends ComponentType<unknown>>(
  importFn: () => Promise<{ default: T } | T>,
  options: {
    loading?: ComponentType<unknown>;
    error?: ComponentType<{ error: Error; retry: () => void }>;
    ssr?: boolean;
    preload?: boolean;
    chunkName?: string;
  } = {}
): ComponentType<T extends ComponentType<infer P> ? P : never> {
  const {
    loading: LoadingComponent,
    ssr = false,
    preload = false,
    chunkName,
  } = options;

  const DynamicComponent = dynamic(
    async () => {
      try {
        const importedModule = await importFn();
        return 'default' in importedModule ? importedModule : { default: importedModule as T };
      } catch (error) {
        console.error(`Failed to load dynamic component${chunkName ? ` (${chunkName})` : ''}:`, error);
        throw error;
      }
    },
    {
      ...(LoadingComponent && { loading: () => React.createElement(LoadingComponent) }),
      ssr,
    }
  );

  // Preload the component if requested
  if (preload && typeof window !== 'undefined') {
    // Preload after a short delay to avoid blocking initial render
    setTimeout(() => {
      importFn().catch(error => {
        console.warn(`Failed to preload component${chunkName ? ` (${chunkName})` : ''}:`, error);
      });
    }, 100);
  }

  return DynamicComponent as ComponentType<T extends ComponentType<infer P> ? P : never>;
}

/**
 * Lazy load component with error boundary
 */
export function createLazyComponent<T extends ComponentType<unknown>>(
  importFn: () => Promise<{ default: T } | T>,
  fallback?: ComponentType<unknown>
): LazyExoticComponent<T> {
  return lazy(async () => {
    try {
      const importedModule = await importFn();
      return 'default' in importedModule ? importedModule : { default: importedModule as T };
    } catch (error) {
      console.error('Failed to lazy load component:', error);
      
      // Return fallback component if loading fails
      if (fallback) {
        return { default: fallback as T };
      }
      
      throw error;
    }
  });
}

/**
 * Preload a dynamic import without rendering
 */
export async function preloadDynamicImport<T>(
  importFn: () => Promise<T>
): Promise<T | null> {
  try {
    return await importFn();
  } catch (error) {
    console.warn('Failed to preload dynamic import:', error);
    return null;
  }
}

/**
 * Batch preload multiple dynamic imports
 */
export async function batchPreloadDynamicImports(
  imports: Array<{
    name: string;
    importFn: () => Promise<unknown>;
    priority?: 'high' | 'medium' | 'low';
    delay?: number;
  }>
): Promise<void> {
  // Sort by priority
  const sortedImports = imports.sort((a, b) => {
    const priorityOrder = { high: 0, medium: 1, low: 2 };
    return priorityOrder[a.priority || 'medium'] - priorityOrder[b.priority || 'medium'];
  });

  const preloadPromises = sortedImports.map(async ({ name, importFn, delay = 0 }) => {
    if (delay > 0) {
      await new Promise(resolve => setTimeout(resolve, delay));
    }

    try {
      await importFn();
      console.debug(`Preloaded dynamic import: ${name}`);
    } catch (error) {
      console.warn(`Failed to preload dynamic import ${name}:`, error);
    }
  });

  await Promise.allSettled(preloadPromises);
}

/**
 * Hook for managing dynamic import preloading
 */
import { useCallback } from 'react';

export function useDynamicImportPreloader() {
  const preloadImports = useCallback(async (imports: Parameters<typeof batchPreloadDynamicImports>[0]) => {
    await batchPreloadDynamicImports(imports);
  }, []);

  const preloadOnIdle = useCallback((imports: Parameters<typeof batchPreloadDynamicImports>[0]) => {
    if (typeof window === 'undefined') return;

    if ('requestIdleCallback' in window) {
      window.requestIdleCallback(() => {
        preloadImports(imports);
      });
    } else {
      // Fallback for browsers without requestIdleCallback
      setTimeout(() => {
        preloadImports(imports);
      }, 1000);
    }
  }, [preloadImports]);

  return {
    preloadImports,
    preloadOnIdle,
  };
}

/**
 * Component performance metrics for dynamic imports
 */
export function measureDynamicImportPerformance<T>(
  componentName: string,
  importFn: () => Promise<T>
): () => Promise<T> {
  return async () => {
    const startTime = performance.now();
    
    try {
      const result = await importFn();
      const endTime = performance.now();
      const loadTime = endTime - startTime;
      
      console.debug(`Dynamic import performance - ${componentName}: ${loadTime.toFixed(2)}ms`);
      
      // Report to analytics if available
      if (typeof window !== 'undefined' && 'gtag' in window) {
        (window as { gtag?: (...args: unknown[]) => void }).gtag?.('event', 'dynamic_import_load', {
          component_name: componentName,
          load_time: Math.round(loadTime),
        });
      }
      
      return result;
    } catch (error) {
      const endTime = performance.now();
      const failTime = endTime - startTime;
      
      console.error(`Dynamic import failed - ${componentName}: ${failTime.toFixed(2)}ms`, error);
      throw error;
    }
  };
}