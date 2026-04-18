import { Skeleton } from '@/components/ui/skeleton';
import { TableSkeleton } from '../TableSkeleton';

/**
 * CharacterPageSkeleton provides a loading state specifically for the Characters page.
 * Matches the layout structure with page title and table content.
 */
export function CharacterPageSkeleton() {
  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
      {/* Page header */}
      <div className="items-center justify-between space-y-2">
        <div>
          <Skeleton className="h-8 w-32" /> {/* "Characters" title */}
        </div>
      </div>
      
      {/* Table content */}
      <div className="mt-4">
        <TableSkeleton 
          rows={12} 
          columns={7} 
          showHeader={true} 
          showActions={true}
        />
      </div>
    </div>
  );
}