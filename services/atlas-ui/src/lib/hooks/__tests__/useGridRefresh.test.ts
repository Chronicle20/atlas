import { renderHook, act } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import * as toast from "@/lib/utils/toast";
import { useGridRefresh, type RefreshableQuery } from "@/lib/hooks/useGridRefresh";

vi.mock("@/lib/utils/toast", () => ({
  success: vi.fn(),
  error: vi.fn(),
}));

function makeQuery(overrides: Partial<RefreshableQuery> = {}): RefreshableQuery {
  return {
    isFetching: false,
    refetch: vi.fn().mockResolvedValue({ isError: false, error: null }),
    ...overrides,
  } as unknown as RefreshableQuery;
}

describe("useGridRefresh", () => {
  beforeEach(() => {
    vi.mocked(toast.success).mockReset();
    vi.mocked(toast.error).mockReset();
  });

  it("isRefreshing is true when any query is fetching", () => {
    const { result } = renderHook(() =>
      useGridRefresh([makeQuery({ isFetching: false }), makeQuery({ isFetching: true })]),
    );
    expect(result.current.isRefreshing).toBe(true);
  });

  it("isRefreshing is false when no query is fetching", () => {
    const { result } = renderHook(() =>
      useGridRefresh([makeQuery(), makeQuery()]),
    );
    expect(result.current.isRefreshing).toBe(false);
  });

  it("onRefresh refetches every query and toasts success once", async () => {
    const q1 = makeQuery();
    const q2 = makeQuery();
    const { result } = renderHook(() => useGridRefresh([q1, q2]));

    await act(async () => {
      await result.current.onRefresh();
    });

    expect(q1.refetch).toHaveBeenCalledTimes(1);
    expect(q2.refetch).toHaveBeenCalledTimes(1);
    expect(toast.success).toHaveBeenCalledTimes(1);
    expect(toast.success).toHaveBeenCalledWith("Data refreshed");
    expect(toast.error).not.toHaveBeenCalled();
  });

  it("uses a custom success message when provided", async () => {
    const { result } = renderHook(() =>
      useGridRefresh([makeQuery()], { successMessage: "Maps refreshed" }),
    );
    await act(async () => {
      await result.current.onRefresh();
    });
    expect(toast.success).toHaveBeenCalledWith("Maps refreshed");
  });

  it("toasts error (not success) when a refetch resolves with isError", async () => {
    const boom = new Error("network down");
    const failing = makeQuery({
      refetch: vi.fn().mockResolvedValue({ isError: true, error: boom }),
    } as Partial<RefreshableQuery>);
    const ok = makeQuery();
    const { result } = renderHook(() => useGridRefresh([ok, failing]));

    await act(async () => {
      await result.current.onRefresh();
    });

    expect(toast.error).toHaveBeenCalledTimes(1);
    expect(toast.error).toHaveBeenCalledWith(boom, { context: { action: "refresh" } });
    expect(toast.success).not.toHaveBeenCalled();
  });
});
