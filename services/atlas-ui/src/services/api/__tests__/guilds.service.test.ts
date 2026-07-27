import { describe, expect, it, vi, beforeEach } from "vitest";

const getMock = vi.fn();
const getListMock = vi.fn();
vi.mock("@/lib/api/client", () => ({
  api: {
    get: (...args: unknown[]) => getMock(...args),
    getList: (...args: unknown[]) => getListMock(...args),
    getOne: vi.fn(),
    delete: vi.fn(),
  },
}));

import { guildsService } from "@/services/api/guilds.service";
import type { Guild, GuildMember } from "@/types/models/guild";

function makeGuild(
  id: string,
  name: string,
  points: number,
  worldId = 0,
): Guild {
  const member: GuildMember = {
    characterId: 1,
    name: "Leader",
    jobId: 100,
    level: 50,
    title: 0,
    online: true,
    allianceTitle: 0,
  };
  return {
    id,
    attributes: {
      worldId,
      name,
      notice: "",
      points,
      capacity: 100,
      logo: 0,
      logoColor: 0,
      logoBackground: 0,
      logoBackgroundColor: 0,
      leaderId: 1,
      members: [member],
      titles: [],
    },
  };
}

describe("guildsService.getPage", () => {
  beforeEach(() => vi.clearAllMocks());

  it("appends page params and returns data + meta, sorted by points desc", async () => {
    getMock.mockResolvedValue({
      data: [makeGuild("2", "Bravo", 500), makeGuild("1", "Alpha", 900)],
      meta: { total: 5, page: { number: 2, size: 2, last: 3 } },
    });

    const result = await guildsService.getPage({ number: 2, size: 2 });

    expect(result.meta).toEqual({
      total: 5,
      page: { number: 2, size: 2, last: 3 },
    });
    // Sorted by points descending within the page.
    expect(result.data.map((g) => g.attributes.name)).toEqual([
      "Alpha",
      "Bravo",
    ]);

    const [calledUrl] = getMock.mock.calls[0] as [string];
    const params = new URL(calledUrl, "http://example.test").searchParams;
    expect(params.get("page[number]")).toBe("2");
    expect(params.get("page[size]")).toBe("2");
  });
});

describe("guildsService.search", () => {
  beforeEach(() => vi.clearAllMocks());

  it("hits the server with filter[name] and page params (not a client-side filter over a full fetch)", async () => {
    getMock.mockResolvedValue({
      data: [makeGuild("1", "Alpha Guild", 900)],
      meta: { total: 1, page: { number: 1, size: 50, last: 1 } },
    });

    const result = await guildsService.search("alpha", { number: 1, size: 50 });

    expect(result.data.map((g) => g.attributes.name)).toEqual(["Alpha Guild"]);
    expect(result.meta).toEqual({
      total: 1,
      page: { number: 1, size: 50, last: 1 },
    });

    // Exactly one server request for the search — no fetch-all-then-filter.
    expect(getMock).toHaveBeenCalledTimes(1);
    const [calledUrl] = getMock.mock.calls[0] as [string];
    const params = new URL(calledUrl, "http://example.test").searchParams;
    expect(params.get("filter[name]")).toBe("alpha");
    expect(params.get("page[number]")).toBe("1");
    expect(params.get("page[size]")).toBe("50");
  });
});

describe("guildsService.getByMemberId", () => {
  beforeEach(() => vi.clearAllMocks());

  it("uses filter[members.id]", async () => {
    getListMock.mockResolvedValue([makeGuild("1", "Alpha", 900)]);

    const result = await guildsService.getByMemberId("42");

    expect(result).toHaveLength(1);
    const [calledUrl] = getListMock.mock.calls[0] as [string];
    expect(calledUrl).toContain("filter%5Bmembers.id%5D=42");
  });
});

describe("guildsService.getByWorld", () => {
  beforeEach(() => vi.clearAllMocks());

  it("drains every page (no filter[worldId] route exists) and filters in-memory", async () => {
    getMock
      .mockResolvedValueOnce({
        data: [makeGuild("1", "OnWorld", 100, 1)],
        meta: { total: 2, page: { number: 1, size: 1, last: 2 } },
      })
      .mockResolvedValueOnce({
        data: [makeGuild("2", "OtherWorld", 900, 2)],
        meta: { total: 2, page: { number: 2, size: 1, last: 2 } },
      });

    const result = await guildsService.getByWorld(1);

    expect(getMock).toHaveBeenCalledTimes(2);
    expect(result.map((g) => g.attributes.name)).toEqual(["OnWorld"]);
  });
});

describe("guildsService.getWithSpace", () => {
  beforeEach(() => vi.clearAllMocks());

  it("drains the collection and filters guilds with open capacity", async () => {
    const full = makeGuild("1", "Full", 100, 0);
    full.attributes.members = Array.from(
      { length: full.attributes.capacity },
      (_, i) => ({
        characterId: i,
        name: `M${i}`,
        jobId: 0,
        level: 1,
        title: 0,
        online: false,
        allianceTitle: 0,
      }),
    );
    const open = makeGuild("2", "Open", 50, 0);

    getMock.mockResolvedValue({ data: [full, open], meta: null });

    const result = await guildsService.getWithSpace();

    expect(result.map((g) => g.attributes.name)).toEqual(["Open"]);
  });
});

describe("guildsService.getRankings", () => {
  beforeEach(() => vi.clearAllMocks());

  it("drains, sorts by points descending, and limits", async () => {
    getMock.mockResolvedValue({
      data: [
        makeGuild("1", "Low", 10),
        makeGuild("2", "High", 999),
        makeGuild("3", "Mid", 500),
      ],
      meta: null,
    });

    const result = await guildsService.getRankings(undefined, 2);

    expect(result.map((g) => g.attributes.name)).toEqual(["High", "Mid"]);
  });
});
