import { describe, it, expect, vi, beforeEach } from "vitest";
import type { ReactNode } from "react";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useReseedTemplate, templateKeys } from "@/lib/hooks/api/useTemplates";
import { templatesService } from "@/services/api/templates.service";

vi.mock("@/services/api/templates.service", () => ({
  templatesService: {
    reseed: vi.fn(),
  },
}));

function makeWrapper(qc: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
  };
}

describe("useReseedTemplate", () => {
  beforeEach(() => vi.clearAllMocks());

  it("calls the service and invalidates the detail and list queries", async () => {
    vi.mocked(templatesService.reseed).mockResolvedValue(undefined);
    const qc = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const invalidate = vi.spyOn(qc, "invalidateQueries");

    const { result } = renderHook(() => useReseedTemplate(), {
      wrapper: makeWrapper(qc),
    });

    await result.current.mutateAsync({ id: "abc-123" });

    expect(templatesService.reseed).toHaveBeenCalledWith("abc-123");
    await waitFor(() => {
      expect(invalidate).toHaveBeenCalledWith({
        queryKey: templateKeys.detail("abc-123"),
      });
      expect(invalidate).toHaveBeenCalledWith({
        queryKey: templateKeys.lists(),
      });
    });
  });

  it("surfaces a rejection and invalidates nothing", async () => {
    vi.mocked(templatesService.reseed).mockRejectedValue(
      new Error("no shipped template"),
    );
    const qc = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });
    const invalidate = vi.spyOn(qc, "invalidateQueries");

    const { result } = renderHook(() => useReseedTemplate(), {
      wrapper: makeWrapper(qc),
    });

    await expect(result.current.mutateAsync({ id: "abc-123" })).rejects.toThrow(
      "no shipped template",
    );
    expect(invalidate).not.toHaveBeenCalled();
  });
});
