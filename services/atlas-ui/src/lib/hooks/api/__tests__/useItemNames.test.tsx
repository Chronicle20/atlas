import { renderHook, waitFor } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";

const getItemStringMock = vi.fn();
vi.mock("@/services/api/item-strings.service", () => ({
  itemStringsService: {
    getItemString: (...a: unknown[]) => getItemStringMock(...a),
  },
}));

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: {
      id: "t1",
      attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
    },
  }),
}));

import { useItemNames } from "../useItemNames";
import { itemStringKeys } from "../useItemStrings";

function wrapper(client: QueryClient) {
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
}

describe("useItemNames", () => {
  it("resolves names per id and keys the shared useItemName cache", async () => {
    getItemStringMock.mockReset();
    getItemStringMock.mockImplementation((id: string) =>
      Promise.resolve({ attributes: { name: `Item ${id}` } }),
    );
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const { result } = renderHook(() => useItemNames([20000, 30030]), {
      wrapper: wrapper(client),
    });
    await waitFor(() =>
      expect(result.current).toEqual({
        20000: "Item 20000",
        30030: "Item 30030",
      }),
    );
    // cache entries share useItemName's key shape → lookups merge across UI
    expect(client.getQueryData(itemStringKeys.byId("20000"))).toBe(
      "Item 20000",
    );
  });

  it("returns undefined for ids whose lookup fails", async () => {
    getItemStringMock.mockReset();
    getItemStringMock.mockRejectedValue(new Error("404"));
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const { result } = renderHook(() => useItemNames([99999]), {
      wrapper: wrapper(client),
    });
    await waitFor(() => expect(getItemStringMock).toHaveBeenCalled());
    expect(result.current[99999]).toBeUndefined();
  });
});
