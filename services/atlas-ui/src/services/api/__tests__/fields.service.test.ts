import { beforeEach, describe, expect, it, vi } from "vitest";

const getListMock = vi.fn();
vi.mock("@/lib/api/client", () => ({
  api: {
    get: vi.fn(),
    getList: (...args: unknown[]) => getListMock(...args),
    getOne: vi.fn(),
  },
}));

import { fieldsService } from "@/services/api/fields.service";

describe("fieldsService", () => {
  beforeEach(() => {
    getListMock.mockReset();
    getListMock.mockResolvedValue([]);
  });

  describe("getFields", () => {
    it("requests page[size]=250 to avoid the backend's silent-truncation default", async () => {
      await fieldsService.getFields({});

      const url = getListMock.mock.calls[0]?.[0] as string;
      expect(url).toContain("page%5Bsize%5D=250");
    });

    it("appends filter[worldId]/filter[channelId]/filter[mapId] only when defined", async () => {
      await fieldsService.getFields({
        worldId: 1,
        channelId: 2,
        mapId: 910340000,
      });

      const url = getListMock.mock.calls[0]?.[0] as string;
      const params = new URLSearchParams(url.split("?")[1]);
      expect(params.get("page[size]")).toBe("250");
      expect(params.get("filter[worldId]")).toBe("1");
      expect(params.get("filter[channelId]")).toBe("2");
      expect(params.get("filter[mapId]")).toBe("910340000");
    });

    it("omits filter params entirely when the corresponding filter is undefined", async () => {
      await fieldsService.getFields({});

      const url = getListMock.mock.calls[0]?.[0] as string;
      expect(url).not.toContain("filter[worldId]");
      expect(url).not.toContain("filter[channelId]");
      expect(url).not.toContain("filter[mapId]");
    });
  });

  describe("getFieldCharacters", () => {
    it("requests page[size]=250 to avoid the backend's silent-truncation default", async () => {
      const instanceId = "00000000-0000-0000-0000-000000000000";

      await fieldsService.getFieldCharacters(1, 2, 910340000, instanceId);

      const url = getListMock.mock.calls[0]?.[0] as string;
      expect(url).toContain(
        `/api/worlds/1/channels/2/maps/910340000/instances/${instanceId}/characters`,
      );
      expect(url).toContain("page%5Bsize%5D=250");
    });
  });
});
