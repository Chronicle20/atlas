import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MonsterSkillChip } from "../MonsterSkillChip";
import type { MobSkillDetailAttributes } from "@/services/api/mob-skills.service";

// task-190: mob-skill `duration` became MILLISECONDS on the wire/backend
// (mobskill reader.go is the single seconds->ms conversion point). The
// component divides by 1000 before rendering. These tests assert the
// RENDERED tooltip text, not a re-derivation of the arithmetic, so a
// regression that drops (or re-adds) the /1000 scaling is actually caught.

const fakeTenant = {
  id: "t1",
  attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
};

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: fakeTenant }),
}));
vi.mock("@/lib/hooks/useMobSkillData", () => ({
  useMobSkillData: () => ({ name: "Disease" }),
}));
vi.mock("@/lib/data/mob-skill-names", () => ({
  getMobSkillCanonicalName: () => undefined,
}));

const getMobSkillDetailMock = vi.fn();
vi.mock("@/services/api/mob-skills.service", () => ({
  mobSkillsService: {
    getMobSkillDetail: (...args: [number, number]) =>
      getMobSkillDetailMock(...args),
  },
}));

function detail(
  over: Partial<MobSkillDetailAttributes> = {},
): MobSkillDetailAttributes {
  return {
    name: "Disease",
    mp_con: 0,
    duration: 0,
    hp: 0,
    x: 0,
    y: 0,
    prop: 0,
    interval: 0,
    count: 0,
    limit: 0,
    lt_x: 0,
    lt_y: 0,
    rb_x: 0,
    rb_y: 0,
    summon_effect: 0,
    summons: [],
    ...over,
  };
}

function renderChip() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MonsterSkillChip skillId={5220000} level={1} />
    </QueryClientProvider>,
  );
}

async function openTooltip() {
  const user = userEvent.setup();
  const trigger = screen.getByText(/Disease/);
  await user.hover(trigger);
}

// Radix renders tooltip content twice — the visible copy plus a
// VisuallyHidden one carrying role="tooltip" for screen readers — so every
// label matches two nodes. The visible copy is rendered first; the *All
// queries are what keep these lookups from throwing "Found multiple elements".
async function findTooltipRow(label: string) {
  const [labelNode] = await screen.findAllByText(label);
  return labelNode!.closest("div");
}

describe("MonsterSkillChip", () => {
  beforeEach(() => {
    getMobSkillDetailMock.mockReset();
  });

  it("renders the Duration row scaled from ms to seconds (30000ms -> 30s), not the raw ms value", async () => {
    getMobSkillDetailMock.mockResolvedValueOnce(detail({ duration: 30000 }));
    renderChip();
    await openTooltip();

    const row = await findTooltipRow("Duration");
    await waitFor(() => {
      expect(row).toHaveTextContent("30s");
    });

    // Regression guard: the bug this task fixed was rendering the raw
    // unscaled millisecond value with the seconds suffix appended.
    expect(screen.queryAllByText(/30,000s/)).toHaveLength(0);
    expect(screen.queryAllByText(/30000s/)).toHaveLength(0);
  });

  it("does not render a Duration row when duration is 0", async () => {
    getMobSkillDetailMock.mockResolvedValueOnce(detail({ duration: 0 }));
    renderChip();
    await openTooltip();

    await screen.findAllByText("No derived effect data.");
    expect(screen.queryAllByText("Duration")).toHaveLength(0);
  });

  it("leaves the Interval row unscaled — interval is a separate WZ field this task does not touch", async () => {
    getMobSkillDetailMock.mockResolvedValueOnce(
      detail({ duration: 30000, interval: 7 }),
    );
    renderChip();
    await openTooltip();

    const row = await findTooltipRow("Interval");
    await waitFor(() => {
      expect(row).toHaveTextContent("7s");
    });

    // If a future edit mistakenly applied the same /1000 scaling to
    // `interval` (as it should NOT), 7 would render as "0.007s" instead.
    expect(row).not.toHaveTextContent("0.007s");
  });
});
