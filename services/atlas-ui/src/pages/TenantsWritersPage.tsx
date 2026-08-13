import { TenantDetailLayout } from "@/components/features/tenants/TenantDetailLayout";
import { DefinitionGridPage } from "@/components/features/socket/DefinitionGridPage";

export function TenantsWritersPage() {
  return (
    <TenantDetailLayout>
      <DefinitionGridPage kind="writer" scope="tenant" />
    </TenantDetailLayout>
  );
}
