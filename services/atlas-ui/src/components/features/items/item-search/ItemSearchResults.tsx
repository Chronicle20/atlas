import type { ReactNode } from "react";
import type { ItemSearchResult } from "@/types/models/item";
import { getAssetIconUrl } from "@/lib/utils/asset-url";
import { useTenant } from "@/context/tenant-context";

interface ItemSearchResultsProps {
  rows: ItemSearchResult[];
  manualId: number | undefined;
  isLoading: boolean;
  isError: boolean;
  settledTerm: string;
  onPick: (id: number) => void;
  /** Ids already in the caller's pool — rendered disabled with an "Added" tag. */
  disabledIds?: number[];
  /** Rendered as the first <li>. Used for ItemPicker's "None" option. */
  leadingRow?: ReactNode;
}

export function ItemSearchResults({
  rows,
  manualId,
  isLoading,
  isError,
  settledTerm,
  onPick,
  disabledIds = [],
  leadingRow,
}: ItemSearchResultsProps) {
  const { activeTenant } = useTenant();

  return (
    <ul role="listbox" className="mt-2 max-h-64 space-y-0.5 overflow-y-auto">
      {leadingRow}
      {rows.map((row) => {
        const id = Number(row.id);
        const inPool = disabledIds.includes(id);
        return (
          <li
            key={row.id}
            role="option"
            aria-selected={false}
            aria-disabled={inPool}
            tabIndex={inPool ? -1 : 0}
            onClick={() => !inPool && onPick(id)}
            onKeyDown={(e) => {
              if ((e.key === "Enter" || e.key === " ") && !inPool) {
                e.preventDefault();
                onPick(id);
              }
            }}
            className={
              inPool
                ? "flex cursor-not-allowed items-center gap-2 rounded px-2 py-1 opacity-50"
                : "flex cursor-pointer items-center gap-2 rounded px-2 py-1 hover:bg-accent focus-visible:bg-accent"
            }
          >
            {activeTenant && (
              <img
                src={getAssetIconUrl(
                  activeTenant.id,
                  activeTenant.attributes.region,
                  activeTenant.attributes.majorVersion,
                  activeTenant.attributes.minorVersion,
                  "item",
                  id,
                )}
                alt=""
                width={24}
                height={24}
                loading="lazy"
                className="[image-rendering:pixelated]"
                onError={(e) => {
                  (e.target as HTMLImageElement).style.visibility = "hidden";
                }}
              />
            )}
            <span className="flex-1 truncate text-sm">{row.name}</span>
            <span className="font-mono text-xs text-muted-foreground">
              {row.id}
            </span>
            {inPool && (
              <span className="text-xs text-muted-foreground">Added</span>
            )}
          </li>
        );
      })}
      {manualId !== undefined && (
        <li
          role="option"
          aria-selected={false}
          tabIndex={0}
          onClick={() => onPick(manualId)}
          onKeyDown={(e) => {
            if (e.key === "Enter" || e.key === " ") {
              e.preventDefault();
              onPick(manualId);
            }
          }}
          className="cursor-pointer rounded px-2 py-1 text-sm hover:bg-accent focus-visible:bg-accent"
        >
          Use id {manualId}
        </li>
      )}
      {isLoading && settledTerm && (
        <li className="px-2 py-1 text-sm text-muted-foreground">Searching…</li>
      )}
      {isError && settledTerm && (
        <li className="px-2 py-1 text-sm text-warning-foreground">
          Search failed — enter an id manually
        </li>
      )}
      {!isLoading &&
        !isError &&
        settledTerm &&
        rows.length === 0 &&
        manualId === undefined && (
          <li className="px-2 py-1 text-sm text-muted-foreground">
            No matches.
          </li>
        )}
    </ul>
  );
}
