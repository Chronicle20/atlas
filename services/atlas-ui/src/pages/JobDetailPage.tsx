import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { ChevronLeft, Sparkles } from "lucide-react";
import { useTenant } from "@/context/tenant-context";
import { useJobSkills } from "@/lib/hooks/api/useJobSkills";
import { useJobSkillDefinitions } from "@/lib/hooks/api/useJobSkillDefinitions";
import type { SkillDefinitionWithIcon } from "@/lib/hooks/api/useSkillDefinition";
import { getJobNameById } from "@/lib/jobs";
import { deriveSkillType } from "@/lib/skills/skill-type";
import { buildLevelTable } from "@/lib/skills/level-table";
import { resolveSkillName } from "@/lib/skills/beginner-skill-names";
import {
  formatSkillDescription,
  type FormattedDescription,
} from "@/lib/skills/format-skill-description";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

function SkillIcon({
  def,
  name,
}: {
  def: SkillDefinitionWithIcon;
  name: string;
}) {
  const [failed, setFailed] = useState(false);
  if (failed) {
    return (
      <span
        data-testid={`skill-icon-fallback-${def.id}`}
        className="inline-flex h-8 w-8 items-center justify-center text-muted-foreground"
      >
        <Sparkles className="h-4 w-4" />
      </span>
    );
  }
  return (
    <img
      src={def.iconUrl}
      alt={name}
      width={32}
      height={32}
      loading="lazy"
      className="object-contain"
      onError={() => setFailed(true)}
    />
  );
}

function SkillDescription({ formatted }: { formatted: FormattedDescription }) {
  if (formatted.lines.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">No description available.</p>
    );
  }
  return (
    <div className="text-sm space-y-1">
      {formatted.lines.map((line, i) => (
        <p key={i}>
          {line.map((seg, j) => (
            <span key={j}>{seg.text}</span>
          ))}
        </p>
      ))}
    </div>
  );
}

function LevelTable({ def }: { def: SkillDefinitionWithIcon }) {
  const table = buildLevelTable(def.effects);
  if (table.rows.length === 0) {
    return <p className="text-sm text-muted-foreground">No per-level data.</p>;
  }
  return (
    <div className="rounded-md border overflow-auto">
      <Table>
        <TableHeader className="sticky top-0 bg-background z-10">
          <TableRow>
            {table.columns.map((c) => (
              <TableHead key={c.key}>{c.label}</TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {table.rows.map((row, i) => (
            <TableRow key={i}>
              {table.columns.map((c) => (
                <TableCell key={c.key}>{row[c.key] ?? ""}</TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function SkillRow({ def }: { def: SkillDefinitionWithIcon }) {
  const type = deriveSkillType(def);
  const name = resolveSkillName(def.id, def.name);
  const formatted = formatSkillDescription(def.description);
  return (
    <Collapsible>
      <div className="flex items-center gap-3 py-2 border-b">
        <SkillIcon def={def} name={name} />
        <CollapsibleTrigger asChild>
          <button className="flex-1 text-left">
            <span className="font-medium">{name}</span>
          </button>
        </CollapsibleTrigger>
        <Badge variant="secondary">{type}</Badge>
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <span
                tabIndex={0}
                className="text-sm text-muted-foreground w-24 text-right cursor-help focus:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded"
              >
                Master Lv <span>{def.maxLevel ?? "—"}</span>
              </span>
            </TooltipTrigger>
            <TooltipContent>
              <p>Skill&#39;s maximum (master) level</p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </div>
      <CollapsibleContent className="py-3 pl-11 space-y-3">
        <SkillDescription formatted={formatted} />
        <div className="flex gap-4 text-xs text-muted-foreground">
          <span>Type: {type}</span>
          {def.element ? <span>Element: {def.element}</span> : null}
          <span>Master Level: {def.maxLevel ?? "—"}</span>
        </div>
        <LevelTable def={def} />
      </CollapsibleContent>
    </Collapsible>
  );
}

export function JobDetailPage() {
  const { jobId } = useParams<{ jobId: string }>();
  const { activeTenant } = useTenant();
  const numericJobId = Number(jobId);
  const jobName = getJobNameById(numericJobId) ?? `Job ${jobId}`;

  const skillsQuery = useJobSkills(activeTenant, numericJobId);
  const skillIds = skillsQuery.data ?? [];
  const {
    definitions,
    isLoading: defsLoading,
    isError: defsError,
  } = useJobSkillDefinitions(activeTenant, skillIds);

  const loading = skillsQuery.isLoading || (skillIds.length > 0 && defsLoading);

  return (
    <div className="flex flex-col flex-1 min-h-0 space-y-6 overflow-y-auto p-10 pb-16">
      <div className="flex items-center gap-2">
        <Link
          to="/jobs"
          className="text-muted-foreground hover:text-foreground"
        >
          <ChevronLeft className="h-5 w-5" />
        </Link>
        <h2 className="text-2xl font-bold tracking-tight">{jobName}</h2>
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <span
                tabIndex={0}
                className="inline-flex items-center rounded border px-2 py-0.5 text-xs font-medium cursor-help focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              >
                {jobId}
              </span>
            </TooltipTrigger>
            <TooltipContent copyable>
              <p>{jobId}</p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </div>

      {!activeTenant ? (
        <Card>
          <CardContent className="py-10 text-center text-muted-foreground">
            Select a tenant to browse its jobs and skills.
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardHeader>
            <CardTitle>Skills</CardTitle>
          </CardHeader>
          <CardContent>
            {loading ? (
              <div data-testid="job-detail-loading" className="space-y-2">
                {[0, 1, 2].map((i) => (
                  <Skeleton key={i} className="h-10 w-full" />
                ))}
              </div>
            ) : skillsQuery.isError ? (
              <p className="text-center py-8 text-destructive">
                Failed to load this job&#39;s skills.
              </p>
            ) : skillIds.length === 0 ? (
              <p className="text-center py-8 text-muted-foreground">
                This job grants no skills.
              </p>
            ) : definitions.length === 0 && defsError ? (
              <p className="text-center py-8 text-destructive">
                Skill details unavailable.
              </p>
            ) : (
              <div>
                {definitions.map((def) => (
                  <SkillRow key={def.id} def={def} />
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
