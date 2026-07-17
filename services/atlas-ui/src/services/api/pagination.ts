/**
 * Shared pagination helpers for atlas-ui list endpoints.
 *
 * The Go services emit a JSON:API-ish envelope for paginated collections:
 * `{ data: T[], meta: { total, page: { number, size, last } }, links: {...} }`.
 * Endpoints not yet converted to pagination emit no `meta` at all — per the
 * compat rule, the single response in that case *is* the whole collection.
 * This mirrors the Go client's `DrainProvider` semantics (task-117 Task 4):
 * drain page 1..meta.page.last, or treat a `meta === null` response as
 * already-complete.
 */
import { api } from "@/lib/api/client";
import type { ApiRequestOptions } from "@/lib/api/client";

export interface PageMeta {
  total: number;
  page: {
    number: number;
    size: number;
    last: number;
  };
}

export interface PagedResult<T> {
  data: T[];
  meta: PageMeta | null;
}

const DEFAULT_DRAIN_SIZE = 250;

function withPageParams(url: string, page: { number: number; size: number }): string {
  const queryIndex = url.indexOf("?");
  const path = queryIndex === -1 ? url : url.slice(0, queryIndex);
  const query = queryIndex === -1 ? "" : url.slice(queryIndex + 1);

  const params = new URLSearchParams(query);
  params.set("page[number]", String(page.number));
  params.set("page[size]", String(page.size));
  return `${path}?${params.toString()}`;
}

/**
 * Fetch a single page from a list endpoint. Appends `page[number]`/`page[size]`
 * to `url` while preserving any existing query params. `meta` is `null` when
 * the server sent no envelope (unconverted endpoint) — the returned `data` is
 * then the whole collection.
 */
export async function fetchPaged<T>(
  url: string,
  page: { number: number; size: number },
  options?: ApiRequestOptions
): Promise<PagedResult<T>> {
  const pagedUrl = withPageParams(url, page);
  const doc = await api.get<{ data: T[]; meta?: PageMeta }>(pagedUrl, options);
  return { data: doc.data ?? [], meta: doc.meta ?? null };
}

/**
 * Drain an entire collection. Fetches page 1 at `size` (default 250); if the
 * response carries no `meta`, that single response is the whole collection
 * (unconverted-endpoint compat). Otherwise iterates pages 2..meta.page.last,
 * stopping early if a page comes back empty.
 */
export async function fetchAll<T>(
  url: string,
  size: number = DEFAULT_DRAIN_SIZE,
  options?: ApiRequestOptions
): Promise<T[]> {
  const first = await fetchPaged<T>(url, { number: 1, size }, options);

  if (first.meta === null) {
    return first.data;
  }

  const results: T[] = [...first.data];
  const lastPage = first.meta.page.last;

  for (let pageNumber = 2; pageNumber <= lastPage; pageNumber++) {
    const next = await fetchPaged<T>(url, { number: pageNumber, size }, options);
    if (next.data.length === 0) break;
    results.push(...next.data);
  }

  return results;
}
