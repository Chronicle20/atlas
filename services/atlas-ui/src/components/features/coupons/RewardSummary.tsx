/**
 * Renders a coupon's rewards as text, with cash items named rather than left
 * as bare serial numbers.
 *
 * One component for all three sites — the coupon list's Rewards column, the
 * detail page's Rewards card, and the redemption history's Rewards Granted
 * column — so the three can never disagree about how a reward reads. Each
 * instance resolves only the serials it shows, and React Query dedupes those
 * lookups across every row on the page.
 *
 * While a name is loading (or if the serial resolves to nothing, e.g. a reward
 * pointing at a commodity a later WZ ingest dropped) the serial is shown, so a
 * row is never blank or misleading.
 */

import { useCashItemNames } from "@/lib/hooks/api/useItemCommodities";
import { cashItemSerials, formatRewards } from "@/lib/utils/coupons";
import type { CouponReward } from "@/services/api/coupons.service";

export interface RewardSummaryProps {
  rewards: CouponReward[];
  className?: string;
}

export function RewardSummary({ rewards, className }: RewardSummaryProps) {
  const names = useCashItemNames(cashItemSerials(rewards));
  return (
    <span className={className}>
      {formatRewards(rewards, (serialNumber) => names[serialNumber])}
    </span>
  );
}
