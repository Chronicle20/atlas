import { Skeleton } from '@/components/ui/skeleton';
import { TableSkeleton } from '../TableSkeleton';

interface TenantPageSkeletonProps {
  /**
   * Animation type for the skeleton elements
   * - 'pulse': Default pulsing animation
   * - 'wave': Shimmer wave animation
   * - 'none': No animation
   */
  animation?: 'pulse' | 'wave' | 'none';
}

/**
 * TenantPageSkeleton provides a loading state specifically for the Tenants page.
 * Matches the layout structure with page title and table content.
 * 
 * @param animation - Animation type for skeleton elements (default: 'pulse')
 */
export function TenantPageSkeleton({ animation = 'pulse' }: TenantPageSkeletonProps = {}) {
  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
      {/* Page header */}
      <div className="items-center justify-between space-y-2">
        <div>
          <Skeleton animation={animation} className="h-8 w-24" /> {/* "Tenants" title */}
        </div>
      </div>
      
      {/* Table content */}
      <div className="mt-4">
        <TableSkeleton 
          rows={8} 
          columns={5} 
          showHeader={true} 
          showActions={true}
          animation={animation}
        />
      </div>
    </div>
  );
}