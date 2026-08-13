/**
 * Picks a cash-shop COMMODITY — for a CASH_ITEM coupon reward, and for a
 * cash-surprise reward-pool entry.
 *
 * Both name their reward by commodity serial number, but an operator thinks in
 * item names — so this searches items by name (or id) and translates the chosen
 * item to one of its cash-shop commodities, whose id IS the serial number
 * (atlas-data commodity/reader.go reads it from the `SN` node).
 *
 * onChange hands back the WHOLE commodity row, not just its id: a cash-surprise
 * entry stores the granted item id and count alongside the serial, and those
 * must be the commodity's own (atlas-cashshop surprise/processor.go grants
 * `ci.ItemId()` × `ci.Count()`, ignoring whatever the entry claims). Callers
 * that only need the serial take `.id`.
 *
 * Only items that ACTUALLY HAVE a commodity are offered. Most items don't: a
 * sword or a red potion has no serial, so picking one used to leave the field
 * blank and fail validation at submit. The filter is membership in the
 * commodity catalog, NOT the "cash" compartment — the live catalog is mostly
 * equipment ids (e.g. serial 20000036 sells item 1002077, a hat), so filtering
 * by compartment would hide most of the cash shop.
 *
 * Where an item has exactly one commodity, clicking the item selects it. Only
 * a genuinely ambiguous item — several serials differing in bundle count,
 * price or rental period — asks the operator for a second click.
 *
 * The value stays a string — that is how a reward row holds it
 * (`rewardRowSchema` is what turns it into a number); numeric callers pass
 * `String(serial)`.
 */

import { useMemo, useState } from "react";
import { Pencil } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { useCommodityCatalog } from "@/lib/hooks/api/useItemCommodities";
import { useItemName } from "@/lib/hooks/api/useItemStrings";
import type { ItemCashShopCommodity } from "@/types/models/npc";
import { ItemSearchResults } from "./ItemSearchResults";
import { useItemSearch } from "./useItemSearch";

export interface CashItemPickerProps {
  /** The serial number as the form holds it — a string, possibly blank. */
  value: string;
  onChange: (commodity: ItemCashShopCommodity) => void;
  /** Ties the trigger to the caller's <Label>. */
  id: string;
  /** Trigger label rendered when nothing is chosen. */
  placeholder?: string;
}

/** "×2 · 3700 NX · 90 days" — the facts that distinguish one serial from another. */
function describeCommodity(commodity: ItemCashShopCommodity): string {
  const parts = [`×${commodity.count}`, `${commodity.price} NX`];
  if (commodity.period > 0) parts.push(`${commodity.period} days`);
  if (!commodity.onSale) parts.push("not on sale");
  return parts.join(" · ");
}

export function CashItemPicker({
  value,
  onChange,
  id,
  placeholder = "Choose a cash item…",
}: CashItemPickerProps) {
  const [open, setOpen] = useState(false);
  // Set only when the chosen item has SEVERAL serials and one must be picked.
  const [ambiguousItemId, setAmbiguousItemId] = useState<number | null>(null);

  const catalogQuery = useCommodityCatalog(open);
  const catalog = catalogQuery.data;

  const search = useItemSearch({
    poolKey: "items",
    open: open && ambiguousItemId === null,
  });

  // Only items the cash shop actually sells can be picked.
  const sellableRows = useMemo(
    () => search.rows.filter((row) => catalog?.byItemId.has(Number(row.id))),
    [search.rows, catalog],
  );
  const hiddenCount = search.rows.length - sellableRows.length;

  const ambiguous = ambiguousItemId
    ? catalog?.byItemId.get(ambiguousItemId)
    : undefined;
  const ambiguousName = useItemName(
    ambiguousItemId ? String(ambiguousItemId) : "",
  );

  // The trigger names the item an already-chosen serial grants — the catalog
  // answers that in memory, so a stored serial needs no extra request.
  const selected = value ? catalog?.bySerial.get(value) : undefined;
  const selectedName = useItemName(selected ? String(selected.itemId) : "");

  const close = () => {
    setOpen(false);
    setAmbiguousItemId(null);
    search.reset();
  };

  const pick = (commodity: ItemCashShopCommodity) => {
    onChange(commodity);
    close();
  };

  /** One serial → done in a click; several → ask which. */
  const pickItem = (itemId: number) => {
    const commodities = catalog?.byItemId.get(itemId) ?? [];
    const only = commodities.length === 1 ? commodities[0] : undefined;
    if (only) {
      pick(only);
      return;
    }
    setAmbiguousItemId(itemId);
  };

  const triggerLabel = () => {
    if (!value) return placeholder;
    if (selectedName.data) return `${selectedName.data} (${value})`;
    if (catalog && !selected) return `Serial ${value} — not in the cash shop`;
    return `Serial ${value}`;
  };

  // A bare number is taken as a serial when the catalog knows it — that is the
  // escape hatch for an operator who already has the SN in hand.
  const typedSerial =
    catalog && /^\d+$/.test(search.search.trim())
      ? catalog.bySerial.get(search.search.trim())
      : undefined;

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
        {catalogQuery.isLoading ? (
          <p className="px-1 py-2 text-sm text-muted-foreground">
            Loading the cash-shop catalog…
          </p>
        ) : catalogQuery.isError ? (
          <p className="px-1 py-2 text-sm text-destructive">
            Could not load the cash-shop catalog, so no serial number can be
            resolved.
          </p>
        ) : ambiguous && ambiguous.length > 0 ? (
          <>
            <div className="flex items-center justify-between gap-2 pb-2">
              <span className="truncate text-sm font-medium">
                {ambiguousName.data ?? `Item ${ambiguousItemId}`}
              </span>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => setAmbiguousItemId(null)}
              >
                Change item
              </Button>
            </div>
            <p className="px-1 pb-1 text-xs text-muted-foreground">
              This item is sold under {ambiguous.length} serial numbers — pick
              the one to grant.
            </p>
            <ul
              role="listbox"
              className="max-h-64 space-y-0.5 overflow-y-auto"
              aria-label="Serial numbers"
            >
              {ambiguous.map((commodity) => (
                <li
                  key={commodity.id}
                  role="option"
                  aria-selected={commodity.id === value}
                  tabIndex={0}
                  onClick={() => pick(commodity)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" || e.key === " ") {
                      e.preventDefault();
                      pick(commodity);
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
          </>
        ) : (
          <>
            <Input
              autoFocus
              aria-label="Search cash items"
              value={search.search}
              onChange={(e) => search.setSearch(e.target.value)}
              placeholder="Search by name, item id or serial…"
            />
            {typedSerial && (
              <button
                type="button"
                onClick={() => pick(typedSerial)}
                className="mt-2 w-full cursor-pointer rounded px-2 py-1 text-left text-sm hover:bg-accent focus-visible:bg-accent"
              >
                Use serial {typedSerial.id} ({describeCommodity(typedSerial)})
              </button>
            )}
            <ItemSearchResults
              rows={sellableRows}
              manualId={undefined}
              isLoading={search.isLoading}
              isError={search.isError}
              settledTerm={search.settledTerm}
              onPick={pickItem}
            />
            {hiddenCount > 0 && (
              <p className="px-2 py-1 text-xs text-muted-foreground">
                {hiddenCount} match{hiddenCount === 1 ? "" : "es"} hidden: the
                cash shop does not sell {hiddenCount === 1 ? "it" : "them"}, so{" "}
                {hiddenCount === 1 ? "it has" : "they have"} no serial number.
              </p>
            )}
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
          </>
        )}
      </PopoverContent>
    </Popover>
  );
}
