import { useMemo } from "react";
import type { CSSProperties } from "react";
import {
  advancementChains,
  jobNodeName,
  jobTreePath,
  tierLabel,
  type JobGraph,
} from "@/lib/jobs/job-graph";
import { cn } from "@/lib/utils";

interface AdvancementFlowProps {
  graph: JobGraph;
  entryId: number;
  selectedJobId: number;
  /** Branch accent token name, e.g. "--c-warrior". */
  accent: string;
  onSelect: (id: number) => void;
}

function FlowChip({
  graph,
  id,
  selected,
  onSelect,
}: {
  graph: JobGraph;
  id: number;
  selected: boolean;
  onSelect: (id: number) => void;
}) {
  const tier = tierLabel(graph, id);
  return (
    <button
      type="button"
      aria-pressed={selected}
      onClick={() => onSelect(id)}
      className={cn(
        "inline-flex items-center gap-1.5 whitespace-nowrap rounded-md border px-2.5 py-1 text-[13px] font-medium transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-ring",
        selected
          ? "border-[hsl(var(--acc))] bg-[hsl(var(--acc))] text-[hsl(var(--acc-fg))]"
          : "bg-card hover:border-[hsl(var(--acc))]",
      )}
    >
      {jobNodeName(graph, id)}
      {tier ? (
        <span
          className={cn(
            "rounded px-1 py-px text-[10px] font-semibold tracking-wide",
            selected
              ? "bg-[hsl(var(--acc-fg)/0.22)] text-[hsl(var(--acc-fg))]"
              : "bg-secondary text-muted-foreground",
          )}
        >
          {tier}
        </span>
      ) : null}
    </button>
  );
}

/**
 * Tier-aligned advancement grid (design D2, ported from the approved mock):
 * ancestors + the entry are "anchor" cells spanning every path row, vertically
 * centered; chain node k of path r lands at column anchors+1+k, row r+1, so
 * same-tier chips share an implicit auto column and align with zero
 * measurement code.
 */
export function AdvancementFlow({
  graph,
  entryId,
  selectedJobId,
  accent,
  onSelect,
}: AdvancementFlowProps) {
  const anchors = useMemo(
    () => jobTreePath(graph, entryId).map((n) => n.id),
    [graph, entryId],
  );
  const chains = useMemo(
    () => advancementChains(graph, entryId),
    [graph, entryId],
  );
  const rows = Math.max(chains.length, 1);
  const anchorCols = anchors.length;
  const sep = (
    <span aria-hidden className="mx-px flex-none text-muted-foreground/55">
      ›
    </span>
  );
  return (
    <div className="overflow-x-auto pb-0.5">
      <div
        className="mx-auto grid w-max gap-x-1 gap-y-1.5"
        style={{ "--acc": `var(${accent})` } as CSSProperties}
      >
        {anchors.map((id, i) => (
          <div
            key={`anchor-${id}`}
            data-testid={`flow-cell-${id}`}
            className="flex items-center gap-1 self-center whitespace-nowrap"
            style={{ gridColumn: `${i + 1}`, gridRow: `1 / span ${rows}` }}
          >
            {i > 0 ? sep : null}
            <FlowChip
              graph={graph}
              id={id}
              selected={selectedJobId === id}
              onSelect={onSelect}
            />
          </div>
        ))}
        {chains.map((chain, r) =>
          chain.map((id, k) => (
            <div
              key={`chain-${id}`}
              data-testid={`flow-cell-${id}`}
              className="flex items-center gap-1 whitespace-nowrap [&>button]:flex-1 [&>button]:justify-center"
              style={{
                gridColumn: `${anchorCols + 1 + k}`,
                gridRow: `${r + 1}`,
              }}
            >
              {sep}
              <FlowChip
                graph={graph}
                id={id}
                selected={selectedJobId === id}
                onSelect={onSelect}
              />
            </div>
          )),
        )}
      </div>
    </div>
  );
}
