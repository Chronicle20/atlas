import { Skeleton } from '@/components/ui/skeleton';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';

interface TableSkeletonProps {
  rows?: number;
  columns?: number;
  showHeader?: boolean;
  showActions?: boolean;
  className?: string;
}

/**
 * TableSkeleton component provides a loading state for data tables
 * that matches the structure of the main DataTable component.
 * 
 * @param rows - Number of skeleton rows to display (default: 5)
 * @param columns - Number of skeleton columns to display (default: 4)
 * @param showHeader - Whether to show the header skeleton (default: true)
 * @param showActions - Whether to show the actions skeleton (default: true)
 * @param className - Additional CSS classes to apply
 */
export function TableSkeleton({ 
  rows = 5, 
  columns = 4, 
  showHeader = true, 
  showActions = true,
  className 
}: TableSkeletonProps) {
  return (
    <div className={`w-full space-y-4 ${className || ''}`}>
      {/* Header actions skeleton */}
      {showActions && (
        <div className="flex items-center justify-between">
          <div className="flex gap-2 items-center">
            <Skeleton className="h-10 w-10" /> {/* Refresh button */}
          </div>
          <div className="flex items-center gap-2">
            <Skeleton className="h-8 w-20" /> {/* Actions dropdown */}
          </div>
        </div>
      )}

      {/* Table skeleton */}
      <div className="w-full h-[calc(100vh-10rem)]">
        <div className="w-full h-auto">
          <div className="rounded-md border flex flex-col">
            {/* Table header skeleton */}
            {showHeader && (
              <div className="w-full">
                <Table className="[&>div]:overflow-visible table-fixed w-full">
                  <TableHeader>
                    <TableRow>
                      {Array.from({ length: columns }).map((_, index) => (
                        <TableHead key={`header-${index}`}>
                          <Skeleton className="h-4 w-full" />
                        </TableHead>
                      ))}
                    </TableRow>
                  </TableHeader>
                </Table>
              </div>
            )}
            
            {/* Table body skeleton */}
            <div className="overflow-auto max-h-[calc(100vh-20rem)]">
              <Table className="[&>div]:overflow-visible table-fixed w-full">
                <TableBody>
                  {Array.from({ length: rows }).map((_, rowIndex) => (
                    <TableRow key={`row-${rowIndex}`}>
                      {Array.from({ length: columns }).map((_, colIndex) => (
                        <TableCell key={`cell-${rowIndex}-${colIndex}`}>
                          <Skeleton className="h-4 w-full" />
                        </TableCell>
                      ))}
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}