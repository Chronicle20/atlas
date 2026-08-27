// services/atlas-ui/src/components/features/accounts/CharactersPanel.tsx
import { useMemo, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { ErrorDisplay } from "@/components/common";
import { useCharacters } from "@/lib/hooks/api/useCharacters";
import { useTenantConfiguration } from "@/lib/hooks/api/useTenants";
import { useServices } from "@/lib/hooks/api/useServices";
import { isChannelService } from "@/types/models/service";
import { ApplyPresetDialog } from "@/components/features/characters/ApplyPresetDialog";
import type { Account } from "@/types/models/account";
import type { Tenant } from "@/types/models/tenant";
import { cn } from "@/lib/utils";
import { WorldCharactersSection } from "./WorldCharactersSection";
import { tileFrameClasses } from "./tile-frame";

interface CharactersPanelProps {
  tenant: Tenant;
  account: Account;
}

export function CharactersPanel({ tenant, account }: CharactersPanelProps) {
  const charactersQuery = useCharacters(tenant);
  const tenantConfigQuery = useTenantConfiguration(tenant.id);
  const servicesQuery = useServices();
  const [addOpen, setAddOpen] = useState(false);

  const worlds = tenantConfigQuery.data?.attributes?.worlds ?? [];
  const templates =
    tenantConfigQuery.data?.attributes?.characters?.templates ?? [];
  const emptyTemplate = templates[0];
  const hasPresets =
    (tenantConfigQuery.data?.attributes?.characters?.presets ?? []).length > 0;

  // World NAMES come from tenant configuration; world EXISTENCE comes from
  // the channel services actually deployed for this tenant. A tenant-config
  // world with no matching channel-service world entry is never rendered.
  const configuredWorldIds = useMemo(() => {
    const ids = new Set<number>();
    for (const service of servicesQuery.data ?? []) {
      if (!isChannelService(service)) {
        continue;
      }
      for (const channelTenant of service.attributes.tenants) {
        if (channelTenant.id !== tenant.id) {
          continue;
        }
        for (const world of channelTenant.worlds) {
          ids.add(world.id);
        }
      }
    }
    return ids;
  }, [servicesQuery.data, tenant.id]);

  // Preserve the tenant-config `worlds` array index as the worldId — that
  // index semantics is relied on throughout (slots lookups, character
  // grouping), so filtering must not renumber the remaining entries.
  const visibleWorlds = worlds
    .map((world, worldId) => ({ world, worldId }))
    .filter(({ worldId }) => configuredWorldIds.has(worldId));

  const accountCharacters = useMemo(() => {
    const list = charactersQuery.data ?? [];
    return list.filter((c) => c.attributes.accountId === Number(account.id));
  }, [charactersQuery.data, account.id]);

  const charactersLoading =
    charactersQuery.isLoading || charactersQuery.isFetching;

  const renderBody = () => {
    if (charactersQuery.error) {
      return <ErrorDisplay error={charactersQuery.error.message} />;
    }
    if (worlds.length === 0) {
      if (tenantConfigQuery.isLoading) {
        return (
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-4">
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton
                key={i}
                className={cn(tileFrameClasses, "animate-pulse")}
              />
            ))}
          </div>
        );
      }
      return (
        <p className="text-sm text-muted-foreground">
          No worlds are configured for this tenant.
        </p>
      );
    }
    if (servicesQuery.isLoading) {
      return (
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton
              key={i}
              className={cn(tileFrameClasses, "animate-pulse")}
            />
          ))}
        </div>
      );
    }
    if (servicesQuery.error) {
      return <ErrorDisplay error={servicesQuery.error.message} />;
    }
    if (visibleWorlds.length === 0) {
      return (
        <p className="text-sm text-muted-foreground">
          No worlds are configured in the channel services for this tenant.
        </p>
      );
    }
    return (
      <div className="space-y-6">
        {visibleWorlds.map(({ world, worldId }) => (
          <WorldCharactersSection
            key={worldId}
            tenant={tenant}
            account={account}
            worldId={worldId}
            worldName={world.name}
            characters={accountCharacters.filter(
              (c) => c.attributes.worldId === worldId,
            )}
            charactersLoading={charactersLoading}
            charactersError={charactersQuery.error}
            {...(emptyTemplate && { emptyTemplate })}
            hasPresets={hasPresets}
            onAddClick={() => setAddOpen(true)}
          />
        ))}
      </div>
    );
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>Characters</CardTitle>
      </CardHeader>
      <CardContent>{renderBody()}</CardContent>

      <ApplyPresetDialog
        tenant={tenant}
        accountId={Number(account.id)}
        open={addOpen}
        onOpenChange={setAddOpen}
      />
    </Card>
  );
}
