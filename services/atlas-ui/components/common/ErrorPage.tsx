import * as React from 'react';
import { 
  AlertTriangle, 
  FileX, 
  Shield, 
  Lock, 
  Server, 
  Home, 
  ArrowLeft, 
  RefreshCw,
  Bug,
  Clock,
  Ban
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert';
import Link from 'next/link';

interface ErrorPageProps {
  statusCode: number;
  title?: string;
  message?: string;
  showHomeButton?: boolean;
  showBackButton?: boolean;
  showRetryButton?: boolean;
  onRetry?: () => void;
  className?: string;
}

interface ErrorConfig {
  icon: React.ComponentType<{ className?: string }>;
  title: string;
  message: string;
  variant: 'destructive' | 'default';
}

const errorConfigs: Record<number, ErrorConfig> = {
  400: {
    icon: AlertTriangle,
    title: 'Bad Request',
    message: 'The request was invalid or could not be understood by the server.',
    variant: 'destructive',
  },
  401: {
    icon: Lock,
    title: 'Unauthorized',
    message: 'You need to sign in to access this resource.',
    variant: 'destructive',
  },
  403: {
    icon: Shield,
    title: 'Forbidden',
    message: 'You don&apos;t have permission to access this resource.',
    variant: 'destructive',
  },
  404: {
    icon: FileX,
    title: 'Page Not Found',
    message: 'The page you&apos;re looking for doesn&apos;t exist or has been moved.',
    variant: 'default',
  },
  408: {
    icon: Clock,
    title: 'Request Timeout',
    message: 'The request took too long to complete. Please try again.',
    variant: 'destructive',
  },
  429: {
    icon: Ban,
    title: 'Too Many Requests',
    message: 'You&apos;ve made too many requests. Please wait a moment and try again.',
    variant: 'destructive',
  },
  500: {
    icon: Server,
    title: 'Internal Server Error',
    message: 'An unexpected error occurred on the server. Please try again later.',
    variant: 'destructive',
  },
  502: {
    icon: Server,
    title: 'Bad Gateway',
    message: 'The server received an invalid response. Please try again later.',
    variant: 'destructive',
  },
  503: {
    icon: Server,
    title: 'Service Unavailable',
    message: 'The service is temporarily unavailable. Please try again later.',
    variant: 'destructive',
  },
  504: {
    icon: Clock,
    title: 'Gateway Timeout',
    message: 'The server took too long to respond. Please try again later.',
    variant: 'destructive',
  },
};

/**
 * Reusable error page component for displaying various HTTP error statuses
 * 
 * @example
 * ```tsx
 * <ErrorPage 
 *   statusCode={404} 
 *   showHomeButton={true}
 *   showBackButton={true}
 * />
 * ```
 * 
 * @example
 * ```tsx
 * <ErrorPage 
 *   statusCode={500}
 *   title="Custom Error Title"
 *   message="Custom error message"
 *   showRetryButton={true}
 *   onRetry={() => window.location.reload()}
 * />
 * ```
 */
export function ErrorPage({
  statusCode,
  title,
  message,
  showHomeButton = true,
  showBackButton = false,
  showRetryButton = false,
  onRetry,
  className,
}: ErrorPageProps) {
  const config = errorConfigs[statusCode] || {
    icon: AlertTriangle,
    title: 'Error',
    message: 'An unexpected error occurred.',
    variant: 'destructive' as const,
  };

  const Icon = config.icon;
  const displayTitle = title || config.title;
  const displayMessage = message || config.message;

  const handleGoBack = () => {
    if (typeof window !== 'undefined') {
      window.history.back();
    }
  };

  const handleRetry = () => {
    if (onRetry) {
      onRetry();
    } else if (typeof window !== 'undefined') {
      window.location.reload();
    }
  };

  return (
    <div className={`flex min-h-screen items-center justify-center p-4 ${className || ''}`}>
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-muted">
            <Icon className="h-8 w-8 text-muted-foreground" />
          </div>
          <CardTitle className="text-2xl">{displayTitle}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <Alert variant={config.variant}>
            <Icon className="h-4 w-4" />
            <AlertTitle>Error {statusCode}</AlertTitle>
            <AlertDescription>{displayMessage}</AlertDescription>
          </Alert>
          
          <div className="flex flex-col gap-2">
            {(showHomeButton || showBackButton || showRetryButton) && (
              <div className="flex flex-col gap-2 sm:flex-row sm:justify-center">
                {showHomeButton && (
                  <Button asChild variant="default" className="flex-1">
                    <Link href="/">
                      <Home className="h-4 w-4 mr-2" />
                      Go Home
                    </Link>
                  </Button>
                )}
                
                {showBackButton && (
                  <Button 
                    variant="outline" 
                    onClick={handleGoBack}
                    className="flex-1"
                  >
                    <ArrowLeft className="h-4 w-4 mr-2" />
                    Go Back
                  </Button>
                )}
                
                {showRetryButton && (
                  <Button 
                    variant="secondary" 
                    onClick={handleRetry}
                    className="flex-1"
                  >
                    <RefreshCw className="h-4 w-4 mr-2" />
                    Try Again
                  </Button>
                )}
              </div>
            )}
            
            {process.env.NODE_ENV === 'development' && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => console.log(`Error ${statusCode}: ${displayMessage}`)}
                className="text-xs"
              >
                <Bug className="h-3 w-3 mr-1" />
                Log to Console
              </Button>
            )}
          </div>
          
          <div className="text-xs text-muted-foreground text-center">
            Error Code: {statusCode}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

/**
 * Pre-configured error page components for common HTTP status codes
 */
export const Error400Page = (props: Omit<ErrorPageProps, 'statusCode'>) => (
  <ErrorPage statusCode={400} showRetryButton={true} {...props} />
);

export const Error401Page = (props: Omit<ErrorPageProps, 'statusCode'>) => (
  <ErrorPage statusCode={401} showHomeButton={true} {...props} />
);

export const Error403Page = (props: Omit<ErrorPageProps, 'statusCode'>) => (
  <ErrorPage statusCode={403} showHomeButton={true} showBackButton={true} {...props} />
);

export const Error404Page = (props: Omit<ErrorPageProps, 'statusCode'>) => (
  <ErrorPage statusCode={404} showHomeButton={true} showBackButton={true} {...props} />
);

export const Error429Page = (props: Omit<ErrorPageProps, 'statusCode'>) => (
  <ErrorPage statusCode={429} showRetryButton={true} {...props} />
);

export const Error500Page = (props: Omit<ErrorPageProps, 'statusCode'>) => (
  <ErrorPage statusCode={500} showRetryButton={true} showHomeButton={true} {...props} />
);

export const Error502Page = (props: Omit<ErrorPageProps, 'statusCode'>) => (
  <ErrorPage statusCode={502} showRetryButton={true} showHomeButton={true} {...props} />
);

export const Error503Page = (props: Omit<ErrorPageProps, 'statusCode'>) => (
  <ErrorPage statusCode={503} showRetryButton={true} showHomeButton={true} {...props} />
);

export const Error504Page = (props: Omit<ErrorPageProps, 'statusCode'>) => (
  <ErrorPage statusCode={504} showRetryButton={true} showHomeButton={true} {...props} />
);