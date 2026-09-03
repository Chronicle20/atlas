import { useParams } from "react-router-dom";
import { TenantDetailLayout } from "@/components/features/tenants/TenantDetailLayout";
import { DefinitionGridPage } from "@/components/features/socket/DefinitionGridPage";
import { TenantSectionResetBar } from "@/components/features/tenants/TenantSectionResetBar";

export function TenantsWritersPage() {
  const { id } = useParams();
  return (
    <TenantDetailLayout>
      <TenantSectionResetBar
        id={id}
        sections={["socket"]}
        sectionLabel="socket handlers and writers"
      />
      <DefinitionGridPage kind="writer" scope="tenant" />
    </TenantDetailLayout>
  );
}
