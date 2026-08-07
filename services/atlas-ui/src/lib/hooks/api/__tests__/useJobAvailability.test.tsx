import { renderHook, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import type { Tenant } from "@/services/api/tenants.service";
import { useJobAvailability, jobAvailabilityKeys } from "../useJobAvailability";

const fakeTenant = {
  id: "t1",
  attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
} as unknown as Tenant;

const getJobAvailabilityMock = vi.fn();
vi.mock("@/services/api/availability.service", () => ({
  availabilityService: {
    getJobAvailability: (...args: unknown[]) => getJobAvailabilityMock(...args),
  },
}));

function wrapper({ children }: { children: ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

describe("useJobAvailability", () => {
  beforeEach(() => vi.clearAllMocks());

  it("is disabled when there is no active tenant", () => {
    const { result } = renderHook(() => useJobAvailability(undefined), {
      wrapper,
    });

    expect(result.current.fetchStatus).toBe("idle");
    expect(result.current.isPending).toBe(true);
    expect(getJobAvailabilityMock).not.toHaveBeenCalled();
  });

  it("is disabled when the tenant is null", () => {
    const { result } = renderHook(() => useJobAvailability(null), {
      wrapper,
    });

    expect(result.current.fetchStatus).toBe("idle");
    expect(getJobAvailabilityMock).not.toHaveBeenCalled();
  });

  it("fetches and returns the tenant's released job identities once a tenant is active", async () => {
    const jobs = [{ id: 500, name: "Gm" }];
    getJobAvailabilityMock.mockResolvedValue(jobs);

    const { result } = renderHook(() => useJobAvailability(fakeTenant), {
      wrapper,
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual({ jobs });
    expect(getJobAvailabilityMock).toHaveBeenCalledTimes(1);
  });

  it("surfaces the error state when the fetch fails", async () => {
    getJobAvailabilityMock.mockRejectedValue(new Error("boom"));

    const { result } = renderHook(() => useJobAvailability(fakeTenant), {
      wrapper,
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBeInstanceOf(Error);
  });

  it("keys the query hierarchically with a no-tenant fallback", () => {
    expect(jobAvailabilityKeys.list(undefined)).toEqual([
      "job-availability",
      "list",
      "no-tenant",
    ]);
    expect(jobAvailabilityKeys.list("t1")).toEqual([
      "job-availability",
      "list",
      "t1",
    ]);
    expect(jobAvailabilityKeys.lists()).toEqual(["job-availability", "list"]);
    expect(jobAvailabilityKeys.all).toEqual(["job-availability"]);
  });
});
