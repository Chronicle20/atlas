"use client"

import { useGachapons } from "@/lib/hooks/api/useGachapons";
import { DataTableWrapper } from "@/components/common/DataTableWrapper";
import { columns } from "./columns";
import { PageLoader } from "@/components/common/PageLoader";

export default function GachaponsPage() {
  const { data: gachapons, isLoading, error, refetch } = useGachapons();

  if (isLoading) {
    return <PageLoader />;
  }

  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
      <div className="items-center justify-between space-y-2">
        <h2 className="text-2xl font-bold tracking-tight">Gachapons</h2>
      </div>
      <div className="mt-4">
        <DataTableWrapper
          columns={columns}
          data={gachapons ?? []}
          error={error}
          onRefresh={() => refetch()}
          emptyState={{
            title: "No gachapons found",
            description: "Gachapon data may not have been seeded yet. Go to Setup to seed gachapons.",
          }}
        />
      </div>
    </div>
  );
}
