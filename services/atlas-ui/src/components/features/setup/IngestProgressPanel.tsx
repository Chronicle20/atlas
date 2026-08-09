import { useEffect, useState } from "react";
import type { IngestRun, IngestRunWorker } from "@/services/api/seed.service";
import {
  formatDuration,
  ingestElapsedMs,
} from "@/components/features/setup/ingest-progress";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

interface IngestProgressPanelProps {
  run?: IngestRun | undefined;
  isError: boolean;
}

const TERMINAL_PHASES = ["succeeded", "failed", "stuck"];

/** Wire states are lower-case; the UI labels them in title case throughout. */
const PHASE_LABEL: Record<string, string> = {
  none: "None",
  running: "Running",
  succeeded: "Succeeded",
  failed: "Failed",
  stuck: "Stuck",
  unknown: "Unknown",
};

const WORKER_STATE_LABEL: Record<string, string> = {
  pending: "Pending",
  running: "Running",
  succeeded: "Succeeded",
  failed: "Failed",
  skipped: "Skipped",
  // Derived, never stored: a worker still `running` under a terminal run
  // phase was the one in flight when the run ended.
  interrupted: "Interrupted",
};

/** Badge styling for the overall run phase. */
function phaseBadgeClass(phase: string): string {
  switch (phase) {
    case "succeeded":
      return "border-emerald-600/30 text-emerald-600 dark:text-emerald-500";
    case "unknown":
      return "border-amber-600/30 text-amber-600 dark:text-amber-500";
    default:
      return "";
  }
}

function phaseBadgeVariant(
  phase: string,
): "secondary" | "destructive" | "outline" {
  switch (phase) {
    case "failed":
    case "stuck":
      return "destructive";
    case "succeeded":
    case "unknown":
      return "outline";
    default:
      return "secondary";
  }
}

function workerStateClass(state: string): string {
  switch (state) {
    case "failed":
    case "interrupted":
      return "text-destructive";
    case "succeeded":
      return "text-emerald-600 dark:text-emerald-500";
    case "running":
      return "text-foreground";
    default:
      return "text-muted-foreground";
  }
}

function workerDuration(w: IngestRunWorker, now: number): string {
  if (!w.startedAt) return "—";
  const start = Date.parse(w.startedAt);
  if (Number.isNaN(start)) return "—";
  const end = w.finishedAt ? Date.parse(w.finishedAt) : now;
  return formatDuration(Math.max(0, end - start));
}

/**
 * A once-per-second clock, live only while `active`.
 *
 * The panel cannot read `Date.now()` at render and rely on the polling hook to
 * re-render it: React Query's structural sharing returns the identical object
 * when the fetched record is unchanged, so a run whose state is momentarily
 * quiet produces no re-render at all and the elapsed readout freezes. Owning
 * the tick here also keeps it independent of whatever else a host page
 * happens to be re-rendering.
 */
function useNow(active: boolean): number {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    // A terminal run's duration is a fixed fact — no timer behind it.
    if (!active) return;
    // Only the interval callback sets state: the first tick lands within a
    // second of activation, so an initial synchronous resync would buy
    // nothing and cost a cascading render.
    const id = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(id);
  }, [active]);
  return now;
}

/**
 * Presentational only — it owns no fetching. Both the Setup page (tenant
 * scope) and the Baselines page (shared scope) mount this same component with
 * a different hook, so the two surfaces cannot drift.
 */
export function IngestProgressPanel({
  run,
  isError,
}: IngestProgressPanelProps) {
  const terminal = run ? TERMINAL_PHASES.includes(run.phase) : true;
  const now = useNow(!!run && !terminal && run.phase !== "none");

  if (isError || !run) {
    return (
      <div className="border-b last:border-0 py-3">
        <p className="text-sm text-muted-foreground">
          Ingest progress unavailable.
        </p>
      </div>
    );
  }

  if (run.phase === "none") {
    return (
      <div className="border-b last:border-0 py-3">
        <p className="text-sm text-muted-foreground">
          No ingest has been run for this region/version yet.
        </p>
      </div>
    );
  }

  const elapsed = ingestElapsedMs(run, now);

  return (
    <div className="border-b last:border-0 py-3" aria-live="polite">
      <div className="flex items-center justify-between gap-4">
        <div className="flex items-center gap-2">
          <p className="text-sm font-medium">Ingest</p>
          <Badge
            variant={phaseBadgeVariant(run.phase)}
            className={cn(phaseBadgeClass(run.phase))}
          >
            {PHASE_LABEL[run.phase] ?? run.phase}
          </Badge>
        </div>
        <p className="text-xs text-muted-foreground tabular-nums">
          {run.workersComplete} / {run.workersTotal} workers
          {elapsed !== null
            ? ` · ${terminal ? "took" : "elapsed"} ${formatDuration(elapsed)}`
            : ""}
        </p>
      </div>

      {run.reason ? (
        <p className="mt-1 text-xs text-destructive">{run.reason}</p>
      ) : null}

      {/* Each row is a fixed three-column grid — name, state, duration — so
          the state and duration columns line up across both grid columns
          regardless of how long a worker's name or state word is. */}
      <ul className="mt-2 grid gap-x-8 gap-y-1 sm:grid-cols-2">
        {run.workers.map((w) => {
          const state =
            terminal && w.state === "running" ? "interrupted" : w.state;
          return (
            <li
              key={w.name}
              className="grid grid-cols-[minmax(0,1fr)_5.5rem_4rem] items-baseline gap-2 text-xs"
            >
              <span className="font-mono truncate">{w.name}</span>
              <span className={workerStateClass(state)}>
                {WORKER_STATE_LABEL[state] ?? state}
              </span>
              <span className="text-right tabular-nums text-muted-foreground">
                {workerDuration(w, now)}
              </span>
              {w.error ? (
                <span className="col-span-3 text-destructive">{w.error}</span>
              ) : null}
            </li>
          );
        })}
      </ul>
    </div>
  );
}
