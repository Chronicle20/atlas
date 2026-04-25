import { renderHook, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import type { Tenant } from "@/services/api/tenants.service";
import { useCharacterSkills } from "../useCharacterSkills";

const fakeTenant = {
  id: "t1",
  attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
} as unknown as Tenant;

const getByCharacterIdMock = vi.fn();
vi.mock("@/services/api/characterSkills.service", () => ({
  characterSkillsService: {
    getByCharacterId: (...a: unknown[]) => getByCharacterIdMock(...a),
  },
}));

function wrapper({ children }: { children: ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

describe("useCharacterSkills", () => {
  beforeEach(() => vi.clearAllMocks());

  it("returns character skill list", async () => {
    getByCharacterIdMock.mockResolvedValue([
      {
        id: "1001004",
        level: 5,
        masterLevel: 20,
        expiration: "0001-01-01T00:00:00Z",
        cooldownExpiresAt: "0001-01-01T00:00:00Z",
      },
    ]);
    const { result } = renderHook(() => useCharacterSkills(fakeTenant, "42"), {
      wrapper,
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.[0].level).toBe(5);
  });
});
