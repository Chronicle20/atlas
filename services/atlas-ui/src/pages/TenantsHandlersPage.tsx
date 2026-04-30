import { TenantDetailLayout } from "@/components/features/tenants/TenantDetailLayout";
import { HandlersForm } from "@/pages/tenants-handlers-form";

export function TenantsHandlersPage() {
    return (
        <TenantDetailLayout>
            <HandlersForm />
        </TenantDetailLayout>
    );
}
