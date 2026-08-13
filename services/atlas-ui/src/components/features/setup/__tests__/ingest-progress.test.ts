import { describe, it, expect } from "vitest";
import {
  INGEST_BLOCKING_PHASES,
  formatDuration,
  ingestElapsedMs,
  ingestPublishBlockReason,
} from "@/components/features/setup/ingest-progress";
import type { IngestPhase, IngestRun } from "@/services/api/seed.service";

function run(phase: IngestPhase, over: Partial<IngestRun> = {}): IngestRun {
  return {
    runId: "r1",
    jobName: "j1",
    scope: "shared",
    region: "GMS",
    version: "83.1",
    phase,
    startedAt: "2026-08-08T10:00:00Z",
    finishedAt: null,
    reason: null,
    workersTotal: 11,
    workersComplete: 0,
    workers: [],
    ...over,
  };
}

describe("ingestPublishBlockReason", () => {
  // FR-5.6: `none` is the pre-existing state for every scope never ingested
  // through this mechanism, and an evicted record degrades to it — neither may
  // wedge the publish control.
  it.each<IngestPhase>(["none", "succeeded"])("does not block on %s", (p) => {
    expect(ingestPublishBlockReason(run(p), false)).toBeNull();
  });

  it.each<IngestPhase>(["running", "stuck", "failed", "unknown"])(
    "blocks on %s with an explanation",
    (p) => {
      const reason = ingestPublishBlockReason(run(p), false);
      expect(reason).toBeTruthy();
      expect(reason).toContain(p);
    },
  );

  it("does not block when the run is unknown to the UI", () => {
    expect(ingestPublishBlockReason(undefined, false)).toBeNull();
    expect(ingestPublishBlockReason(undefined, true)).toBeNull();
  });

  it("surfaces the recorded reason when there is one", () => {
    const r = run("stuck", { reason: "watchdog deleted the ingest Job" });
    expect(ingestPublishBlockReason(r, false)).toContain(
      "watchdog deleted the ingest Job",
    );
  });

  it("exports the blocking set", () => {
    expect([...INGEST_BLOCKING_PHASES].sort()).toEqual([
      "failed",
      "running",
      "stuck",
      "unknown",
    ]);
  });
});

describe("ingestElapsedMs", () => {
  it("measures to now while in flight", () => {
    const now = Date.parse("2026-08-08T10:05:00Z");
    expect(ingestElapsedMs(run("running"), now)).toBe(5 * 60 * 1000);
  });

  it("measures to finishedAt once terminal", () => {
    const r = run("succeeded", { finishedAt: "2026-08-08T10:02:00Z" });
    const now = Date.parse("2026-08-08T11:00:00Z");
    expect(ingestElapsedMs(r, now)).toBe(2 * 60 * 1000);
  });

  it("is null without a start", () => {
    expect(
      ingestElapsedMs(run("none", { startedAt: null }), Date.now()),
    ).toBeNull();
  });
});

describe("formatDuration", () => {
  it("formats sub-minute, minute, and hour spans", () => {
    expect(formatDuration(4_000)).toBe("4s");
    expect(formatDuration(125_000)).toBe("2m 5s");
    expect(formatDuration(3_725_000)).toBe("1h 2m");
  });
});
