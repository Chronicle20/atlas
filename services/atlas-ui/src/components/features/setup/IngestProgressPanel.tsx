import type { IngestRun, IngestRunWorker } from "@/services/api/seed.service";
import {
  formatDuration,
  ingestElapsedMs,
} from "@/components/features/setup/ingest-progress";

interface IngestProgressPanelProps {
  run?: IngestRun;
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
 * Presentational only — it owns no fetching. Both the Setup page (tenant
 * scope) and the Baselines page (shared scope) mount this same component with
 * a different hook, so the two surfaces cannot drift.
 */
export function IngestProgressPanel({
  run,
  isError,
}: IngestProgressPanelProps) {
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

  // Presentational-only: no internal ticking clock is owned here (the
  // polling hooks in Task 11 re-render this panel on their own cadence),
  // so "now" is read once per render as a snapshot for the elapsed/duration
  // display rather than driven by a timer.
  // eslint-disable-next-line react-hooks/purity -- deliberate snapshot read, not a stateful clock
  const now = Date.now();
  const elapsed = ingestElapsedMs(run, now);
  const terminal = TERMINAL_PHASES.includes(run.phase);

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

      {/* When a specific worker already carries its own .error, that is the
          more precise diagnostic — surfacing the run-level reason too would
          just repeat the same text (the reason is usually derived from the
          failing worker) in a second place. */}
      {run.reason && !run.workers.some((w) => w.error) ? (
        <p className="mt-1 text-xs text-destructive">{run.reason}</p>
      ) : null}

      <ul className="mt-2 grid gap-1 sm:grid-cols-2">
        {run.workers.map((w) => {
          // A worker still `running` under a terminal run phase was the one
          // in flight when the run ended — a derived presentation concern,
          // requiring no extra stored state (PRD Q1).
          const interrupted = terminal && w.state === "running";
          // While the run itself is actively `running`, an individual
          // worker's `running` state is redundant with the phase indicator
          // above (and would otherwise collide with it as a duplicate text
          // match) — a progress ellipsis conveys the same thing.
          const stateLabel = interrupted
            ? "interrupted"
            : w.state === "running"
              ? "…"
              : w.state;
          return (
            <li
              key={w.name}
              className="flex items-baseline justify-between gap-2 text-xs"
            >
              <span className="font-mono">{w.name}</span>
              <span className="text-muted-foreground">
                <span>{stateLabel}</span>
                {w.error ? <span> — {w.error}</span> : null} ·{" "}
                {workerDuration(w, now)}
              </span>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
