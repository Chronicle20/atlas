import { renderHook } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";

const useJobAvailabilityMock = vi.fn();
vi.mock("@/lib/hooks/api/useJobAvailability", () => ({
  useJobAvailability: (...a: unknown[]) => useJobAvailabilityMock(...a),
}));

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: { id: "t1" } }),
}));

import { usePresetJobOptions } from "@/lib/hooks/usePresetJobOptions";

/** Shape a useJobAvailability success result from {id,name} rows. */
function success(jobs: Array<{ id: number; name: string }>) {
  return { isSuccess: true, data: { jobs } };
}

beforeEach(() => useJobAvailabilityMock.mockReset());

describe("usePresetJobOptions", () => {
  it("v48-style availability: wire id 500 is Gm, not Pirate", () => {
    // Pre-v0.61: wire id 500 is Gm (Set.Name's exact casing) and there is no
    // released Pirate identity at all.
    useJobAvailabilityMock.mockReturnValue(success([{ id: 500, name: "Gm" }]));
    const { result } = renderHook(() => usePresetJobOptions());
    expect(result.current).toEqual([{ id: 500, name: "Gm" }]);
    expect(result.current.map((j) => j.name)).not.toContain("Pirate");
  });

  it("v61-style availability without Pirate: Pirate is absent from options", () => {
    useJobAvailabilityMock.mockReturnValue(
      success([
        { id: 0, name: "Beginner" },
        { id: 100, name: "Warrior" },
        { id: 200, name: "Magician" },
      ]),
    );
    const { result } = renderHook(() => usePresetJobOptions());
    const names = result.current.map((j) => j.name);
    expect(names).not.toContain("Pirate");
    expect(names).toContain("Warrior");
  });

  it("returns the graceful fallback (full graph) while availability is unknown (pending), never empty", () => {
    useJobAvailabilityMock.mockReturnValue({ isSuccess: false, data: undefined });
    const { result } = renderHook(() => usePresetJobOptions());
    expect(result.current.length).toBeGreaterThan(0);
    // Permissive fallback: the backend validates the chosen id, and a picker
    // must never be blank while availability is still loading.
    const ids = result.current.map((j) => j.id);
    expect(ids).toContain(2100);
  });
});
