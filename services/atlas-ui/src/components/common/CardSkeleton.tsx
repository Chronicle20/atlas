import { Skeleton } from '@/components/ui/skeleton';
import { Card, CardContent, CardHeader } from '@/components/ui/card';
import { cn } from '@/lib/utils';

interface CardSkeletonProps {
  /**
   * Variant determines the structure of the card skeleton
   * - 'default': Standard card with header and content
   * - 'grid': Small card for grid layouts (like asset cards)
   * - 'detailed': Card with header, content, and footer sections
   */
  variant?: 'default' | 'grid' | 'detailed';
  
  /**
   * Number of content lines to show in the skeleton
   */
  lines?: number;
  
  /**
   * Whether to show a header skeleton
   */
  showHeader?: boolean;
  
  /**
   * Whether to show an action button in the header
   */
  showHeaderAction?: boolean;
  
  /**
   * Whether to show a footer skeleton
   */
  showFooter?: boolean;
  
  /**
   * Additional CSS classes to apply
   */
  className?: string;
  
  /**
   * Width of the card (useful for grid layouts)
   */
  width?: string;
}

/**
 * CardSkeleton component provides loading states for card components
 * with different variants to match various card layouts throughout the app.
 * 
 * @param variant - Determines the card structure (default: 'default')
 * @param lines - Number of content lines to display (default: 3)
 * @param showHeader - Whether to show the header skeleton (default: true)
 * @param showHeaderAction - Whether to show action button in header (default: false)
 * @param showFooter - Whether to show the footer skeleton (default: false for default/grid, true for detailed)
 * @param className - Additional CSS classes to apply
 * @param width - Width of the card (default: auto)
 */
export function CardSkeleton({
  variant = 'default',
  lines = 3,
  showHeader = true,
  showHeaderAction = false,
  showFooter = variant === 'detailed',
  className,
  width = 'auto'
}: CardSkeletonProps) {
  
  // Grid variant - small cards for asset grids
  if (variant === 'grid') {
    return (
      <Card 
        className={cn('overflow-hidden relative py-0', className)} 
        style={{ width }}
      >
        {showHeader && (
          <CardHeader className="p-1 pl-3 pb-1">
            <Skeleton className="h-4 w-8" />
          </CardHeader>
        )}
        <CardContent className="p-2 pt-0 text-center">
          <Skeleton className="h-5 w-full" />
        </CardContent>
        {showHeaderAction && (
          <div className="absolute top-0 right-0 p-1">
            <Skeleton className="h-6 w-6 rounded" />
          </div>
        )}
      </Card>
    );
  }

  // Default and detailed variants
  return (
    <Card className={cn('w-full', className)} style={{ width }}>
      {showHeader && (
        <CardHeader>
          <div className={cn(
            'flex items-center',
            showHeaderAction ? 'justify-between' : 'justify-start'
          )}>
            <div className="space-y-2">
              <Skeleton className="h-6 w-32" /> {/* Title */}
              {variant === 'detailed' && (
                <Skeleton className="h-4 w-48" /> /* Description */
              )}
            </div>
            {showHeaderAction && (
              <Skeleton className="h-10 w-24" /> /* Action button */
            )}
          </div>
        </CardHeader>
      )}
      
      <CardContent className="space-y-3">
        {variant === 'detailed' ? (
          // Detailed variant with structured content
          <div className="grid grid-cols-2 gap-4">
            {Array.from({ length: Math.ceil(lines / 2) }).map((_, rowIndex) => (
              <div key={`row-${rowIndex}`} className="space-y-2">
                <Skeleton className="h-4 w-full" />
                {rowIndex * 2 + 1 < lines && (
                  <Skeleton className="h-4 w-3/4" />
                )}
              </div>
            ))}
          </div>
        ) : (
          // Default variant with simple lines
          <div className="space-y-2">
            {Array.from({ length: lines }).map((_, index) => (
              <Skeleton 
                key={`line-${index}`} 
                className={cn(
                  'h-4',
                  index === lines - 1 ? 'w-2/3' : 'w-full' // Last line is shorter
                )} 
              />
            ))}
          </div>
        )}
      </CardContent>
      
      {showFooter && (
        <div className="px-6 pb-6">
          <div className="flex items-center justify-between pt-4 border-t">
            <Skeleton className="h-4 w-20" />
            <Skeleton className="h-8 w-16" />
          </div>
        </div>
      )}
    </Card>
  );
}