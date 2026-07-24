import { useState } from "react";
import type { CSSProperties } from "react";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import type { SkillDefinitionWithIcon } from "@/lib/hooks/api/useSkillDefinition";
import { deriveSkillType } from "@/lib/skills/skill-type";
import { resolveSkillName } from "@/lib/skills/beginner-skill-names";
import { SkillIcon } from "@/components/features/jobs/skill-icon";

export type SkillListState =
  "loading" | "error" | "empty" | "defs-failed" | "ready";

interface SkillListProps {
  jobName: string;
  defs: SkillDefinitionWithIcon[];
  state: SkillListState;
  selectedSkillId: number | null;
  /** Branch accent token name, e.g. "--c-warrior". */
  accent: string;
  onSelect: (id: number) => void;
}

// Filter text is intentionally local (not URL) state; the page remounts this
// component with key={jobId} so the filter resets on job change (FR-3.4).
export function SkillList({
  jobName,
  defs,
  state,
  selectedSkillId,
  accent,
  onSelect,
}: SkillListProps) {
  const [filter, setFilter] = useState("");
  const q = filter.trim().toLowerCase();
  const filtered = defs.filter((d) => {
    if (!q) return true;
    return (
      resolveSkillName(d.id, d.name).toLowerCase().includes(q) ||
      String(d.id).includes(q)
    );
  });

  let body;
  if (state === "loading") {
    body = (
      <div data-testid="skill-list-loading" className="space-y-2 px-1 py-1">
        {[0, 1, 2].map((i) => (
          <Skeleton key={i} className="h-10 w-full" />
        ))}
      </div>
    );
  } else if (state === "error") {
    body = (
      <p className="py-8 text-center text-destructive">
        Failed to load this job&#39;s skills.
      </p>
    );
  } else if (state === "empty") {
    body = (
      <p className="py-8 text-center text-muted-foreground">
        This job grants no skills.
      </p>
    );
  } else if (state === "defs-failed") {
    body = (
      <p className="py-8 text-center text-destructive">
        Skill details unavailable.
      </p>
    );
  } else if (filtered.length === 0) {
    body = (
      <p className="py-8 text-center text-muted-foreground">
        No skills match &ldquo;{filter}&rdquo;.
      </p>
    );
  } else {
    body = filtered.map((d) => {
      const name = resolveSkillName(d.id, d.name);
      return (
        <button
          key={d.id}
          type="button"
          aria-pressed={selectedSkillId === d.id}
          onClick={() => onSelect(d.id)}
          className="flex w-full items-center gap-3 rounded-md px-2 py-1.5 text-left hover:bg-accent focus:outline-none focus-visible:ring-2 focus-visible:ring-ring aria-pressed:bg-[hsl(var(--acc)/0.13)]"
        >
          <SkillIcon def={d} name={name} />
          <span className="min-w-0">
            <span className="block truncate text-[13.5px] font-medium">
              {name}
            </span>
            <span className="block font-mono text-[11px] text-muted-foreground">
              {d.id}
            </span>
          </span>
          <span className="ml-auto flex flex-none items-center gap-2">
            <Badge variant="secondary">{deriveSkillType(d)}</Badge>
            <span className="whitespace-nowrap text-xs tabular-nums text-muted-foreground">
              Master {d.maxLevel ?? "—"}
            </span>
          </span>
        </button>
      );
    });
  }

  return (
    <div
      className="flex min-h-0 flex-1 flex-col"
      style={{ "--acc": `var(${accent})` } as CSSProperties}
    >
      <div className="flex flex-none items-center gap-3 px-4 pb-2 pt-3">
        <h4 className="text-[13px] font-semibold text-muted-foreground">
          {jobName} — Skills
        </h4>
        <Input
          type="search"
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          placeholder="Filter skills…"
          aria-label="Filter skills"
          className="ml-auto h-8 w-[190px] text-[13px]"
        />
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto px-2 pb-3">{body}</div>
    </div>
  );
}
