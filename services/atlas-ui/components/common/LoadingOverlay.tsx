import React from 'react';
import { LoadingSpinner } from './LoadingSpinner';
import { cn } from '@/lib/utils';

interface LoadingOverlayProps {
  children: React.ReactNode;
  loading: boolean;
  className?: string;
  spinnerSize?: 'sm' | 'md' | 'lg';
}

export function LoadingOverlay({ 
  children, 
  loading, 
  className,
  spinnerSize = 'md' 
}: LoadingOverlayProps) {
  return (
    <div className={cn('relative', className)}>
      {children}
      {loading && (
        <div className="absolute inset-0 bg-background/50 flex items-center justify-center z-10">
          <LoadingSpinner size={spinnerSize} />
        </div>
      )}
    </div>
  );
}