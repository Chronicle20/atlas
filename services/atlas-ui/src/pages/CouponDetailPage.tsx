/**
 * One coupon code, with its redemption history.
 *
 * History is fetched per-code (`GET /coupons/{id}/redemptions`). There is no
 * global redemption list to link to — the only other audit view is per-account
 * — so this page is where a code's trail lives.
 */

import { useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { ErrorDisplay } from "@/components/common/ErrorDisplay";
import { Pager } from "@/components/common/Pager";
import { CouponPageSkeleton } from "@/components/common/skeletons/CouponPageSkeleton";
import { TableSkeleton } from "@/components/common/TableSkeleton";
import { useCoupon, useCouponRedemptions } from "@/lib/hooks/api/useCoupons";
import {
  couponStatus,
  formatRewards,
  formatTimestamp,
  formatUses,
  formatWindow,
} from "@/lib/utils/coupons";

const PAGE_SIZE = 50;

export function CouponDetailPage() {
  const { couponId } = useParams();
  const id = couponId ?? "";
  const [pageNumber, setPageNumber] = useState(1);

  const couponQuery = useCoupon(id);
  const redemptionPage = useMemo(
    () => ({ number: pageNumber, size: PAGE_SIZE }),
    [pageNumber],
  );
  const redemptionsQuery = useCouponRedemptions(
    useMemo(() => ({ couponId: id }), [id]),
    redemptionPage,
  );

  if (couponQuery.isLoading) return <CouponPageSkeleton />;

  const coupon = couponQuery.data;
  if (couponQuery.error || !coupon) {
    return (
      <div className="p-10">
        <ErrorDisplay
          error={couponQuery.error ?? "Coupon not found"}
          retry={() => void couponQuery.refetch()}
        />
      </div>
    );
  }

  const attrs = coupon.attributes;
  const status = couponStatus(attrs);
  // PagedResult again: rows under `.data`, pager off `.meta`.
  const redemptions = redemptionsQuery.data?.data ?? [];
  const meta = redemptionsQuery.data?.meta ?? null;

  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h2 className="font-mono text-2xl font-bold tracking-tight">
            {attrs.code}
          </h2>
          <Badge variant={status === "Active" ? "secondary" : "outline"}>
            {status}
          </Badge>
        </div>
        <Button variant="outline" asChild>
          <Link to="/coupons">Back to coupons</Link>
        </Button>
      </div>

      {attrs.description && (
        <p className="text-sm text-muted-foreground">{attrs.description}</p>
      )}

      <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium">Uses</CardTitle>
          </CardHeader>
          <CardContent className="text-lg tabular-nums">
            {formatUses(attrs)}
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium">Window</CardTitle>
          </CardHeader>
          <CardContent className="text-sm">{formatWindow(attrs)}</CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium">Rewards</CardTitle>
          </CardHeader>
          <CardContent className="text-sm">
            {formatRewards(attrs.rewards)}
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-medium">
            Redemption History
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {redemptionsQuery.isLoading ? (
            <TableSkeleton rows={5} columns={5} showHeader />
          ) : redemptionsQuery.error ? (
            <ErrorDisplay
              error={redemptionsQuery.error}
              retry={() => void redemptionsQuery.refetch()}
            />
          ) : redemptions.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              This code has not been redeemed yet.
            </p>
          ) : (
            <>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Redeemed At</TableHead>
                    <TableHead>Account</TableHead>
                    <TableHead>Character</TableHead>
                    <TableHead>Rewards Granted</TableHead>
                    <TableHead>Transaction</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {redemptions.map((redemption) => (
                    <TableRow key={redemption.id}>
                      <TableCell>
                        {formatTimestamp(redemption.attributes.redeemedAt)}
                      </TableCell>
                      <TableCell>
                        <Link
                          to={`/accounts/${redemption.attributes.accountId}`}
                          className="text-primary hover:underline"
                        >
                          {redemption.attributes.accountId}
                        </Link>
                      </TableCell>
                      <TableCell>
                        <Link
                          to={`/characters/${redemption.attributes.characterId}`}
                          className="text-primary hover:underline"
                        >
                          {redemption.attributes.characterId}
                        </Link>
                      </TableCell>
                      <TableCell>
                        {formatRewards(redemption.attributes.rewardsGranted)}
                      </TableCell>
                      <TableCell className="font-mono text-xs">
                        {redemption.attributes.transactionId}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
              {meta && (
                <Pager
                  page={meta.page.number}
                  lastPage={meta.page.last}
                  total={meta.total}
                  pageSize={meta.page.size}
                  onPageChange={setPageNumber}
                />
              )}
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
