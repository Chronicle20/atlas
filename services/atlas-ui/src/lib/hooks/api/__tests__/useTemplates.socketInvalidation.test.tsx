import { describe, it, expect, vi, beforeEach } from "vitest";
import type { ReactNode } from "react";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  useCreateTemplate,
  useUpdateTemplate,
  usePatchTemplate,
  useDeleteTemplate,
  useReseedTemplate,
  useCreateTemplatesBatch,
  useUpdateTemplatesBatch,
  useDeleteTemplatesBatch,
} from "@/lib/hooks/api/useTemplates";
import { socketKeys } from "@/lib/hooks/api/socketKeys";
import { templatesService } from "@/services/api/templates.service";
import type { Template, TemplateAttributes } from "@/types/models/template";
import type { BatchResult } from "@/lib/api/query-params";

vi.mock("@/services/api/templates.service", () => ({
  templatesService: {
    create: vi.fn(),
    update: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
    reseed: vi.fn(),
    createBatch: vi.fn(),
    updateBatch: vi.fn(),
    deleteBatch: vi.fn(),
  },
}));

function makeWrapper(qc: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
  };
}

function newClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
}

const template: Template = {
  id: "t1",
  attributes: {
    region: "GMS",
    majorVersion: 95,
    minorVersion: 1,
  } as TemplateAttributes,
};

describe("useTemplates mutations invalidate socketKeys.all (Packet Matrix)", () => {
  beforeEach(() => vi.clearAllMocks());

  it("useCreateTemplate invalidates socketKeys.all on success", async () => {
    vi.mocked(templatesService.create).mockResolvedValue(template);
    const qc = newClient();
    const invalidate = vi.spyOn(qc, "invalidateQueries");

    const { result } = renderHook(() => useCreateTemplate(), {
      wrapper: makeWrapper(qc),
    });
    await result.current.mutateAsync(template.attributes);

    await waitFor(() => {
      const keys = invalidate.mock.calls.map((c) => c[0]?.queryKey);
      expect(keys).toContainEqual(socketKeys.all);
    });
  });

  it("useUpdateTemplate invalidates socketKeys.all on settle", async () => {
    vi.mocked(templatesService.update).mockResolvedValue(template);
    const qc = newClient();
    const invalidate = vi.spyOn(qc, "invalidateQueries");

    const { result } = renderHook(() => useUpdateTemplate(), {
      wrapper: makeWrapper(qc),
    });
    await result.current.mutateAsync({
      id: "t1",
      updates: template.attributes,
    });

    await waitFor(() => {
      const keys = invalidate.mock.calls.map((c) => c[0]?.queryKey);
      expect(keys).toContainEqual(socketKeys.all);
    });
  });

  it("usePatchTemplate invalidates socketKeys.all on success", async () => {
    vi.mocked(templatesService.patch).mockResolvedValue(template);
    const qc = newClient();
    const invalidate = vi.spyOn(qc, "invalidateQueries");

    const { result } = renderHook(() => usePatchTemplate(), {
      wrapper: makeWrapper(qc),
    });
    await result.current.mutateAsync({ id: "t1", updates: { region: "GMS" } });

    await waitFor(() => {
      const keys = invalidate.mock.calls.map((c) => c[0]?.queryKey);
      expect(keys).toContainEqual(socketKeys.all);
    });
  });

  it("useDeleteTemplate invalidates socketKeys.all on settle", async () => {
    vi.mocked(templatesService.delete).mockResolvedValue(undefined);
    const qc = newClient();
    const invalidate = vi.spyOn(qc, "invalidateQueries");

    const { result } = renderHook(() => useDeleteTemplate(), {
      wrapper: makeWrapper(qc),
    });
    await result.current.mutateAsync({ id: "t1" });

    await waitFor(() => {
      const keys = invalidate.mock.calls.map((c) => c[0]?.queryKey);
      expect(keys).toContainEqual(socketKeys.all);
    });
  });

  it("useReseedTemplate invalidates socketKeys.all on success", async () => {
    vi.mocked(templatesService.reseed).mockResolvedValue(undefined);
    const qc = newClient();
    const invalidate = vi.spyOn(qc, "invalidateQueries");

    const { result } = renderHook(() => useReseedTemplate(), {
      wrapper: makeWrapper(qc),
    });
    await result.current.mutateAsync({ id: "t1" });

    await waitFor(() => {
      const keys = invalidate.mock.calls.map((c) => c[0]?.queryKey);
      expect(keys).toContainEqual(socketKeys.all);
    });
  });

  it("useReseedTemplate invalidates nothing on failure (FR-5.6, onSuccess-only)", async () => {
    vi.mocked(templatesService.reseed).mockRejectedValue(
      new Error("no shipped template"),
    );
    const qc = newClient();
    const invalidate = vi.spyOn(qc, "invalidateQueries");

    const { result } = renderHook(() => useReseedTemplate(), {
      wrapper: makeWrapper(qc),
    });
    await expect(result.current.mutateAsync({ id: "t1" })).rejects.toThrow(
      "no shipped template",
    );

    expect(invalidate).not.toHaveBeenCalled();
  });

  it("useCreateTemplatesBatch invalidates socketKeys.all on success", async () => {
    const batchResult: BatchResult<Template> = {
      successes: [template],
      failures: [],
      total: 1,
      successCount: 1,
      failureCount: 0,
    };
    vi.mocked(templatesService.createBatch).mockResolvedValue(batchResult);
    const qc = newClient();
    const invalidate = vi.spyOn(qc, "invalidateQueries");

    const { result } = renderHook(() => useCreateTemplatesBatch(), {
      wrapper: makeWrapper(qc),
    });
    await result.current.mutateAsync({ templates: [template.attributes] });

    await waitFor(() => {
      const keys = invalidate.mock.calls.map((c) => c[0]?.queryKey);
      expect(keys).toContainEqual(socketKeys.all);
    });
  });

  it("useUpdateTemplatesBatch invalidates socketKeys.all on success", async () => {
    const batchResult: BatchResult<Template> = {
      successes: [template],
      failures: [],
      total: 1,
      successCount: 1,
      failureCount: 0,
    };
    vi.mocked(templatesService.updateBatch).mockResolvedValue(batchResult);
    const qc = newClient();
    const invalidate = vi.spyOn(qc, "invalidateQueries");

    const { result } = renderHook(() => useUpdateTemplatesBatch(), {
      wrapper: makeWrapper(qc),
    });
    await result.current.mutateAsync({
      updates: [{ id: "t1", data: template.attributes }],
    });

    await waitFor(() => {
      const keys = invalidate.mock.calls.map((c) => c[0]?.queryKey);
      expect(keys).toContainEqual(socketKeys.all);
    });
  });

  it("useDeleteTemplatesBatch invalidates socketKeys.all on settle", async () => {
    const batchResult: BatchResult<string> = {
      successes: ["t1"],
      failures: [],
      total: 1,
      successCount: 1,
      failureCount: 0,
    };
    vi.mocked(templatesService.deleteBatch).mockResolvedValue(batchResult);
    const qc = newClient();
    const invalidate = vi.spyOn(qc, "invalidateQueries");

    const { result } = renderHook(() => useDeleteTemplatesBatch(), {
      wrapper: makeWrapper(qc),
    });
    await result.current.mutateAsync({ ids: ["t1"] });

    await waitFor(() => {
      const keys = invalidate.mock.calls.map((c) => c[0]?.queryKey);
      expect(keys).toContainEqual(socketKeys.all);
    });
  });
});
