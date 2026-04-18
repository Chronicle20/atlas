import { Skeleton } from '@/components/ui/skeleton';
import { TableSkeleton } from '../TableSkeleton';

/**
 * GuildPageSkeleton provides a loading state specifically for the Guilds page.
 * Matches the layout structure with page title and table content.
 */
export function GuildPageSkeleton() {
  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
      {/* Page header */}
      <div className="items-center justify-between space-y-2">
        <div>
          <Skeleton className="h-8 w-20" /> {/* "Guilds" title */}
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