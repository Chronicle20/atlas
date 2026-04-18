import { useTenant } from "@/context/tenant-context";
import { DataTableWrapper } from "@/components/common/DataTableWrapper";
import { getColumns, hiddenColumns } from "@/pages/guilds-columns";
import { useGuilds, useInvalidateGuilds } from "@/lib/hooks/api/useGuilds";
import { useCharacters, useInvalidateCharacters } from "@/lib/hooks/api/useCharacters";
import { useTenantConfiguration } from "@/lib/hooks/api/useTenants";
import { Toaster } from "sonner";
import { GuildPageSkeleton } from "@/components/common/skeletons/GuildPageSkeleton";

export function GuildsPage() {
  const { activeTenant } = useTenant();
  const guildsQuery = useGuilds(activeTenant);
  const charactersQuery = useCharacters(activeTenant!);
  const tenantConfigQuery = useTenantConfiguration(activeTenant?.id ?? "");
  const { invalidateAll: invalidateGuilds } = useInvalidateGuilds();
  const { invalidateAll: invalidateCharacters } = useInvalidateCharacters();

  const guilds = guildsQuery.data ?? [];
  const characters = charactersQuery.data ?? [];
  const tenantConfig = tenantConfigQuery.data ?? null;

  const loading = guildsQuery.isLoading || charactersQuery.isLoading || tenantConfigQuery.isLoading;
  const error = guildsQuery.error?.message ?? charactersQuery.error?.message ?? tenantConfigQuery.error?.message ?? null;

  const refresh = () => {
    invalidateGuilds();
    invalidateCharacters();
  };

  const characterMap = new Map(characters.map(c => [c.id, c]));
  const columns = getColumns({ tenant: tenantConfig, characterMap });

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
      <div className="mt-4">
        <DataTableWrapper
          columns={columns}
          data={guilds}
          error={error}
          onRefresh={refresh}
          initialVisibilityState={hiddenColumns}
          emptyState={{
            title: "No guilds found",
            description: "There are no guilds to display at this time.",
          }}
        />
      </div>
      <Toaster richColors />
    </div>
  );
}
