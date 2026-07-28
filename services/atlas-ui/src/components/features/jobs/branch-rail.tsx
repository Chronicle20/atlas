import type { CSSProperties } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import type { VisibleRailGroup } from "@/components/features/jobs/rail-groups";

interface BranchRailProps {
  groups: VisibleRailGroup[];
  selectedEntryId: number;
  onSelect: (id: number) => void;
  /** True while the tenant job set is still loading — renders a skeleton. */
  isPending?: boolean;
}

export function BranchRail({
  groups,
  selectedEntryId,
  onSelect,
  isPending = false,
}: BranchRailProps) {
  if (isPending) {
    return (
      <Card className="flex min-h-0 flex-col">
        <CardHeader className="pb-1">
          <CardTitle className="text-[15px]">Branches</CardTitle>
        </CardHeader>
        <CardContent
          data-testid="branch-rail-skeleton"
          className="min-h-0 flex-1 space-y-2 px-2 pb-3 pt-2"
        >
          {Array.from({ length: 8 }, (_, i) => (
            <Skeleton key={i} className="h-7 w-full" />
          ))}
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="flex min-h-0 flex-col">
      <CardHeader className="pb-1">
        <CardTitle className="text-[15px]">Branches</CardTitle>
      </CardHeader>
      <CardContent className="min-h-0 flex-1 overflow-y-auto px-2 pb-3">
        {groups.map((g) => (
          <div key={g.label}>
            <h3 className="mx-2 mb-1 mt-2.5 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
              {g.label}
            </h3>
            {g.entries.map((e) => (
              <button
                key={e.id}
                type="button"
                aria-pressed={selectedEntryId === e.id}
                onClick={() => onSelect(e.id)}
                style={{ "--acc": `var(${e.accent})` } as CSSProperties}
                className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-[13.5px] hover:bg-accent focus:outline-none focus-visible:ring-2 focus-visible:ring-ring aria-pressed:bg-[hsl(var(--acc)/0.14)] aria-pressed:font-medium"
              >
                <span
                  aria-hidden
                  className="h-2 w-2 flex-none rounded-[3px] bg-[hsl(var(--acc))]"
                />
                <span className="truncate">{e.name}</span>
                {` `}
                <span className="ml-auto text-[11.5px] tabular-nums text-muted-foreground">
                  {e.count}
                </span>
              </button>
            ))}
          </div>
        ))}
      </CardContent>
    </Card>
  );
}
