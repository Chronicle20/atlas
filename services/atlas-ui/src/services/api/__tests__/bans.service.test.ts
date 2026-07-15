import { describe, expect, it, vi, beforeEach } from "vitest";

const getMock = vi.fn();
vi.mock("@/lib/api/client", () => ({
  api: {
    get: (...args: unknown[]) => getMock(...args),
    getList: vi.fn(),
    getOne: vi.fn(),
    post: vi.fn(),
    delete: vi.fn(),
  },
}));

import { bansService } from "@/services/api/bans.service";
import { BanType } from "@/types/models/ban";
import type { Ban } from "@/types/models/ban";

function makeBan(id: string): Ban {
  return {
    id,
    attributes: {
      banType: BanType.IP,
      value: "1.2.3.4",
      reason: "test",
      reasonCode: 0,
      permanent: true,
      expiresAt: "0001-01-01T00:00:00Z",
      issuedBy: "admin",
    },
  };
}

describe("bansService.getBansPage", () => {
  beforeEach(() => vi.clearAllMocks());

  it("appends page params and the type filter, sorts by id descending", async () => {
    getMock.mockResolvedValue({
      data: [makeBan("1"), makeBan("2")],
      meta: { total: 2, page: { number: 1, size: 50, last: 1 } },
    });

    const result = await bansService.getBansPage({ number: 1, size: 50 }, { type: BanType.Account });

    expect(result.data.map((b) => b.id)).toEqual(["2", "1"]);
    expect(result.meta).toEqual({ total: 2, page: { number: 1, size: 50, last: 1 } });

    const [calledUrl] = getMock.mock.calls[0] as [string];
    const url = new URL(calledUrl, "http://example.test");
    expect(url.searchParams.get("type")).toBe(String(BanType.Account));
    expect(url.searchParams.get("page[number]")).toBe("1");
    expect(url.searchParams.get("page[size]")).toBe("50");
  });
});

describe("bansService.getAllBans", () => {
  beforeEach(() => vi.clearAllMocks());

  it("drains every page", async () => {
    getMock
      .mockResolvedValueOnce({
        data: [makeBan("1")],
        meta: { total: 2, page: { number: 1, size: 1, last: 2 } },
      })
      .mockResolvedValueOnce({
        data: [makeBan("2")],
        meta: { total: 2, page: { number: 2, size: 1, last: 2 } },
      });

    const result = await bansService.getAllBans();

    expect(result.map((b) => b.id).sort()).toEqual(["1", "2"]);
    expect(getMock).toHaveBeenCalledTimes(2);
  });
});
