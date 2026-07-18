import { Badge } from "@/components/ui/badge";
import type { RewardPoolKind } from "@/types/models/reward-pool";

/**
 * Shared "Incubator" / "Gachapon" pool-kind badge. The amber utility classes
 * match the existing amber-badge convention used across the codebase — keep
 * them here rather than inventing a new semantic token.
 */
export function KindBadge({ kind }: { kind: RewardPoolKind }) {
  return kind === "incubator" ? (
    <Badge className="bg-amber-500/15 text-amber-600 dark:text-amber-400 border-transparent">
      Incubator
    </Badge>
  ) : (
    <Badge variant="secondary">Gachapon</Badge>
  );
}
