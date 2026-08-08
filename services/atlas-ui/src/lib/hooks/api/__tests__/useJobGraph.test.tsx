import { describe, expect, it, vi, beforeEach } from "vitest";
import { renderHook } from "@testing-library/react";

const useJobs = vi.fn();
const useJobAvailability = vi.fn();

vi.mock("@/lib/hooks/api/useJobs", () => ({
  useJobs: (...a: unknown[]) => useJobs(...a),
}));
vi.mock("@/lib/hooks/api/useJobAvailability", () => ({
  useJobAvailability: (...a: unknown[]) => useJobAvailability(...a),
  jobAvailabilityKeys: { list: () => ["job-availability", "list"] },
}));
vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: { id: "t-1" } }),
}));

import { useJobGraph, useJobNameLookup } from "@/lib/hooks/api/useJobGraph";

function q(over: Record<string, unknown>) {
  return {
    isSuccess: false,
    isPending: false,
    isError: false,
    data: undefined,
    ...over,
  };
}

const AVAILABILITY = [
  { id: 0, name: "Beginner", parent: null, identity: 0 },
  { id: 500, name: "Gm", parent: 0, identity: 900 },
  { id: 510, name: "Super Gm", parent: 500, identity: 910 },
];

beforeEach(() => {
  useJobs.mockReset();
  useJobAvailability.mockReset();
});

describe("useJobGraph", () => {
  it("is pending — with an EMPTY graph — while either query is pending", () => {
    useJobAvailability.mockReturnValue(q({ isPending: true }));
    useJobs.mockReturnValue(
      q({ isSuccess: true, data: { jobs: [{ id: "0" }] } }),
    );

    const { result } = renderHook(() => useJobGraph());
    expect(result.current.isPending).toBe(true);
    expect(result.current.isSuccess).toBe(false);
    expect(result.current.graph.size).toBe(0);
  });

  it("is an error when either query errors", () => {
    useJobAvailability.mockReturnValue(q({ isError: true }));
    useJobs.mockReturnValue(q({ isSuccess: true, data: { jobs: [] } }));

    const { result } = renderHook(() => useJobGraph());
    expect(result.current.isError).toBe(true);
    expect(result.current.isSuccess).toBe(false);
  });

  it("intersects availability with the WZ job set once both succeed", () => {
    useJobAvailability.mockReturnValue(
      q({ isSuccess: true, data: { jobs: AVAILABILITY } }),
    );
    useJobs.mockReturnValue(
      q({ isSuccess: true, data: { jobs: [{ id: "0" }, { id: "500" }] } }),
    );

    const { result } = renderHook(() => useJobGraph());
    expect(result.current.isSuccess).toBe(true);
    expect([...result.current.graph.keys()].sort((a, b) => a - b)).toEqual([
      0, 500,
    ]);
    expect(result.current.graph.get(500)?.identity).toBe(900);
  });
});

describe("useJobNameLookup", () => {
  it("resolves version-correct names and falls back to `Job <id>`", () => {
    useJobAvailability.mockReturnValue(
      q({ isSuccess: true, data: { jobs: AVAILABILITY } }),
    );
    useJobs.mockReturnValue(
      q({ isSuccess: true, data: { jobs: [{ id: "500" }] } }),
    );

    const { result } = renderHook(() => useJobNameLookup());
    expect(result.current(500)).toBe("Gm");
    expect(result.current(999)).toBe("Job 999");
  });
});
