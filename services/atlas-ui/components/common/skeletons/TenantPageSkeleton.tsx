import { Skeleton } from '@/components/ui/skeleton';
import { TableSkeleton } from '../TableSkeleton';

/**
 * TenantPageSkeleton provides a loading state specifically for the Tenants page.
 * Matches the layout structure with page title and table content.
 */
export function TenantPageSkeleton() {
  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
      {/* Page header */}
      <div className="items-center justify-between space-y-2">
        <div>
          <Skeleton className="h-8 w-24" /> {/* "Tenants" title */}
        </div>
      </div>
      
      {/* Table content */}
      <div className="mt-4">
        <TableSkeleton 
          rows={8} 
          columns={5} 
          showHeader={true} 
          showActions={true}
        />
      </div>
    </div>
  );
}