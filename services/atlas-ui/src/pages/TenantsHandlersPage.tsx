import { TenantDetailLayout } from "@/components/features/tenants/TenantDetailLayout";
import { DefinitionGridPage } from "@/components/features/socket/DefinitionGridPage";

export function TenantsHandlersPage() {
  return (
    <TenantDetailLayout>
      <DefinitionGridPage kind="handler" scope="tenant" />
    </TenantDetailLayout>
  );
}
