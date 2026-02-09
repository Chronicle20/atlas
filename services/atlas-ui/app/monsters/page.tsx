"use client"

import { useMonsters } from "@/lib/hooks/api/useMonsters";
import { DataTableWrapper } from "@/components/common/DataTableWrapper";
import { columns, hiddenColumns } from "./columns";
import { PageLoader } from "@/components/common/PageLoader";

export default function MonstersPage() {
  const { data: monsters, isLoading, error, refetch } = useMonsters();

  if (isLoading) {
    return <PageLoader />;
  }

  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
      <div className="items-center justify-between space-y-2">
        <h2 className="text-2xl font-bold tracking-tight">Monsters</h2>
      </div>
      <div className="mt-4">
        <DataTableWrapper
          columns={columns}
          data={monsters ?? []}
          error={error}
          onRefresh={() => refetch()}
          initialVisibilityState={hiddenColumns}
          emptyState={{
            title: "No monsters found",
            description: "Game data may not have been uploaded yet. Go to Setup to upload game data.",
          }}
        />
      </div>
    </div>
  );
}
