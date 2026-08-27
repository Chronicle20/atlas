// services/atlas-ui/src/components/features/accounts/CharactersPanel.tsx
import { useMemo, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { ErrorDisplay } from "@/components/common";
import { useCharacters } from "@/lib/hooks/api/useCharacters";
import { useTenantConfiguration } from "@/lib/hooks/api/useTenants";
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
  const [addOpen, setAddOpen] = useState(false);

  const worlds = tenantConfigQuery.data?.attributes?.worlds ?? [];
  const templates =
    tenantConfigQuery.data?.attributes?.characters?.templates ?? [];
  const emptyTemplate = templates[0];
  const hasPresets =
    (tenantConfigQuery.data?.attributes?.characters?.presets ?? []).length > 0;

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
    return (
      <div className="space-y-6">
        {worlds.map((world, worldId) => (
          <WorldCharactersSection
            key={worldId}
            tenant={tenant}
            account={account}
            worldId={worldId}
            worldName={world.name}
            worlds={worlds}
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
