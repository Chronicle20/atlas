import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter } from "react-router-dom";

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

vi.mock("@/components/features/tenants/TenantDetailLayout", () => ({
  TenantDetailLayout: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="tenant-layout">{children}</div>
  ),
}));

// Capture the adapter handed to the shared editor.
const editorMock = vi.fn();
vi.mock(
  "@/components/features/characters/presets/CharacterPresetsEditor",
  () => ({
    CharacterPresetsEditor: (props: unknown) => {
      editorMock(props);
      return <div data-testid="shared-editor" />;
    },
  }),
);

const useTenantConfigurationMock = vi.fn();
const useTenantMock = vi.fn();
const mutateMock = vi.fn();
const refetchMock = vi.fn();
vi.mock("@/lib/hooks/api/useTenants", () => ({
  useTenantConfiguration: (...a: unknown[]) => useTenantConfigurationMock(...a),
  useUpdateTenantConfiguration: () => ({
    mutate: mutateMock,
    isPending: false,
  }),
  useTenant: (...a: unknown[]) => useTenantMock(...a),
}));

import { TenantsCharacterPresetsPage } from "../TenantsCharacterPresetsPage";

const templates = [
  {
    jobIndex: 1,
    subJobIndex: 0,
    gender: 0,
    mapId: 0,
    faces: [20000],
    hairs: [],
    hairColors: [],
    skinColors: [],
    tops: [],
    bottoms: [],
    shoes: [],
    weapons: [],
    items: [],
    skills: [],
  },
];
const presets = [{ id: "p1", attributes: { name: "keep-me" } }];
const tenant = {
  id: "t1",
  attributes: {
    region: "GMS",
    majorVersion: 83,
    minorVersion: 1,
    usesPin: false,
    characters: { templates, presets },
    npcs: [],
    socket: { handlers: [], writers: [] },
    worlds: [],
  },
};
const tenantBasic = { id: "t1", attributes: { name: "Tenant One" } };

beforeEach(() => {
  vi.clearAllMocks();
  useTenantConfigurationMock.mockReturnValue({
    data: tenant,
    isLoading: false,
    error: null,
    refetch: refetchMock,
  });
  useTenantMock.mockReturnValue({
    data: tenantBasic,
    isLoading: false,
    error: null,
  });
});

describe("TenantsCharacterPresetsPage", () => {
  it("renders the shared editor inside the tenant layout with adapter data", () => {
    render(
      <MemoryRouter initialEntries={["/tenants/t1/character/presets"]}>
        <TenantsCharacterPresetsPage />
      </MemoryRouter>,
    );
    expect(screen.getByTestId("tenant-layout")).toBeInTheDocument();
    expect(screen.getByTestId("shared-editor")).toBeInTheDocument();
    const { adapter } = editorMock.mock.calls[0]![0] as {
      adapter: { presets: unknown };
    };
    expect(adapter.presets).toEqual(presets);
  });

  it("save spreads characters so the sibling templates array survives, and sends no key", () => {
    render(
      <MemoryRouter initialEntries={["/tenants/t1/character/presets"]}>
        <TenantsCharacterPresetsPage />
      </MemoryRouter>,
    );
    const { adapter } = editorMock.mock.calls.at(-1)![0] as {
      adapter: {
        save: (
          p: unknown[],
          onSuccess: (persisted?: unknown[]) => void,
        ) => void;
      };
    };
    adapter.save([{ attributes: { name: "P" } }], () => {});
    expect(mutateMock).toHaveBeenCalledWith(
      {
        tenant,
        updates: {
          characters: expect.objectContaining({
            templates,
            presets: [{ attributes: { name: "P" } }],
          }),
        },
      },
      expect.objectContaining({ onSuccess: expect.any(Function) }),
    );
    const sentUpdates = mutateMock.mock.calls[0]![0].updates.characters;
    expect(sentUpdates.presets[0]).not.toHaveProperty("key");
  });

  it("re-reads the config after the PATCH lands, so onSuccess receives presets with server-assigned ids", async () => {
    render(
      <MemoryRouter initialEntries={["/tenants/t1/character/presets"]}>
        <TenantsCharacterPresetsPage />
      </MemoryRouter>,
    );
    const { adapter } = editorMock.mock.calls.at(-1)![0] as {
      adapter: {
        save: (
          p: unknown[],
          onSuccess: (persisted?: unknown[]) => void,
        ) => void;
      };
    };
    const onSuccess = vi.fn();

    // The PATCH itself carries no response body (204). The follow-up GET is
    // what surfaces the id atlas-configurations assigned to the previously
    // id-less preset, in the same order it was sent.
    const persistedPresets = [
      { id: "server-uuid-1", attributes: { name: "P" } },
    ];
    refetchMock.mockResolvedValue({
      data: {
        ...tenant,
        attributes: {
          ...tenant.attributes,
          characters: { templates, presets: persistedPresets },
        },
      },
    });

    adapter.save([{ attributes: { name: "P" } }], onSuccess);

    // Drive the mutation's own onSuccess callback the way react-query would.
    const [, options] = mutateMock.mock.calls[0]!;
    await (options as { onSuccess: () => Promise<void> }).onSuccess();

    expect(refetchMock).toHaveBeenCalled();
    expect(onSuccess).toHaveBeenCalledWith(persistedPresets);
  });

  it("degrades gracefully when the follow-up read fails, still invoking onSuccess without crashing", async () => {
    render(
      <MemoryRouter initialEntries={["/tenants/t1/character/presets"]}>
        <TenantsCharacterPresetsPage />
      </MemoryRouter>,
    );
    const { adapter } = editorMock.mock.calls.at(-1)![0] as {
      adapter: {
        save: (
          p: unknown[],
          onSuccess: (persisted?: unknown[]) => void,
        ) => void;
      };
    };
    const onSuccess = vi.fn();
    refetchMock.mockRejectedValue(new Error("network unreachable"));

    adapter.save([{ attributes: { name: "P" } }], onSuccess);

    const [, options] = mutateMock.mock.calls[0]!;
    await expect(
      (options as { onSuccess: () => Promise<void> }).onSuccess(),
    ).resolves.toBeUndefined();

    expect(onSuccess).toHaveBeenCalledWith();
  });

  it("supplies apply.tenant capability", () => {
    render(
      <MemoryRouter initialEntries={["/tenants/t1/character/presets"]}>
        <TenantsCharacterPresetsPage />
      </MemoryRouter>,
    );
    const { adapter } = editorMock.mock.calls.at(-1)![0] as {
      adapter: { apply?: { tenant: unknown } };
    };
    expect(adapter.apply?.tenant).toEqual(tenantBasic);
  });
});
