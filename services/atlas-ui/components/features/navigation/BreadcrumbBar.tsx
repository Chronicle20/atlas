"use client";

import { memo, useState, useEffect } from 'react';
import Link from 'next/link';
import { useBreadcrumbs } from '@/lib/hooks/useBreadcrumbs';
import { useTenant } from '@/context/tenant-context';
import {
  Breadcrumb,
  BreadcrumbList,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbPage,
  BreadcrumbSeparator,
  BreadcrumbEllipsis,
} from '@/components/ui/breadcrumb';
import { LoadingSpinner } from '@/components/common/LoadingSpinner';
import { cn } from '@/lib/utils';

interface BreadcrumbBarProps {
  /** Custom CSS classes */
  className?: string;
  /** Maximum number of breadcrumb items to display */
  maxItems?: number;
  /** Whether to show ellipsis when truncating */
  showEllipsis?: boolean;
  /** Routes to hide from breadcrumbs */
  hiddenRoutes?: string[];
  /** Whether to show loading states for dynamic labels */
  showLoadingStates?: boolean;
  /** Custom label overrides for specific routes */
  labelOverrides?: Record<string, string>;
}

/**
 * BreadcrumbBar component with tenant context support
 * 
 * Features:
 * - Dynamic label resolution for entity names (characters, guilds, etc.)
 * - Tenant context integration for data fetching
 * - Loading states for async label resolution
 * - Responsive design with ellipsis support
 * - Accessibility compliance (ARIA labels, semantic HTML)
 * - Error handling with fallbacks
 */
export const BreadcrumbBar = memo<BreadcrumbBarProps>(({
  className,
  maxItems = 5,
  showEllipsis = true,
  hiddenRoutes = [],
  showLoadingStates = true,
  labelOverrides = {},
}) => {
  const { activeTenant, loading: tenantLoading } = useTenant();
  const [isClient, setIsClient] = useState(false);
  
  // Initialize client-side rendering flag
  useEffect(() => {
    setIsClient(true);
  }, []);

  const {
    breadcrumbs,
    loading: breadcrumbsLoading,
    error,
    utils: { isValidRoute },
  } = useBreadcrumbs({
    maxItems,
    showEllipsis,
    hiddenRoutes,
    autoResolve: !!activeTenant, // Only resolve labels when tenant is available
    enablePreloading: false, // Disable preloading to prevent unnecessary API calls
    resolverOptions: {
      fallback: 'Unknown',
      timeout: 5000,
      useCache: true,
    },
  });

  // Don't render during SSR or if tenant is loading
  if (!isClient || tenantLoading) {
    return (
      <nav aria-label="breadcrumb" className={cn("flex items-center space-x-2", className)}>
        <div className="flex items-center space-x-1">
          <LoadingSpinner size="sm" />
          <span className="text-sm text-muted-foreground">Loading navigation...</span>
        </div>
      </nav>
    );
  }

  // Handle errors gracefully
  if (error) {
    console.warn('Breadcrumb error:', error);
    return (
      <nav aria-label="breadcrumb" className={cn("flex items-center", className)}>
        <span className="text-sm text-muted-foreground">Navigation unavailable</span>
      </nav>
    );
  }

  // Don't render if no valid route or breadcrumbs
  if (!isValidRoute || breadcrumbs.length === 0) {
    return null;
  }

  // Apply label overrides
  const processedBreadcrumbs = breadcrumbs.map(breadcrumb => ({
    ...breadcrumb,
    label: labelOverrides[breadcrumb.href] || breadcrumb.label,
  }));

  // Calculate if we need ellipsis
  const needsEllipsis = showEllipsis && processedBreadcrumbs.length > maxItems;
  const visibleBreadcrumbs = needsEllipsis 
    ? [
        processedBreadcrumbs[0], // First item
        ...processedBreadcrumbs.slice(-2) // Last two items
      ]
    : processedBreadcrumbs.slice(0, maxItems);

  return (
    <div className={cn("flex items-center space-x-2", className)}>
      <Breadcrumb>
        <BreadcrumbList>
          {visibleBreadcrumbs.map((breadcrumb, index) => {
            const isLast = index === visibleBreadcrumbs.length - 1;
            const showEllipsisHere = needsEllipsis && index === 1 && visibleBreadcrumbs.length > 2;
            
            return (
              <div key={`${breadcrumb?.href}-${index}`} className="flex items-center">
                {/* Show ellipsis between first and last items */}
                {showEllipsisHere && (
                  <>
                    <BreadcrumbItem>
                      <BreadcrumbEllipsis />
                    </BreadcrumbItem>
                    <BreadcrumbSeparator />
                  </>
                )}
                
                <BreadcrumbItem>
                  {isLast || breadcrumb?.isCurrentPage ? (
                    <BreadcrumbPage>
                      <span className="flex items-center gap-2">
                        {showLoadingStates && breadcrumbsLoading && breadcrumb?.dynamic && (
                          <LoadingSpinner size="sm" />
                        )}
                        {breadcrumb?.label}
                      </span>
                    </BreadcrumbPage>
                  ) : (
                    <BreadcrumbLink asChild>
                      <Link href={breadcrumb?.href || '#'} className="flex items-center gap-2">
                        {showLoadingStates && breadcrumbsLoading && breadcrumb?.dynamic && (
                          <LoadingSpinner size="sm" />
                        )}
                        {breadcrumb?.label}
                      </Link>
                    </BreadcrumbLink>
                  )}
                </BreadcrumbItem>
                
                {/* Add separator if not the last item */}
                {!isLast && <BreadcrumbSeparator />}
              </div>
            );
          })}
        </BreadcrumbList>
      </Breadcrumb>

      {/* Show loading indicator for overall breadcrumb loading */}
      {showLoadingStates && breadcrumbsLoading && (
        <div className="ml-2">
          <LoadingSpinner size="sm" />
        </div>
      )}
    </div>
  );
});

BreadcrumbBar.displayName = 'BreadcrumbBar';

// Export a simplified version for basic usage
export const SimpleBreadcrumbBar = memo<Pick<BreadcrumbBarProps, 'className'>>(({ 
  className 
}) => (
  <BreadcrumbBar 
    className={className ?? undefined}
    maxItems={3}
    showEllipsis={false}
    showLoadingStates={false}
  />
));

SimpleBreadcrumbBar.displayName = 'SimpleBreadcrumbBar';

export default BreadcrumbBar;