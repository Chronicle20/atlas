import { renderHook, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import type { Tenant } from "@/services/api/tenants.service";
import { useJobSkillDefinitions } from "../useJobSkillDefinitions";

const fakeTenant = {
  id: "t1",
  attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
} as unknown as Tenant;

const getSkillByIdMock = vi.fn();
vi.mock("@/services/api/skills.service", () => ({
  skillsService: { getSkillById: (...a: unknown[]) => getSkillByIdMock(...a) },
}));

function wrapper({ children }: { children: ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

describe("useJobSkillDefinitions", () => {
  beforeEach(() => vi.clearAllMocks());

  it("fetches every skill id in parallel and exposes an icon url", async () => {
    getSkillByIdMock.mockImplementation((id: string) =>
      Promise.resolve({ id: Number(id), name: `Skill ${id}`, description: "", action: true, element: "", animationTime: 0, effects: [] }),
    );

    const { result } = renderHook(() => useJobSkillDefinitions(fakeTenant, [1101000, 1101001]), { wrapper });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.definitions).toHaveLength(2);
    expect(result.current.definitions.map((d) => d.id).sort()).toEqual([1101000, 1101001]);
    expect(result.current.definitions[0]?.iconUrl).toContain("/skill/");
  });

  it("returns an empty result for no skill ids and fires no requests", async () => {
    const { result } = renderHook(() => useJobSkillDefinitions(fakeTenant, []), { wrapper });
    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.definitions).toEqual([]);
    expect(getSkillByIdMock).not.toHaveBeenCalled();
  });
});
