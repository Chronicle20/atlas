import { useEffect, useState } from "react";
import type { IngestRun, IngestRunWorker } from "@/services/api/seed.service";
import {
  formatDuration,
  ingestElapsedMs,
} from "@/components/features/setup/ingest-progress";

interface IngestProgressPanelProps {
  run?: IngestRun | undefined;
  isError: boolean;
}

const TERMINAL_PHASES = ["succeeded", "failed", "stuck"];

function phaseClass(phase: string): string {
  switch (phase) {
    case "failed":
    case "stuck":
      return "text-destructive font-medium";
    case "unknown":
      return "text-amber-600 dark:text-amber-500 font-medium";
    case "succeeded":
      return "text-emerald-600 dark:text-emerald-500 font-medium";
    default:
      return "text-muted-foreground font-medium";
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
        <p className="text-sm">
          Ingest <span className={phaseClass(run.phase)}>{run.phase}</span>
        </p>
        <p className="text-xs text-muted-foreground">
          {run.workersComplete} / {run.workersTotal} workers
          {elapsed !== null
            ? ` · ${terminal ? "took" : "elapsed"} ${formatDuration(elapsed)}`
            : ""}
        </p>
      </div>

      {run.reason ? (
        <p className="mt-1 text-xs text-destructive">{run.reason}</p>
      ) : null}

      <ul className="mt-2 grid gap-1 sm:grid-cols-2">
        {run.workers.map((w) => (
          <li
            key={w.name}
            className="flex items-baseline justify-between gap-2 text-xs"
          >
            <span className="font-mono">{w.name}</span>
            <span className="text-muted-foreground">
              {/* A worker still `running` under a terminal run phase was the
                  one in flight when the run ended — a derived presentation
                  concern, requiring no extra stored state. */}
              {terminal && w.state === "running" ? "interrupted" : w.state}
              {w.error ? ` — ${w.error}` : ""} · {workerDuration(w, now)}
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}
