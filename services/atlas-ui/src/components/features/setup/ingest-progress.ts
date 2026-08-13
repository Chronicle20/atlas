import type { IngestPhase, IngestRun } from "@/services/api/seed.service";

/**
 * Phases that must block publishing a canonical baseline. `none` and
 * `succeeded` are deliberately absent: `none` is the pre-existing state for
 * every scope never ingested through this mechanism, and it is also what an
 * evicted Redis record degrades to — neither may wedge the control.
 */
export const INGEST_BLOCKING_PHASES: readonly IngestPhase[] = [
  "running",
  "stuck",
  "failed",
  "unknown",
];

const PHASE_EXPLANATION: Record<string, string> = {
  running: "an ingest is currently running for this region/version",
  stuck: "the last ingest was terminated by the watchdog",
  failed: "the last ingest failed",
  unknown: "the last ingest's outcome could not be determined",
};

/**
 * Returns the reason publishing is blocked, or null when it is allowed.
 * An unknown run (no data yet, or the endpoint is unreachable) never blocks —
 * progress is telemetry, and losing it must not take the control with it.
 */
export function ingestPublishBlockReason(
  run: IngestRun | undefined,
  _isError: boolean,
): string | null {
  if (!run) return null;
  if (!INGEST_BLOCKING_PHASES.includes(run.phase)) return null;
  const base = `Cannot publish: ${PHASE_EXPLANATION[run.phase] ?? run.phase} (${run.phase}).`;
  return run.reason ? `${base} ${run.reason}` : base;
}

/** Elapsed time for an in-flight run, or total duration once terminal. */
export function ingestElapsedMs(run: IngestRun, now: number): number | null {
  if (!run.startedAt) return null;
  const start = Date.parse(run.startedAt);
  if (Number.isNaN(start)) return null;
  const end = run.finishedAt ? Date.parse(run.finishedAt) : now;
  if (Number.isNaN(end)) return null;
  return Math.max(0, end - start);
}

export function formatDuration(ms: number): string {
  const total = Math.floor(ms / 1000);
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  const s = total % 60;
  if (h > 0) return `${h}h ${m}m`;
  if (m > 0) return `${m}m ${s}s`;
  return `${s}s`;
}
