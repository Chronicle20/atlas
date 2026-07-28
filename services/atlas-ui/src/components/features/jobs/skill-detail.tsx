import { useMemo, useState } from "react";
import type { CSSProperties } from "react";
import { ChevronRight } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
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
import type { SkillDefinitionWithIcon } from "@/lib/hooks/api/useSkillDefinition";
import { buildLevelTable } from "@/lib/skills/level-table";
import { deriveSkillType } from "@/lib/skills/skill-type";
import { resolveSkillName } from "@/lib/skills/beginner-skill-names";
import { formatSkillDescription } from "@/lib/skills/format-skill-description";
import { SkillIcon } from "@/components/features/jobs/skill-icon";
import { cn } from "@/lib/utils";

interface SkillDetailProps {
  def: SkillDefinitionWithIcon;
  /** Branch accent token name, e.g. "--c-warrior". */
  accent: string;
}

// Hosts render this with key={def.id} so the slider level resets per skill
// (design D3) — no effect juggling.
export function SkillDetail({ def, accent }: SkillDetailProps) {
  const [level, setLevel] = useState(1);
  const name = resolveSkillName(def.id, def.name);
  const type = deriveSkillType(def);
  const formatted = formatSkillDescription(def.description);
  const table = useMemo(() => buildLevelTable(def.effects), [def.effects]);
  const maxLevel = def.maxLevel ?? 0;
  const hasLevels = maxLevel > 1 && table.rows.length > 0;
  const row = table.rows[level - 1];
  const statColumns = table.columns.filter((c) => c.key !== "level");

  return (
    <div
      className="space-y-3 px-4 pb-5 pt-1"
      style={{ "--acc": `var(${accent})` } as CSSProperties}
    >
      <div className="flex items-start gap-3">
        <SkillIcon def={def} name={name} />
        <div>
          <h5 className="text-base font-semibold">{name}</h5>
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <span
                  tabIndex={0}
                  className="cursor-help rounded font-mono text-[11.5px] text-muted-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                >
                  ID {def.id}
                </span>
              </TooltipTrigger>
              <TooltipContent copyable>
                <p>{def.id}</p>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </div>
      </div>

      <div className="flex flex-wrap gap-1.5">
        <Badge variant="secondary">{type}</Badge>
        <Badge variant="secondary">Master Lv {def.maxLevel ?? "—"}</Badge>
        {def.element ? (
          <Badge variant="outline">Element {def.element}</Badge>
        ) : null}
      </div>

      {formatted.lines.length === 0 ? (
        <p className="text-sm text-muted-foreground">
          No description available.
        </p>
      ) : (
        <div className="max-w-[60ch] space-y-1 text-sm">
          {formatted.lines.map((line, i) => (
            <p key={i}>
              {line.map((seg, j) => (
                <span key={j}>{seg.text}</span>
              ))}
            </p>
          ))}
        </div>
      )}

      {hasLevels ? (
        <>
          <div className="rounded-lg border bg-[hsl(var(--sidebar-background))] p-3">
            <div className="mb-1.5 flex items-baseline gap-2">
              <span className="text-[13px] font-semibold">Level {level}</span>
              <span className="text-xs tabular-nums text-muted-foreground">
                / {maxLevel}
              </span>
            </div>
            <input
              type="range"
              min={1}
              max={maxLevel}
              value={level}
              onChange={(e) => setLevel(Number(e.target.value))}
              aria-label="Skill level"
              className="mb-2.5 mt-0.5 w-full accent-[hsl(var(--acc))]"
            />
            {row ? (
              <div
                data-testid="stat-readout"
                className="grid grid-cols-2 gap-x-3.5 gap-y-1.5"
              >
                {statColumns.map((c) => (
                  <div
                    key={c.key}
                    className="flex justify-between gap-2.5 text-[13px]"
                  >
                    <span className="text-muted-foreground">{c.label}</span>
                    <span className="font-medium tabular-nums">
                      {row[c.key] ?? ""}
                    </span>
                  </div>
                ))}
              </div>
            ) : null}
          </div>

          <Collapsible defaultOpen>
            <CollapsibleTrigger className="group flex cursor-pointer items-center gap-1.5 rounded text-[12.5px] font-medium text-muted-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-ring">
              <ChevronRight className="h-3.5 w-3.5 transition-transform group-data-[state=open]:rotate-90" />
              All {table.rows.length} levels
            </CollapsibleTrigger>
            <CollapsibleContent>
              <div className="mt-2 overflow-x-auto rounded-md border">
                <Table>
                  <TableHeader>
                    <TableRow>
                      {table.columns.map((c) => (
                        <TableHead key={c.key}>{c.label}</TableHead>
                      ))}
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {table.rows.map((r, i) => (
                      <TableRow
                        key={i}
                        data-selected={i === level - 1}
                        className={cn(
                          i === level - 1 && "bg-[hsl(var(--acc)/0.14)]",
                        )}
                      >
                        {table.columns.map((c) => (
                          <TableCell key={c.key} className="tabular-nums">
                            {r[c.key] ?? ""}
                          </TableCell>
                        ))}
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </CollapsibleContent>
          </Collapsible>
        </>
      ) : (
        <p className="text-sm text-muted-foreground">No per-level data.</p>
      )}
    </div>
  );
}
