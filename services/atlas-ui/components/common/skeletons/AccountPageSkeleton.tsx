import { Skeleton } from '@/components/ui/skeleton';
import { TableSkeleton } from '../TableSkeleton';

/**
 * AccountPageSkeleton provides a loading state specifically for the Accounts page.
 * Matches the layout structure with page title and table content with refresh functionality.
 */
export function AccountPageSkeleton() {
  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
      {/* Page header */}
      <div className="items-center justify-between space-y-2">
        <div>
          <Skeleton className="h-8 w-28" /> {/* "Accounts" title */}
        </div>
      </div>
      
      {/* Table content */}
      <div className="mt-4">
        <TableSkeleton 
          rows={10} 
          columns={6} 
          showHeader={true} 
          showActions={true}
        />
      </div>
    </div>
  );
}