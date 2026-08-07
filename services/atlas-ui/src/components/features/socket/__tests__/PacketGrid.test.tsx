import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { PacketGrid } from "@/components/features/socket/PacketGrid";
import { buildRows } from "@/lib/socket/matrix";
import type { Binding, SocketObject } from "@/lib/socket/model";
import { parseOpcode } from "@/lib/socket/opcode";

function binding(opCode: string, extra: Partial<Binding> = {}): Binding {
  return {
    opCode,
    opCodeValue: parseOpcode(opCode),
    services: ["channel"],
    index: 0,
    ...extra,
  };
}

function obj(
  key: string,
  major: number,
  writers: Record<string, Binding[]>,
  unsupportedWriters: string[] = [],
): SocketObject {
  return {
    key,
    label: `GMS v${major}.1`,
    source: "template",
    region: "GMS",
    majorVersion: major,
    minorVersion: 1,
    handlers: new Map(),
    writers: new Map(Object.entries(writers)),
    unsupportedHandlers: new Set(),
    unsupportedWriters: new Set(unsupportedWriters),
  };
}

const a = obj(
  "a",
  83,
  {
    AuthSuccess: [binding("0x00", { fname: "CLogin::OnCheckPasswordResult" })],
    CharacterEffect: [binding("0xE0"), binding("0xE9")],
    MiniRoom: [binding("0xB8"), binding("0x0B8")],
    CharacterMovement: [binding("0xB9", { options: { types: ["WALK"] } })],
  },
  ["MonsterCarnival"],
);
const b = obj("b", 95, { CharacterMovement: [binding("0xC0")] });

function renderGrid(overrides: Partial<Parameters<typeof PacketGrid>[0]> = {}) {
  const objects = [a, b];
  const rows = buildRows({ objects, kind: "writer", baselineKey: "a" });
  const onSelect = vi.fn();
  render(
    <PacketGrid
      rows={rows}
      objects={objects}
      baselineKey="a"
      showFName={false}
      selection={null}
      onSelect={onSelect}
      {...overrides}
    />,
  );
  return { onSelect, rows };
}

describe("PacketGrid", () => {
  it("renders one column per object and no duplicate baseline column", () => {
    renderGrid();
    expect(screen.getAllByText("GMS v83.1")).toHaveLength(1);
    expect(screen.getAllByText("GMS v95.1")).toHaveLength(1);
  });

  it("marks the baseline column in place rather than duplicating it", () => {
    renderGrid();
    const header = screen.getByRole("columnheader", { name: /GMS v83\.1/ });
    expect(within(header).getByText(/baseline/i)).toBeInTheDocument();
  });

  it("renders a single-binding cell as its opcode", () => {
    renderGrid();
    const row = screen.getByRole("row", { name: /AuthSuccess/ });
    expect(within(row).getByText("0x00")).toBeInTheDocument();
  });

  it("renders a multi-binding cell as the lowest opcode plus a count", () => {
    renderGrid();
    const row = screen.getByRole("row", { name: /CharacterEffect/ });
    expect(within(row).getByText("0xE0")).toBeInTheDocument();
    expect(within(row).getByText("+1")).toBeInTheDocument();
  });

  it("marks a cell whose bindings collide numerically", () => {
    renderGrid();
    const row = screen.getByRole("row", { name: /MiniRoom/ });
    expect(within(row).getByLabelText(/duplicate opcode/i)).toBeInTheDocument();
  });

  // State is never conveyed by colour alone.
  it("renders Unsupported as the literal n/a", () => {
    renderGrid();
    const row = screen.getByRole("row", { name: /MonsterCarnival/ });
    expect(within(row).getByText("n/a")).toBeInTheDocument();
  });

  it("renders an options-omission glyph with an accessible label", () => {
    renderGrid();
    const row = screen.getByRole("row", { name: /CharacterMovement/ });
    expect(
      within(row).getByLabelText(/supplies no options/i),
    ).toBeInTheDocument();
  });

  it("hides the fname column until it is toggled on", () => {
    renderGrid();
    expect(
      screen.queryByRole("columnheader", { name: /fname/i }),
    ).not.toBeInTheDocument();
    renderGrid({ showFName: true });
    expect(
      screen.getAllByRole("columnheader", { name: /fname/i }).length,
    ).toBeGreaterThan(0);
  });

  // FR-5.2/5.3: clicking a CELL scopes to that object; clicking the NAME leaves
  // the scope on the baseline.
  it("scopes the selection to the clicked cell's object", async () => {
    const { onSelect } = renderGrid();
    const row = screen.getByRole("row", { name: /CharacterMovement/ });
    await userEvent.click(
      within(row).getByRole("button", { name: /GMS v95\.1/ }),
    );
    expect(onSelect).toHaveBeenCalledWith({
      name: "CharacterMovement",
      scopeKey: "b",
    });
  });

  it("leaves the scope on the baseline when the definition name is clicked", async () => {
    const { onSelect } = renderGrid();
    await userEvent.click(
      screen.getByRole("button", { name: "CharacterMovement" }),
    );
    expect(onSelect).toHaveBeenCalledWith({
      name: "CharacterMovement",
      scopeKey: "a",
    });
  });

  it("exposes the grid and its selection to assistive technology", () => {
    renderGrid({ selection: { name: "AuthSuccess", scopeKey: "a" } });
    expect(screen.getByRole("grid")).toBeInTheDocument();
    const row = screen.getByRole("row", { name: /AuthSuccess/ });
    expect(row).toHaveAttribute("aria-selected", "true");
  });

  // An Undefined cell is where you go to define the definition here, so it
  // has to be clickable - it used to render an empty, zero-height button.
  it("opens the drawer from an Undefined cell", async () => {
    const { onSelect } = renderGrid();
    const row = screen.getByRole("row", { name: /AuthSuccess/ });
    await userEvent.click(
      within(row).getByRole("button", { name: /AuthSuccess in GMS v95\.1/ }),
    );
    expect(onSelect).toHaveBeenCalledWith({
      name: "AuthSuccess",
      scopeKey: "b",
    });
  });

  it("opens the drawer from an Unsupported cell", async () => {
    const { onSelect } = renderGrid();
    const row = screen.getByRole("row", { name: /MonsterCarnival/ });
    await userEvent.click(
      within(row).getByRole("button", {
        name: /MonsterCarnival in GMS v83\.1/,
      }),
    );
    expect(onSelect).toHaveBeenCalledWith({
      name: "MonsterCarnival",
      scopeKey: "a",
    });
  });

  function renderWithGap() {
    const objects = [a, b];
    const rows = buildRows({ objects, kind: "writer", baselineKey: "a" });
    render(
      <PacketGrid
        rows={[...rows, { gap: true, opCodeValue: 0xe5 }]}
        objects={objects}
        baselineKey="a"
        showFName={false}
        selection={null}
        onSelect={vi.fn()}
      />,
    );
    return screen.getByRole("row", { name: /0xE5 — no definition/ });
  }

  it("renders an interleaved opcode-gap row as an unclickable row", () => {
    expect(
      within(renderWithGap()).queryByRole("button"),
    ).not.toBeInTheDocument();
  });

  // The opcode belongs in the baseline column, not spliced into the name -
  // that is the column every other row shows its opcode in, so the baseline's
  // table reads as one column of numbers with visible holes.
  it("says No Definition in the name column and puts the opcode in the baseline cell", () => {
    const gap = renderWithGap();
    expect(
      within(gap).getByRole("rowheader", { name: "No Definition" }),
    ).toBeInTheDocument();
    const cells = within(gap).getAllByRole("gridcell");
    // objects = [a (baseline), b]; no fname column.
    expect(cells[0]).toHaveTextContent("0xE5");
    expect(cells[1]).toHaveTextContent("");
  });

  it("renders an empty-state message when there are no rows", () => {
    render(
      <PacketGrid
        rows={[]}
        objects={[a, b]}
        baselineKey="a"
        showFName={false}
        selection={null}
        onSelect={vi.fn()}
      />,
    );
    expect(screen.getByText(/no definitions match/i)).toBeInTheDocument();
  });
});
