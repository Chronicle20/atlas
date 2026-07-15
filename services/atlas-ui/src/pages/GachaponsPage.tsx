
import { useGachaponsPage } from "@/lib/hooks/api/useGachapons";
import { DataTableWrapper } from "@/components/common/DataTableWrapper";
import { columns } from "./gachapons-columns";
import { PageLoader } from "@/components/common/PageLoader";
import { useGridRefresh } from "@/lib/hooks/useGridRefresh";
import { Pager } from "@/components/common/Pager";
import { useSearchParams } from "react-router-dom";

const PAGE_SIZE = 50;

export function GachaponsPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const pageNumber = Math.max(1, Number.parseInt(searchParams.get("page") ?? "1", 10) || 1);

  const gachaponsQuery = useGachaponsPage({ number: pageNumber, size: PAGE_SIZE });
  const { isRefreshing, onRefresh } = useGridRefresh([gachaponsQuery]);

  const gachapons = gachaponsQuery.data?.data ?? [];
  const meta = gachaponsQuery.data?.meta ?? null;
  const loading = gachaponsQuery.isLoading;
  const error = gachaponsQuery.error?.message ?? null;

  const handlePageChange = (nextPage: number) => {
    const next = new URLSearchParams(searchParams);
    if (nextPage > 1) next.set("page", String(nextPage));
    else next.delete("page");
    setSearchParams(next, { replace: false });
  };

  if (loading) {
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
          data={gachapons}
          error={error}
          onRefresh={onRefresh}
          isRefreshing={isRefreshing}
          emptyState={{
            title: "No gachapons found",
            description: "Gachapon data may not have been seeded yet. Go to Setup to seed gachapons.",
          }}
        />
        {meta && gachapons.length > 0 && (
          <Pager
            page={meta.page.number}
            lastPage={meta.page.last}
            total={meta.total}
            pageSize={meta.page.size}
            onPageChange={handlePageChange}
          />
        )}
      </div>
    </div>
  );
}
