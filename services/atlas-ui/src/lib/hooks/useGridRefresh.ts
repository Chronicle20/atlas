import * as toast from "@/lib/utils/toast";

/**
 * Minimal structural contract the refresh hook needs from a data source.
 *
 * A hand-written interface rather than `Pick<UseQueryResult, …>` (D1): a
 * non-React-Query source — e.g. `TenantsPage`'s tenant-context shim — must
 * still be expressible as a `RefreshableQuery`.
 */
export interface RefreshableQuery {
  isFetching: boolean;
  dataUpdatedAt: number;
  refetch: () => Promise<{ isError: boolean; error: unknown }>;
}

export interface UseGridRefreshResult {
  isRefreshing: boolean;
  onRefresh: () => Promise<void>;
  lastUpdatedAt: number | null;
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
  options?: { successMessage?: string; alsoRefresh?: () => Promise<unknown> },
): UseGridRefreshResult {
  const isRefreshing = queries.some((q) => q.isFetching);

  const stamps = queries.map((q) => q.dataUpdatedAt).filter((t) => t > 0);
  const lastUpdatedAt = stamps.length ? Math.min(...stamps) : null;

  const onRefresh = async (): Promise<void> => {
    const [results] = await Promise.all([
      Promise.all(queries.map((q) => q.refetch())),
      options?.alsoRefresh?.(),
    ]);
    const failed = results.find((r) => r.isError);
    if (failed) {
      toast.error(failed.error, { context: { action: "refresh" } });
      return;
    }
    toast.success(options?.successMessage ?? "Data refreshed");
  };

  return { isRefreshing, onRefresh, lastUpdatedAt };
}
