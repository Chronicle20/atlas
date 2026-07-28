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
  "@/components/features/characters/templates/CharacterTemplatesEditor",
  () => ({
    CharacterTemplatesEditor: (props: unknown) => {
      editorMock(props);
      return <div data-testid="shared-editor" />;
    },
  }),
);

const useTenantConfigurationMock = vi.fn();
const mutateMock = vi.fn();
vi.mock("@/lib/hooks/api/useTenants", () => ({
  useTenantConfiguration: (...a: unknown[]) => useTenantConfigurationMock(...a),
  useUpdateTenantConfiguration: () => ({
    mutate: mutateMock,
    isPending: false,
  }),
}));

import { TenantsCharacterTemplatesPage } from "../TenantsCharacterTemplatesPage";

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
const presets = [{ attributes: { name: "keep-me" } }];
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

beforeEach(() => {
  vi.clearAllMocks();
  useTenantConfigurationMock.mockReturnValue({
    data: tenant,
    isLoading: false,
    error: null,
  });
});

describe("TenantsCharacterTemplatesPage", () => {
  it("renders the shared editor inside the tenant layout with adapter data", () => {
    render(
      <MemoryRouter initialEntries={["/tenants/t1/character/templates"]}>
        <TenantsCharacterTemplatesPage />
      </MemoryRouter>,
    );
    expect(screen.getByTestId("tenant-layout")).toBeInTheDocument();
    expect(screen.getByTestId("shared-editor")).toBeInTheDocument();
    const { adapter } = editorMock.mock.calls[0]![0] as {
      adapter: { templates: unknown };
    };
    expect(adapter.templates).toEqual(templates);
  });

  it("save PATCHes the full characters object, preserving presets", () => {
    render(
      <MemoryRouter initialEntries={["/tenants/t1/character/templates"]}>
        <TenantsCharacterTemplatesPage />
      </MemoryRouter>,
    );
    const { adapter } = editorMock.mock.calls[0]![0] as {
      adapter: {
        save: (t: unknown[], onSuccess: () => void) => void;
      };
    };
    const newTemplates = [...templates, { jobIndex: 2 }];
    adapter.save(newTemplates, vi.fn());
    expect(mutateMock).toHaveBeenCalledWith(
      {
        tenant,
        updates: {
          characters: { templates: newTemplates, presets },
        },
      },
      expect.objectContaining({ onSuccess: expect.any(Function) }),
    );
  });
});
