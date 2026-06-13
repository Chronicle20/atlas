import type { UseQueryResult } from "@tanstack/react-query";
import * as toast from "@/lib/utils/toast";

/** Minimal slice of a React Query result the refresh hook needs. */
export type RefreshableQuery = Pick<UseQueryResult, "isFetching" | "refetch">;

export interface UseGridRefreshResult {
  isRefreshing: boolean;
  onRefresh: () => Promise<void>;
}

/**
 * Centralizes grid refresh feedback. Accepts the page's query/queries,
 * refetches them in parallel, and surfaces success/error via the app toast.
 *
 * `isRefreshing` is sourced from React Query's own `isFetching` (FR-1.2), not a
 * local timer, so it covers user-clicked and background refetches alike and
 * returns to idle exactly when React Query says fetching ended.
 *
 * NOTE: `refetch()` RESOLVES (it does not reject — React Query v5 default
 * `throwOnError: false`). Error detection therefore inspects each resolved
 * result's `isError`/`error`; do not rely on a thrown exception.
 */
export function useGridRefresh(
  queries: RefreshableQuery[],
  options?: { successMessage?: string },
): UseGridRefreshResult {
  const isRefreshing = queries.some((q) => q.isFetching);

  const onRefresh = async (): Promise<void> => {
    const results = await Promise.all(queries.map((q) => q.refetch()));
    const failed = results.find((r) => r.isError);
    if (failed) {
      toast.error(failed.error, { context: { action: "refresh" } });
      return;
    }
    toast.success(options?.successMessage ?? "Data refreshed");
  };

  return { isRefreshing, onRefresh };
}
