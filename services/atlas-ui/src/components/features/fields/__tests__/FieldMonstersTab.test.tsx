import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";

import { FieldMonstersTab } from "@/components/features/fields/FieldMonstersTab";
import type { LiveMonsterData } from "@/services/api/live-monsters.service";

const MAP_ID = 910340000;

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
      mapId: MAP_ID,
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
    spawnSourceType: "MAP",
    spawnSourceId: "spawn-a",
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
    spawnSourceType: "EVENT",
    spawnSourceId: "boss-1",
  }),
];

function renderTab(data: LiveMonsterData[] | undefined, error?: Error) {
  return render(
    <MemoryRouter>
      <FieldMonstersTab monsters={data} error={error} mapId={MAP_ID} />
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

  it("monster links to the definition", () => {
    renderTab(monsters);

    const row = screen.getByText("9001").closest("tr");
    expect(row).not.toBeNull();
    const link = row!.querySelector("a[href='/monsters/100100']");
    expect(link).not.toBeNull();
  });

  it("spawn is blank-tolerant", () => {
    renderTab(monsters);

    const row = screen.getByText("9002").closest("tr");
    expect(row).not.toBeNull();
    expect(row).toHaveTextContent("—");
  });

  it("spawn links to the definition tab", () => {
    renderTab(monsters);

    const row = screen.getByText("9001").closest("tr");
    expect(row).not.toBeNull();
    const link = row!.querySelector("a[href='/maps/910340000?tab=monsters']");
    expect(link).not.toBeNull();
  });

  it("empty tab", () => {
    renderTab([]);

    expect(
      screen.getByText("No monsters are currently in this field."),
    ).toBeInTheDocument();
  });
});
