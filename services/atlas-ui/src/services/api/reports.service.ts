import { api } from "@/lib/api/client";
import { type ServiceOptions, type QueryOptions } from "@/lib/api/query-params";
import type { Report, ReportStatus } from "@/types/models/report";

const BASE_PATH = "/api/reports";

export interface ReportQueryOptions extends QueryOptions {
  status?: ReportStatus;
}

function sortReports(reports: Report[]): Report[] {
  return reports.sort(
    (a, b) =>
      new Date(b.attributes.createdAt).getTime() -
      new Date(a.attributes.createdAt).getTime(),
  );
}

export const reportsService = {
  async getAllReports(options?: ReportQueryOptions): Promise<Report[]> {
    let url = BASE_PATH;
    if (options?.status !== undefined) {
      url += `?status=${options.status}`;
    }
    const reports = await api.getList<Report>(url, options);
    return sortReports(reports);
  },

  async getReportById(id: string, options?: ServiceOptions): Promise<Report> {
    return api.getOne<Report>(`${BASE_PATH}/${id}`, options);
  },

  async updateReportStatus(
    id: string,
    status: ReportStatus,
    options?: ServiceOptions,
  ): Promise<void> {
    // JSON:API envelope — bare attribute bodies are rejected with 400 by
    // RegisterInputHandler endpoints.
    await api.patch<void>(
      `${BASE_PATH}/${id}`,
      { data: { type: "reports", id, attributes: { status } } },
      options,
    );
  },
};
