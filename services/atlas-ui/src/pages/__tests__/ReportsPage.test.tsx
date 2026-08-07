import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ReportsPage } from "@/pages/ReportsPage";
import { reportsService } from "@/services/api/reports.service";
import { ReportKind, ReportStatus } from "@/types/models/report";
import type { Report } from "@/types/models/report";

// Radix Select relies on DOM APIs jsdom does not implement.
Element.prototype.hasPointerCapture ||= () => false;
Element.prototype.scrollIntoView ||= () => {};

vi.mock("@/services/api/reports.service", () => ({
  reportsService: {
    getAllReports: vi.fn(async () => []),
  },
}));

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: {
      id: "test-tenant",
      attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
    },
  }),
}));

function makeReport(
  id: string,
  overrides?: Partial<Report["attributes"]>,
): Report {
  return {
    id,
    type: "reports",
    attributes: {
      kind: ReportKind.Sue,
      reporterId: 1,
      reporterName: "Reporter",
      accusedId: 2,
      accusedName: "Accused",
      reasonType: 3,
      description: "test",
      chatLog: null,
      serverTranscript: null,
      status: ReportStatus.Open,
      createdAt: "2024-01-01T00:00:00Z",
      updatedAt: "2024-01-01T00:00:00Z",
      ...overrides,
    },
  };
}

function renderAt(initial: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[initial]}>
        <Routes>
          <Route path="/reports" element={<ReportsPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("ReportsPage", () => {
  beforeEach(() => {
    vi.mocked(reportsService.getAllReports).mockReset();
  });

  it("requests all reports with no status filter on mount", async () => {
    vi.mocked(reportsService.getAllReports).mockResolvedValue([
      makeReport("1"),
    ]);

    renderAt("/reports");

    await waitFor(() => {
      expect(reportsService.getAllReports).toHaveBeenCalledWith(undefined);
    });
  });

  it("renders the reports the hook returns", async () => {
    vi.mocked(reportsService.getAllReports).mockResolvedValue([
      makeReport("1", { reporterName: "Alice", accusedName: "Bob" }),
    ]);

    renderAt("/reports");

    await screen.findByText("Alice");
    expect(screen.getByText("Bob")).toBeInTheDocument();
  });

  it("renders a claim's reason code as its client label, not the raw byte", async () => {
    vi.mocked(reportsService.getAllReports).mockResolvedValue([
      makeReport("1", { kind: ReportKind.Claim, reasonType: 6 }),
    ]);

    renderAt("/reports");

    expect(await screen.findByText("Cash trade")).toBeInTheDocument();
    expect(screen.queryByText("6")).not.toBeInTheDocument();
  });

  it("renders an unrecognised claim reason with the code still visible", async () => {
    vi.mocked(reportsService.getAllReports).mockResolvedValue([
      makeReport("1", { kind: ReportKind.Claim, reasonType: 42 }),
    ]);

    renderAt("/reports");

    expect(await screen.findByText("Unknown (42)")).toBeInTheDocument();
  });

  // Sue stores a player-typed slash-command number in the same column, so it
  // must NOT be read through the claim reason table.
  it("renders a sue's reason as a bare category, not a claim label", async () => {
    vi.mocked(reportsService.getAllReports).mockResolvedValue([
      makeReport("1", { kind: ReportKind.Sue, reasonType: 6 }),
    ]);

    renderAt("/reports");

    expect(await screen.findByText("Category 6")).toBeInTheDocument();
    expect(screen.queryByText("Cash trade")).not.toBeInTheDocument();
  });

  it("re-requests with the selected status filter", async () => {
    vi.mocked(reportsService.getAllReports).mockResolvedValue([
      makeReport("1"),
    ]);

    renderAt("/reports");

    await waitFor(() => {
      expect(reportsService.getAllReports).toHaveBeenCalledWith(undefined);
    });

    const user = userEvent.setup();
    await user.click(await screen.findByRole("combobox"));
    await user.click(await screen.findByRole("option", { name: "Open" }));

    await waitFor(() => {
      const lastCall = vi
        .mocked(reportsService.getAllReports)
        .mock.calls.at(-1)!;
      expect(lastCall[0]).toEqual({ status: ReportStatus.Open });
    });
  });
});
