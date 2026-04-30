import { TenantDetailLayout } from "@/components/features/tenants/TenantDetailLayout";
import { WorldsForm } from "@/pages/tenants-worlds-form";

export function TenantsWorldsPage() {
    return (
        <TenantDetailLayout>
            <WorldsForm />
        </TenantDetailLayout>
    );
}
