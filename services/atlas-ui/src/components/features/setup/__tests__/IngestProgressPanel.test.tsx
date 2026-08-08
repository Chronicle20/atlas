import { describe, it, expect } from "vitest";
import { render, screen, within } from "@testing-library/react";
import { IngestProgressPanel } from "@/components/features/setup/IngestProgressPanel";
import type { IngestPhase, IngestRun } from "@/services/api/seed.service";

function run(phase: IngestPhase, over: Partial<IngestRun> = {}): IngestRun {
  return {
    runId: "r1",
    jobName: "ingest-shared-gms-83-1-x7f2qa",
    scope: "shared",
    region: "GMS",
    version: "83.1",
    phase,
    startedAt: "2026-08-08T10:00:00Z",
    finishedAt: null,
    reason: null,
    workersTotal: 2,
    workersComplete: 1,
    workers: [
      {
        name: "STRING",
        state: "succeeded",
        startedAt: "2026-08-08T10:00:01Z",
        finishedAt: "2026-08-08T10:02:41Z",
        error: null,
      },
      {
        name: "MAP",
        state: "running",
        startedAt: "2026-08-08T10:02:42Z",
        finishedAt: null,
        error: null,
      },
    ],
    ...over,
  };
}

describe("IngestProgressPanel", () => {
  it("shows the phase and the completed-of-total count", () => {
    render(<IngestProgressPanel run={run("running")} isError={false} />);
    // Scoped outside the worker list: the fixture's MAP worker is also
    // `running`, which independently and correctly renders that same word
    // inside the <ul> — the phase indicator itself never lives there.
    const phaseText = screen.getByText(
      (content, element) => /running/i.test(content) && !element?.closest("ul"),
    );
    expect(phaseText).toBeInTheDocument();
    expect(screen.getByText(/1\s*\/\s*2/)).toBeInTheDocument();
  });

  it("lists every worker with its state", () => {
    render(<IngestProgressPanel run={run("running")} isError={false} />);
    expect(screen.getByText("STRING")).toBeInTheDocument();
    expect(screen.getByText("MAP")).toBeInTheDocument();
    // The worker row renders state and duration as one text node
    // ("succeeded · 2m 40s"), so an exact match on the bare word never
    // hits — match the substring instead.
    expect(screen.getByText(/succeeded/)).toBeInTheDocument();
  });

  it("surfaces the reason on a stuck run", () => {
    const r = run("stuck", {
      reason: "watchdog deleted the ingest Job after 7200s without a heartbeat",
    });
    render(<IngestProgressPanel run={r} isError={false} />);
    expect(screen.getByText(/without a heartbeat/)).toBeInTheDocument();
  });

  it("surfaces a worker's error text on a failed run", () => {
    const r = run("failed", {
      reason: "MAP open Map.wz: boom",
      workers: [
        {
          name: "MAP",
          state: "failed",
          startedAt: "2026-08-08T10:00:01Z",
          finishedAt: "2026-08-08T10:00:09Z",
          error: "boom",
        },
      ],
    });
    render(<IngestProgressPanel run={r} isError={false} />);
    // Scoped to the MAP worker row: the fixture's run-level `reason` also
    // contains "boom" here, since a real reason is typically derived from
    // the failing worker's own error — this asserts specifically that the
    // worker row itself surfaces it, not just the reason banner.
    const workerRow = screen.getByText("MAP").closest("li");
    expect(workerRow).not.toBeNull();
    expect(
      within(workerRow as HTMLElement).getByText(/boom/),
    ).toBeInTheDocument();
  });

  it("surfaces both a distinct run-level reason and a worker's own error", () => {
    const r = run("stuck", {
      reason: "watchdog deleted the ingest Job after 7200s without a heartbeat",
      workers: [
        {
          name: "STRING",
          state: "failed",
          startedAt: "2026-08-08T10:00:01Z",
          finishedAt: "2026-08-08T10:00:09Z",
          error: "rate limited by WZ source",
        },
      ],
    });
    render(<IngestProgressPanel run={r} isError={false} />);
    expect(screen.getByText(/without a heartbeat/)).toBeInTheDocument();
    expect(screen.getByText(/rate limited by WZ source/)).toBeInTheDocument();
  });

  it("renders a worker still running under a terminal phase as interrupted", () => {
    render(<IngestProgressPanel run={run("stuck")} isError={false} />);
    expect(screen.getByText(/interrupted/i)).toBeInTheDocument();
  });

  it("says nothing has been run for phase none", () => {
    const r = run("none", { workers: [], workersTotal: 0, workersComplete: 0 });
    render(<IngestProgressPanel run={r} isError={false} />);
    expect(screen.getByText(/no ingest has been run/i)).toBeInTheDocument();
  });

  it("degrades to progress unavailable on error", () => {
    render(<IngestProgressPanel isError={true} />);
    expect(screen.getByText(/progress unavailable/i)).toBeInTheDocument();
  });
});
