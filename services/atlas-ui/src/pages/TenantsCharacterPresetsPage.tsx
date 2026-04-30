import { TenantDetailLayout } from "@/components/features/tenants/TenantDetailLayout";
import { TenantsPresetsForm } from "@/pages/tenants-character-presets-form";

export function TenantsCharacterPresetsPage() {
    return (
        <TenantDetailLayout>
            <TenantsPresetsForm />
        </TenantDetailLayout>
    );
}
