import { renderHook, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import type { Tenant } from "@/services/api/tenants.service";
import { useJobs, jobsKeys } from "../useJobs";

const fakeTenant = {
  id: "t1",
  attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
} as unknown as Tenant;

const getJobsMock = vi.fn();
vi.mock("@/services/api/jobs.service", () => ({
  jobsService: {
    getJobs: (...args: unknown[]) => getJobsMock(...args),
  },
}));

function wrapper({ children }: { children: ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

describe("useJobs", () => {
  beforeEach(() => vi.clearAllMocks());

  it("is disabled when there is no active tenant", () => {
    const { result } = renderHook(() => useJobs(undefined), { wrapper });

    expect(result.current.fetchStatus).toBe("idle");
    expect(result.current.isPending).toBe(true);
    expect(getJobsMock).not.toHaveBeenCalled();
  });

  it("is disabled when the tenant is null", () => {
    const { result } = renderHook(() => useJobs(null), { wrapper });

    expect(result.current.fetchStatus).toBe("idle");
    expect(getJobsMock).not.toHaveBeenCalled();
  });

  it("fetches and returns the tenant's job set once a tenant is active", async () => {
    const jobsResult = {
      jobs: [{ id: "100", type: "jobs", attributes: { skills: [1001000] } }],
      skillsById: new Map(),
    };
    getJobsMock.mockResolvedValue(jobsResult);

    const { result } = renderHook(() => useJobs(fakeTenant), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(jobsResult);
    expect(getJobsMock).toHaveBeenCalledTimes(1);
  });

  it("surfaces the error state when the fetch fails", async () => {
    getJobsMock.mockRejectedValue(new Error("boom"));

    const { result } = renderHook(() => useJobs(fakeTenant), { wrapper });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBeInstanceOf(Error);
  });

  it("keys the query hierarchically with a no-tenant fallback", () => {
    // Regression coverage for the missing-fallback finding: every sibling
    // tenant-scoped key factory falls back to "no-tenant" so an undefined
    // tenant id can never collide with (or be confused for) a real one.
    expect(jobsKeys.list(undefined)).toEqual(["jobs", "list", "no-tenant"]);
    expect(jobsKeys.list("t1")).toEqual(["jobs", "list", "t1"]);
    expect(jobsKeys.lists()).toEqual(["jobs", "list"]);
    expect(jobsKeys.all).toEqual(["jobs"]);
  });
});
