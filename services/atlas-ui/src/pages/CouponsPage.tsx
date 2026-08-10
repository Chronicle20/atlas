/**
 * Cash-shop coupon codes.
 *
 * Note there is no global redemption list here, by design: redemption history
 * is per-code (the detail page) and per-account only, so this page never
 * fetches the whole audit trail.
 */

import { useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { DataTableWrapper } from "@/components/common/DataTableWrapper";
import { Pager } from "@/components/common/Pager";
import { CouponPageSkeleton } from "@/components/common/skeletons/CouponPageSkeleton";
import { CreateCouponDialog } from "@/components/features/coupons/CreateCouponDialog";
import { GenerateCouponBatchDialog } from "@/components/features/coupons/GenerateCouponBatchDialog";
import { getColumns, hiddenColumns } from "@/pages/coupons-columns";
import { useGridRefresh } from "@/lib/hooks/useGridRefresh";
import {
  useCoupons,
  useDeleteCoupon,
  useUpdateCoupon,
} from "@/lib/hooks/api/useCoupons";
import {
  CouponConflictError,
  type Coupon,
  type CouponFilters,
} from "@/services/api/coupons.service";
import { createErrorFromUnknown } from "@/types/api/errors";

const PAGE_SIZE = 50;

type StatusFilter = "all" | "active" | "inactive";

const STATUS_FILTERS: [StatusFilter, string][] = [
  ["all", "All"],
  ["active", "Active only"],
  ["inactive", "Inactive only"],
];

export function CouponsPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const pageNumber = Math.max(
    1,
    Number.parseInt(searchParams.get("page") ?? "1", 10) || 1,
  );

  const [status, setStatus] = useState<StatusFilter>("all");
  const [codeDraft, setCodeDraft] = useState("");
  const [codeFilter, setCodeFilter] = useState("");

  const [createOpen, setCreateOpen] = useState(false);
  const [batchOpen, setBatchOpen] = useState(false);
  const [pendingToggleId, setPendingToggleId] = useState<string | null>(null);
  const [couponToDelete, setCouponToDelete] = useState<Coupon | null>(null);
  const [conflict, setConflict] = useState<string | null>(null);

  const filters = useMemo<CouponFilters | undefined>(() => {
    const next: CouponFilters = {};
    if (status !== "all") next.active = status === "active";
    if (codeFilter) next.code = codeFilter;
    return Object.keys(next).length > 0 ? next : undefined;
  }, [status, codeFilter]);

  const page = useMemo(
    () => ({ number: pageNumber, size: PAGE_SIZE }),
    [pageNumber],
  );

  const couponsQuery = useCoupons(page, filters);
  const { isRefreshing, onRefresh } = useGridRefresh([couponsQuery]);
  const updateCoupon = useUpdateCoupon();
  const deleteCoupon = useDeleteCoupon();

  // The list hooks return PagedResult<T>, not a bare array: rows live under
  // `.data` and the pager is driven off `.meta` (null for an endpoint that
  // sent no envelope, in which case the one response is the whole collection).
  const coupons = couponsQuery.data?.data ?? [];
  const meta = couponsQuery.data?.meta ?? null;

  const goToPage = (nextPage: number) => {
    const next = new URLSearchParams(searchParams);
    if (nextPage > 1) next.set("page", String(nextPage));
    else next.delete("page");
    setSearchParams(next, { replace: false });
  };

  const resetToFirstPage = () => {
    const next = new URLSearchParams(searchParams);
    next.delete("page");
    setSearchParams(next, { replace: true });
  };

  const handleStatusFilter = (next: StatusFilter) => {
    setStatus(next);
    resetToFirstPage();
  };

  /**
   * PATCH semantics are genuinely partial — an omitted key preserves the
   * stored value. This sends `{ active }` alone: including any other field
   * (description, window, maxUses) would overwrite it with whatever this row
   * happened to be holding.
   */
  const handleToggleActive = async (coupon: Coupon, next: boolean) => {
    setPendingToggleId(coupon.id);
    try {
      await updateCoupon.mutateAsync({
        id: coupon.id,
        patch: { active: next },
      });
      toast.success(
        `${coupon.attributes.code} is now ${next ? "active" : "inactive"}`,
      );
    } catch (e) {
      toast.error(createErrorFromUnknown(e).message);
    } finally {
      setPendingToggleId(null);
    }
  };

  /**
   * Delete is already disabled once `redemptionCount > 0`, but a redemption
   * can land between the render and the click; the server answers 409 and the
   * service turns that into a CouponConflictError, which is reported here
   * rather than surfacing as a generic failure.
   */
  const handleConfirmDelete = async () => {
    if (!couponToDelete) return;
    const { id, attributes } = couponToDelete;
    setCouponToDelete(null);
    setConflict(null);
    try {
      await deleteCoupon.mutateAsync({ id });
      toast.success(`Coupon ${attributes.code} deleted`);
    } catch (e) {
      if (e instanceof CouponConflictError) {
        setConflict(
          `${attributes.code} was redeemed while you were deleting it, so it can no longer be removed.`,
        );
        return;
      }
      toast.error(createErrorFromUnknown(e).message);
    }
  };

  const columns = getColumns({
    onToggleActive: (coupon, next) => void handleToggleActive(coupon, next),
    onDelete: setCouponToDelete,
    pendingToggleId,
  });

  if (couponsQuery.isLoading) {
    return <CouponPageSkeleton />;
  }

  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
      <div className="flex items-center justify-between space-y-2">
        <h2 className="text-2xl font-bold tracking-tight">Coupons</h2>
        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={() => setBatchOpen(true)}>
            Generate Batch
          </Button>
          <Button onClick={() => setCreateOpen(true)}>New Coupon</Button>
        </div>
      </div>

      {conflict && (
        <Alert variant="destructive">
          <AlertTitle>Delete rejected</AlertTitle>
          <AlertDescription>{conflict}</AlertDescription>
        </Alert>
      )}

      <div className="flex flex-wrap items-center gap-2">
        {STATUS_FILTERS.map(([value, label]) => (
          <Button
            key={value}
            size="sm"
            variant={status === value ? "default" : "outline"}
            aria-pressed={status === value}
            onClick={() => handleStatusFilter(value)}
          >
            {label}
          </Button>
        ))}
        <form
          className="flex items-center gap-2"
          onSubmit={(e) => {
            e.preventDefault();
            setCodeFilter(codeDraft.trim());
            resetToFirstPage();
          }}
        >
          <Input
            aria-label="Search by code"
            placeholder="Search by code"
            className="h-9 w-56"
            value={codeDraft}
            onChange={(e) => setCodeDraft(e.target.value)}
          />
          <Button type="submit" size="sm" variant="outline">
            Search
          </Button>
        </form>
      </div>

      <div>
        <DataTableWrapper
          columns={columns}
          data={coupons}
          error={couponsQuery.error?.message ?? null}
          onRefresh={() => void onRefresh()}
          isRefreshing={isRefreshing}
          initialVisibilityState={hiddenColumns}
          emptyState={{
            title: "No coupons found",
            description:
              "Create a coupon, or generate a batch of codes to hand out.",
          }}
        />
        {meta && coupons.length > 0 && (
          <Pager
            page={meta.page.number}
            lastPage={meta.page.last}
            total={meta.total}
            pageSize={meta.page.size}
            onPageChange={goToPage}
          />
        )}
      </div>

      <CreateCouponDialog open={createOpen} onOpenChange={setCreateOpen} />
      <GenerateCouponBatchDialog open={batchOpen} onOpenChange={setBatchOpen} />

      <AlertDialog
        open={!!couponToDelete}
        onOpenChange={(open) => !open && setCouponToDelete(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete this coupon?</AlertDialogTitle>
            <AlertDialogDescription>
              {couponToDelete?.attributes.code} will be removed permanently.
              This cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={() => void handleConfirmDelete()}>
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
