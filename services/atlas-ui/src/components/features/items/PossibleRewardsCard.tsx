import { Link } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { useItemData } from "@/lib/hooks/useItemData";
import type { RewardModel } from "@/types/models/item";

interface PossibleRewardsCardProps {
  rewards: RewardModel[];
}

interface RewardRow extends RewardModel {
  chance: number; // 0..1, computed from prob / Σprob
}

export function PossibleRewardsCard({ rewards }: PossibleRewardsCardProps) {
  if (rewards.length === 0) return null;

  const total = rewards.reduce((sum, r) => sum + r.prob, 0);
  const rows: RewardRow[] = rewards
    .map((r) => ({ ...r, chance: total > 0 ? r.prob / total : 0 }))
    .sort((a, b) => b.chance - a.chance);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm font-medium">
          Possible Rewards ({rewards.length})
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid gap-2 grid-cols-[repeat(auto-fill,minmax(min(400px,100%),1fr))]">
          {rows.map((row, idx) => (
            <RewardRowWidget key={`${row.itemId}-${idx}`} reward={row} />
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

function RewardRowWidget({ reward }: { reward: RewardRow }) {
  const { name, iconUrl, isLoading } = useItemData(reward.itemId);
  // 3 decimals: the rarest canonical reward is ~0.005% (1 / 19,864); 2 decimals
  // would round it to 0.01%, overstating it 2×. 3 decimals renders it faithfully
  // and nothing rounds down to a false 0.000%.
  const pct = (reward.chance * 100).toFixed(3);
  const displayName =
    isLoading && !name
      ? `Item #${reward.itemId}`
      : name || `Item #${reward.itemId}`;

  return (
    <Link
      to={`/items/${reward.itemId}`}
      className="flex items-center gap-3 rounded-md border bg-card p-2 hover:bg-accent transition-colors"
    >
      <div className="h-8 w-8 shrink-0 flex items-center justify-center">
        {iconUrl && (
          <img
            src={iconUrl}
            alt={name || String(reward.itemId)}
            width={32}
            height={32}
            loading="lazy"
            className="max-h-full max-w-full object-contain"
          />
        )}
      </div>
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium truncate">
          {displayName}
          {reward.count > 1 && (
            <span className="ml-1 text-muted-foreground">×{reward.count}</span>
          )}
        </p>
      </div>
      <div className="flex items-center gap-2 shrink-0">
        {reward.period > 0 && <Badge variant="secondary">time-limited</Badge>}
        {reward.worldMsg !== "" && <Badge variant="secondary">announces</Badge>}
        <p className="text-sm font-medium tabular-nums">{pct}%</p>
      </div>
    </Link>
  );
}
