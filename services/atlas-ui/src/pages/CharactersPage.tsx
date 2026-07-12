import { useQueryClient } from "@tanstack/react-query";
import { useTenant } from "@/context/tenant-context";
import { DataTableWrapper } from "@/components/common/DataTableWrapper";
import { getColumns, hiddenColumns } from "@/pages/characters-columns";
import { useCharacters } from "@/lib/hooks/api/useCharacters";
import { useAccounts } from "@/lib/hooks/api/useAccounts";
import { useTenantConfiguration } from "@/lib/hooks/api/useTenants";
import { characterLocationKeys } from "@/lib/hooks/api/useCharacterLocation";
import { useGridRefresh } from "@/lib/hooks/useGridRefresh";
import { CharacterPageSkeleton } from "@/components/common/skeletons/CharacterPageSkeleton";

export function CharactersPage() {
  const { activeTenant } = useTenant();
  const charactersQuery = useCharacters(activeTenant!);
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

  const characters = charactersQuery.data ?? [];
  const accounts = accountsQuery.data ?? [];
  const tenantConfig = tenantConfigQuery.data ?? null;

  const loading = charactersQuery.isLoading || accountsQuery.isLoading || tenantConfigQuery.isLoading;
  const error = charactersQuery.error?.message ?? accountsQuery.error?.message ?? tenantConfigQuery.error?.message ?? null;

  const accountMap = new Map(accounts.map(a => [a.id, a]));
  const columns = getColumns({ tenant: activeTenant, tenantConfig, accountMap, onRefresh });

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
      </div>
    </div>
  );
}
