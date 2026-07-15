import { useEffect, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { useTenant } from "@/context/tenant-context";
import { DataTableWrapper } from "@/components/common/DataTableWrapper";
import { getColumns, hiddenColumns } from "@/pages/guilds-columns";
import { useGuildsPage, useGuildSearch } from "@/lib/hooks/api/useGuilds";
import { useCharacters } from "@/lib/hooks/api/useCharacters";
import { useTenantConfiguration } from "@/lib/hooks/api/useTenants";
import { useGridRefresh } from "@/lib/hooks/useGridRefresh";
import { Toaster } from "sonner";
import { GuildPageSkeleton } from "@/components/common/skeletons/GuildPageSkeleton";
import { Pager } from "@/components/common/Pager";
import { Input } from "@/components/ui/input";
import { useDebounce } from "@/lib/utils/debounce";

const PAGE_SIZE = 50;
const DEBOUNCE_MS = 250;

export function GuildsPage() {
  const { activeTenant } = useTenant();
  const [searchParams, setSearchParams] = useSearchParams();
  const pageNumber = Math.max(1, Number.parseInt(searchParams.get("page") ?? "1", 10) || 1);
  const urlQ = searchParams.get("q") ?? "";
  const isSearching = urlQ.trim().length > 0;

  const [searchInput, setSearchInput] = useState(urlQ);
  const debounced = useDebounce(searchInput.trim(), DEBOUNCE_MS);

  // Search input -> URL (`q`), resetting to page 1 on every new term.
  useEffect(() => {
    if (debounced === urlQ) return;
    const next = new URLSearchParams(searchParams);
    if (debounced.length > 0) next.set("q", debounced);
    else next.delete("q");
    next.delete("page");
    setSearchParams(next, { replace: true });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [debounced]);

  const page = { number: pageNumber, size: PAGE_SIZE };

  // Only one of these two is enabled at a time based on whether a search
  // term is active (task-117: server-side filter[name] via guildsService.search,
  // plain server-side paging via guildsService.getPage).
  const browseQuery = useGuildsPage(activeTenant, page, undefined, !isSearching);
  const searchQuery = useGuildSearch(activeTenant, urlQ, page, undefined, isSearching);
  const guildsQuery = isSearching ? searchQuery : browseQuery;

  const charactersQuery = useCharacters(activeTenant!);
  const tenantConfigQuery = useTenantConfiguration(activeTenant?.id ?? "");
  const { isRefreshing, onRefresh } = useGridRefresh([
    guildsQuery,
    charactersQuery,
    tenantConfigQuery,
  ]);

  const guilds = guildsQuery.data?.data ?? [];
  const meta = guildsQuery.data?.meta ?? null;
  const characters = charactersQuery.data ?? [];
  const tenantConfig = tenantConfigQuery.data ?? null;

  const loading = guildsQuery.isLoading || charactersQuery.isLoading || tenantConfigQuery.isLoading;
  const error = guildsQuery.error?.message ?? charactersQuery.error?.message ?? tenantConfigQuery.error?.message ?? null;

  const characterMap = new Map(characters.map(c => [c.id, c]));
  const columns = getColumns({ tenant: tenantConfig, characterMap });

  const handlePageChange = (nextPage: number) => {
    const next = new URLSearchParams(searchParams);
    if (nextPage > 1) next.set("page", String(nextPage));
    else next.delete("page");
    setSearchParams(next, { replace: false });
  };

  if (loading) {
    return <GuildPageSkeleton />;
  }

  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
      <div className="items-center justify-between space-y-2">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">Guilds</h2>
        </div>
      </div>
      <div className="max-w-sm">
        <Input
          placeholder="Search guilds by name..."
          value={searchInput}
          onChange={(e) => setSearchInput(e.target.value)}
          aria-label="Search guilds"
        />
      </div>
      <div className="mt-4">
        <DataTableWrapper
          columns={columns}
          data={guilds}
          error={error}
          onRefresh={onRefresh}
          isRefreshing={isRefreshing}
          initialVisibilityState={hiddenColumns}
          emptyState={{
            title: "No guilds found",
            description: "There are no guilds to display at this time.",
          }}
        />
        {meta && guilds.length > 0 && (
          <Pager
            page={meta.page.number}
            lastPage={meta.page.last}
            total={meta.total}
            pageSize={meta.page.size}
            onPageChange={handlePageChange}
          />
        )}
      </div>
      <Toaster richColors />
    </div>
  );
}
