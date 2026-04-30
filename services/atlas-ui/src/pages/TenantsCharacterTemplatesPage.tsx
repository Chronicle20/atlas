import { TenantDetailLayout } from "@/components/features/tenants/TenantDetailLayout";
import { TemplatesForm } from "@/pages/tenants-character-templates-form";

export function TenantsCharacterTemplatesPage() {
    return (
        <TenantDetailLayout>
            <TemplatesForm />
        </TenantDetailLayout>
    );
}
