import { describe, expect, it, vi, beforeEach } from "vitest";

const getMock = vi.fn();
vi.mock("@/lib/api/client", () => ({
  api: {
    get: (...args: unknown[]) => getMock(...args),
    getList: vi.fn(),
    getOne: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
  },
}));

import { charactersService } from "@/services/api/characters.service";
import type { Character } from "@/types/models/character";

function makeCharacter(id: string): Character {
  return {
    id,
    attributes: {
      accountId: 1,
      worldId: 0,
      name: `Char${id}`,
      level: 1,
      experience: 0,
      gachaponExperience: 0,
      strength: 4,
      dexterity: 4,
      intelligence: 4,
      luck: 4,
      hp: 50,
      maxHp: 50,
      mp: 5,
      maxMp: 5,
      meso: 0,
      hpMpUsed: 0,
      jobId: 0,
      skinColor: 0,
      gender: 0,
      fame: 0,
      hair: 30000,
      face: 20000,
      ap: 0,
      sp: "0",
      spawnPoint: 0,
      gm: 0,
      x: 0,
      y: 0,
      stance: 0,
    },
  };
}

describe("charactersService.getPage", () => {
  beforeEach(() => vi.clearAllMocks());

  it("appends page params and returns the envelope's data + meta", async () => {
    getMock.mockResolvedValue({
      data: [makeCharacter("1")],
      meta: { total: 3, page: { number: 1, size: 2, last: 2 } },
    });

    const result = await charactersService.getPage({ number: 1, size: 2 });

    expect(result.data).toEqual([makeCharacter("1")]);
    expect(result.meta).toEqual({ total: 3, page: { number: 1, size: 2, last: 2 } });

    const [calledUrl] = getMock.mock.calls[0] as [string];
    const params = new URL(calledUrl, "http://example.test").searchParams;
    expect(params.get("page[number]")).toBe("1");
    expect(params.get("page[size]")).toBe("2");
  });
});

describe("charactersService.getAll", () => {
  beforeEach(() => vi.clearAllMocks());

  it("drains every page into a single array", async () => {
    getMock
      .mockResolvedValueOnce({
        data: [makeCharacter("1")],
        meta: { total: 2, page: { number: 1, size: 1, last: 2 } },
      })
      .mockResolvedValueOnce({
        data: [makeCharacter("2")],
        meta: { total: 2, page: { number: 2, size: 1, last: 2 } },
      });

    const result = await charactersService.getAll();

    expect(result.map((c) => c.id)).toEqual(["1", "2"]);
    expect(getMock).toHaveBeenCalledTimes(2);
  });

  it("treats a no-envelope response as the whole collection", async () => {
    getMock.mockResolvedValue({ data: [makeCharacter("1"), makeCharacter("2")] });

    const result = await charactersService.getAll();

    expect(result.map((c) => c.id)).toEqual(["1", "2"]);
    expect(getMock).toHaveBeenCalledTimes(1);
  });
});
