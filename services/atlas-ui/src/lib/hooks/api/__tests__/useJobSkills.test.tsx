import { renderHook, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import type { Tenant } from "@/services/api/tenants.service";
import { useJobSkills } from "../useJobSkills";

const fakeTenant = {
  id: "t1",
  attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
} as unknown as Tenant;

const getSkillsByJobMock = vi.fn();
vi.mock("@/services/api/jobs.service", () => ({
  jobsService: { getSkillsByJobId: (...args: unknown[]) => getSkillsByJobMock(...args) },
}));

function wrapper({ children }: { children: ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

describe("useJobSkills", () => {
  beforeEach(() => vi.clearAllMocks());

  it("returns the skill list for a job", async () => {
    getSkillsByJobMock.mockResolvedValue([1101000, 1101001]);
    const { result } = renderHook(() => useJobSkills(fakeTenant, 110), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual([1101000, 1101001]);
  });
});
