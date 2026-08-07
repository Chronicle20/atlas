import { useEffect, useMemo, useState } from "react";
import { useQuery, keepPreviousData } from "@tanstack/react-query";
import { itemsService } from "@/services/api/items.service";
import type { ItemSearchFilters } from "@/services/api/items.service";
import type { ItemSearchResult } from "@/types/models/item";
import { useTenant } from "@/context/tenant-context";
import {
  POOL_SEARCH_CONFIGS,
  type SearchPoolKey,
} from "@/lib/items/poolSearchConfig";

const PAGE_SIZE = 50;

// Mirrors the itemStringKeys / itemCommoditiesKeys shape (`all` + one leaf,
// `as const`) with the tenant id folded into the leaf like mtsListingsKeys:
// without it, switching tenants while a picker stays open could serve one
// tenant's cached search results under another until the next debounce
// settles a new term/page — today that's only avoided because
// TenantProvider calls `queryClient.clear()` on every tenant switch.
export const itemSearchKeys = {
  all: ["item-search"] as const,
  search: (
    tenantId: string | undefined,
    poolKey: SearchPoolKey,
    term: string,
    page: number,
  ) => [...itemSearchKeys.all, tenantId ?? "", poolKey, term, page] as const,
};

export interface UseItemSearchOptions {
  poolKey: SearchPoolKey;
  /** Queries only fire while the consumer's popover is open. */
  open: boolean;
  /** Test hook: pass 0 to disable debouncing. */
  debounceMs?: number;
}

export interface UseItemSearchResult {
  search: string;
  setSearch: (value: string) => void;
  rows: ItemSearchResult[];
  /** The search box parsed as a raw id, for the "Use id N" escape hatch. */
  manualId: number | undefined;
  hasMore: boolean;
  loadMore: () => void;
  isLoading: boolean;
  isError: boolean;
  /** The debounced term the current results belong to. */
  settledTerm: string;
  reset: () => void;
}

export function useItemSearch({
  poolKey,
  open,
  debounceMs = 300,
}: UseItemSearchOptions): UseItemSearchResult {
  const { activeTenant } = useTenant();
  const [search, setSearch] = useState("");
  // The settled query term and its page are held TOGETHER so they can only
  // ever change atomically — the page must never move independently of the
  // term it belongs to. `term` updates only from the debounce timer's
  // callback (async — not a synchronous setState-in-effect, so this stays
  // clean under react-hooks/set-state-in-effect); "Load more" advances
  // `page` via a functional update that leaves `term` untouched. Raw
  // keystrokes update only `search` (below), never `settled` directly —
  // that decoupling is exactly what caused the prior regression: a
  // synchronous page reset on every keystroke could pair the OLD settled
  // term with a NEW page number and fire an un-debounced query.
  const [settled, setSettled] = useState({ term: "", page: 1 });

  useEffect(() => {
    const handle = setTimeout(() => {
      setSettled({ term: search, page: 1 });
    }, debounceMs);
    return () => clearTimeout(handle);
  }, [search, debounceMs]);

  const cfg = POOL_SEARCH_CONFIGS[poolKey];

  const filters: ItemSearchFilters = {
    pageNumber: settled.page,
    pageSize: PAGE_SIZE,
    ...(settled.term ? { q: settled.term } : {}),
    ...(cfg.compartment ? { compartment: cfg.compartment } : {}),
    ...(cfg.subcategory ? { subcategory: cfg.subcategory } : {}),
  };

  const query = useQuery({
    queryKey: itemSearchKeys.search(
      activeTenant?.id,
      poolKey,
      settled.term,
      settled.page,
    ),
    queryFn: () => itemsService.searchItems(filters),
    enabled: open && !!activeTenant && settled.term.trim().length > 0,
    placeholderData: keepPreviousData,
    staleTime: 10 * 60 * 1000,
  });

  const rows = useMemo(() => {
    const items = query.data?.items ?? [];
    return cfg.clientSubcategories
      ? items.filter((r) => cfg.clientSubcategories!.has(r.subcategory))
      : items;
  }, [query.data, cfg.clientSubcategories]);

  const manualId = /^\d+$/.test(search.trim())
    ? Number(search.trim())
    : undefined;

  return {
    search,
    setSearch,
    rows,
    manualId,
    hasMore: (query.data?.lastPage ?? 1) > settled.page,
    loadMore: () => setSettled((s) => ({ ...s, page: s.page + 1 })),
    isLoading: query.isLoading,
    isError: query.isError,
    settledTerm: settled.term,
    reset: () => setSearch(""),
  };
}
