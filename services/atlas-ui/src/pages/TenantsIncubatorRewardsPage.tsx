import { TenantDetailLayout } from "@/components/features/tenants/TenantDetailLayout";
import { IncubatorRewardsForm } from "./tenants-incubator-rewards-form";

export function TenantsIncubatorRewardsPage() {
  return (
    <TenantDetailLayout>
      <IncubatorRewardsForm />
    </TenantDetailLayout>
  );
}
