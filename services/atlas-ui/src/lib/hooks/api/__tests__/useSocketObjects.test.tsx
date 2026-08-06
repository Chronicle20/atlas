import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const getById = vi.fn();
const update = vi.fn();
const getSocketMatrix = vi.fn();

vi.mock("@/services/api/templates.service", () => ({
  templatesService: {
    getById: (...a: unknown[]) => getById(...a),
    update: (...a: unknown[]) => update(...a),
    getSocketMatrix: (...a: unknown[]) => getSocketMatrix(...a),
  },
}));

const getTenantConfigurationById = vi.fn();
const updateTenantConfiguration = vi.fn();
const getTenantSocketMatrix = vi.fn();

vi.mock("@/services/api/tenants.service", () => ({
  tenantsService: {
    getTenantConfigurationById: (...a: unknown[]) =>
      getTenantConfigurationById(...a),
    updateTenantConfiguration: (...a: unknown[]) =>
      updateTenantConfiguration(...a),
    getSocketMatrix: (...a: unknown[]) => getTenantSocketMatrix(...a),
  },
}));

import {
  socketKeys,
  useSocketMatrixTemplates,
  useSocketMutation,
} from "@/lib/hooks/api/useSocketObjects";
import { templateKeys } from "@/lib/hooks/api/useTemplates";
import { tenantKeys } from "@/lib/hooks/api/useTenants";
import type { SocketConfig } from "@/types/models/socket";

function makeWrapper(client?: QueryClient) {
  const qc =
    client ??
    new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });
  function wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
  }
  return { qc, wrapper };
}

const fullTemplate = {
  id: "t1",
  attributes: {
    region: "GMS",
    majorVersion: 83,
    minorVersion: 1,
    usesPin: false,
    characters: { templates: [], presets: [{ id: "p1" }] },
    npcs: [{ npcId: 1, impl: "x" }],
    worlds: [{ name: "Scania" }],
    cashShop: { commodities: {} },
    socket: {
      handlers: [],
      writers: [],
      unsupported: { handlers: [], writers: [] },
    },
  },
};

describe("useSocketMutation", () => {
  beforeEach(() => {
    getById.mockReset().mockResolvedValue(structuredClone(fullTemplate));
    update.mockReset().mockResolvedValue(structuredClone(fullTemplate));
    getSocketMatrix.mockReset().mockResolvedValue([]);
    getTenantConfigurationById.mockReset();
    updateTenantConfiguration.mockReset();
    getTenantSocketMatrix.mockReset().mockResolvedValue([]);
  });

  // The core rule: a mutation NEVER writes the cached (possibly sparse)
  // document. It re-fetches the full one first.
  it("re-fetches the full document before writing", async () => {
    const { wrapper } = makeWrapper();
    const { result } = renderHook(() => useSocketMutation(), { wrapper });
    await result.current.mutateAsync({
      target: { source: "template", id: "t1" },
      apply: (cfg: SocketConfig) => cfg,
    });
    expect(getById).toHaveBeenCalledWith("t1");
    expect(getById.mock.invocationCallOrder[0]!).toBeLessThan(
      update.mock.invocationCallOrder[0]!,
    );
  });

  // Structural proof of the sparse-cache-hazard rule: even when the sparse
  // matrix cache is already populated (as it would be after the grid has
  // loaded), the mutation ignores it entirely and still re-fetches by id.
  // socketKeys.matrix() never holds full TemplateAttributes (no characters/
  // worlds/cashShop at all - see normalize.ts's SocketObject shape), so
  // there is no code path by which this hook could read a sparse document
  // as its write input.
  it("never reads from the sparse matrix cache, even when it is populated", async () => {
    const { qc, wrapper } = makeWrapper();
    qc.setQueryData(socketKeys.matrix(), [
      {
        key: "t1",
        label: "GMS v83.1",
        source: "template",
        region: "GMS",
        majorVersion: 83,
        minorVersion: 1,
        handlers: new Map(),
        writers: new Map(),
        unsupportedHandlers: new Set(),
        unsupportedWriters: new Set(),
      },
    ]);

    const { result } = renderHook(() => useSocketMutation(), { wrapper });
    await result.current.mutateAsync({
      target: { source: "template", id: "t1" },
      apply: (cfg: SocketConfig) => cfg,
    });

    expect(getById).toHaveBeenCalledWith("t1");
    const sent = update.mock.calls[0]![1] as typeof fullTemplate.attributes;
    // These fields exist only on the freshly-fetched full document, never on
    // the cached sparse SocketObject - if the sparse cache had leaked into
    // the write path there would be nothing here to assert on.
    expect(sent.characters.presets).toHaveLength(1);
    expect(sent.worlds).toHaveLength(1);
    expect(sent.cashShop).toBeDefined();
  });

  it("sends the whole attribute document, with only socket changed", async () => {
    const { wrapper } = makeWrapper();
    const { result } = renderHook(() => useSocketMutation(), { wrapper });
    await result.current.mutateAsync({
      target: { source: "template", id: "t1" },
      apply: (cfg: SocketConfig) => ({
        ...cfg,
        writers: [
          { opCode: "0x00", writer: "AuthSuccess", services: ["login"] },
        ],
      }),
    });
    const sent = update.mock.calls[0]![1] as typeof fullTemplate.attributes;
    expect(sent.characters.presets).toHaveLength(1);
    expect(sent.npcs).toHaveLength(1);
    expect(sent.worlds).toHaveLength(1);
    expect(sent.cashShop).toBeDefined();
    expect(sent.socket.writers).toHaveLength(1);
  });

  it("propagates a MutationError from the apply function without writing", async () => {
    const { wrapper } = makeWrapper();
    const { result } = renderHook(() => useSocketMutation(), { wrapper });
    await expect(
      result.current.mutateAsync({
        target: { source: "template", id: "t1" },
        apply: () => {
          throw new Error("binding did not resolve");
        },
      }),
    ).rejects.toThrow("binding did not resolve");
    expect(update).not.toHaveBeenCalled();
  });

  it("onSuccess invalidates both the sparse matrix key and the template detail key", async () => {
    const { qc, wrapper } = makeWrapper();
    const invalidateSpy = vi.spyOn(qc, "invalidateQueries");
    const { result } = renderHook(() => useSocketMutation(), { wrapper });

    await result.current.mutateAsync({
      target: { source: "template", id: "t1" },
      apply: (cfg: SocketConfig) => cfg,
    });

    await waitFor(() => {
      const keys = invalidateSpy.mock.calls.map((c) => c[0]?.queryKey);
      expect(keys).toContainEqual(socketKeys.all);
      expect(keys).toContainEqual(templateKeys.detail("t1"));
    });
  });

  it("onSuccess invalidates both the sparse matrix key and the tenant config detail key", async () => {
    getTenantConfigurationById.mockResolvedValue(
      structuredClone({
        id: "tn1",
        attributes: {
          region: "GMS",
          majorVersion: 95,
          minorVersion: 1,
          usesPin: false,
          characters: { templates: [], presets: [] },
          npcs: [],
          worlds: [{ name: "Scania" }],
          socket: { handlers: [], writers: [] },
        },
      }),
    );
    updateTenantConfiguration.mockResolvedValue({});

    const { qc, wrapper } = makeWrapper();
    const invalidateSpy = vi.spyOn(qc, "invalidateQueries");
    const { result } = renderHook(() => useSocketMutation(), { wrapper });

    await result.current.mutateAsync({
      target: { source: "tenant", id: "tn1" },
      apply: (cfg: SocketConfig) => cfg,
    });

    await waitFor(() => {
      const keys = invalidateSpy.mock.calls.map((c) => c[0]?.queryKey);
      expect(keys).toContainEqual(socketKeys.all);
      expect(keys).toContainEqual(tenantKeys.configDetail("tn1"));
    });

    // The tenant write path re-fetched the full document and handed it back
    // to updateTenantConfiguration - never the sparse tenant matrix cache.
    expect(getTenantConfigurationById).toHaveBeenCalledWith("tn1");
    const [tenantArg, updatesArg] = updateTenantConfiguration.mock.calls[0]!;
    expect(tenantArg.attributes.worlds).toHaveLength(1);
    expect(updatesArg).toEqual({ socket: { handlers: [], writers: [] } });
  });
});

describe("useSocketMatrixTemplates", () => {
  it("fetches the sparse matrix and normalizes it, under socketKeys.matrix()", async () => {
    getSocketMatrix.mockResolvedValue([structuredClone(fullTemplate)]);
    const { wrapper } = makeWrapper();
    const { result } = renderHook(() => useSocketMatrixTemplates(), {
      wrapper,
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(getSocketMatrix).toHaveBeenCalledTimes(1);
    expect(result.current.data).toHaveLength(1);
    expect(result.current.data![0]!.key).toBe("t1");
  });
});
