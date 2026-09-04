import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";

import { FieldMonstersTab } from "@/components/features/fields/FieldMonstersTab";
import type { LiveMonsterData } from "@/services/api/live-monsters.service";

const MOB_NAMES: Record<number, string | undefined> = {
  100100: "Snail",
  100101: undefined,
};

vi.mock("@/lib/hooks/useMobData", () => ({
  useMobData: (mobId: number) => ({ name: MOB_NAMES[mobId] }),
}));

function makeMonster(
  id: string,
  overrides: Partial<LiveMonsterData["attributes"]> = {},
): LiveMonsterData {
  return {
    id,
    type: "live-monsters",
    attributes: {
      worldId: 0,
      channelId: 1,
      mapId: 910340000,
      instance: "00000000-0000-0000-0000-000000000000",
      monsterId: 100100,
      controlCharacterId: 0,
      x: 0,
      y: 0,
      fh: 0,
      stance: 0,
      team: -1,
      maxHp: 250,
      hp: 250,
      maxMp: 0,
      mp: 0,
      damageEntries: [],
      experienceEntries: [],
      statusEffects: [],
      controllerHasAggro: false,
      ...overrides,
    },
  };
}

const monsters: LiveMonsterData[] = [
  makeMonster("9001", {
    monsterId: 100100,
    hp: 250,
    maxHp: 250,
    x: 640,
    y: 120,
  }),
  makeMonster("9002", {
    monsterId: 100100,
    hp: 0,
    maxHp: 250,
    x: 700,
    y: 120,
  }),
  makeMonster("9003", {
    monsterId: 100101,
    hp: 40,
    maxHp: 500,
    x: -30,
    y: 45,
  }),
];

function renderTab(data: LiveMonsterData[] | undefined, error?: Error) {
  return render(
    <MemoryRouter>
      <FieldMonstersTab monsters={data} error={error} />
    </MemoryRouter>,
  );
}

describe("FieldMonstersTab", () => {
  it("renders one row per monster", () => {
    renderTab(monsters);

    expect(screen.getByText("9001")).toBeInTheDocument();
    expect(screen.getByText("9002")).toBeInTheDocument();
    expect(screen.getByText("9003")).toBeInTheDocument();
  });

  it("shows current and max HP", () => {
    renderTab(monsters);

    const row = screen.getByText("9003").closest("tr");
    expect(row).not.toBeNull();
    expect(row).toHaveTextContent("40");
    expect(row).toHaveTextContent("500");
  });

  it("shows position", () => {
    renderTab(monsters);

    const row = screen.getByText("9003").closest("tr");
    expect(row).not.toBeNull();
    expect(row).toHaveTextContent("-30");
    expect(row).toHaveTextContent("45");
  });

  it("dead monsters are visually distinguished", () => {
    renderTab(monsters);

    const deadRow = screen.getByText("9002").closest("tr");
    const aliveRow = screen.getByText("9001").closest("tr");
    expect(deadRow).not.toBeNull();
    expect(aliveRow).not.toBeNull();
    expect(deadRow).toHaveAttribute("data-dead", "true");
    expect(aliveRow).not.toHaveAttribute("data-dead", "true");
  });

  it("has no State column", () => {
    renderTab(monsters);

    expect(screen.queryByText("State")).not.toBeInTheDocument();
  });

  it("has no Spawn column", () => {
    renderTab(monsters);

    expect(
      screen.queryByRole("columnheader", { name: "Spawn" }),
    ).not.toBeInTheDocument();
  });

  it("renders the resolved monster name as a badge, linked to the definition", () => {
    renderTab(monsters);

    const row = screen.getByText("9001").closest("tr");
    expect(row).not.toBeNull();
    const link = row!.querySelector("a[href='/monsters/100100']");
    expect(link).not.toBeNull();
    expect(link).toHaveTextContent("Snail");
  });

  it("falls back to the raw template id when the name resolver has no name", () => {
    renderTab(monsters);

    const row = screen.getByText("9003").closest("tr");
    expect(row).not.toBeNull();
    const link = row!.querySelector("a[href='/monsters/100101']");
    expect(link).not.toBeNull();
    expect(link).toHaveTextContent("100101");
  });

  it("carries the template id in a copyable tooltip", async () => {
    const user = userEvent.setup();
    renderTab(monsters);

    expect(screen.queryByText("100100")).not.toBeInTheDocument();
    const row = screen.getByText("9001").closest("tr");
    expect(row).not.toBeNull();
    await user.hover(row!.querySelector("a[href='/monsters/100100']")!);
    expect(await screen.findByText("100100")).toBeInTheDocument();
  });

  it("empty tab", () => {
    renderTab([]);

    expect(
      screen.getByText("No monsters are currently in this field."),
    ).toBeInTheDocument();
  });

  it("does not show the empty state while the query is still loading (B3)", () => {
    renderTab(undefined);

    expect(
      screen.queryByText("No monsters are currently in this field."),
    ).not.toBeInTheDocument();
    expect(screen.getByText("Loading monsters...")).toBeInTheDocument();
  });

  it("shows the empty state once the query resolves with no monsters", () => {
    renderTab([]);

    expect(screen.queryByText("Loading monsters...")).not.toBeInTheDocument();
    expect(
      screen.getByText("No monsters are currently in this field."),
    ).toBeInTheDocument();
  });
});
