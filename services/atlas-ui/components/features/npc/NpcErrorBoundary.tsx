"use client"

import React from 'react';
import { AlertTriangle, RefreshCw, Database, Wifi } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert';
import { ErrorBoundary } from '@/components/common/ErrorBoundary';
import { cn } from '@/lib/utils';
import { errorLogger } from '@/services/errorLogger';

interface NpcErrorFallbackProps {
  error: Error;
  errorInfo: React.ErrorInfo | null;
  resetError: () => void;
  npcId?: number;
  className?: string;
}

/**
 * Specialized error fallback for NPC cards that provides context-specific recovery options
 */
function NpcErrorFallback({ 
  error, 
  errorInfo: _errorInfo, // eslint-disable-line @typescript-eslint/no-unused-vars
  resetError, 
  npcId,
  className 
}: NpcErrorFallbackProps) {
  const isNetworkError = error.message.includes('fetch') || 
                        error.message.includes('network') ||
                        error.message.includes('Failed to fetch');
  
  const isApiError = error.message.includes('404') || 
                    error.message.includes('API') ||
                    error.message.includes('not found');

  const handleRetryWithLog = () => {
    errorLogger.logUserAction('npc_error_retry', { 
      npcId, 
      errorType: error.name,
      errorMessage: error.message 
    });
    resetError();
  };

  const getErrorIcon = () => {
    if (isNetworkError) return <Wifi className="h-4 w-4" />;
    if (isApiError) return <Database className="h-4 w-4" />;
    return <AlertTriangle className="h-4 w-4" />;
  };

  const getErrorTitle = () => {
    if (isNetworkError) return "Connection Issue";
    if (isApiError) return "NPC Data Unavailable";
    return "Display Error";
  };

  const getErrorMessage = () => {
    if (isNetworkError) {
      return "Unable to connect to MapleStory.io API. Please check your internet connection.";
    }
    if (isApiError) {
      return npcId 
        ? `NPC data for ID ${npcId} is not available in the MapleStory.io database.`
        : "The requested NPC data is not available.";
    }
    return "There was an error displaying this NPC card. This might be a temporary issue.";
  };

  return (
    <Card className={cn("border-destructive/50 bg-destructive/5", className)}>
      <CardHeader className="pb-2">
        <CardTitle className="text-sm flex items-center gap-2 text-destructive">
          {getErrorIcon()}
          {getErrorTitle()}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <Alert variant="destructive" className="border-destructive/20">
          <AlertTriangle className="h-4 w-4" />
          <AlertTitle className="text-sm">Error Details</AlertTitle>
          <AlertDescription className="text-xs">
            {getErrorMessage()}
          </AlertDescription>
        </Alert>

        {/* Show NPC ID as fallback information */}
        {npcId && (
          <div className="bg-muted p-2 rounded text-xs">
            <span className="font-medium">NPC ID:</span> {npcId}
          </div>
        )}

        <div className="flex gap-2">
          <Button
            onClick={handleRetryWithLog}
            size="sm"
            variant="outline"
            className="flex-1 text-xs h-8"
          >
            <RefreshCw className="h-3 w-3 mr-1" />
            Retry
          </Button>
        </div>

        {/* Development mode: Show technical details */}
        {process.env.NODE_ENV === 'development' && (
          <details className="text-xs">
            <summary className="cursor-pointer text-muted-foreground hover:text-foreground">
              Technical Details
            </summary>
            <div className="mt-2 p-2 bg-muted rounded font-mono text-xs break-all">
              <div><strong>Error:</strong> {error.name}</div>
              <div><strong>Message:</strong> {error.message}</div>
              {error.stack && (
                <div className="mt-1">
                  <strong>Stack:</strong>
                  <pre className="whitespace-pre-wrap text-xs">{error.stack}</pre>
                </div>
              )}
            </div>
          </details>
        )}
      </CardContent>
    </Card>
  );
}

interface NpcErrorBoundaryProps {
  children: React.ReactNode;
  npcId?: number;
  className?: string;
  onError?: (error: Error, errorInfo: React.ErrorInfo) => void;
}

/**
 * Specialized error boundary for NPC components with NPC-specific error handling
 */
export function NpcErrorBoundary({ 
  children, 
  npcId, 
  className,
  onError 
}: NpcErrorBoundaryProps) {
  const handleError = (error: Error, errorInfo: React.ErrorInfo) => {
    // Log NPC-specific error context
    errorLogger.logError(error, errorInfo, {
      ...(npcId && { userId: npcId.toString() }), // Use userId field for NPC ID since it's the closest match
      url: `npc_card_${npcId || 'unknown'}`,
    });

    // Call optional error handler
    onError?.(error, errorInfo);
  };

  return (
    <ErrorBoundary
      fallback={({ error, errorInfo, resetError }) => (
        <NpcErrorFallback
          error={error}
          errorInfo={errorInfo}
          resetError={resetError}
          {...(npcId !== undefined && { npcId })}
          {...(className !== undefined && { className })}
        />
      )}
      onError={handleError}
      enableErrorReporting={true}
      context={{
        ...(npcId && { userId: npcId.toString() }), // Use userId field for NPC ID
        url: `npc_card_${npcId || 'unknown'}`,
      }}
    >
      {children}
    </ErrorBoundary>
  );
}

/**
 * HOC to wrap components with NPC-specific error boundary
 */
export function withNpcErrorBoundary<T extends { npcId?: number }>(
  Component: React.ComponentType<T>
) {
  const WrappedComponent = (props: T) => (
    <NpcErrorBoundary {...(props.npcId !== undefined && { npcId: props.npcId })}>
      <Component {...props} />
    </NpcErrorBoundary>
  );

  WrappedComponent.displayName = `withNpcErrorBoundary(${Component.displayName || Component.name})`;
  
  return WrappedComponent;
}