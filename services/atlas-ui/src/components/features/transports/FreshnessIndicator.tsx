import { AlertTriangle } from "lucide-react";

import { cn } from "@/lib/utils";
import { useClock } from "@/lib/utils/clock";

interface FreshnessIndicatorProps {
  /** React Query's dataUpdatedAt for the page's primary query. */
  dataUpdatedAt: number;
  isFetching: boolean;
  isError: boolean;
}

/**
 * Says how fresh the board is. The age ticks off the same shared clock as the
 * countdowns, so this adds no timer of its own.
 */
export function FreshnessIndicator({
  dataUpdatedAt,
  isFetching,
  isError,
}: FreshnessIndicatorProps) {
  const now = useClock();

  if (isError) {
    return (
      <span className="inline-flex items-center gap-1.5 text-sm text-destructive">
        <AlertTriangle className="h-4 w-4" aria-hidden="true" />
        Stale — last refresh failed
      </span>
    );
  }

  if (!dataUpdatedAt) {
    return <span className="text-sm text-muted-foreground">Loading…</span>;
  }

  const ageSeconds = Math.max(0, Math.floor((now - dataUpdatedAt) / 1000));

  return (
    <span className="inline-flex items-center gap-1.5 text-sm text-muted-foreground">
      <span
        className={cn(
          "h-2 w-2 rounded-full bg-emerald-500",
          isFetching && "animate-pulse",
        )}
        aria-hidden="true"
      />
      <span>Updated {ageSeconds}s ago</span>
    </span>
  );
}
