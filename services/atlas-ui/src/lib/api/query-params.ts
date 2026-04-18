/**
 * Shared types and helpers used by the service layer. Extracted from the
 * (now-deleted) `services/api/base.service.ts` during task-004.
 *
 * Services call `api.*` directly for HTTP — this module only exists to
 * keep the `ServiceOptions` / `QueryOptions` / `BatchOptions` /
 * `BatchResult` / `ValidationError` vocabulary that hooks and pages
 * already import.
 */

import type { ApiRequestOptions, CacheOptions } from "@/lib/api/client";

/** Base options accepted by every service method. */
export interface ServiceOptions extends ApiRequestOptions {
  /** @deprecated React Query owns caching; ignored. */
  useCache?: boolean;
  /** @deprecated React Query owns caching; ignored. */
  cacheConfig?: CacheOptions;
  /** Whether to validate payloads client-side before POST/PUT/PATCH. */
  validate?: boolean;
}

/** Options for list-style (getAll-style) queries. */
export interface QueryOptions extends ServiceOptions {
  /** Full-text search term. */
  search?: string;
  /** Field to sort by. */
  sortBy?: string;
  /** Sort direction. */
  sortOrder?: "asc" | "desc";
  /** Max items per page. */
  limit?: number;
  /** Page offset or page number. */
  offset?: number;
  /** Additional filters serialised as `filter[key]=value`. */
  filters?: Record<string, unknown>;
  /** Sparse fieldsets per resource, e.g. `{ maps: ["name", "streetName"] }`. */
  fields?: Record<string, string[]>;
}

/** Configuration for batch operations. */
export interface BatchOptions {
  concurrency?: number;
  failFast?: boolean;
  delay?: number;
}

/** Shape returned by batch helpers. */
export interface BatchResult<T> {
  successes: T[];
  failures: Array<{ item: unknown; error: Error }>;
  total: number;
  successCount: number;
  failureCount: number;
}

export interface ValidationError {
  field: string;
  message: string;
  value?: unknown;
}

/**
 * Build a `?foo=bar&baz=qux` query string from a QueryOptions object.
 * Returns an empty string if no params apply, so callers can append directly.
 */
export function buildQueryString(options?: QueryOptions): string {
  if (!options) return "";
  const params = new URLSearchParams();

  if (options.search) params.append("search", options.search);
  if (options.sortBy) params.append("sortBy", options.sortBy);
  if (options.sortOrder) params.append("sortOrder", options.sortOrder);
  if (options.limit !== undefined) params.append("limit", String(options.limit));
  if (options.offset !== undefined) params.append("offset", String(options.offset));

  if (options.filters) {
    for (const [key, value] of Object.entries(options.filters)) {
      if (value !== null && value !== undefined) {
        params.append(`filter[${key}]`, String(value));
      }
    }
  }

  if (options.fields) {
    for (const [resource, fieldList] of Object.entries(options.fields)) {
      params.append(`fields[${resource}]`, fieldList.join(","));
    }
  }

  const qs = params.toString();
  return qs ? `?${qs}` : "";
}

/**
 * Run an async operation across an array with bounded concurrency and
 * structured success/failure reporting. Mirrors the batch helpers that
 * used to live on BaseService.
 */
export async function runBatch<TItem, TResult>(
  items: TItem[],
  fn: (item: TItem) => Promise<TResult>,
  options?: BatchOptions,
): Promise<BatchResult<TResult>> {
  const concurrency = options?.concurrency ?? 5;
  const failFast = options?.failFast ?? false;
  const delayMs = options?.delay ?? 0;

  const successes: TResult[] = [];
  const failures: Array<{ item: unknown; error: Error }> = [];

  for (let i = 0; i < items.length; i += concurrency) {
    const batch = items.slice(i, i + concurrency);
    const results = await Promise.all(
      batch.map(async (item) => {
        try {
          return { ok: true, result: await fn(item) } as const;
        } catch (err) {
          return { ok: false, item, error: err as Error } as const;
        }
      }),
    );

    for (const result of results) {
      if (result.ok) {
        successes.push(result.result);
      } else {
        failures.push({ item: result.item, error: result.error });
        if (failFast) throw result.error;
      }
    }

    if (delayMs > 0 && i + concurrency < items.length) {
      await new Promise((resolve) => setTimeout(resolve, delayMs));
    }
  }

  return {
    successes,
    failures,
    total: items.length,
    successCount: successes.length,
    failureCount: failures.length,
  };
}
