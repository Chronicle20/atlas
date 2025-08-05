import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';

interface FormSkeletonProps {
  /**
   * Number of form fields to show in the skeleton
   */
  fields?: number;
  
  /**
   * Whether to show label placeholders for each field
   */
  showLabels?: boolean;
  
  /**
   * Whether to show help text placeholders below fields
   */
  showHelpText?: boolean;
  
  /**
   * Whether to show submit button placeholder
   */
  showSubmitButton?: boolean;
  
  /**
   * Whether to show additional action buttons (like cancel)
   */
  showActionButtons?: boolean;
  
  /**
   * Additional CSS classes to apply
   */
  className?: string;
  
  /**
   * Variant determines the layout style
   * - 'default': Standard form fields with normal spacing
   * - 'compact': Smaller spacing, suitable for inline forms
   * - 'wide': Wider form fields for desktop layouts
   */
  variant?: 'default' | 'compact' | 'wide';
}

/**
 * FormSkeleton component provides loading states for form components
 * that can accommodate various form layouts throughout the application.
 * 
 * @param fields - Number of skeleton fields to display (default: 4)
 * @param showLabels - Whether to show label placeholders (default: true)
 * @param showHelpText - Whether to show help text placeholders (default: false)
 * @param showSubmitButton - Whether to show submit button placeholder (default: true)
 * @param showActionButtons - Whether to show additional action buttons (default: false)
 * @param className - Additional CSS classes to apply
 * @param variant - Layout style variant (default: 'default')
 */
export function FormSkeleton({
  fields = 4,
  showLabels = true,
  showHelpText = false,
  showSubmitButton = true,
  showActionButtons = false,
  className,
  variant = 'default'
}: FormSkeletonProps) {
  const getFieldSpacing = () => {
    switch (variant) {
      case 'compact':
        return 'space-y-3';
      case 'wide':
        return 'space-y-8';
      default:
        return 'space-y-6';
    }
  };

  const getFieldHeight = () => {
    switch (variant) {
      case 'compact':
        return 'h-8';
      case 'wide':
        return 'h-12';
      default:
        return 'h-10';
    }
  };

  const getLabelWidth = () => {
    switch (variant) {
      case 'compact':
        return 'w-16';
      case 'wide':
        return 'w-32';
      default:
        return 'w-24';
    }
  };

  const getButtonHeight = () => {
    switch (variant) {
      case 'compact':
        return 'h-8';
      case 'wide':
        return 'h-12';
      default:
        return 'h-10';
    }
  };

  return (
    <div 
      className={cn(getFieldSpacing(), className)} 
      data-testid="form-skeleton"
    >
      {/* Form fields */}
      {Array.from({ length: fields }).map((_, index) => (
        <div key={`skeleton-field-${index}`} className="space-y-2">
          {/* Field label */}
          {showLabels && (
            <Skeleton 
              className={cn('h-4', getLabelWidth())} 
            />
          )}
          
          {/* Form field input */}
          <Skeleton 
            variant="rectangular"
            className={cn('w-full', getFieldHeight())} 
          />
          
          {/* Help text */}
          {showHelpText && (
            <Skeleton className="h-3 w-3/4" />
          )}
        </div>
      ))}
      
      {/* Action buttons section */}
      {(showSubmitButton || showActionButtons) && (
        <div className="flex items-center justify-end space-x-3 pt-4">
          {/* Cancel/Secondary button */}
          {showActionButtons && (
            <Skeleton 
              variant="rectangular"
              className={cn('w-20', getButtonHeight())} 
            />
          )}
          
          {/* Submit/Primary button */}
          {showSubmitButton && (
            <Skeleton 
              variant="rectangular"
              className={cn('w-24', getButtonHeight())} 
            />
          )}
        </div>
      )}
    </div>
  );
}