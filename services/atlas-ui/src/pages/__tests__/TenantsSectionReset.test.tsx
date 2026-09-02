import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import type { ReactElement } from "react";

import { TenantsPropertiesPage } from "@/pages/TenantsPropertiesPage";
import { TenantsHandlersPage } from "@/pages/TenantsHandlersPage";
import { TenantsWritersPage } from "@/pages/TenantsWritersPage";
import { TenantsCharacterTemplatesPage } from "@/pages/TenantsCharacterTemplatesPage";
import { TenantsCharacterPresetsPage } from "@/pages/TenantsCharacterPresetsPage";
import { TenantsMapleLifePage } from "@/pages/TenantsMapleLifePage";
import {
  useTenantConfiguration,
  useUpdateTenantConfiguration,
  useTenant,
} from "@/lib/hooks/api/useTenants";
import { MAPLE_LIFE_HANDLER } from "@/components/features/characters/maple-life/mapleLifeSupport";

vi.mock("@/components/features/config/ConfigExportButton", () => ({
  ConfigExportButton: () => <button type="button">Export</button>,
}));

vi.mock("@/components/features/tenants/TenantResetButton", () => ({
  TenantResetButton: (props: {
    id?: string;
    sections?: string[];
    sectionLabel?: string;
  }) => (
    <div
      data-testid="tenant-reset-button"
      data-id={props.id}
      {...(props.sections
        ? { "data-sections": JSON.stringify(props.sections) }
        : {})}
      {...(props.sectionLabel ? { "data-label": props.sectionLabel } : {})}
    />
  ),
}));

vi.mock("@/lib/hooks/api/useTenants", () => ({
  useTenantConfiguration: vi.fn(),
  useUpdateTenantConfiguration: vi.fn(),
  useTenant: vi.fn(),
}));

vi.mock("@/components/features/socket/DefinitionGridPage", () => ({
  DefinitionGridPage: ({ kind, scope }: { kind: string; scope: string }) => (
    <div
      data-testid="definition-grid-page"
      data-kind={kind}
      data-scope={scope}
    />
  ),
}));

vi.mock(
  "@/components/features/characters/templates/CharacterTemplatesEditor",
  () => ({
    CharacterTemplatesEditor: () => (
      <div data-testid="character-templates-editor" />
    ),
  }),
);

vi.mock(
  "@/components/features/characters/presets/CharacterPresetsEditor",
  () => ({
    CharacterPresetsEditor: () => (
      <div data-testid="character-presets-editor" />
    ),
  }),
);

vi.mock("@/components/features/characters/maple-life/MapleLifeEditor", () => ({
  MapleLifeEditor: () => <div data-testid="maple-life-editor" />,
}));

const baseTenant = (overrides: Record<string, unknown> = {}) => ({
  id: "tnt-1",
  attributes: {
    region: "GMS",
    majorVersion: 83,
    minorVersion: 1,
    usesPin: false,
    socket: { handlers: [], writers: [] },
    ...overrides,
  },
});

function renderAt(page: ReactElement, id: string, path: string) {
  render(
    <MemoryRouter initialEntries={[`/tenants/${id}/${path}`]}>
      <Routes>
        <Route path={`/tenants/:id/${path}`} element={page} />
      </Routes>
    </MemoryRouter>,
  );
}

/** The section-scoped reset button - excludes the header's whole-document one. */
function getSectionResetButton() {
  return screen
    .getAllByTestId("tenant-reset-button")
    .find((el) => el.hasAttribute("data-sections"));
}

describe("per-section tenant reset buttons", () => {
  beforeEach(() => {
    vi.mocked(useUpdateTenantConfiguration).mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
    } as never);
    vi.mocked(useTenant).mockReturnValue({ data: undefined } as never);
  });

  it("properties page resets properties", () => {
    vi.mocked(useTenantConfiguration).mockReturnValue({
      data: baseTenant(),
      isLoading: false,
      error: null,
    } as never);
    renderAt(<TenantsPropertiesPage />, "tnt-1", "properties");

    const button = getSectionResetButton();
    expect(button).toBeDefined();
    expect(JSON.parse(button?.getAttribute("data-sections") ?? "[]")).toEqual([
      "properties",
    ]);
    expect(button).toHaveAttribute("data-label", "global properties");
  });

  it("handlers page resets socket", () => {
    vi.mocked(useTenantConfiguration).mockReturnValue({
      data: baseTenant(),
      isLoading: false,
      error: null,
    } as never);
    renderAt(<TenantsHandlersPage />, "tnt-1", "handlers");

    const button = getSectionResetButton();
    expect(button).toBeDefined();
    expect(JSON.parse(button?.getAttribute("data-sections") ?? "[]")).toEqual([
      "socket",
    ]);
    expect(button).toHaveAttribute("data-label", "socket handlers and writers");
  });

  it("writers page resets socket", () => {
    vi.mocked(useTenantConfiguration).mockReturnValue({
      data: baseTenant(),
      isLoading: false,
      error: null,
    } as never);
    renderAt(<TenantsWritersPage />, "tnt-1", "writers");

    const button = getSectionResetButton();
    expect(button).toBeDefined();
    expect(JSON.parse(button?.getAttribute("data-sections") ?? "[]")).toEqual([
      "socket",
    ]);
    expect(button).toHaveAttribute("data-label", "socket handlers and writers");
  });

  it("character templates page resets characters", () => {
    vi.mocked(useTenantConfiguration).mockReturnValue({
      data: baseTenant({ characters: { templates: [], presets: [] } }),
      isLoading: false,
      error: null,
    } as never);
    renderAt(<TenantsCharacterTemplatesPage />, "tnt-1", "character/templates");

    const button = getSectionResetButton();
    expect(button).toBeDefined();
    expect(JSON.parse(button?.getAttribute("data-sections") ?? "[]")).toEqual([
      "characters",
    ]);
    expect(button).toHaveAttribute(
      "data-label",
      "character templates and presets",
    );
  });

  it("character presets page resets characters", () => {
    vi.mocked(useTenantConfiguration).mockReturnValue({
      data: baseTenant({ characters: { templates: [], presets: [] } }),
      isLoading: false,
      error: null,
    } as never);
    renderAt(<TenantsCharacterPresetsPage />, "tnt-1", "character/presets");

    const button = getSectionResetButton();
    expect(button).toBeDefined();
    expect(JSON.parse(button?.getAttribute("data-sections") ?? "[]")).toEqual([
      "characters",
    ]);
    expect(button).toHaveAttribute(
      "data-label",
      "character templates and presets",
    );
  });

  it("maple life page resets mapleLife", () => {
    vi.mocked(useTenantConfiguration).mockReturnValue({
      data: baseTenant({
        socket: {
          handlers: [
            { opCode: "0x12D", validator: "V", handler: MAPLE_LIFE_HANDLER },
          ],
          writers: [],
        },
        mapleLife: undefined,
      }),
      isLoading: false,
      error: null,
    } as never);
    renderAt(<TenantsMapleLifePage />, "tnt-1", "character/maple-life");

    const button = getSectionResetButton();
    expect(button).toBeDefined();
    expect(JSON.parse(button?.getAttribute("data-sections") ?? "[]")).toEqual([
      "mapleLife",
    ]);
    expect(button).toHaveAttribute("data-label", "Maple Life configuration");
  });

  it("maple life page renders no reset button on an unsupported client", () => {
    vi.mocked(useTenantConfiguration).mockReturnValue({
      data: baseTenant({ socket: { handlers: [], writers: [] } }),
      isLoading: false,
      error: null,
    } as never);
    renderAt(<TenantsMapleLifePage />, "tnt-1", "character/maple-life");

    expect(getSectionResetButton()).toBeUndefined();
  });
});
