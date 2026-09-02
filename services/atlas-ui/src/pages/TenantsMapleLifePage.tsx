import { useParams } from "react-router-dom";
import { toast } from "sonner";
import { TenantDetailLayout } from "@/components/features/tenants/TenantDetailLayout";
import {
  MapleLifeEditor,
  type MapleLifeEditorAdapter,
} from "@/components/features/characters/maple-life/MapleLifeEditor";
import { supportsMapleLife } from "@/components/features/characters/maple-life/mapleLifeSupport";
import { TenantResetButton } from "@/components/features/tenants/TenantResetButton";
import {
  useTenantConfiguration,
  useUpdateTenantConfiguration,
} from "@/lib/hooks/api/useTenants";

export function TenantsMapleLifePage() {
  const { id } = useParams();
  const tenantQuery = useTenantConfiguration(id ?? "");
  const updateTenantConfig = useUpdateTenantConfiguration();
  const tenant = tenantQuery.data;

  if (
    !tenantQuery.isLoading &&
    !tenantQuery.error &&
    tenant &&
    !supportsMapleLife(tenant.attributes.socket)
  ) {
    return (
      <TenantDetailLayout>
        <p className="text-sm text-muted-foreground">
          This client version has no Maple Life dialog.
        </p>
      </TenantDetailLayout>
    );
  }

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
      <div className="flex justify-end">
        <TenantResetButton
          id={id}
          sections={["mapleLife"]}
          sectionLabel="Maple Life configuration"
        />
      </div>
      <MapleLifeEditor adapter={adapter} />
    </TenantDetailLayout>
  );
}
