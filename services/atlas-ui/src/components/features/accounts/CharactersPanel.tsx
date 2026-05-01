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
import { FilledSlotTile } from "./FilledSlotTile";
import { EmptySlotTile } from "./EmptySlotTile";
import { tileFrameClasses } from "./tile-frame";

interface CharactersPanelProps {
  tenant: Tenant;
  account: Account;
}

export function CharactersPanel({ tenant, account }: CharactersPanelProps) {
  const charactersQuery = useCharacters(tenant);
  const tenantConfigQuery = useTenantConfiguration(tenant.id);
  const [addOpen, setAddOpen] = useState(false);

  const slots = account.attributes.characterSlots;
  const worlds = tenantConfigQuery.data?.attributes?.worlds ?? [];
  const hasPresets =
    (tenantConfigQuery.data?.attributes?.characters?.presets ?? []).length > 0;

  const filtered = useMemo(() => {
    const list = charactersQuery.data ?? [];
    return list.filter((c) => c.attributes.accountId === Number(account.id));
  }, [charactersQuery.data, account.id]);

  const overCapacity = filtered.length > slots;
  const emptyCount = Math.max(0, slots - filtered.length);

  const renderBody = () => {
    if (charactersQuery.isLoading || charactersQuery.isFetching) {
      return (
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-4">
          {Array.from({ length: slots }).map((_, i) => (
            <Skeleton key={i} className={`${tileFrameClasses} animate-pulse`} />
          ))}
        </div>
      );
    }
    if (charactersQuery.error) {
      return <ErrorDisplay error={charactersQuery.error.message} />;
    }
    return (
      <>
        {overCapacity && (
          <p className="text-xs text-muted-foreground mb-2">
            Over capacity: this account has more characters than allocated slots.
          </p>
        )}
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-4">
          {filtered.map((c) => (
            <FilledSlotTile key={c.id} character={c} worlds={worlds} />
          ))}
          {Array.from({ length: emptyCount }).map((_, i) => (
            <EmptySlotTile
              key={`empty-${i}`}
              onClick={() => setAddOpen(true)}
              disabled={!hasPresets}
            />
          ))}
        </div>
      </>
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
