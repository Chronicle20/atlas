import { Badge } from "@/components/ui/badge";
import type { RewardPoolKind } from "@/types/models/reward-pool";

/**
 * Pool-kind badge. A Record keyed by RewardPoolKind rather than a ternary:
 * the previous binary ternary had no default branch, so a new kind rendered
 * silently as "Gachapon". With the Record, adding a kind without a badge is
 * a type error.
 *
 * The amber utility classes match the existing amber-badge convention used
 * across the codebase — keep them here rather than inventing a new token.
 */
const BADGES: Record<RewardPoolKind, React.ReactElement> = {
  incubator: (
    <Badge className="bg-amber-500/15 text-amber-600 dark:text-amber-400 border-transparent">
      Incubator
    </Badge>
  ),
  "cash-surprise": (
    <Badge className="bg-violet-500/15 text-violet-600 dark:text-violet-400 border-transparent">
      Cash Surprise
    </Badge>
  ),
  gachapon: <Badge variant="secondary">Gachapon</Badge>,
};

export function KindBadge({ kind }: { kind: RewardPoolKind }) {
  return BADGES[kind];
}
