import { describe, it, expect, vi, beforeEach } from "vitest";

const getMock = vi.fn();
vi.mock("@/lib/api/client", () => ({
  api: { get: (...args: unknown[]) => getMock(...args) },
}));

import { fetchPaged, fetchAll } from "@/services/api/pagination";

interface Widget {
  id: string;
  name: string;
}

describe("fetchPaged", () => {
  beforeEach(() => vi.clearAllMocks());

  it("returns data and meta for a single page and appends page params", async () => {
    getMock.mockResolvedValue({
      data: [{ id: "1", name: "a" }],
      meta: { total: 1, page: { number: 1, size: 250, last: 1 } },
    });

    const result = await fetchPaged<Widget>("/api/data/widgets", { number: 1, size: 250 });

    expect(result.data).toEqual([{ id: "1", name: "a" }]);
    expect(result.meta).toEqual({ total: 1, page: { number: 1, size: 250, last: 1 } });

    expect(getMock).toHaveBeenCalledTimes(1);
    const [calledUrl] = getMock.mock.calls[0] as [string];
    const params = new URL(calledUrl, "http://example.test").searchParams;
    expect(params.get("page[number]")).toBe("1");
    expect(params.get("page[size]")).toBe("250");
  });

  it("preserves existing query params on the url", async () => {
    getMock.mockResolvedValue({ data: [], meta: { total: 0, page: { number: 1, size: 50, last: 1 } } });

    await fetchPaged<Widget>("/api/data/widgets?filter[name]=x", { number: 2, size: 50 });

    const [calledUrl] = getMock.mock.calls[0] as [string];
    const url = new URL(calledUrl, "http://example.test");
    expect(url.pathname).toBe("/api/data/widgets");
    expect(url.searchParams.get("filter[name]")).toBe("x");
    expect(url.searchParams.get("page[number]")).toBe("2");
    expect(url.searchParams.get("page[size]")).toBe("50");
  });

  it("passes options through to api.get and defaults meta to null when absent", async () => {
    getMock.mockResolvedValue({ data: [{ id: "1", name: "a" }] });

    const options = { maxRetries: 0 };
    const result = await fetchPaged<Widget>("/api/data/widgets", { number: 1, size: 10 }, options);

    expect(result.meta).toBeNull();
    expect(result.data).toEqual([{ id: "1", name: "a" }]);
    const [, calledOptions] = getMock.mock.calls[0] as [string, unknown];
    expect(calledOptions).toBe(options);
  });
});

describe("fetchAll", () => {
  beforeEach(() => vi.clearAllMocks());

  it("drains multiple pages until the last page", async () => {
    getMock
      .mockResolvedValueOnce({
        data: [{ id: "1", name: "a" }],
        meta: { total: 3, page: { number: 1, size: 1, last: 3 } },
      })
      .mockResolvedValueOnce({
        data: [{ id: "2", name: "b" }],
        meta: { total: 3, page: { number: 2, size: 1, last: 3 } },
      })
      .mockResolvedValueOnce({
        data: [{ id: "3", name: "c" }],
        meta: { total: 3, page: { number: 3, size: 1, last: 3 } },
      });

    const all = await fetchAll<Widget>("/api/data/widgets", 1);

    expect(all).toEqual([
      { id: "1", name: "a" },
      { id: "2", name: "b" },
      { id: "3", name: "c" },
    ]);
    expect(getMock).toHaveBeenCalledTimes(3);
  });

  it("treats a no-envelope (meta null) response as the whole collection and stops after page 1", async () => {
    getMock.mockResolvedValue({ data: [{ id: "1", name: "a" }, { id: "2", name: "b" }] });

    const all = await fetchAll<Widget>("/api/data/widgets");

    expect(all).toEqual([{ id: "1", name: "a" }, { id: "2", name: "b" }]);
    expect(getMock).toHaveBeenCalledTimes(1);
  });

  it("stops early when a page comes back empty before reaching meta.page.last", async () => {
    getMock
      .mockResolvedValueOnce({
        data: [{ id: "1", name: "a" }],
        meta: { total: 3, page: { number: 1, size: 1, last: 3 } },
      })
      .mockResolvedValueOnce({
        data: [],
        meta: { total: 3, page: { number: 2, size: 1, last: 3 } },
      });

    const all = await fetchAll<Widget>("/api/data/widgets", 1);

    expect(all).toEqual([{ id: "1", name: "a" }]);
    expect(getMock).toHaveBeenCalledTimes(2);
  });

  it("uses default drain size of 250 when not specified", async () => {
    getMock.mockResolvedValue({
      data: [],
      meta: { total: 0, page: { number: 1, size: 250, last: 1 } },
    });

    await fetchAll<Widget>("/api/data/widgets");

    const [calledUrl] = getMock.mock.calls[0] as [string];
    const params = new URL(calledUrl, "http://example.test").searchParams;
    expect(params.get("page[size]")).toBe("250");
  });

  it("preserves existing query params across all drained pages", async () => {
    getMock
      .mockResolvedValueOnce({
        data: [{ id: "1", name: "a" }],
        meta: { total: 2, page: { number: 1, size: 1, last: 2 } },
      })
      .mockResolvedValueOnce({
        data: [{ id: "2", name: "b" }],
        meta: { total: 2, page: { number: 2, size: 1, last: 2 } },
      });

    await fetchAll<Widget>("/api/data/widgets?filter[name]=x", 1);

    for (const call of getMock.mock.calls) {
      const url = new URL(call[0] as string, "http://example.test");
      expect(url.searchParams.get("filter[name]")).toBe("x");
    }
  });
});
