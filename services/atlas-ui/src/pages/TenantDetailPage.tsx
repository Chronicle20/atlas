import { TenantDetailLayout } from "@/components/features/tenants/TenantDetailLayout";
import { PropertiesForm } from "@/pages/tenants-properties-form";

export function TenantDetailPage() {
    return (
        <TenantDetailLayout>
            <PropertiesForm />
        </TenantDetailLayout>
    );
}
