import { useParams } from "react-router-dom";
import { toast } from "sonner";
import { TenantDetailLayout } from "@/components/features/tenants/TenantDetailLayout";
import {
  MapleLifeEditor,
  type MapleLifeEditorAdapter,
} from "@/components/features/characters/maple-life/MapleLifeEditor";
import {
  useTenantConfiguration,
  useUpdateTenantConfiguration,
} from "@/lib/hooks/api/useTenants";

export function TenantsMapleLifePage() {
  const { id } = useParams();
  const tenantQuery = useTenantConfiguration(id ?? "");
  const updateTenantConfig = useUpdateTenantConfiguration();
  const tenant = tenantQuery.data;

  const adapter: MapleLifeEditorAdapter = {
    mapleLife: tenant?.attributes.mapleLife,
    isLoading: tenantQuery.isLoading,
    error: tenantQuery.error ?? null,
    isSaving: updateTenantConfig.isPending,
    ...(tenant
      ? {
          seedFrom: {
            region: tenant.attributes.region,
            majorVersion: tenant.attributes.majorVersion,
            minorVersion: tenant.attributes.minorVersion,
          },
        }
      : {}),
    save: (mapleLife, onSuccess) => {
      if (!tenant) return;
      updateTenantConfig.mutate(
        { tenant, updates: { mapleLife } },
        {
          onSuccess: () => {
            toast.success("Successfully saved Maple Life configuration.");
            onSuccess();
          },
          onError: (error) =>
            toast.error(
              `Failed to update Maple Life configuration: ${error.message}`,
            ),
        },
      );
    },
  };

  return (
    <TenantDetailLayout>
      <MapleLifeEditor adapter={adapter} />
    </TenantDetailLayout>
  );
}
