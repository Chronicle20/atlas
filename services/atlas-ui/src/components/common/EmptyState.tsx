import * as React from "react";
import { RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface EmptyStateProps {
  icon?: React.ReactNode;
  title: string;
  description?: string;
  action?: {
    label: string;
    onClick: () => void;
  };
  onRefresh?: () => void | Promise<void>;
  isRefreshing?: boolean;
  lastUpdatedAt?: number | null;
  className?: string;
}

export function EmptyState({
  icon,
  title,
  description,
  action,
  onRefresh,
  isRefreshing,
  lastUpdatedAt,
  className,
}: EmptyStateProps) {
  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center p-8 text-center",
        className,
      )}
      data-testid="empty-state"
    >
      {icon && <div className="mb-4 text-muted-foreground">{icon}</div>}
      <h3 className="text-lg font-semibold">{title}</h3>
      {description && (
        <p className="mt-2 text-sm text-muted-foreground max-w-sm">
          {description}
        </p>
      )}
      {(action || onRefresh) && (
        <div className="mt-4 flex items-center justify-center gap-2">
          {action && <Button onClick={action.onClick}>{action.label}</Button>}
          {onRefresh && (
            <Button
              variant="outline"
              onClick={() => void onRefresh()}
              disabled={isRefreshing}
              aria-busy={isRefreshing}
              data-testid="empty-state-refresh"
            >
              <RefreshCw
                className={cn("h-4 w-4", isRefreshing && "animate-spin")}
              />
              Refresh
            </Button>
          )}
        </div>
      )}
      {lastUpdatedAt != null && lastUpdatedAt > 0 && (
        <p
          className="mt-3 text-xs text-muted-foreground"
          data-testid="empty-state-last-updated"
          title={new Date(lastUpdatedAt).toISOString()}
        >
          Last updated{" "}
          {new Date(lastUpdatedAt).toLocaleTimeString(undefined, {
            hour: "2-digit",
            minute: "2-digit",
          })}
        </p>
      )}
    </div>
  );
}
