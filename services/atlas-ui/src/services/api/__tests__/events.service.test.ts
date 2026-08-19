import { describe, it, expect, vi, beforeEach } from "vitest";
import { eventsService } from "../events.service";
import { api } from "@/lib/api/client";

vi.mock("@/lib/api/client", () => ({
  api: {
    get: vi.fn(),
    getOne: vi.fn(),
    getList: vi.fn(),
    getListDocument: vi.fn(),
    patch: vi.fn(),
  },
}));

describe("eventsService", () => {
  beforeEach(() => vi.clearAllMocks());

  it("sends JSON:API filter params for the occurrence list", async () => {
    (api.get as ReturnType<typeof vi.fn>).mockResolvedValue({ data: [] });

    await eventsService.getOccurrences(
      { type: "CRIMSON_BALROG", state: "ACTIVE", worldId: 1, channelId: 4 },
      { number: 1, size: 25 },
    );

    const calledUrl = (api.get as ReturnType<typeof vi.fn>).mock
      .calls[0]![0] as string;
    const url = new URL(calledUrl, "http://x");
    expect(url.searchParams.get("filter[type]")).toBe("CRIMSON_BALROG");
    expect(url.searchParams.get("filter[state]")).toBe("ACTIVE");
    expect(url.searchParams.get("filter[worldId]")).toBe("1");
    expect(url.searchParams.get("filter[channelId]")).toBe("4");
    expect(url.searchParams.get("page[number]")).toBe("1");
    expect(url.searchParams.get("page[size]")).toBe("25");
  });

  it("PATCHes only enabled when toggling a definition", async () => {
    (api.patch as ReturnType<typeof vi.fn>).mockResolvedValue({
      data: { id: "d1", type: "event-definitions", attributes: {} },
    });

    await eventsService.setDefinitionEnabled("d1", true);

    expect(api.patch).toHaveBeenCalledTimes(1);
    const [url, body] = (api.patch as ReturnType<typeof vi.fn>).mock
      .calls[0]! as [string, { data: { attributes: Record<string, unknown> } }];
    expect(url).toBe("/api/events/definitions/d1");
    expect(Object.keys(body.data.attributes)).toEqual(["enabled"]);
    expect(body.data.attributes.enabled).toBe(true);
  });

  it("surfaces the included transitions on the occurrence detail", async () => {
    (api.get as ReturnType<typeof vi.fn>).mockResolvedValue({
      data: {
        id: "o1",
        type: "event-occurrences",
        attributes: { state: "COMPLETED" },
      },
      included: [
        {
          id: "t1",
          type: "event-occurrence-transitions",
          attributes: { toStage: "ATTACKING" },
        },
      ],
    });

    const o = await eventsService.getOccurrence("o1");

    expect(o.transitions).toHaveLength(1);
    expect(o.transitions[0]!.toStage).toBe("ATTACKING");
  });
});
