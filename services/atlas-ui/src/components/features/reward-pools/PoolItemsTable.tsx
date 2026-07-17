import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { ItemNameCell } from "@/components/item-name-cell";
import { useTenant } from "@/context/tenant-context";
import { gachaponChances, incubatorChances, tierHasMixedWeights } from "@/lib/utils/reward-pool-chance";
import type { RewardPoolItemData } from "@/types/models/reward-pool-item";
import type { GlobalRewardItemData } from "@/types/models/global-reward-item";

const TIERS = ["common", "uncommon", "rare"] as const;

function pct(v: number): string {
  return `${(v * 100).toFixed(2)}%`;
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

export function PoolItemsTable({ kind, tierWeights, items, globalItems, onEdit, onDelete }: PoolItemsTableProps) {
  const { activeTenant } = useTenant();

  if (kind === "incubator") {
    const chances = incubatorChances(items.map((i) => ({ id: i.id, weight: i.attributes.weight })));
    return (
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Item</TableHead>
            <TableHead>Quantity</TableHead>
            <TableHead>Weight</TableHead>
            <TableHead>Chance</TableHead>
            <TableHead className="w-24" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {items.map((it) => (
            <TableRow key={it.id}>
              <TableCell><ItemNameCell itemId={String(it.attributes.itemId)} tenant={activeTenant} /></TableCell>
              <TableCell>{it.attributes.quantity}</TableCell>
              <TableCell>{it.attributes.weight}</TableCell>
              <TableCell>{pct(chances.get(it.id) ?? 0)}</TableCell>
              <TableCell className="space-x-2 text-right">
                <Button variant="ghost" size="sm" onClick={() => onEdit(it)}>Edit</Button>
                <Button variant="ghost" size="sm" onClick={() => onDelete(it)}>Delete</Button>
              </TableCell>
            </TableRow>
          ))}
          {items.length === 0 && (
            <TableRow><TableCell colSpan={5} className="text-muted-foreground">No items in this pool.</TableCell></TableRow>
          )}
        </TableBody>
      </Table>
    );
  }

  // Gachapon: merged rows (machine + global) grouped by tier, chances via the
  // exact selectTier×selectItem model. Global rows are read-only here.
  const machineRows = items.map((it) => ({
    key: `m-${it.id}`,
    tier: it.attributes.tier as (typeof TIERS)[number],
    weight: it.attributes.weight,
    itemId: it.attributes.itemId,
    quantity: it.attributes.quantity,
    source: "machine" as const,
    item: it as RewardPoolItemData | undefined,
  }));
  const globalRows = globalItems.map((gi) => ({
    key: `g-${gi.id}`,
    tier: gi.attributes.tier as (typeof TIERS)[number],
    weight: 0, // global items always roll with weight 0 (reward/processor.go getMergedPool)
    itemId: gi.attributes.itemId,
    quantity: gi.attributes.quantity,
    source: "global" as const,
    item: undefined as RewardPoolItemData | undefined,
  }));
  const rows = [...machineRows, ...globalRows];
  const chances = gachaponChances(tierWeights, rows.map(({ key, tier, weight }) => ({ key, tier, weight })));

  return (
    <div className="space-y-6">
      {TIERS.map((tier) => {
        const tierRows = rows.filter((r) => r.tier === tier);
        if (tierRows.length === 0) return null;
        const mixed = tierHasMixedWeights(rows, tier);
        return (
          <div key={tier} className="space-y-2">
            <div className="flex items-center gap-2">
              <Badge variant={tier === "rare" ? "destructive" : tier === "uncommon" ? "secondary" : "outline"}>{tier}</Badge>
              <span className="text-sm text-muted-foreground">{tierRows.length} item{tierRows.length === 1 ? "" : "s"}</span>
            </div>
            {mixed && (
              <Alert variant="destructive">
                <AlertDescription>
                  This tier mixes weighted and unweighted items — the weighted roll excludes every zero-weight item
                  (including all global items) from this tier.
                </AlertDescription>
              </Alert>
            )}
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Item</TableHead>
                  <TableHead>Quantity</TableHead>
                  <TableHead>Weight</TableHead>
                  <TableHead>Chance</TableHead>
                  <TableHead className="w-24" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {tierRows.map((r) => {
                  const c = chances.get(r.key);
                  return (
                    <TableRow key={r.key} className={c?.excluded ? "opacity-60" : undefined}>
                      <TableCell className="space-x-2">
                        <ItemNameCell itemId={String(r.itemId)} tenant={activeTenant} />
                        {r.source === "global" && <Badge variant="outline">Global</Badge>}
                      </TableCell>
                      <TableCell>{r.quantity}</TableCell>
                      <TableCell>{r.weight > 0 ? r.weight : "—"}</TableCell>
                      <TableCell>
                        <TooltipProvider>
                          <Tooltip>
                            <TooltipTrigger asChild><span>{pct(c?.chance ?? 0)}</span></TooltipTrigger>
                            <TooltipContent>
                              <p>tier chance × within-tier share (mirrors the server roll)</p>
                            </TooltipContent>
                          </Tooltip>
                        </TooltipProvider>
                      </TableCell>
                      <TableCell className="space-x-2 text-right">
                        {r.source === "machine" && r.item ? (
                          <>
                            <Button variant="ghost" size="sm" onClick={() => onEdit(r.item as RewardPoolItemData)}>Edit</Button>
                            <Button variant="ghost" size="sm" onClick={() => onDelete(r.item as RewardPoolItemData)}>Delete</Button>
                          </>
                        ) : (
                          <span className="text-xs text-muted-foreground">Managed centrally</span>
                        )}
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </div>
        );
      })}
      {rows.length === 0 && <p className="text-sm text-muted-foreground">No items in this pool.</p>}
    </div>
  );
}
