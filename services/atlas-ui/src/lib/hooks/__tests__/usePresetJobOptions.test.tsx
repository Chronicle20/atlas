import { renderHook } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";

const useJobsMock = vi.fn();
vi.mock("@/lib/hooks/api/useJobs", () => ({
  useJobs: (...a: unknown[]) => useJobsMock(...a),
}));

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: { id: "t1" } }),
}));

import { usePresetJobOptions } from "@/lib/hooks/usePresetJobOptions";

/** Shape a useJobs success result from a bare list of job ids. */
function success(ids: number[]) {
  return {
    isSuccess: true,
    data: { jobs: ids.map((id) => ({ id: String(id) })) },
  };
}

beforeEach(() => useJobsMock.mockReset());

describe("usePresetJobOptions", () => {
  it("includes Aran/Evan when the tenant's job set has those roots", () => {
    useJobsMock.mockReturnValue(success([0, 100, 200, 2000, 2100, 2110, 2001]));
    const { result } = renderHook(() => usePresetJobOptions());
    const ids = result.current.map((j) => j.id);
    expect(ids).toContain(2100);
    expect(ids).toContain(2001);
    expect(ids).toContain(100);
  });

  it("hides Aran/Evan when the tenant's job set lacks them", () => {
    // A version without Aran/Evan: explorer roots only.
    useJobsMock.mockReturnValue(success([0, 100, 110, 111, 112, 200]));
    const { result } = renderHook(() => usePresetJobOptions());
    const ids = result.current.map((j) => j.id);
    expect(ids).toContain(112);
    expect(ids).not.toContain(2100);
    expect(ids).not.toContain(2001);
  });

  it("returns the full graph while the job set is unknown (pending), never empty", () => {
    useJobsMock.mockReturnValue({ isSuccess: false, data: undefined });
    const { result } = renderHook(() => usePresetJobOptions());
    const ids = result.current.map((j) => j.id);
    expect(result.current.length).toBeGreaterThan(0);
    // Permissive fallback: the backend validates the chosen id, and a picker
    // must never be blank while the tenant's job set is still loading.
    expect(ids).toContain(2100);
  });
});
