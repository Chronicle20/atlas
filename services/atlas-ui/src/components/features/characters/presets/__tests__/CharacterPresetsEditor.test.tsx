import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { MemoryRouter, useSearchParams } from "react-router-dom";
import { CharacterPresetsEditor, type PresetsEditorAdapter } from "../CharacterPresetsEditor";
import { DEFAULT_PRESET_ATTRIBUTES } from "../presetEditorState";
import type { Tenant } from "@/types/models/tenant";

// Mock the action bar + heavy leaves; keep library + editor real enough to assert flow.
vi.mock("@/components/DetailActionBarContext", () => ({ useRegisterDetailActionBar: vi.fn() }));
vi.mock("sonner", () => ({
  toast: { error: vi.fn(), warning: vi.fn(), success: vi.fn() },
}));

interface MockWorkingPreset {
  key: string;
  id?: string;
  attributes: { name: string };
}

vi.mock("../PresetEditor", () => ({
  PresetEditor: ({
    preset,
    onBack,
    onDuplicate,
    onRemove,
    onApply,
    onSetField,
  }: {
    preset: MockWorkingPreset;
    onBack: () => void;
    onDuplicate: () => void;
    onRemove: () => void;
    onApply?: () => void;
    onSetField: (path: string, value: string) => void;
  }) => (
    <div>
      <span>editor:{preset.attributes.name}</span>
      <button onClick={onBack}>back</button>
      <button onClick={onDuplicate}>duplicate-open</button>
      <button onClick={onRemove}>remove-open</button>
      {onApply && <button onClick={onApply}>apply-open</button>}
      <button onClick={() => onSetField("name", "Changed")}>make-dirty</button>
    </div>
  ),
}));
vi.mock("../PresetLibrary", () => ({
  PresetLibrary: ({
    presets,
    onOpen,
    onNew,
    onDuplicate,
    onApply,
    canApply,
  }: {
    presets: MockWorkingPreset[];
    onOpen: (k: string) => void;
    onNew: () => void;
    onDuplicate: (k: string) => void;
    onApply: (k: string) => void;
    canApply: boolean;
  }) => (
    <div>
      {presets.map((p) => (
        <div key={p.key}>
          <button onClick={() => onOpen(p.key)}>open:{p.attributes.name}</button>
          <button onClick={() => onDuplicate(p.key)}>duplicate:{p.attributes.name}</button>
          {canApply && (
            <button onClick={() => onApply(p.key)}>apply:{p.attributes.name}</button>
          )}
        </div>
      ))}
      <button onClick={onNew}>new</button>
    </div>
  ),
}));
vi.mock("../AccountPickerDialog", () => ({
  AccountPickerDialog: ({
    open,
    onPick,
  }: {
    open: boolean;
    onPick: (accountId: number) => void;
  }) => (open ? <button onClick={() => onPick(42)}>pick-account</button> : null),
}));
vi.mock("@/components/features/characters/ApplyPresetDialog", () => ({
  ApplyPresetDialog: ({
    open,
    accountId,
    initialPresetId,
  }: {
    open: boolean;
    accountId: number;
    initialPresetId?: string;
  }) =>
    open ? (
      <div>
        apply-dialog:{accountId}:{initialPresetId ?? "none"}
      </div>
    ) : null,
}));

const presets = [
  { id: "a1", attributes: { ...DEFAULT_PRESET_ATTRIBUTES, name: "One" } },
  { id: "b2", attributes: { ...DEFAULT_PRESET_ATTRIBUTES, name: "Two" } },
];
const adapter = (over: Partial<PresetsEditorAdapter> = {}): PresetsEditorAdapter => ({
  presets, isLoading: false, error: null, isSaving: false, save: vi.fn(), ...over,
});

const renderAt = (url: string, a: PresetsEditorAdapter) =>
  render(<MemoryRouter initialEntries={[url]}><CharacterPresetsEditor adapter={a} /></MemoryRouter>);

describe("CharacterPresetsEditor", () => {
  it("shows the library when no ?preset=", () => {
    renderAt("/", adapter());
    expect(screen.getByText("open:One")).toBeInTheDocument();
    expect(screen.queryByText(/^editor:/)).toBeNull();
  });

  it("deep-links ?preset=<id> into the focused editor", async () => {
    renderAt("/?preset=b2", adapter());
    expect(await screen.findByText("editor:Two")).toBeInTheDocument();
  });

  it("unresolvable ?preset= falls back to library without error", () => {
    renderAt("/?preset=nope", adapter());
    expect(screen.getByText("open:One")).toBeInTheDocument();
  });

  it("opening a card then back toggles editor/library", async () => {
    renderAt("/", adapter());
    await userEvent.click(screen.getByText("open:Two"));
    expect(await screen.findByText("editor:Two")).toBeInTheDocument();
    await userEvent.click(screen.getByText("back"));
    await waitFor(() => expect(screen.getByText("open:One")).toBeInTheDocument());
  });

  it("registers the action bar and Save projects the array (id-only, no key)", async () => {
    const { useRegisterDetailActionBar } = await import("@/components/DetailActionBarContext");
    const save = vi.fn();
    renderAt("/", adapter({ save }));
    // grab the last registration's onSave
    const calls = (useRegisterDetailActionBar as unknown as { mock: { calls: unknown[][] } }).mock.calls;
    const reg = calls.map((c) => c[0]).filter(Boolean).at(-1) as { onSave: () => void };
    reg.onSave();
    expect(save).toHaveBeenCalledWith(
      [
        { id: "a1", attributes: expect.objectContaining({ name: "One" }) },
        { id: "b2", attributes: expect.objectContaining({ name: "Two" }) },
      ],
      expect.any(Function),
    );
  });
});

function PresetParamProbe() {
  const [params] = useSearchParams();
  return <output data-testid="preset-param">{params.get("preset") ?? ""}</output>;
}

function renderWithProbe(url: string, a: PresetsEditorAdapter) {
  return render(
    <MemoryRouter initialEntries={[url]}>
      <PresetParamProbe />
      <CharacterPresetsEditor adapter={a} />
    </MemoryRouter>,
  );
}

describe("CharacterPresetsEditor URL sync (syncSelection, no length-watcher)", () => {
  it("adding a preset selects it and writes its (id-less) key to the URL", async () => {
    renderWithProbe("/", adapter());
    await userEvent.click(screen.getByText("new"));
    expect(await screen.findByText(/^editor:/)).toBeInTheDocument();
    expect(screen.getByTestId("preset-param")).toHaveTextContent("local-0");
  });

  it("back from the editor deletes the ?preset= param", async () => {
    renderWithProbe("/?preset=a1", adapter());
    await screen.findByText("editor:One");
    expect(screen.getByTestId("preset-param")).toHaveTextContent("a1");
    await userEvent.click(screen.getByText("back"));
    await waitFor(() => expect(screen.getByTestId("preset-param")).toHaveTextContent(""));
    expect(screen.getByText("open:One")).toBeInTheDocument();
  });

  it("duplicating from the library selects the new (id-less) copy, not the source's id", async () => {
    renderWithProbe("/", adapter());
    await userEvent.click(screen.getByText("duplicate:One"));
    expect(await screen.findByText(/^editor:/)).toBeInTheDocument();
    expect(screen.getByTestId("preset-param")).toHaveTextContent("local-0");
  });

  it("removing the open preset returns to the library and clears the param", async () => {
    renderWithProbe("/?preset=b2", adapter());
    await screen.findByText("editor:Two");
    await userEvent.click(screen.getByText("remove-open"));
    await waitFor(() => expect(screen.getByTestId("preset-param")).toHaveTextContent(""));
    expect(screen.getByText("open:One")).toBeInTheDocument();
    expect(screen.queryByText("open:Two")).toBeNull();
  });

  it("discard (via the action bar) clears selection and the param", async () => {
    const { useRegisterDetailActionBar } = await import("@/components/DetailActionBarContext");
    renderWithProbe("/?preset=a1", adapter());
    await screen.findByText("editor:One");
    await userEvent.click(screen.getByText("make-dirty"));

    const calls = (useRegisterDetailActionBar as unknown as { mock: { calls: unknown[][] } }).mock.calls;
    const lastReg = () => calls.map((c) => c[0]).filter(Boolean).at(-1) as { onDiscard: () => void; dirty: boolean };
    expect(lastReg().dirty).toBe(true);

    lastReg().onDiscard();
    await waitFor(() => expect(screen.getByTestId("preset-param")).toHaveTextContent(""));
    expect(screen.getByText("open:One")).toBeInTheDocument();
  });
});

describe("CharacterPresetsEditor apply orchestration", () => {
  const tenant = { id: "t1" } as Tenant;

  it("hides Apply affordances entirely when the adapter has no apply (template context)", async () => {
    renderAt("/", adapter());
    expect(screen.queryByText("apply:One")).toBeNull();
    await userEvent.click(screen.getByText("open:One"));
    expect(await screen.findByText("editor:One")).toBeInTheDocument();
    expect(screen.queryByText("apply-open")).toBeNull();
  });

  it("shows Apply affordances when the adapter provides tenant context", () => {
    renderAt("/", adapter({ apply: { tenant } }));
    expect(screen.getByText("apply:One")).toBeInTheDocument();
  });

  it("blocks Apply for a preset with no persisted id and does not open the picker", async () => {
    const { toast } = await import("sonner");
    renderAt("/", adapter({ apply: { tenant } }));
    await userEvent.click(screen.getByText("new")); // unsaved preset, no id
    await userEvent.click(await screen.findByText("apply-open"));
    expect(toast.error).toHaveBeenCalledWith(
      expect.stringMatching(/save this preset before applying/i),
    );
    expect(screen.queryByText("pick-account")).toBeNull();
  });

  it("warns (non-blocking) for a dirty saved preset, then still opens the picker and apply dialog", async () => {
    const { toast } = await import("sonner");
    renderAt("/?preset=a1", adapter({ apply: { tenant } }));
    await screen.findByText(/^editor:/);
    await userEvent.click(screen.getByText("make-dirty"));
    await userEvent.click(screen.getByText("apply-open"));

    expect(toast.warning).toHaveBeenCalledWith(
      expect.stringMatching(/last saved/i),
    );
    await userEvent.click(await screen.findByText("pick-account"));
    expect(await screen.findByText("apply-dialog:42:a1")).toBeInTheDocument();
  });

  it("applying a clean saved preset from the library opens the picker then the apply dialog with initialPresetId", async () => {
    renderAt("/", adapter({ apply: { tenant } }));
    await userEvent.click(screen.getByText("apply:Two"));
    await userEvent.click(await screen.findByText("pick-account"));
    expect(await screen.findByText("apply-dialog:42:b2")).toBeInTheDocument();
  });
});
