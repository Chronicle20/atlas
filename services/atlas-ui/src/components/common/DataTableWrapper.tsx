import * as React from 'react';
import { ColumnDef } from '@tanstack/react-table';
import { DataTable } from '@/components/data-table';
import { PageLoader } from './PageLoader';
import { ErrorDisplay } from './ErrorDisplay';
import { EmptyState } from './EmptyState';
import { cn } from '@/lib/utils';
import { Database } from 'lucide-react';

interface DataTableWrapperProps<TData, TValue> {
  columns: ColumnDef<TData, TValue>[];
  data: TData[];
  loading?: boolean;
  error?: Error | { message: string } | string | null;
  initialVisibilityState?: string[];
  onRefresh?: () => void;
  headerActions?: Array<{
    icon?: React.ReactNode;
    label: string;
    onClick: () => void;
  }>;
  emptyState?: {
    title: string;
    description?: string;
    icon?: React.ReactNode;
    action?: {
      label: string;
      onClick: () => void;
    };
  };
  className?: string;
}

export function DataTableWrapper<TData, TValue>({
  columns,
  data,
  loading = false,
  error,
  initialVisibilityState,
  onRefresh,
  headerActions,
  emptyState,
  className,
}: DataTableWrapperProps<TData, TValue>) {
  // Show loading state
  if (loading) {
    return (
      <div className={cn('w-full', className)}>
        <PageLoader />
      </div>
    );
  }

  // Show error state
  if (error) {
    return (
      <div className={cn('w-full', className)}>
        <ErrorDisplay 
          error={error} 
          {...(onRefresh && { retry: onRefresh })}
        />
      </div>
    );
  }

  // Show empty state when no data
  if (!data.length) {
    const defaultEmptyState = {
      title: 'No data available',
      description: 'There are no items to display at this time.',
      icon: <Database className="h-12 w-12" />,
      ...emptyState,
    };

    return (
      <div className={cn('w-full', className)}>
        <EmptyState {...defaultEmptyState} />
      </div>
    );
  }

  // Show data table with data
  return (
    <div className={cn('w-full', className)}>
      <DataTable
        columns={columns}
        data={data}
        {...(initialVisibilityState && { initialVisibilityState })}
        {...(onRefresh && { onRefresh })}
        {...(headerActions && { headerActions })}
      />
    </div>
  );
}