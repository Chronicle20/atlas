import * as React from 'react';
import { AlertCircle, RefreshCw } from 'lucide-react';
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

interface ErrorDisplayProps {
  error: Error | { message: string } | string;
  retry?: () => void;
  className?: string;
  title?: string;
  showIcon?: boolean;
}

export function ErrorDisplay({ 
  error, 
  retry, 
  className, 
  title = 'Error',
  showIcon = true 
}: ErrorDisplayProps) {
  const message = typeof error === 'string' ? error : error.message;
  
  return (
    <Alert variant="destructive" className={cn('', className)}>
      {showIcon && <AlertCircle />}
      <AlertTitle>{title}</AlertTitle>
      <AlertDescription>
        <div className="flex items-center justify-between gap-4">
          <span className="flex-1">{message}</span>
          {retry && (
            <Button 
              size="sm" 
              variant="outline" 
              onClick={retry}
              className="shrink-0"
            >
              <RefreshCw className="h-3 w-3 mr-1" />
              Try Again
            </Button>
          )}
        </div>
      </AlertDescription>
    </Alert>
  );
}