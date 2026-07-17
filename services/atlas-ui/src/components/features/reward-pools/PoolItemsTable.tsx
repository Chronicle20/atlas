import { useState } from "react";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription } from "@/components/ui/alert";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ItemNameCell } from "@/components/item-name-cell";
import { PoolItemActions } from "@/components/features/reward-pools/PoolItemActions";
import { useTenant } from "@/context/tenant-context";
import { getAssetIconUrl } from "@/lib/utils/asset-url";
import {
  gachaponChances,
  incubatorChances,
  tierHasMixedWeights,
} from "@/lib/utils/reward-pool-chance";
import type { RewardPoolItemData } from "@/types/models/reward-pool-item";
import type { GlobalRewardItemData } from "@/types/models/global-reward-item";

const TIERS = ["common", "uncommon", "rare"] as const;
type Tier = (typeof TIERS)[number];

function pct(v: number): string {
  return `${(v * 100).toFixed(2)}%`;
}

function tierBadgeVariant(
  tier: string,
): "destructive" | "secondary" | "outline" {
  if (tier === "rare") return "destructive";
  if (tier === "uncommon") return "secondary";
  return "outline";
}

interface PoolItemsTableProps {
  kind: "gachapon" | "incubator";
  poolId: string;
  tierWeights: { common: number; uncommon: number; rare: number };
  items: RewardPoolItemData[];
  globalItems: GlobalRewardItemData[];
  onEdit: (item: RewardPoolItemData) => void;
  onDelete: (item: RewardPoolItemData) => void;
}

export function PoolItemsTable({
  kind,
  tierWeights,
  items,
  globalItems,
  onEdit,
  onDelete,
}: PoolItemsTableProps) {
  const { activeTenant } = useTenant();
  const [tierFilter, setTierFilter] = useState<"all" | Tier>("all");

  // Plain helper (not a component) so re-renders don't remount the icon/name pair.
  function renderItem(itemId: number) {
    const iconUrl = activeTenant
      ? getAssetIconUrl(
          activeTenant.id,
          activeTenant.attributes.region,
          activeTenant.attributes.majorVersion,
          activeTenant.attributes.minorVersion,
          "item",
          itemId,
        )
      : null;
    return (
      <span className="inline-flex items-center gap-2">
        {iconUrl && (
          <img src={iconUrl} alt="" width={24} height={24} loading="lazy" />
        )}
        <ItemNameCell itemId={String(itemId)} tenant={activeTenant} />
      </span>
    );
  }

  if (kind === "incubator") {
    const chances = incubatorChances(
      items.map((i) => ({ id: i.id, weight: i.attributes.weight })),
    );
    return (
      <div className="flex flex-col min-h-0 flex-1">
        <div className="flex-1 min-h-0 overflow-auto rounded-md border">
          <Table>
            <TableHeader className="sticky top-0 bg-background z-10">
              <TableRow>
                <TableHead>Item</TableHead>
                <TableHead>Quantity</TableHead>
                <TableHead>Weight</TableHead>
                <TableHead>Chance</TableHead>
                <TableHead className="w-16" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((it) => (
                <TableRow key={it.id}>
                  <TableCell>{renderItem(it.attributes.itemId)}</TableCell>
                  <TableCell>{it.attributes.quantity}</TableCell>
                  <TableCell>
                    {it.attributes.weight > 0 ? it.attributes.weight : "—"}
                  </TableCell>
                  <TableCell>{pct(chances.get(it.id) ?? 0)}</TableCell>
                  <TableCell className="text-right">
                    <PoolItemActions
                      onEdit={() => onEdit(it)}
                      onDelete={() => onDelete(it)}
                    />
                  </TableCell>
                </TableRow>
              ))}
              {items.length === 0 && (
                <TableRow>
                  <TableCell colSpan={5} className="text-muted-foreground">
                    No items in this pool.
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>
      </div>
    );
  }

  // Gachapon: merged rows (machine + global), flat grid, chances via the exact
  // selectTier×selectItem model. Global rows are read-only here.
  const machineRows = items.map((it) => ({
    key: `m-${it.id}`,
    tier: it.attributes.tier as Tier,
    weight: it.attributes.weight,
    itemId: it.attributes.itemId,
    quantity: it.attributes.quantity,
    source: "machine" as const,
    item: it as RewardPoolItemData | undefined,
  }));
  const globalRows = globalItems.map((gi) => ({
    key: `g-${gi.id}`,
    tier: gi.attributes.tier as Tier,
    weight: 0, // global items always roll with weight 0 (reward/processor.go getMergedPool)
    itemId: gi.attributes.itemId,
    quantity: gi.attributes.quantity,
    source: "global" as const,
    item: undefined as RewardPoolItemData | undefined,
  }));
  const rows = [...machineRows, ...globalRows];
  // Chances are computed over the FULL pool so percentages stay correct
  // regardless of the tier filter applied below.
  const chances = gachaponChances(
    tierWeights,
    rows.map(({ key, tier, weight }) => ({ key, tier, weight })),
  );
  const visibleRows =
    tierFilter === "all" ? rows : rows.filter((r) => r.tier === tierFilter);
  const mixedTiers = TIERS.filter((tier) => tierHasMixedWeights(rows, tier));

  return (
    <div className="flex flex-col min-h-0 flex-1 gap-3">
      <div className="shrink-0 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-2">
          <Select
            value={tierFilter}
            onValueChange={(v) => setTierFilter(v as "all" | Tier)}
          >
            <SelectTrigger aria-label="Tier" className="w-[160px]">
              <SelectValue placeholder="All Tiers" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Tiers</SelectItem>
              {TIERS.map((tier) => (
                <SelectItem key={tier} value={tier}>
                  {tier}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        {mixedTiers.length > 0 && (
          <Alert variant="destructive" className="sm:max-w-md">
            <AlertDescription>
              {mixedTiers.join(", ")}{" "}
              {mixedTiers.length === 1 ? "tier mixes" : "tiers mix"} weighted
              and unweighted items — the weighted roll excludes every
              zero-weight item (including all global items) in{" "}
              {mixedTiers.length === 1 ? "that tier" : "those tiers"}.
            </AlertDescription>
          </Alert>
        )}
      </div>
      <div className="flex-1 min-h-0 overflow-auto rounded-md border">
        <Table>
          <TableHeader className="sticky top-0 bg-background z-10">
            <TableRow>
              <TableHead>Item</TableHead>
              <TableHead>Tier</TableHead>
              <TableHead>Quantity</TableHead>
              <TableHead>Weight</TableHead>
              <TableHead>Chance</TableHead>
              <TableHead className="w-16" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {visibleRows.map((r) => {
              const c = chances.get(r.key);
              return (
                <TableRow
                  key={r.key}
                  className={c?.excluded ? "opacity-60" : undefined}
                >
                  <TableCell>
                    <div className="flex items-center gap-2">
                      {renderItem(r.itemId)}
                      {r.source === "global" && (
                        <Badge variant="outline">Global</Badge>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant={tierBadgeVariant(r.tier)}>{r.tier}</Badge>
                  </TableCell>
                  <TableCell>{r.quantity}</TableCell>
                  <TableCell>{r.weight > 0 ? r.weight : "—"}</TableCell>
                  <TableCell>
                    <TooltipProvider>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <span>{pct(c?.chance ?? 0)}</span>
                        </TooltipTrigger>
                        <TooltipContent>
                          <p>
                            tier chance × within-tier share (mirrors the server
                            roll)
                          </p>
                        </TooltipContent>
                      </Tooltip>
                    </TooltipProvider>
                  </TableCell>
                  <TableCell className="text-right">
                    {r.source === "machine" && r.item && (
                      <PoolItemActions
                        onEdit={() => onEdit(r.item as RewardPoolItemData)}
                        onDelete={() => onDelete(r.item as RewardPoolItemData)}
                      />
                    )}
                  </TableCell>
                </TableRow>
              );
            })}
            {visibleRows.length === 0 && (
              <TableRow>
                <TableCell colSpan={6} className="text-muted-foreground">
                  No items in this pool.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
