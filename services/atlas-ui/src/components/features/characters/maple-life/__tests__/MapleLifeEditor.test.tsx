import { act } from "react";
import { render, screen, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { MemoryRouter, useLocation } from "react-router-dom";

import type { MapleLifeConfig } from "@/types/models/template";
import {
  MapleLifeEditor,
  type MapleLifeEditorAdapter,
} from "../MapleLifeEditor";

// Radix Select relies on DOM APIs jsdom does not implement.
Element.prototype.hasPointerCapture ||= () => false;
Element.prototype.scrollIntoView ||= () => {};

// Mock the action bar + heavy leaves; keep IdentitySection, ProgressionSection,
// SpSkillSection, and ClassSelector real enough to assert wiring.
vi.mock("@/components/DetailActionBarContext", () => ({
  useRegisterDetailActionBar: vi.fn(),
}));
vi.mock("sonner", () => ({
  toast: { error: vi.fn(), warning: vi.fn(), success: vi.fn() },
}));
vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: {
      id: "t1",
      attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
    },
  }),
}));
vi.mock("@/lib/hooks/usePresetJobOptions", () => ({
  usePresetJobOptions: () => ({
    options: [],
    isPending: false,
    isError: false,
  }),
}));
vi.mock("@/lib/hooks/api/useMaps", () => ({
  useMap: () => ({ data: undefined, isError: false }),
  useMapsByName: () => ({ data: [], isLoading: false }),
}));
vi.mock("@/lib/hooks/api/useTemplates", () => ({
  useTemplates: () => ({ data: undefined }),
}));
vi.mock("../AppearancePoolsSection", () => ({
  AppearancePoolsSection: () => <div data-testid="appearance-pools-section" />,
}));
vi.mock("../MapleLifePreviewCard", () => ({
  MapleLifePreviewCard: () => <div data-testid="preview-card" />,
}));
vi.mock("../StartingKitSection", () => ({
  StartingKitSection: () => <div data-testid="starting-kit-section" />,
}));

import { useRegisterDetailActionBar } from "@/components/DetailActionBarContext";

// Shipped seed row, reproduced verbatim from
// services/atlas-configurations/seed-data/templates/template_gms_83_1.json
// `mapleLife.looks[0]` and `mapleLife.classes[0]`, extended with the same
// second (synthetic) class row used by mapleLifeEditorState.test.ts, so this
// suite exercises the same canonical fixture as the reducer/schema tests it
// composes.
const SEED: MapleLifeConfig = {
  looks: [
    {
      gender: 0,
      faces: [20000, 20001, 20002],
      hairs: [30030, 30020, 30000],
      hairColors: [0, 7, 3, 2],
      skinColors: [0, 1, 2, 3],
    },
    {
      gender: 1,
      faces: [21000],
      hairs: [31050],
      hairColors: [0],
      skinColors: [0],
    },
  ],
  classes: [
    {
      ordinal: 0,
      gender: 0,
      jobId: 100,
      level: 30,
      mapId: 102000000,
      stats: { str: 35, dex: 4, int: 4, luk: 4, hp: 804, mp: 150 },
      ap: 123,
      sp: "61,0,0,0,0,0,0,0,0,0",
      spSkillId: 1000001,
      meso: 100000,
      equipment: [
        { templateId: 1040021, useAverageStats: true },
        { templateId: 1060016, useAverageStats: true },
      ],
      inventory: [
        { templateId: 2000002, quantity: 100 },
        { templateId: 2000006, quantity: 100 },
      ],
    },
    {
      ordinal: 3,
      gender: 1,
      jobId: 400,
      level: 30,
      mapId: 103000000,
      stats: { str: 4, dex: 25, int: 4, luk: 20, hp: 520, mp: 130 },
      ap: 121,
      sp: "61,0,0,0,0,0,0,0,0,0",
      meso: 100000,
      equipment: [],
      inventory: [],
    },
  ],
};

// Synthetic fixture (not shipped seed data): class 0 carries an SP skill but
// book 0 is below 6, which the schema rejects (MSG.spPoolTooSmall -- "the
// server needs sp + 5 for the prerequisite"). Chosen over an unparseable
// `sp` string (MSG.spNotTenBooks) so the ten book inputs stay ENABLED
// (ProgressionSection disables them while the pool isn't exactly ten
// books), making the "fix it via the book inputs" case exercisable through
// the real UI rather than a disabled control.
const BROKEN_SP_CONFIG: MapleLifeConfig = {
  looks: [SEED.looks[0]!],
  classes: [
    {
      ordinal: 0,
      gender: 0,
      jobId: 100,
      level: 30,
      mapId: 102000000,
      stats: { str: 4, dex: 4, int: 4, luk: 4, hp: 50, mp: 50 },
      ap: 0,
      sp: "0,0,0,0,0,0,0,0,0,0",
      spSkillId: 1000001,
      meso: 0,
      equipment: [],
      inventory: [],
    },
  ],
};

function buildAdapter(
  overrides: Partial<MapleLifeEditorAdapter> = {},
): MapleLifeEditorAdapter {
  return {
    mapleLife: undefined,
    isLoading: false,
    error: null,
    save: vi.fn(),
    isSaving: false,
    ...overrides,
  };
}

function renderAt(url: string, adapter: MapleLifeEditorAdapter) {
  return render(
    <MemoryRouter initialEntries={[url]}>
      <MapleLifeEditor adapter={adapter} />
    </MemoryRouter>,
  );
}

interface CapturedBarConfig {
  dirty: boolean;
  isSaving: boolean;
  blockingIssues?: number;
  onSave: () => void;
  onDiscard: () => void;
}

function lastBarConfig(): CapturedBarConfig | null {
  const mockFn = useRegisterDetailActionBar as unknown as {
    mock: { calls: unknown[][] };
  };
  const calls = mockFn.mock.calls;
  const last = calls.at(-1);
  return (last?.[0] as CapturedBarConfig | null) ?? null;
}

function LocationSearchProbe() {
  const location = useLocation();
  return <output data-testid="location-search">{location.search}</output>;
}

function renderWithProbe(url: string, adapter: MapleLifeEditorAdapter) {
  return render(
    <MemoryRouter initialEntries={[url]}>
      <LocationSearchProbe />
      <MapleLifeEditor adapter={adapter} />
    </MemoryRouter>,
  );
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe("MapleLifeEditor", () => {
  it("renders a skeleton before the adapter delivers", () => {
    renderAt("/", buildAdapter({ isLoading: true, mapleLife: undefined }));
    expect(screen.getByTestId("form-skeleton")).toBeInTheDocument();
    expect(screen.queryByRole("tablist")).toBeNull();
  });

  it("renders the error display when the query fails before load", () => {
    renderAt(
      "/",
      buildAdapter({
        isLoading: false,
        mapleLife: undefined,
        error: new Error("boom"),
      }),
    );
    expect(screen.getByText(/boom/)).toBeInTheDocument();
  });

  it("seeds the reducer exactly once", async () => {
    const user = userEvent.setup();
    const adapter = buildAdapter({ mapleLife: SEED });
    const { rerender } = renderAt("/", adapter);

    const apInput = screen.getByRole("spinbutton", { name: "AP" });
    await user.clear(apInput);
    await user.type(apInput, "124");
    expect(apInput).toHaveValue(124);

    // Re-deliver a NEW object identity with the original ap -- must NOT
    // clobber the edit, since the reducer is already loaded.
    const reseeded = buildAdapter({
      mapleLife: { ...SEED, classes: SEED.classes.map((c) => ({ ...c })) },
    });
    rerender(
      <MemoryRouter initialEntries={["/"]}>
        <MapleLifeEditor adapter={reseeded} />
      </MemoryRouter>,
    );

    expect(screen.getByRole("spinbutton", { name: "AP" })).toHaveValue(124);
  });

  it("applies ?ordinal and ?gender on load", () => {
    renderAt("/?ordinal=3&gender=1", buildAdapter({ mapleLife: SEED }));
    const selected = screen.getAllByRole("tab", { selected: true });
    expect(selected.some((el) => el.textContent?.includes("Female"))).toBe(
      true,
    );
    expect(selected.some((el) => el.textContent?.startsWith("3 ·"))).toBe(true);
  });

  it("clamps an out-of-range ordinal to 0", () => {
    renderWithProbe("/?ordinal=9", buildAdapter({ mapleLife: SEED }));
    const selected = screen.getAllByRole("tab", { selected: true });
    expect(selected.some((el) => el.textContent?.startsWith("0 ·"))).toBe(true);
    expect(screen.getByTestId("location-search")).toHaveTextContent(
      "ordinal=0",
    );
  });

  it("clamps an unparseable gender to 0", () => {
    renderWithProbe("/?gender=abc", buildAdapter({ mapleLife: SEED }));
    const selected = screen.getAllByRole("tab", { selected: true });
    expect(selected.some((el) => el.textContent?.includes("Male"))).toBe(true);
    expect(screen.getByTestId("location-search")).toHaveTextContent("gender=0");
  });

  it("a selection click writes the URL", async () => {
    const user = userEvent.setup();
    renderWithProbe("/", buildAdapter({ mapleLife: SEED }));
    await user.click(screen.getByRole("tab", { name: /female/i }));
    expect(screen.getByTestId("location-search")).toHaveTextContent("gender=1");
  });

  it("registers a clean bar after load", () => {
    renderAt("/", buildAdapter({ mapleLife: SEED }));
    const config = lastBarConfig();
    expect(config?.dirty).toBe(false);
    expect(config?.blockingIssues).toBe(0);
  });

  it("reports blocking errors in the bar", () => {
    renderAt("/", buildAdapter({ mapleLife: BROKEN_SP_CONFIG }));
    const config = lastBarConfig();
    expect(config?.blockingIssues).toBeGreaterThanOrEqual(1);
  });

  it("clears blockingIssues once the error is fixed", () => {
    renderAt("/", buildAdapter({ mapleLife: BROKEN_SP_CONFIG }));
    expect(lastBarConfig()?.blockingIssues).toBeGreaterThanOrEqual(1);

    const book0 = screen.getByRole("spinbutton", { name: "Book 0" });
    fireEvent.change(book0, { target: { value: "61" } });

    expect(lastBarConfig()?.blockingIssues).toBe(0);
  });

  it("saves only the mapleLife block", async () => {
    const user = userEvent.setup();
    const save = vi.fn();
    renderAt("/", buildAdapter({ mapleLife: SEED, save }));

    const apInput = screen.getByRole("spinbutton", { name: "AP" });
    await user.clear(apInput);
    await user.type(apInput, "124");

    lastBarConfig()!.onSave();

    expect(save).toHaveBeenCalledTimes(1);
    const [savedConfig] = save.mock.calls[0]!;
    expect(Object.keys(savedConfig).sort()).toEqual(["classes", "looks"]);
    expect(savedConfig.classes[0].ap).toBe(124);
  });

  it("a successful save clears dirty", async () => {
    const user = userEvent.setup();
    const save = vi.fn();
    renderAt("/", buildAdapter({ mapleLife: SEED, save }));

    const apInput = screen.getByRole("spinbutton", { name: "AP" });
    await user.clear(apInput);
    await user.type(apInput, "124");

    act(() => {
      lastBarConfig()!.onSave();
    });
    const onSuccess = save.mock.calls[0]![1] as () => void;
    act(() => {
      onSuccess();
    });

    expect(lastBarConfig()?.dirty).toBe(false);
  });

  it("registers no bar for an empty configuration", () => {
    renderAt("/", buildAdapter({ mapleLife: undefined, isLoading: false }));
    expect(lastBarConfig()).toBeNull();
  });

  it("an empty tenant configuration offers Seed from template", () => {
    renderAt(
      "/",
      buildAdapter({
        mapleLife: undefined,
        isLoading: false,
        seedFrom: { region: "GMS", majorVersion: 83, minorVersion: 1 },
      }),
    );
    expect(
      screen.getByRole("button", { name: /seed from template/i }),
    ).toBeInTheDocument();
  });

  it("an empty template configuration does not", () => {
    renderAt("/", buildAdapter({ mapleLife: undefined, isLoading: false }));
    expect(
      screen.queryByRole("button", { name: /seed from template/i }),
    ).toBeNull();
    expect(
      screen.getByRole("button", { name: /Add the ten class rows/i }),
    ).toBeInTheDocument();
  });

  it("adding the ten rows marks the page dirty", async () => {
    const user = userEvent.setup();
    renderAt("/", buildAdapter({ mapleLife: undefined, isLoading: false }));
    await user.click(
      screen.getByRole("button", { name: /Add the ten class rows/i }),
    );
    expect(lastBarConfig()?.dirty).toBe(true);
    expect(screen.getAllByRole("tab", { name: /^\d ·/ })).toHaveLength(5);
  });
});
