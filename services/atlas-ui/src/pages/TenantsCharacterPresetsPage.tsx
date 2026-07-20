import { useParams } from "react-router-dom";
import { toast } from "sonner";
import { TenantDetailLayout } from "@/components/features/tenants/TenantDetailLayout";
import {
  CharacterPresetsEditor,
  type PresetsEditorAdapter,
} from "@/components/features/characters/presets/CharacterPresetsEditor";
import {
  useTenantConfiguration,
  useUpdateTenantConfiguration,
  useTenant,
} from "@/lib/hooks/api/useTenants";

export function TenantsCharacterPresetsPage() {
  const { id } = useParams();
  const tenantQuery = useTenantConfiguration(id ?? "");
  const updateTenantConfig = useUpdateTenantConfiguration();
  const tenantBasicQuery = useTenant(id ?? "");
  const tenant = tenantQuery.data;

  const adapter: PresetsEditorAdapter = {
    presets: tenant?.attributes.characters.presets,
    isLoading: tenantQuery.isLoading,
    error: tenantQuery.error ?? null,
    isSaving: updateTenantConfig.isPending,
    ...(tenantBasicQuery.data ? { apply: { tenant: tenantBasicQuery.data } } : {}),
    save: (presets, onSuccess) => {
      if (!tenant) return;
      updateTenantConfig.mutate(
        {
          tenant,
          updates: {
            characters: { ...tenant.attributes.characters, presets },
          },
        },
        {
          onSuccess: (updated) => {
            toast.success("Successfully saved presets.");
            onSuccess(updated.attributes.characters.presets);
          },
          onError: (error) =>
            toast.error(`Failed to update presets: ${error.message}`),
        },
      );
    },
  };

  return (
    <TenantDetailLayout>
      <CharacterPresetsEditor adapter={adapter} />
    </TenantDetailLayout>
  );
}
