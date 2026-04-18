import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';

interface ListSkeletonProps {
  /**
   * Number of list items to show in the skeleton
   */
  items?: number;
  
  /**
   * Whether to show avatar/icon placeholder for each item
   */
  showAvatar?: boolean;
  
  /**
   * Whether to show action buttons/controls for each item
   */
  showActions?: boolean;
  
  /**
   * Whether to show secondary text line for each item
   */
  showSubtext?: boolean;
  
  /**
   * Additional CSS classes to apply
   */
  className?: string;
  
  /**
   * Variant determines the layout style
   * - 'default': Standard list items with padding and borders
   * - 'compact': Smaller spacing, suitable for dense lists
   * - 'card': Each item wrapped in a card-like container
   */
  variant?: 'default' | 'compact' | 'card';
  
  /**
   * Animation type for the skeleton elements
   * - 'pulse': Default pulsing animation
   * - 'wave': Shimmer wave animation
   * - 'none': No animation
   */
  animation?: 'pulse' | 'wave' | 'none';
}

/**
 * ListSkeleton component provides loading states for list components
 * that can accommodate various list layouts throughout the application.
 * 
 * @param items - Number of skeleton items to display (default: 5)
 * @param showAvatar - Whether to show avatar placeholder (default: true)
 * @param showActions - Whether to show action buttons (default: false)
 * @param showSubtext - Whether to show secondary text line (default: true)
 * @param className - Additional CSS classes to apply
 * @param variant - Layout style variant (default: 'default')
 * @param animation - Animation type for skeleton elements (default: 'pulse')
 */
export function ListSkeleton({
  items = 5,
  showAvatar = true,
  showActions = false,
  showSubtext = true,
  className,
  variant = 'default',
  animation = 'pulse'
}: ListSkeletonProps) {
  const getItemClassName = () => {
    switch (variant) {
      case 'compact':
        return 'flex items-center space-x-3 py-2 px-3';
      case 'card':
        return 'flex items-center space-x-4 p-4 border rounded-lg shadow-sm';
      default:
        return 'flex items-center space-x-4 py-3 px-4 border-b last:border-b-0';
    }
  };

  const getContainerClassName = () => {
    switch (variant) {
      case 'compact':
        return 'space-y-1';
      case 'card':
        return 'space-y-3';
      default:
        return 'border rounded-lg overflow-hidden';
    }
  };

  return (
    <div 
      className={cn(getContainerClassName(), className)} 
      data-testid="list-skeleton"
    >
      {Array.from({ length: items }).map((_, index) => (
        <div key={`skeleton-item-${index}`} className={getItemClassName()}>
          {/* Avatar/Icon placeholder */}
          {showAvatar && (
            <Skeleton 
              variant="circular" 
              animation={animation}
              className={cn(
                variant === 'compact' ? 'h-8 w-8' : 'h-10 w-10'
              )} 
            />
          )}
          
          {/* Content area */}
          <div className="flex-1 space-y-2">
            {/* Primary text line */}
            <Skeleton 
              animation={animation}
              className={cn(
                'h-4',
                variant === 'compact' ? 'w-24' : 'w-32'
              )} 
            />
            
            {/* Secondary text line */}
            {showSubtext && (
              <Skeleton 
                animation={animation}
                className={cn(
                  'h-3',
                  variant === 'compact' ? 'w-16' : 'w-48'
                )} 
              />
            )}
          </div>
          
          {/* Action buttons placeholder */}
          {showActions && (
            <div className="flex items-center space-x-2">
              <Skeleton 
                animation={animation}
                className={cn(
                  variant === 'compact' ? 'h-6 w-12' : 'h-8 w-16'
                )} 
              />
              <Skeleton 
                animation={animation}
                className={cn(
                  variant === 'compact' ? 'h-6 w-6' : 'h-8 w-8'
                )} 
              />
            </div>
          )}
        </div>
      ))}
    </div>
  );
}