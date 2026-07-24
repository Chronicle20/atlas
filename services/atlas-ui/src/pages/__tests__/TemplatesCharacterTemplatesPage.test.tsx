import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter } from "react-router-dom";

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

vi.mock("@/components/features/templates/TemplateDetailLayout", () => ({
  TemplateDetailLayout: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="template-layout">{children}</div>
  ),
}));

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

const useTemplateMock = vi.fn();
const mutateMock = vi.fn();
vi.mock("@/lib/hooks/api/useTemplates", () => ({
  useTemplate: (...a: unknown[]) => useTemplateMock(...a),
  useUpdateTemplate: () => ({ mutate: mutateMock, isPending: false }),
}));

import { TemplatesCharacterTemplatesPage } from "../TemplatesCharacterTemplatesPage";

const templates = [
  {
    jobIndex: 0,
    subJobIndex: 0,
    gender: 1,
    mapId: 0,
    faces: [],
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
const template = {
  id: "tmpl-1",
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
  useTemplateMock.mockReturnValue({
    data: template,
    isLoading: false,
    error: null,
  });
});

describe("TemplatesCharacterTemplatesPage", () => {
  it("renders the shared editor inside the template layout", () => {
    render(
      <MemoryRouter initialEntries={["/templates/tmpl-1/character/templates"]}>
        <TemplatesCharacterTemplatesPage />
      </MemoryRouter>,
    );
    expect(screen.getByTestId("template-layout")).toBeInTheDocument();
    expect(screen.getByTestId("shared-editor")).toBeInTheDocument();
  });

  it("save PATCHes by id, preserving presets", () => {
    render(
      <MemoryRouter initialEntries={["/templates/tmpl-1/character/templates"]}>
        <TemplatesCharacterTemplatesPage />
      </MemoryRouter>,
    );
    const { adapter } = editorMock.mock.calls[0]![0] as {
      adapter: { save: (t: unknown[], onSuccess: () => void) => void };
    };
    adapter.save(templates, vi.fn());
    expect(mutateMock).toHaveBeenCalledWith(
      {
        id: "tmpl-1",
        updates: { characters: { templates, presets } },
      },
      expect.objectContaining({ onSuccess: expect.any(Function) }),
    );
  });
});
