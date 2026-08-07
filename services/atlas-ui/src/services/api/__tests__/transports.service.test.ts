import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/lib/api/client";
import { transportsService } from "@/services/api/transports.service";

vi.mock("@/lib/api/client", () => ({
  api: { get: vi.fn(), getOne: vi.fn() },
}));

const mockedGet = vi.mocked(api.get);
const mockedGetOne = vi.mocked(api.getOne);

function pagedDocument<T>(data: T[]) {
  return {
    data,
    meta: { total: data.length, page: { number: 1, size: 250, last: 1 } },
  };
}

describe("transportsService", () => {
  beforeEach(() => {
    mockedGet.mockReset();
    mockedGetOne.mockReset();
  });

  it("drains the scheduled route list without asking for the schedule", async () => {
    mockedGet.mockResolvedValueOnce(
      pagedDocument([{ id: "r1", attributes: {} }]),
    );

    const routes = await transportsService.getScheduledRoutes();

    expect(routes).toHaveLength(1);
    const url = mockedGet.mock.calls[0]?.[0] as string;
    expect(url).toContain("/api/transports/routes");
    expect(url).not.toContain("include=schedule");
  });

  it("reads one route by id without dragging its schedule along", async () => {
    // The breadcrumb resolver only needs the name; `include=schedule` would
    // attach ~96 trip resources to every crumb resolution.
    mockedGetOne.mockResolvedValueOnce({
      id: "r1",
      attributes: { name: "boat-orbis-ellinia" },
    });

    const route = await transportsService.getScheduledRouteById("r1");

    expect(route.attributes.name).toBe("boat-orbis-ellinia");
    const url = mockedGetOne.mock.calls[0]?.[0] as string;
    expect(url).toBe("/api/transports/routes/r1");
    expect(url).not.toContain("include=schedule");
  });

  it("asks for the compound document on the detail read and normalises included trips", async () => {
    mockedGet.mockResolvedValueOnce({
      data: { id: "r1", type: "routes", attributes: { name: "Orbis" } },
      included: [
        {
          id: "t1",
          type: "trip-schedule",
          attributes: { boardingOpen: "2023-01-01T08:00:00Z" },
        },
        { id: "x1", type: "something-else", attributes: {} },
      ],
    });

    const detail = await transportsService.getScheduledRoute("r1");

    expect(mockedGet.mock.calls[0]?.[0]).toBe(
      "/api/transports/routes/r1?include=schedule",
    );
    expect(detail.route.id).toBe("r1");
    expect(detail.schedule).toEqual([
      { id: "t1", attributes: { boardingOpen: "2023-01-01T08:00:00Z" } },
    ]);
  });

  it("returns an empty schedule when the document carries no included array", async () => {
    mockedGet.mockResolvedValueOnce({ data: { id: "r1", attributes: {} } });

    const detail = await transportsService.getScheduledRoute("r1");

    expect(detail.schedule).toEqual([]);
  });

  it("reads instance statuses from the per-route status endpoint", async () => {
    mockedGet.mockResolvedValueOnce(pagedDocument([]));

    await transportsService.getInstanceStatuses("ir1");

    expect(mockedGet.mock.calls[0]?.[0]).toContain(
      "/api/transports/instance-routes/ir1/status",
    );
  });

  it("reads vessels from the tenant configuration endpoint", async () => {
    mockedGet.mockResolvedValueOnce(pagedDocument([]));

    await transportsService.getVessels("tenant-1");

    expect(mockedGet.mock.calls[0]?.[0]).toContain(
      "/api/tenants/tenant-1/configurations/vessels",
    );
  });
});
