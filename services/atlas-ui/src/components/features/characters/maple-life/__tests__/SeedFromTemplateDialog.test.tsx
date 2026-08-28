import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, beforeEach } from "vitest";

import type {
  MapleLifeConfig,
  Template,
  TemplateAttributes,
} from "@/types/models/template";

const useTemplates = vi.fn();
vi.mock("@/lib/hooks/api/useTemplates", () => ({
  useTemplates: () => useTemplates(),
}));

import { SeedFromTemplateDialog } from "../SeedFromTemplateDialog";

const SEED_ML: MapleLifeConfig = {
  looks: [
    {
      gender: 0,
      faces: [20000, 20001, 20002],
      hairs: [30030, 30020, 30000],
      hairColors: [0, 7, 3, 2],
      skinColors: [0, 1, 2, 3],
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
      equipment: [{ templateId: 1040021, useAverageStats: true }],
      inventory: [{ templateId: 2000002, quantity: 100 }],
    },
  ],
};

// Copied from src/services/api/__tests__/templates-update.test.ts:23-38 — the
// minimum full TemplateAttributes shape shared across template-authoring tests.
function fullAttributes(
  overrides: Partial<TemplateAttributes> = {},
): TemplateAttributes {
  return {
    region: "GMS",
    majorVersion: 83,
    minorVersion: 1,
    usesPin: false,
    characters: { templates: [], presets: [] },
    npcs: [],
    worlds: [],
    socket: {
      handlers: [],
      writers: [],
      unsupported: { handlers: [], writers: [] },
    },
    ...overrides,
  } as TemplateAttributes;
}

function template(
  id: string,
  overrides: Partial<TemplateAttributes> = {},
): Template {
  return { id, attributes: fullAttributes(overrides) };
}

const seedFrom = { region: "GMS", majorVersion: 83, minorVersion: 1 };

function setup(
  data: Template[] | undefined,
  onSeed = vi.fn(),
  onOpenChange = vi.fn(),
) {
  useTemplates.mockReturnValue({ data, isLoading: false, isError: false });
  render(
    <SeedFromTemplateDialog
      open
      onOpenChange={onOpenChange}
      seedFrom={seedFrom}
      onSeed={onSeed}
    />,
  );
  return { onSeed, onOpenChange };
}

describe("SeedFromTemplateDialog", () => {
  beforeEach(() => {
    useTemplates.mockReset();
  });

  it("an in-flight fetch shows a loading state, not the empty-state copy", () => {
    useTemplates.mockReturnValue({
      data: undefined,
      isLoading: true,
      isError: false,
    });
    render(
      <SeedFromTemplateDialog
        open
        onOpenChange={vi.fn()}
        seedFrom={seedFrom}
        onSeed={vi.fn()}
      />,
    );

    expect(
      screen.queryByText(
        /No template of this region and version carries a Maple Life block/,
      ),
    ).not.toBeInTheDocument();
  });

  it("a failed fetch surfaces an error, not the empty-state copy", () => {
    useTemplates.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
    });
    render(
      <SeedFromTemplateDialog
        open
        onOpenChange={vi.fn()}
        seedFrom={seedFrom}
        onSeed={vi.fn()}
      />,
    );

    expect(screen.getByText(/Failed to load templates/i)).toBeInTheDocument();
    expect(
      screen.queryByText(
        /No template of this region and version carries a Maple Life block/,
      ),
    ).not.toBeInTheDocument();
  });

  it("zero matches states it plainly and offers no action", () => {
    setup([
      template("t1", { region: "GMS", majorVersion: 87, minorVersion: 1 }),
    ]);

    expect(
      screen.getByText(
        /No template of this region and version carries a Maple Life block/,
      ),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /seed/i }),
    ).not.toBeInTheDocument();
  });

  it("a matched template with an empty block is listed as ineligible, not hidden", () => {
    setup([template("t1", { mapleLife: { looks: [], classes: [] } })]);

    expect(screen.getByText(/t1/)).toBeInTheDocument();
    expect(screen.getByText(/no Maple Life block/i)).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /seed/i }),
    ).not.toBeInTheDocument();
  });

  it("a matched template with no mapleLife key at all is ineligible", () => {
    setup([template("t1")]);

    expect(screen.getByText(/t1/)).toBeInTheDocument();
    expect(screen.getByText(/no Maple Life block/i)).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /seed/i }),
    ).not.toBeInTheDocument();
  });

  it("exactly one eligible match seeds after confirmation", async () => {
    const user = userEvent.setup();
    const { onSeed } = setup([template("t1", { mapleLife: SEED_ML })]);

    const confirm = screen.getByRole("button", { name: /t1/ });
    await user.click(confirm);

    expect(onSeed).toHaveBeenCalledTimes(1);
    expect(onSeed.mock.calls[0]![0]).toEqual(SEED_ML);
  });

  it("more than one eligible match presents a picker", async () => {
    const user = userEvent.setup();
    const second: MapleLifeConfig = {
      looks: SEED_ML.looks,
      classes: [...SEED_ML.classes, { ...SEED_ML.classes[0]!, ordinal: 1 }],
    };
    const { onSeed } = setup([
      template("t1", { mapleLife: SEED_ML }),
      template("t2", { mapleLife: second }),
    ]);

    expect(screen.getByText(/t1/)).toBeInTheDocument();
    expect(screen.getByText(/t2/)).toBeInTheDocument();
    expect(screen.getByText(/1 classes · 1 looks/)).toBeInTheDocument();
    expect(screen.getByText(/2 classes · 1 looks/)).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /t2/ }));

    expect(onSeed).toHaveBeenCalledTimes(1);
    expect(onSeed.mock.calls[0]![0]).toEqual(second);
  });

  it("a mixed result lists the ineligible alongside the eligible", () => {
    setup([
      template("t1", { mapleLife: SEED_ML }),
      template("t2", { mapleLife: { looks: [], classes: [] } }),
    ]);

    expect(screen.getByText(/t1/)).toBeInTheDocument();
    expect(screen.getByText(/t2/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /t1/ })).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /t2/ }),
    ).not.toBeInTheDocument();
  });

  it("the donor template is never mutated", async () => {
    const user = userEvent.setup();
    const donorAttrs = fullAttributes({ mapleLife: SEED_ML });
    const donor: Template = { id: "t1", attributes: donorAttrs };
    const originalLength = donorAttrs.mapleLife!.classes.length;

    const onSeed = vi.fn((config: MapleLifeConfig) => {
      config.classes.push({ ...config.classes[0]! });
    });
    setup([donor], onSeed);

    await user.click(screen.getByRole("button", { name: /t1/ }));

    expect(donorAttrs.mapleLife!.classes.length).toBe(originalLength);
  });

  it("filters on all three of region, major and minor", () => {
    setup([
      template("gms-83-1", {
        region: "GMS",
        majorVersion: 83,
        minorVersion: 1,
        mapleLife: SEED_ML,
      }),
      template("gms-83-2", {
        region: "GMS",
        majorVersion: 83,
        minorVersion: 2,
        mapleLife: SEED_ML,
      }),
      template("jms-83-1", {
        region: "JMS",
        majorVersion: 83,
        minorVersion: 1,
        mapleLife: SEED_ML,
      }),
    ]);

    expect(screen.getByText(/gms-83-1/)).toBeInTheDocument();
    expect(screen.queryByText(/gms-83-2/)).not.toBeInTheDocument();
    expect(screen.queryByText(/jms-83-1/)).not.toBeInTheDocument();
  });
});
