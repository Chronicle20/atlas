import { useParams } from "react-router-dom";
import { toast } from "sonner";
import { TenantDetailLayout } from "@/components/features/tenants/TenantDetailLayout";
import { TenantResetButton } from "@/components/features/tenants/TenantResetButton";
import {
  CharacterTemplatesEditor,
  type TemplatesEditorAdapter,
} from "@/components/features/characters/templates/CharacterTemplatesEditor";
import {
  useTenantConfiguration,
  useUpdateTenantConfiguration,
} from "@/lib/hooks/api/useTenants";

export function TenantsCharacterTemplatesPage() {
  const { id } = useParams();
  const tenantQuery = useTenantConfiguration(id ?? "");
  const updateTenantConfig = useUpdateTenantConfiguration();
  const tenant = tenantQuery.data;

  const adapter: TemplatesEditorAdapter = {
    templates: tenant?.attributes.characters.templates,
    isLoading: tenantQuery.isLoading,
    error: tenantQuery.error ?? null,
    isSaving: updateTenantConfig.isPending,
    save: (templates, onSuccess) => {
      if (!tenant) return;
      updateTenantConfig.mutate(
        {
          tenant,
          updates: {
            characters: { ...tenant.attributes.characters, templates },
          },
        },
        {
          onSuccess: () => {
            toast.success("Successfully saved tenant configuration.");
            onSuccess();
          },
          onError: (error) =>
            toast.error(
              `Failed to update tenant configuration: ${error.message}`,
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
          sections={["characters"]}
          sectionLabel="character templates and presets"
        />
      </div>
      <CharacterTemplatesEditor adapter={adapter} />
    </TenantDetailLayout>
  );
}
