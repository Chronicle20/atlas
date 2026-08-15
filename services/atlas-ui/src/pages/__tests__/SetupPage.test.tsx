import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { SetupPage } from "@/pages/SetupPage";

const mockTenant = {
  id: "11111111-1111-1111-1111-111111111111",
  attributes: {
    name: "Test Tenant",
    region: "GMS",
    majorVersion: 83,
    minorVersion: 1,
  },
};

const idleMutation = { mutate: vi.fn(), isPending: false };
const emptyStatus = { data: undefined };

// Mutable per-test data-status so individual tests can flip documentCount.
let dataStatusData: {
  documentCount: number;
  updatedAt: string | null;
  baselineRestoredAt: string | null;
  baselineSha256: string | null;
} = {
  documentCount: 0,
  updatedAt: null,
  baselineRestoredAt: null,
  baselineSha256: null,
};

// Mutable per-test useIngestRun return so one test can exercise a realistic
// in-flight run without perturbing every other test's default (undefined,
// which only guards against the panel crashing when there's nothing to show).
let ingestRunResult: {
  data:
    | {
        runId: string;
        jobName: string;
        scope: string;
        region: string;
        version: string;
        tenant?: string;
        phase:
          "none" | "running" | "succeeded" | "failed" | "stuck" | "unknown";
        startedAt: string | null;
        finishedAt: string | null;
        reason: string | null;
        workersTotal: number;
        workersComplete: number;
        workers: {
          name: string;
          state: "pending" | "running" | "succeeded" | "failed" | "skipped";
          startedAt: string | null;
          finishedAt: string | null;
          error: string | null;
        }[];
      }
    | undefined;
  isError: boolean;
} = { data: undefined, isError: false };

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: mockTenant }),
}));

vi.mock("@/lib/hooks/api/useBaseline", () => ({
  useRestoreBaseline: () => idleMutation,
}));

vi.mock("@/lib/hooks/api/useSeed", () => ({
  useSeedDrops: () => idleMutation,
  useSeedGachapons: () => idleMutation,
  useSeedNpcConversations: () => idleMutation,
  useSeedQuestConversations: () => idleMutation,
  useSeedItemConversations: () => idleMutation,
  useSeedNpcShops: () => idleMutation,
  useSeedPortalScripts: () => idleMutation,
  useSeedReactorScripts: () => idleMutation,
  useSeedMapActionScripts: () => idleMutation,
  useSeedTransportRoutes: () => idleMutation,
  useSeedTransportVessels: () => idleMutation,
  useSeedInstanceRoutes: () => idleMutation,
  useUploadWzFiles: () => idleMutation,
  useRunDataProcessing: () => idleMutation,
  useWzInputStatus: () => ({
    data: { fileCount: 2, totalBytes: 1024, updatedAt: null },
  }),
  useDataStatus: () => ({ data: dataStatusData }),
  useIngestRun: () => ingestRunResult,
  useDropsSeedStatus: () => emptyStatus,
  useGachaponsSeedStatus: () => emptyStatus,
  useNpcConversationsSeedStatus: () => emptyStatus,
  useQuestConversationsSeedStatus: () => emptyStatus,
  useItemConversationsSeedStatus: () => emptyStatus,
  useNpcShopsSeedStatus: () => emptyStatus,
  usePortalScriptsSeedStatus: () => emptyStatus,
  useReactorScriptsSeedStatus: () => emptyStatus,
  useMapActionScriptsSeedStatus: () => emptyStatus,
  useTransportRoutesSeedStatus: () => emptyStatus,
  useTransportVesselsSeedStatus: () => emptyStatus,
  useInstanceRoutesSeedStatus: () => emptyStatus,
  showWzUploadErrorToast: vi.fn(),
}));

describe("SetupPage (tenant-only)", () => {
  it("is titled Setup and has no scope toggle and no publish row", () => {
    render(<SetupPage />);
    expect(screen.getByRole("heading", { name: "Setup" })).toBeInTheDocument();
    expect(screen.queryByTestId("scope-toggle")).not.toBeInTheDocument();
    expect(
      screen.queryByText(/Publish Canonical Baseline/i),
    ).not.toBeInTheDocument();
  });

  it("shows the restore row when the tenant document count is 0", () => {
    dataStatusData = {
      documentCount: 0,
      updatedAt: null,
      baselineRestoredAt: null,
      baselineSha256: null,
    };
    render(<SetupPage />);
    expect(screen.getByText(/Restore Canonical Baseline/i)).toBeInTheDocument();
  });

  it("hides the restore row when documents exist", () => {
    dataStatusData = {
      documentCount: 5,
      updatedAt: null,
      baselineRestoredAt: null,
      baselineSha256: null,
    };
    render(<SetupPage />);
    expect(
      screen.queryByText(/Restore Canonical Baseline/i),
    ).not.toBeInTheDocument();
  });

  it("wires useIngestRun's live data into the ingest progress panel", () => {
    ingestRunResult = {
      data: {
        runId: "11111111-2222-3333-4444-555555555555",
        jobName: "ingest-t-11111111-gms-83-1-abcdef",
        scope: "tenants/11111111-1111-1111-1111-111111111111",
        region: "GMS",
        version: "83.1",
        tenant: "11111111-1111-1111-1111-111111111111",
        phase: "running",
        startedAt: "2026-08-08T00:00:00Z",
        finishedAt: null,
        reason: null,
        workersTotal: 2,
        workersComplete: 1,
        workers: [
          {
            name: "STRING",
            state: "succeeded",
            startedAt: "2026-08-08T00:00:00Z",
            finishedAt: "2026-08-08T00:00:05Z",
            error: null,
          },
          {
            name: "MAP",
            state: "running",
            startedAt: "2026-08-08T00:00:05Z",
            finishedAt: null,
            error: null,
          },
        ],
      },
      isError: false,
    };
    render(<SetupPage />);
    // The panel's own rendering logic is exercised by
    // IngestProgressPanel.test.tsx; this only asserts the Setup page passes
    // useIngestRun's live data through rather than dropping it.
    // "Running" appears twice — the run's phase badge and the MAP worker's
    // state — so this asserts presence, not uniqueness.
    expect(screen.getAllByText("Running").length).toBeGreaterThan(0);
    expect(
      screen.getByText("1 / 2 workers", { exact: false }),
    ).toBeInTheDocument();
    expect(screen.getByText("STRING")).toBeInTheDocument();
    expect(screen.getByText("MAP")).toBeInTheDocument();

    // Reset for subsequent tests in this file.
    ingestRunResult = { data: undefined, isError: false };
  });

  it("renders all eleven seed rows", () => {
    render(<SetupPage />);
    for (const label of [
      "Monster & Reactor Drops",
      "Reward Pools",
      "NPC Conversations",
      "Quest Conversations",
      "NPC Shops",
      "Portal Scripts",
      "Reactor Scripts",
      "Map Action Scripts",
      "Transport Routes",
      "Transport Vessels",
      "Instance Transport Routes",
    ]) {
      expect(screen.getByText(label)).toBeInTheDocument();
    }
  });
});
