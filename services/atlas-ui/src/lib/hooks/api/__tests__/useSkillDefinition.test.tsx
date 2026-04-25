import { renderHook, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import type { Tenant } from "@/services/api/tenants.service";
import { useSkillDefinition } from "../useSkillDefinition";

const fakeTenant = {
  id: "t1",
  attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
} as unknown as Tenant;

const getSkillByIdMock = vi.fn();
vi.mock("@/services/api/skills.service", () => ({
  skillsService: { getSkillById: (...args: unknown[]) => getSkillByIdMock(...args) },
}));

function wrapper({ children }: { children: ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

describe("useSkillDefinition", () => {
  beforeEach(() => vi.clearAllMocks());

  it("fetches and exposes the full skill definition with iconUrl", async () => {
    getSkillByIdMock.mockResolvedValue({
      id: 1101000,
      name: "Iron Body",
      description: "Boost defense.",
      action: false,
      element: "",
      animationTime: 600,
      effects: [],
    });
    const { result } = renderHook(() => useSkillDefinition(fakeTenant, 1101000), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.name).toBe("Iron Body");
    expect(result.current.data?.iconUrl).toContain("/skill/1101000/icon.png");
  });

  it("is disabled when skillId is 0", () => {
    const { result } = renderHook(() => useSkillDefinition(fakeTenant, 0), { wrapper });
    expect(result.current.fetchStatus).toBe("idle");
    expect(getSkillByIdMock).not.toHaveBeenCalled();
  });
});
