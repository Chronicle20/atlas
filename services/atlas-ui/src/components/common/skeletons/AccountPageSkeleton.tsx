import { Skeleton } from '@/components/ui/skeleton';
import { TableSkeleton } from '../TableSkeleton';

interface AccountPageSkeletonProps {
  /**
   * Animation type for the skeleton elements
   * - 'pulse': Default pulsing animation
   * - 'wave': Shimmer wave animation
   * - 'none': No animation
   */
  animation?: 'pulse' | 'wave' | 'none';
}

/**
 * AccountPageSkeleton provides a loading state specifically for the Accounts page.
 * Matches the layout structure with page title and table content with refresh functionality.
 * 
 * @param animation - Animation type for skeleton elements (default: 'pulse')
 */
export function AccountPageSkeleton({ animation = 'pulse' }: AccountPageSkeletonProps = {}) {
  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
      {/* Page header */}
      <div className="items-center justify-between space-y-2">
        <div>
          <Skeleton animation={animation} className="h-8 w-28" /> {/* "Accounts" title */}
        </div>
      </div>
      
      {/* Table content */}
      <div className="mt-4">
        <TableSkeleton 
          rows={10} 
          columns={6} 
          showHeader={true} 
          showActions={true}
          animation={animation}
        />
      </div>
    </div>
  );
}