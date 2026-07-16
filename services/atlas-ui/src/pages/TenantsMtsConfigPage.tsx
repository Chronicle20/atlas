import { TenantDetailLayout } from "@/components/features/tenants/TenantDetailLayout";
import { MtsConfigForm } from "@/pages/tenants-mts-config-form";

export function TenantsMtsConfigPage() {
    return (
        <TenantDetailLayout>
            <MtsConfigForm />
        </TenantDetailLayout>
    );
}
