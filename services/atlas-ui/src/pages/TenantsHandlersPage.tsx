import { useParams } from "react-router-dom";
import { TenantDetailLayout } from "@/components/features/tenants/TenantDetailLayout";
import { DefinitionGridPage } from "@/components/features/socket/DefinitionGridPage";
import { TenantResetButton } from "@/components/features/tenants/TenantResetButton";

export function TenantsHandlersPage() {
  const { id } = useParams();
  return (
    <TenantDetailLayout>
      <div className="flex justify-end">
        <TenantResetButton
          id={id}
          sections={["socket"]}
          sectionLabel="socket handlers and writers"
        />
      </div>
      <DefinitionGridPage kind="handler" scope="tenant" />
    </TenantDetailLayout>
  );
}
