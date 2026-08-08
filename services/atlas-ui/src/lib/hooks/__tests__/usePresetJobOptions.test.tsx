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
  return {
    isSuccess: true,
    isPending: false,
    isError: false,
    data: { jobs },
  };
}

beforeEach(() => useJobAvailabilityMock.mockReset());

describe("usePresetJobOptions", () => {
  it("v48-style availability: wire id 500 is Gm, not Pirate", () => {
    // Pre-v0.61: wire id 500 is Gm (Set.Name's exact casing) and there is no
    // released Pirate identity at all.
    useJobAvailabilityMock.mockReturnValue(success([{ id: 500, name: "Gm" }]));
    const { result } = renderHook(() => usePresetJobOptions());
    expect(result.current.options).toEqual([{ id: 500, name: "Gm" }]);
    expect(result.current.options.map((j) => j.name)).not.toContain("Pirate");
    expect(result.current.isPending).toBe(false);
    expect(result.current.isError).toBe(false);
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
    const names = result.current.options.map((j) => j.name);
    expect(names).not.toContain("Pirate");
    expect(names).toContain("Warrior");
  });

  it("returns an empty options list plus isPending while availability is unknown, never a static fallback", () => {
    useJobAvailabilityMock.mockReturnValue({
      isSuccess: false,
      isPending: true,
      isError: false,
      data: undefined,
    });
    const { result } = renderHook(() => usePresetJobOptions());
    // Pending means "unknown", not "empty" — but offering the static v83
    // list here would show wrong names on a non-v83 tenant (task-202
    // FR-4.2), so the picker gets an empty list plus explicit isPending for
    // the caller to render its own pending affordance, instead of a
    // plausible-looking wrong answer.
    expect(result.current.options).toEqual([]);
    expect(result.current.isPending).toBe(true);
  });

  it("returns an empty options list plus isError when availability fails", () => {
    useJobAvailabilityMock.mockReturnValue({
      isSuccess: false,
      isPending: false,
      isError: true,
      data: undefined,
    });
    const { result } = renderHook(() => usePresetJobOptions());
    expect(result.current.options).toEqual([]);
    expect(result.current.isError).toBe(true);
  });
});
