import * as React from 'react';
import { AlertTriangle, RefreshCw, Home } from 'lucide-react';
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { cn } from '@/lib/utils';

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
  errorInfo: React.ErrorInfo | null;
}

interface ErrorBoundaryProps {
  children: React.ReactNode;
  fallback?: React.ComponentType<{
    error: Error;
    errorInfo: React.ErrorInfo | null;
    resetError: () => void;
  }>;
  onError?: (error: Error, errorInfo: React.ErrorInfo) => void;
  showDetails?: boolean;
  className?: string;
}

interface ErrorFallbackProps {
  error: Error;
  errorInfo: React.ErrorInfo | null;
  resetError: () => void;
  showDetails?: boolean;
  className?: string;
}

/**
 * Default error fallback component that displays error information
 * and provides options to retry or go home
 */
function DefaultErrorFallback({ 
  error, 
  errorInfo, 
  resetError, 
  showDetails = false,
  className 
}: ErrorFallbackProps) {
  const [detailsExpanded, setDetailsExpanded] = React.useState(false);

  const handleGoHome = () => {
    window.location.href = '/';
  };

  return (
    <div className={cn('flex items-center justify-center min-h-[50vh] p-4', className)}>
      <Card className="w-full max-w-lg">
        <CardHeader className="text-center">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-destructive/10">
            <AlertTriangle className="h-6 w-6 text-destructive" />
          </div>
          <CardTitle className="text-xl">Something went wrong</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <Alert variant="destructive">
            <AlertTriangle />
            <AlertTitle>Application Error</AlertTitle>
            <AlertDescription>
              {error.message || 'An unexpected error occurred while rendering this component.'}
            </AlertDescription>
          </Alert>

          {(showDetails || process.env.NODE_ENV === 'development') && (
            <div className="space-y-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setDetailsExpanded(!detailsExpanded)}
                className="w-full"
              >
                {detailsExpanded ? 'Hide' : 'Show'} Technical Details
              </Button>
              
              {detailsExpanded && (
                <div className="rounded-md bg-muted p-3 text-sm">
                  <div className="font-medium mb-2">Error Details:</div>
                  <div className="font-mono text-xs break-all">
                    <div className="mb-2">
                      <span className="text-muted-foreground">Name:</span> {error.name}
                    </div>
                    <div className="mb-2">
                      <span className="text-muted-foreground">Message:</span> {error.message}
                    </div>
                    {error.stack && (
                      <div className="mb-2">
                        <span className="text-muted-foreground">Stack:</span>
                        <pre className="mt-1 whitespace-pre-wrap text-xs">
                          {error.stack}
                        </pre>
                      </div>
                    )}
                    {errorInfo?.componentStack && (
                      <div>
                        <span className="text-muted-foreground">Component Stack:</span>
                        <pre className="mt-1 whitespace-pre-wrap text-xs">
                          {errorInfo.componentStack}
                        </pre>
                      </div>
                    )}
                  </div>
                </div>
              )}
            </div>
          )}

          <div className="flex gap-2">
            <Button
              onClick={resetError}
              className="flex-1"
              variant="default"
            >
              <RefreshCw className="h-4 w-4 mr-2" />
              Try Again
            </Button>
            <Button
              onClick={handleGoHome}
              variant="outline"
              className="flex-1"
            >
              <Home className="h-4 w-4 mr-2" />
              Go Home
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

/**
 * React Error Boundary component that catches JavaScript errors anywhere in the child component tree,
 * logs those errors, and displays a fallback UI instead of the component tree that crashed.
 * 
 * @example
 * ```tsx
 * <ErrorBoundary>
 *   <MyComponent />
 * </ErrorBoundary>
 * ```
 * 
 * @example
 * ```tsx
 * <ErrorBoundary 
 *   fallback={CustomErrorFallback}
 *   onError={(error, errorInfo) => console.error('Error caught:', error)}
 *   showDetails={true}
 * >
 *   <MyComponent />
 * </ErrorBoundary>
 * ```
 */
export class ErrorBoundary extends React.Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = {
      hasError: false,
      error: null,
      errorInfo: null,
    };
  }

  static getDerivedStateFromError(error: Error): Partial<ErrorBoundaryState> {
    // Update state so the next render will show the fallback UI
    return {
      hasError: true,
      error,
    };
  }

  override componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    // Update state with error info
    this.setState({
      error,
      errorInfo,
    });

    // Call optional error handler
    this.props.onError?.(error, errorInfo);

    // Log error to console in development
    if (process.env.NODE_ENV === 'development') {
      console.error('ErrorBoundary caught an error:', error);
      console.error('Error Info:', errorInfo);
    }
  }

  resetError = () => {
    this.setState({
      hasError: false,
      error: null,
      errorInfo: null,
    });
  };

  override render() {
    if (this.state.hasError && this.state.error) {
      const FallbackComponent = this.props.fallback || DefaultErrorFallback;
      
      const fallbackProps: ErrorFallbackProps = {
        error: this.state.error,
        errorInfo: this.state.errorInfo,
        resetError: this.resetError,
        showDetails: this.props.showDetails ?? false,
        ...(this.props.className ? { className: this.props.className } : {}),
      };

      return <FallbackComponent {...fallbackProps} />;
    }

    return this.props.children;
  }
}

/**
 * Hook-based wrapper for functional components that provides error boundary functionality
 */
export function withErrorBoundary<T extends Record<string, unknown>>(
  Component: React.ComponentType<T>,
  errorBoundaryProps?: Omit<ErrorBoundaryProps, 'children'>
) {
  const WrappedComponent = (props: T) => (
    <ErrorBoundary {...errorBoundaryProps}>
      <Component {...props} />
    </ErrorBoundary>
  );

  WrappedComponent.displayName = `withErrorBoundary(${Component.displayName || Component.name})`;
  
  return WrappedComponent;
}

/**
 * Context for accessing error boundary reset functionality from child components
 */
export const ErrorBoundaryContext = React.createContext<{
  resetError: () => void;
} | null>(null);

/**
 * Hook to access error boundary reset functionality
 */
export function useErrorBoundary() {
  const context = React.useContext(ErrorBoundaryContext);
  
  if (!context) {
    throw new Error('useErrorBoundary must be used within an ErrorBoundary');
  }
  
  return context;
}

/**
 * Provider version of ErrorBoundary that exposes reset functionality through context
 */
export function ErrorBoundaryProvider({ children, ...props }: ErrorBoundaryProps) {
  const boundaryRef = React.useRef<ErrorBoundary>(null);
  
  const resetError = React.useCallback(() => {
    boundaryRef.current?.resetError();
  }, []);

  return (
    <ErrorBoundary ref={boundaryRef} {...props}>
      <ErrorBoundaryContext.Provider value={{ resetError }}>
        {children}
      </ErrorBoundaryContext.Provider>
    </ErrorBoundary>
  );
}