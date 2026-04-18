import * as React from "react";
import { AlertTriangle, RefreshCw, Home, Bug } from "lucide-react";
import { Link } from "react-router-dom";
import { Alert, AlertTitle, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { errorLogger } from "@/services/errorLogger";

interface ErrorFallbackProps {
  error: Error & { digest?: string };
  reset: () => void;
}

function ErrorFallback({ error, reset }: ErrorFallbackProps) {
  const [detailsExpanded, setDetailsExpanded] = React.useState(false);
  const [reportSent, setReportSent] = React.useState(false);
  const [reportLoading, setReportLoading] = React.useState(false);

  React.useEffect(() => {
    if (import.meta.env.DEV) {
      console.error("Route Error:", error);
    }
    errorLogger.logError(error).catch((logError) => {
      console.warn("Failed to log error:", logError);
    });
  }, [error]);

  const handleReportError = async () => {
    if (reportSent || reportLoading) return;
    setReportLoading(true);
    try {
      await errorLogger.logError(error);
      setReportSent(true);
    } catch (reportError) {
      console.error("Failed to send error report:", reportError);
    } finally {
      setReportLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center p-4">
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
              {error.message || "An unexpected error occurred while loading this page."}
            </AlertDescription>
          </Alert>

          {import.meta.env.DEV && (
            <div className="space-y-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setDetailsExpanded(!detailsExpanded)}
                className="w-full"
              >
                {detailsExpanded ? "Hide" : "Show"} Technical Details
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
                        <pre className="mt-1 whitespace-pre-wrap text-xs">{error.stack}</pre>
                      </div>
                    )}
                  </div>
                </div>
              )}
            </div>
          )}

          <div className="space-y-2">
            <Button
              onClick={handleReportError}
              disabled={reportSent || reportLoading}
              variant="ghost"
              size="sm"
              className="w-full"
            >
              <Bug className={`h-4 w-4 mr-2 ${reportLoading ? "animate-spin" : ""}`} />
              {reportLoading ? "Sending Report..." : reportSent ? "Report Sent" : "Report This Error"}
            </Button>

            <div className="flex gap-2">
              <Button onClick={reset} className="flex-1" variant="default">
                <RefreshCw className="h-4 w-4 mr-2" />
                Try Again
              </Button>
              <Button asChild variant="outline" className="flex-1">
                <Link to="/">
                  <Home className="h-4 w-4 mr-2" />
                  Go Home
                </Link>
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

interface Props {
  children: React.ReactNode;
}

interface State {
  error: (Error & { digest?: string }) | null;
}

export class RouteErrorBoundary extends React.Component<Props, State> {
  override state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error: error as Error & { digest?: string } };
  }

  reset = () => this.setState({ error: null });

  override render() {
    if (this.state.error) {
      return <ErrorFallback error={this.state.error} reset={this.reset} />;
    }
    return this.props.children;
  }
}
