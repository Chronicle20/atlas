import { TenantDetailLayout } from "@/components/features/tenants/TenantDetailLayout";
import { WritersForm } from "@/pages/tenants-writers-form";

export function TenantsWritersPage() {
    return (
        <TenantDetailLayout>
            <WritersForm />
        </TenantDetailLayout>
    );
}
