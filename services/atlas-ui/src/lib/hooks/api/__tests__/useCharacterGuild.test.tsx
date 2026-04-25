import { renderHook, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import type { Tenant } from "@/services/api/tenants.service";
import { useCharacterGuild } from "../useCharacterGuild";

const fakeTenant = {
  id: "t1",
  attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
} as unknown as Tenant;

const getByMemberIdMock = vi.fn();
vi.mock("@/services/api/guilds.service", () => ({
  guildsService: {
    getByMemberId: (...a: unknown[]) => getByMemberIdMock(...a),
  },
}));

function wrapper({ children }: { children: ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

describe("useCharacterGuild", () => {
  beforeEach(() => vi.clearAllMocks());

  it("returns the first matching guild", async () => {
    getByMemberIdMock.mockResolvedValue([
      { id: "5", attributes: { name: "Heroes" } },
    ]);
    const { result } = renderHook(() => useCharacterGuild(fakeTenant, "42"), {
      wrapper,
    });
    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.guild?.attributes.name).toBe("Heroes");
  });

  it("returns null when no guild matches", async () => {
    getByMemberIdMock.mockResolvedValue([]);
    const { result } = renderHook(() => useCharacterGuild(fakeTenant, "42"), {
      wrapper,
    });
    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.guild).toBeNull();
  });

  it("flattens errors to guild=null", async () => {
    getByMemberIdMock.mockRejectedValue(new Error("network"));
    const { result } = renderHook(() => useCharacterGuild(fakeTenant, "42"), {
      wrapper,
    });
    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.guild).toBeNull();
    expect(result.current.error).not.toBeNull();
  });
});
