import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import {
  usePendingChanges,
  useCancelPendingChange,
  pendingChangeKeys,
} from "../usePendingChanges";
import { pendingChangesService } from "@/services/api/pending-changes.service";
import type { Tenant } from "@/types/models/tenant";

vi.mock("@/services/api/pending-changes.service", () => ({
  pendingChangesService: {
    getByCharacterId: vi.fn().mockResolvedValue([]),
    cancel: vi.fn().mockResolvedValue(undefined),
  },
}));

function wrapper(qc: QueryClient) {
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  );
}

const tenant = { id: "t1" } as Tenant;

describe("usePendingChanges", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("is disabled until a tenant is selected", () => {
    const qc = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const { result } = renderHook(() => usePendingChanges(null, "1"), {
      wrapper: wrapper(qc),
    });
    expect(pendingChangesService.getByCharacterId).not.toHaveBeenCalled();
    expect(result.current.isPending).toBe(true);
  });

  it("invalidates the detail key after a cancel so the panel reflects CANCELLED without a reload", async () => {
    const qc = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const invalidate = vi.spyOn(qc, "invalidateQueries");
    const { result } = renderHook(() => useCancelPendingChange(), {
      wrapper: wrapper(qc),
    });

    await result.current.mutateAsync({ tenant, characterId: "1", id: "pc-1" });

    await waitFor(() =>
      expect(invalidate).toHaveBeenCalledWith({
        queryKey: pendingChangeKeys.detail(tenant.id, "1"),
      }),
    );
  });
});
