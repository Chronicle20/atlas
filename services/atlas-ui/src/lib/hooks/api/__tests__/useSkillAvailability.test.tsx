import { renderHook, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import type { Tenant } from "@/services/api/tenants.service";
import {
  useSkillAvailability,
  skillAvailabilityKeys,
} from "../useSkillAvailability";

const fakeTenant = {
  id: "t1",
  attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
} as unknown as Tenant;

const getSkillAvailabilityMock = vi.fn();
vi.mock("@/services/api/availability.service", () => ({
  availabilityService: {
    getSkillAvailability: (...args: unknown[]) =>
      getSkillAvailabilityMock(...args),
  },
}));

function wrapper({ children }: { children: ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

describe("useSkillAvailability", () => {
  beforeEach(() => vi.clearAllMocks());

  it("is disabled when there is no active tenant", () => {
    const { result } = renderHook(() => useSkillAvailability(undefined), {
      wrapper,
    });

    expect(result.current.fetchStatus).toBe("idle");
    expect(result.current.isPending).toBe(true);
    expect(getSkillAvailabilityMock).not.toHaveBeenCalled();
  });

  it("is disabled when the tenant is null", () => {
    const { result } = renderHook(() => useSkillAvailability(null), {
      wrapper,
    });

    expect(result.current.fetchStatus).toBe("idle");
    expect(getSkillAvailabilityMock).not.toHaveBeenCalled();
  });

  it("fetches and returns the tenant's released skill identities once a tenant is active", async () => {
    const skills = [{ id: 1121000, name: "Power Strike" }];
    getSkillAvailabilityMock.mockResolvedValue(skills);

    const { result } = renderHook(() => useSkillAvailability(fakeTenant), {
      wrapper,
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual({ skills });
    expect(getSkillAvailabilityMock).toHaveBeenCalledTimes(1);
  });

  it("surfaces the error state when the fetch fails", async () => {
    getSkillAvailabilityMock.mockRejectedValue(new Error("boom"));

    const { result } = renderHook(() => useSkillAvailability(fakeTenant), {
      wrapper,
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBeInstanceOf(Error);
  });

  it("keys the query hierarchically with a no-tenant fallback", () => {
    expect(skillAvailabilityKeys.list(undefined)).toEqual([
      "skill-availability",
      "list",
      "no-tenant",
    ]);
    expect(skillAvailabilityKeys.list("t1")).toEqual([
      "skill-availability",
      "list",
      "t1",
    ]);
    expect(skillAvailabilityKeys.lists()).toEqual([
      "skill-availability",
      "list",
    ]);
    expect(skillAvailabilityKeys.all).toEqual(["skill-availability"]);
  });
});
