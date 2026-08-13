import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { UpdateReportStatusDialog } from "../UpdateReportStatusDialog";
import { reportsService } from "@/services/api/reports.service";
import { ReportKind, ReportStatus } from "@/types/models/report";
import type { Report } from "@/types/models/report";

// Radix Select relies on DOM APIs jsdom does not implement.
Element.prototype.hasPointerCapture ||= () => false;
Element.prototype.scrollIntoView ||= () => {};

vi.mock("@/services/api/reports.service", () => ({
  reportsService: {
    updateReportStatus: vi.fn().mockResolvedValue(undefined),
  },
}));

const report: Report = {
  id: "1",
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
  },
};

function setup() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const onOpenChange = vi.fn();
  const utils = render(
    <QueryClientProvider client={qc}>
      <UpdateReportStatusDialog
        report={report}
        open
        onOpenChange={onOpenChange}
      />
    </QueryClientProvider>,
  );
  const rerenderWithOpen = (open: boolean) =>
    utils.rerender(
      <QueryClientProvider client={qc}>
        <UpdateReportStatusDialog
          report={report}
          open={open}
          onOpenChange={onOpenChange}
        />
      </QueryClientProvider>,
    );
  return { ...utils, onOpenChange, rerenderWithOpen };
}

describe("UpdateReportStatusDialog", () => {
  beforeEach(() => vi.clearAllMocks());

  it("shows the report's current status on the initial open", () => {
    setup();
    expect(screen.getByRole("combobox")).toHaveTextContent("Open");
    expect(screen.getByRole("button", { name: /save/i })).toBeDisabled();
  });

  it("resets the selected status to the report's current status on reopen after a cancelled edit", async () => {
    const user = userEvent.setup();
    const { rerenderWithOpen } = setup();

    // Pick a different status but don't save.
    await user.click(screen.getByRole("combobox"));
    await user.click(await screen.findByRole("option", { name: "Reviewed" }));
    expect(screen.getByRole("combobox")).toHaveTextContent("Reviewed");
    expect(screen.getByRole("button", { name: /save/i })).not.toBeDisabled();

    // Cancel — the dialog never unmounts (ReportDetailPage renders it
    // unconditionally and only toggles `open`), so only closing+reopening
    // via the `open` prop can reveal whether local state was reset.
    await user.click(screen.getByRole("button", { name: /cancel/i }));
    rerenderWithOpen(false);
    rerenderWithOpen(true);

    // Without a reopen resync, the Select would still show "Reviewed" and
    // Save would be wrongly enabled even though the report is still "Open".
    expect(screen.getByRole("combobox")).toHaveTextContent("Open");
    expect(screen.getByRole("button", { name: /save/i })).toBeDisabled();
  });

  it("saves the newly selected status and calls onOpenChange(false) on success", async () => {
    const user = userEvent.setup();
    const { onOpenChange } = setup();

    await user.click(screen.getByRole("combobox"));
    await user.click(await screen.findByRole("option", { name: "Actioned" }));
    await user.click(screen.getByRole("button", { name: /save/i }));

    await waitFor(() => {
      expect(reportsService.updateReportStatus).toHaveBeenCalledWith(
        "1",
        ReportStatus.Actioned,
      );
    });
    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
  });
});
