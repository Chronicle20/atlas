import { Skeleton } from "@/components/ui/skeleton";
import { TableSkeleton } from "../TableSkeleton";

interface CouponPageSkeletonProps {
  animation?: "pulse" | "wave" | "none";
}

/** Loading state for the coupons list: header, filter bar, table. */
export function CouponPageSkeleton({
  animation = "pulse",
}: CouponPageSkeletonProps = {}) {
  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
      <div className="flex items-center justify-between">
        <Skeleton animation={animation} className="h-8 w-32" />
        <div className="flex gap-2">
          <Skeleton animation={animation} className="h-9 w-32" />
          <Skeleton animation={animation} className="h-9 w-28" />
        </div>
      </div>
      <div className="flex gap-2">
        <Skeleton animation={animation} className="h-9 w-20" />
        <Skeleton animation={animation} className="h-9 w-24" />
        <Skeleton animation={animation} className="h-9 w-24" />
      </div>
      <TableSkeleton
        rows={10}
        columns={6}
        showHeader={true}
        showActions={true}
        animation={animation}
      />
    </div>
  );
}
