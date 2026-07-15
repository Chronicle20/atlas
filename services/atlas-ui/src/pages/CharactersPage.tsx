import { useQueryClient } from "@tanstack/react-query";
import { useTenant } from "@/context/tenant-context";
import { DataTableWrapper } from "@/components/common/DataTableWrapper";
import { getColumns, hiddenColumns } from "@/pages/characters-columns";
import { useCharactersPage } from "@/lib/hooks/api/useCharacters";
import { useAccounts } from "@/lib/hooks/api/useAccounts";
import { useTenantConfiguration } from "@/lib/hooks/api/useTenants";
import { characterLocationKeys } from "@/lib/hooks/api/useCharacterLocation";
import { useGridRefresh } from "@/lib/hooks/useGridRefresh";
import { CharacterPageSkeleton } from "@/components/common/skeletons/CharacterPageSkeleton";
import { Pager } from "@/components/common/Pager";
import { useSearchParams } from "react-router-dom";

const PAGE_SIZE = 50;

export function CharactersPage() {
  const { activeTenant } = useTenant();
  const [searchParams, setSearchParams] = useSearchParams();
  const pageNumber = Math.max(1, Number.parseInt(searchParams.get("page") ?? "1", 10) || 1);

  const charactersQuery = useCharactersPage(activeTenant, { number: pageNumber, size: PAGE_SIZE });
  // Accounts are fetched in full (not paged) here — every character row on
  // the current page needs its owning account joined in, regardless of
  // which page of accounts that account would fall on.
  const accountsQuery = useAccounts(activeTenant!);
  const tenantConfigQuery = useTenantConfiguration(activeTenant?.id ?? "");
  const queryClient = useQueryClient();
  const { isRefreshing, onRefresh } = useGridRefresh(
    [charactersQuery, accountsQuery, tenantConfigQuery],
    {
      alsoRefresh: () =>
        queryClient.invalidateQueries({ queryKey: characterLocationKeys.all }),
    },
  );

  const characters = charactersQuery.data?.data ?? [];
  const meta = charactersQuery.data?.meta ?? null;
  const accounts = accountsQuery.data ?? [];
  const tenantConfig = tenantConfigQuery.data ?? null;

  const loading = charactersQuery.isLoading || accountsQuery.isLoading || tenantConfigQuery.isLoading;
  const error = charactersQuery.error?.message ?? accountsQuery.error?.message ?? tenantConfigQuery.error?.message ?? null;

  const accountMap = new Map(accounts.map(a => [a.id, a]));
  const columns = getColumns({ tenant: activeTenant, tenantConfig, accountMap, onRefresh });

  const handlePageChange = (nextPage: number) => {
    const next = new URLSearchParams(searchParams);
    if (nextPage > 1) next.set("page", String(nextPage));
    else next.delete("page");
    setSearchParams(next, { replace: false });
  };

  if (loading) {
    return <CharacterPageSkeleton />;
  }

  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
      <div className="items-center justify-between space-y-2">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">Characters</h2>
        </div>
      </div>
      <div className="mt-4">
        <DataTableWrapper
          columns={columns}
          data={characters}
          error={error}
          onRefresh={onRefresh}
          isRefreshing={isRefreshing}
          initialVisibilityState={hiddenColumns}
          emptyState={{
            title: "No characters found",
            description: "There are no characters to display at this time.",
          }}
        />
        {meta && characters.length > 0 && (
          <Pager
            page={meta.page.number}
            lastPage={meta.page.last}
            total={meta.total}
            pageSize={meta.page.size}
            onPageChange={handlePageChange}
          />
        )}
      </div>
    </div>
  );
}
