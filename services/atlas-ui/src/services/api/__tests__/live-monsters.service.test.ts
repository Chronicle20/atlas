import { beforeEach, describe, expect, it, vi } from "vitest";

const getListMock = vi.fn();
vi.mock("@/lib/api/client", () => ({
  api: {
    get: vi.fn(),
    getList: (...args: unknown[]) => getListMock(...args),
    getOne: vi.fn(),
  },
}));

import { liveMonstersService } from "@/services/api/live-monsters.service";

describe("liveMonstersService.getMonsters", () => {
  beforeEach(() => {
    getListMock.mockReset();
    getListMock.mockResolvedValue([]);
  });

  it("requests page[size]=250 to avoid the backend's silent-truncation default", async () => {
    const instanceId = "00000000-0000-0000-0000-000000000000";

    await liveMonstersService.getMonsters(1, 2, 910340000, instanceId);

    const url = getListMock.mock.calls[0]?.[0] as string;
    expect(url).toContain(
      `/api/worlds/1/channels/2/maps/910340000/instances/${instanceId}/monsters`,
    );
    expect(url).toContain("page%5Bsize%5D=250");
  });
});
