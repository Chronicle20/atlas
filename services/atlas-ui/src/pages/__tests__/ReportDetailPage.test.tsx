import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ReportDetailPage } from "../ReportDetailPage";
import { ReportKind, ReportStatus } from "@/types/models/report";
import type { Report } from "@/types/models/report";

// Radix Select (rendered inside UpdateReportStatusDialog, which mounts
// unconditionally alongside the report content) relies on DOM APIs jsdom
// does not implement.
Element.prototype.hasPointerCapture ||= () => false;
Element.prototype.scrollIntoView ||= () => {};

const useReportMock = vi.fn();
vi.mock("@/lib/hooks/api/useReports", () => ({
  useReport: (...args: unknown[]) => useReportMock(...args),
  useUpdateReportStatus: () => ({ mutate: vi.fn(), isPending: false }),
}));

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: {
      id: "t1",
      attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
    },
  }),
}));

vi.mock("sonner", () => ({
  Toaster: () => null,
  toast: { success: vi.fn(), error: vi.fn() },
}));

import { toast } from "sonner";

function makeReport(overrides?: Partial<Report["attributes"]>): Report {
  return {
    id: "1",
    type: "reports",
    attributes: {
      kind: ReportKind.Sue,
      reporterId: 1,
      reporterName: "Reporter",
      accusedId: 2,
      accusedName: "Accused",
      reasonType: 3,
      description: "test description",
      chatLog: null,
      serverTranscript: null,
      status: ReportStatus.Open,
      createdAt: "2024-01-01T00:00:00Z",
      updatedAt: "2024-01-01T00:00:00Z",
      ...overrides,
    },
  };
}

function renderPage() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={["/reports/1"]}>
        <Routes>
          <Route path="/reports/:reportId" element={<ReportDetailPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("ReportDetailPage", () => {
  beforeEach(() => {
    useReportMock.mockReset();
    vi.mocked(toast.error).mockReset();
  });

  it("renders the error branch and toasts on a genuine fetch failure — not the not-found branch", () => {
    useReportMock.mockReturnValue({
      data: undefined,
      error: new Error("network down"),
      isLoading: false,
    });

    renderPage();

    expect(screen.getByText("Failed to load report")).toBeInTheDocument();
    expect(screen.queryByText("Report not found")).not.toBeInTheDocument();
    expect(toast.error).toHaveBeenCalledWith(
      "Failed to load report: network down",
    );
  });

  it("renders the not-found branch without toasting when the fetch succeeds with no report", () => {
    useReportMock.mockReturnValue({
      data: undefined,
      error: undefined,
      isLoading: false,
    });

    renderPage();

    expect(screen.getByText("Report not found")).toBeInTheDocument();
    expect(screen.queryByText("Failed to load report")).not.toBeInTheDocument();
    expect(toast.error).not.toHaveBeenCalled();
  });

  it("renders muted empty states for null chatLog and null serverTranscript", () => {
    useReportMock.mockReturnValue({
      data: makeReport({ chatLog: null, serverTranscript: null }),
      error: undefined,
      isLoading: false,
    });

    renderPage();

    expect(screen.getByText("No chat log submitted.")).toBeInTheDocument();
    expect(
      screen.getByText("No server transcript captured."),
    ).toBeInTheDocument();
  });

  it("renders muted empty states for an empty-array serverTranscript and empty-string chatLog", () => {
    useReportMock.mockReturnValue({
      data: makeReport({ chatLog: "", serverTranscript: [] }),
      error: undefined,
      isLoading: false,
    });

    renderPage();

    expect(screen.getByText("No chat log submitted.")).toBeInTheDocument();
    expect(
      screen.getByText("No server transcript captured."),
    ).toBeInTheDocument();
  });

  it("renders a populated chat log and server transcript", () => {
    useReportMock.mockReturnValue({
      data: makeReport({
        chatLog: "Player: hello\nAccused: get lost",
        serverTranscript: [
          {
            timestamp: 1704067200000,
            senderId: 9,
            senderName: "Witness",
            chatType: "general",
            text: "saw the whole thing",
          },
        ],
      }),
      error: undefined,
      isLoading: false,
    });

    renderPage();

    expect(screen.getByText(/Player: hello/)).toBeInTheDocument();
    expect(screen.getByText("Witness")).toBeInTheDocument();
    expect(screen.getByText("saw the whole thing")).toBeInTheDocument();
    expect(screen.getByText("general")).toBeInTheDocument();
    expect(
      screen.queryByText("No server transcript captured."),
    ).not.toBeInTheDocument();
  });
});
