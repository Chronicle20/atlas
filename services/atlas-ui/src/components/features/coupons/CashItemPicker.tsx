/**
 * Picks the SERIAL NUMBER a CASH_ITEM coupon reward grants.
 *
 * A coupon names its cash-item reward by commodity serial number, but an
 * operator thinks in item names — so this searches items by name (or id, the
 * same combobox the character-preset editor uses) and then translates the
 * chosen item to one of its cash-shop commodities, whose id IS the serial
 * number (atlas-data commodity/reader.go reads it from the `SN` node).
 *
 * The translation is a genuine one-to-many: an item can be sold under several
 * commodities that differ in bundle count, price and rental period, and they
 * grant different things. So the item pick narrows the list and the operator
 * still picks the row. A serial can also be typed straight in, for the case
 * where it is already known (or the item is not searchable by name).
 *
 * The field's value stays a string, like every other reward-row field —
 * `rewardRowSchema` is what turns it into a number.
 */

import { useState } from "react";
import { Pencil } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { ItemSearchResults } from "@/components/features/items/item-search/ItemSearchResults";
import { useItemSearch } from "@/components/features/items/item-search/useItemSearch";
import {
  useCommodityBySerial,
  useItemCommodities,
} from "@/lib/hooks/api/useItemCommodities";
import { useItemName } from "@/lib/hooks/api/useItemStrings";
import type { ItemCashShopCommodity } from "@/types/models/npc";

export interface CashItemPickerProps {
  /** The serial number as the form holds it — a string, possibly blank. */
  value: string;
  onChange: (serialNumber: string) => void;
  /** Ties the trigger to the caller's <Label>. */
  id: string;
}

/** "×2 · 3,000 NX · 30 days" — the facts that distinguish one SN from another. */
function describeCommodity(commodity: ItemCashShopCommodity): string {
  const parts = [`×${commodity.count}`, `${commodity.price} NX`];
  if (commodity.period > 0) parts.push(`${commodity.period} days`);
  if (!commodity.onSale) parts.push("not on sale");
  return parts.join(" · ");
}

export function CashItemPicker({ value, onChange, id }: CashItemPickerProps) {
  const [open, setOpen] = useState(false);
  // The item whose commodities are being listed. Null = still searching.
  const [itemId, setItemId] = useState<number | null>(null);
  const [manualSerial, setManualSerial] = useState("");

  const search = useItemSearch({ poolKey: "items", open: open && !itemId });
  const commodities = useItemCommodities(itemId ? String(itemId) : "");
  const pickedItemName = useItemName(itemId ? String(itemId) : "");

  // What the trigger shows for an already-chosen serial: the item it grants.
  const selected = useCommodityBySerial(value);
  const selectedName = useItemName(
    selected.data ? String(selected.data.itemId) : "",
  );

  const close = () => {
    setOpen(false);
    setItemId(null);
    setManualSerial("");
    search.reset();
  };

  const pick = (serialNumber: number | string) => {
    onChange(String(serialNumber));
    close();
  };

  const triggerLabel = () => {
    if (!value) return "Choose a cash item…";
    if (selectedName.data) return `${selectedName.data} (${value})`;
    if (selected.isError) return `Serial ${value} — no such commodity`;
    return `Serial ${value}`;
  };

  return (
    <Popover
      open={open}
      onOpenChange={(next) => (next ? setOpen(true) : close())}
    >
      <PopoverTrigger asChild>
        <Button
          type="button"
          id={id}
          variant="outline"
          className="w-full justify-between font-normal"
        >
          <span className="truncate">{triggerLabel()}</span>
          <Pencil className="ml-2 size-3.5 shrink-0 opacity-60" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-96 p-2" align="start">
        {itemId === null ? (
          <>
            <Input
              autoFocus
              aria-label="Search cash items"
              value={search.search}
              onChange={(e) => search.setSearch(e.target.value)}
              placeholder="Search by name or enter an item id…"
            />
            <ItemSearchResults
              rows={search.rows}
              manualId={search.manualId}
              isLoading={search.isLoading}
              isError={search.isError}
              settledTerm={search.settledTerm}
              onPick={(picked) => setItemId(picked)}
            />
            {search.hasMore && (
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="mt-1 w-full"
                onClick={search.loadMore}
              >
                Load more
              </Button>
            )}
            <div className="mt-2 space-y-1 border-t pt-2">
              <Label htmlFor={`${id}-manual`} className="text-xs">
                Or enter a serial number directly
              </Label>
              <div className="flex gap-2">
                <Input
                  id={`${id}-manual`}
                  type="number"
                  className="h-8"
                  value={manualSerial}
                  onChange={(e) => setManualSerial(e.target.value)}
                />
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  disabled={!manualSerial.trim()}
                  onClick={() => pick(manualSerial.trim())}
                >
                  Use
                </Button>
              </div>
            </div>
          </>
        ) : (
          <>
            <div className="flex items-center justify-between gap-2 pb-2">
              <span className="truncate text-sm font-medium">
                {pickedItemName.data ?? `Item ${itemId}`}
              </span>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => setItemId(null)}
              >
                Change item
              </Button>
            </div>
            {commodities.isLoading ? (
              <p className="px-1 py-2 text-sm text-muted-foreground">
                Loading cash-shop entries…
              </p>
            ) : commodities.isError ? (
              <p className="px-1 py-2 text-sm text-destructive">
                Could not load cash-shop entries for this item.
              </p>
            ) : (commodities.data?.length ?? 0) === 0 ? (
              <p className="px-1 py-2 text-sm text-muted-foreground">
                This item has no cash-shop entry, so it has no serial number to
                grant.
              </p>
            ) : (
              <ul
                role="listbox"
                className="max-h-64 space-y-0.5 overflow-y-auto"
              >
                {commodities.data?.map((commodity) => (
                  <li
                    key={commodity.id}
                    role="option"
                    aria-selected={commodity.id === value}
                    tabIndex={0}
                    onClick={() => pick(commodity.id)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter" || e.key === " ") {
                        e.preventDefault();
                        pick(commodity.id);
                      }
                    }}
                    className="flex cursor-pointer items-center justify-between gap-2 rounded px-2 py-1 hover:bg-accent focus-visible:bg-accent"
                  >
                    <span className="font-mono text-xs">{commodity.id}</span>
                    <span className="text-xs text-muted-foreground">
                      {describeCommodity(commodity)}
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </>
        )}
      </PopoverContent>
    </Popover>
  );
}
