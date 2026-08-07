import { describe, expect, it, vi, beforeEach } from "vitest";

const getListMock = vi.fn();
vi.mock("@/lib/api/client", () => ({
  api: {
    getList: (...args: unknown[]) => getListMock(...args),
    getOne: vi.fn(),
    patch: vi.fn(),
  },
}));

import { api } from "@/lib/api/client";
import { reportsService } from "@/services/api/reports.service";
import { ReportKind, ReportStatus } from "@/types/models/report";
import type { Report } from "@/types/models/report";

function makeReport(id: string, createdAt: string): Report {
  return {
    id,
    type: "reports",
    attributes: {
      kind: ReportKind.Sue,
      reporterId: 1001,
      reporterName: "Reporter",
      accusedId: 2002,
      accusedName: "Accused",
      reasonType: 3,
      description: "desc",
      chatLog: null,
      serverTranscript: null,
      status: ReportStatus.Open,
      createdAt,
      updatedAt: createdAt,
    },
  };
}

describe("reportsService.getAllReports", () => {
  beforeEach(() => vi.clearAllMocks());

  it("sorts by createdAt descending", async () => {
    getListMock.mockResolvedValue([
      makeReport("1", "2026-01-01T00:00:00Z"),
      makeReport("2", "2026-03-01T00:00:00Z"),
      makeReport("3", "2026-02-01T00:00:00Z"),
    ]);

    const result = await reportsService.getAllReports();

    expect(result.map((r) => r.id)).toEqual(["2", "3", "1"]);
  });

  it("requests the base path with no query string when no status filter is given", async () => {
    getListMock.mockResolvedValue([]);

    await reportsService.getAllReports();

    expect(getListMock).toHaveBeenCalledWith("/api/reports", undefined);
  });

  it("appends ?status= when a status filter is given", async () => {
    getListMock.mockResolvedValue([]);

    await reportsService.getAllReports({ status: ReportStatus.Reviewed });

    expect(getListMock).toHaveBeenCalledWith("/api/reports?status=reviewed", {
      status: ReportStatus.Reviewed,
    });
  });
});

describe("reportsService.getReportById", () => {
  beforeEach(() => vi.clearAllMocks());

  it("fetches a single report by id", async () => {
    const report = makeReport("1", "2026-01-01T00:00:00Z");
    (api.getOne as ReturnType<typeof vi.fn>).mockResolvedValue(report);

    const result = await reportsService.getReportById("1");

    expect(api.getOne).toHaveBeenCalledWith("/api/reports/1", undefined);
    expect(result).toEqual(report);
  });
});

describe("reportsService.updateReportStatus", () => {
  beforeEach(() => vi.clearAllMocks());

  it("PATCHes with a JSON:API envelope carrying only the status attribute", async () => {
    await reportsService.updateReportStatus("1", ReportStatus.Actioned);

    expect(api.patch).toHaveBeenCalledWith(
      "/api/reports/1",
      {
        data: {
          type: "reports",
          id: "1",
          attributes: { status: ReportStatus.Actioned },
        },
      },
      undefined,
    );
  });
});
