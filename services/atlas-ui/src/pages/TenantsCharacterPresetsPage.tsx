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
          onSuccess: async () => {
            toast.success("Successfully saved presets.");
            // The PATCH response body carries no data (atlas-configurations
            // returns 204 on success), and the assigned preset ids only
            // exist server-side after the PATCH validator runs. Re-read the
            // configuration so newly-created presets pick up their
            // server-assigned ids in the same order they were sent.
            try {
              const fresh = await tenantQuery.refetch();
              onSuccess(fresh.data?.attributes.characters.presets);
            } catch {
              // Follow-up read failed - don't block the save UX. The
              // reducer still reconciles positionally, just without ids.
              onSuccess();
            }
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
