"use client"

import { useMaps } from "@/lib/hooks/api/useMaps";
import { DataTableWrapper } from "@/components/common/DataTableWrapper";
import { columns } from "./columns";
import { PageLoader } from "@/components/common/PageLoader";

export default function MapsPage() {
  const { data: maps, isLoading, error, refetch } = useMaps();

  if (isLoading) {
    return <PageLoader />;
  }

  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
      <div className="items-center justify-between space-y-2">
        <h2 className="text-2xl font-bold tracking-tight">Maps</h2>
      </div>
      <div className="mt-4">
        <DataTableWrapper
          columns={columns}
          data={maps ?? []}
          error={error}
          onRefresh={() => refetch()}
          emptyState={{
            title: "No maps found",
            description: "Game data may not have been uploaded yet. Go to Setup to upload game data.",
          }}
        />
      </div>
    </div>
  );
}
