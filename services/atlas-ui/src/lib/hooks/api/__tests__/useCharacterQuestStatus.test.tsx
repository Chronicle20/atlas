import { renderHook, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import type { Tenant } from "@/services/api/tenants.service";
import { useCharacterQuestStatus } from "../useCharacterQuestStatus";

const fakeTenant = { id: "t1", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } } as unknown as Tenant;
const getStartedMock = vi.fn();
const getCompletedMock = vi.fn();
vi.mock("@/services/api/quest-status.service", () => ({
  questStatusService: {
    getStartedQuests: (...a: unknown[]) => getStartedMock(...a),
    getCompletedQuests: (...a: unknown[]) => getCompletedMock(...a),
  },
}));

function wrapper({ children }: { children: ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

describe("useCharacterQuestStatus", () => {
  beforeEach(() => vi.clearAllMocks());

  it("issues both fetches in parallel and groups results", async () => {
    getStartedMock.mockResolvedValue([{ id: "100100" }]);
    getCompletedMock.mockResolvedValue([{ id: "100200" }, { id: "100201" }]);
    const { result } = renderHook(() => useCharacterQuestStatus(fakeTenant, "42"), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.started).toHaveLength(1);
    expect(result.current.data?.completed).toHaveLength(2);
    expect(getStartedMock).toHaveBeenCalledTimes(1);
    expect(getCompletedMock).toHaveBeenCalledTimes(1);
  });
});
