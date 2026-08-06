import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { DefinitionDrawer } from "@/components/features/socket/DefinitionDrawer";
import { buildRows } from "@/lib/socket/matrix";
import type { Binding, SocketObject } from "@/lib/socket/model";
import { parseOpcode } from "@/lib/socket/opcode";

function binding(opCode: string, extra: Partial<Binding> = {}): Binding {
  return {
    opCode,
    opCodeValue: parseOpcode(opCode),
    validator: "LoggedInValidator",
    services: ["channel"],
    index: 0,
    ...extra,
  };
}

function obj(
  key: string,
  major: number,
  handlers: Record<string, Binding[]>,
  unsupportedHandlers: string[] = [],
): SocketObject {
  return {
    key,
    label: `GMS v${major}.1`,
    source: "template",
    region: "GMS",
    majorVersion: major,
    minorVersion: 1,
    handlers: new Map(Object.entries(handlers)),
    writers: new Map(),
    unsupportedHandlers: new Set(unsupportedHandlers),
    unsupportedWriters: new Set(),
  };
}

const a = obj("a", 83, {
  NoOpHandler: [binding("0x17"), binding("0x19"), binding("0x22"), binding("0x24")],
  Move: [binding("0x29", { options: { types: ["WALK", "STAND"] } })],
});
const b = obj("b", 87, { Move: [binding("0x2A", { options: { types: ["WALK"] } })] });
const objects = [a, b];

function renderDrawer(
  name: string,
  scopeKey: string,
  overrides: Partial<Parameters<typeof DefinitionDrawer>[0]> = {},
) {
  const rows = buildRows({ objects, kind: "handler", baselineKey: "a" });
  const row = rows.find((r) => r.name === name)!;
  const onAction = vi.fn();
  render(
    <DefinitionDrawer
      row={row}
      objects={objects}
      kind="handler"
      baselineKey="a"
      selection={{ name, scopeKey }}
      onClose={vi.fn()}
      onAction={onAction}
      {...overrides}
    />,
  );
  return { onAction };
}

describe("DefinitionDrawer", () => {
  it("names the scoped object in every action label", () => {
    renderDrawer("NoOpHandler", "a");
    expect(screen.getByRole("button", { name: /edit in GMS v83\.1/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /delete in GMS v83\.1/i })).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /mark unsupported in GMS v83\.1/i }),
    ).toBeInTheDocument();
  });

  it("relabels the actions when the scope moves to another object", () => {
    renderDrawer("Move", "b");
    expect(screen.getByRole("button", { name: /edit in GMS v87\.1/i })).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /edit in GMS v83\.1/i }),
    ).not.toBeInTheDocument();
  });

  // FR-5.4
  it("disables Edit, Delete and Open where the definition is undefined for the scope", () => {
    renderDrawer("NoOpHandler", "b");
    expect(screen.getByRole("button", { name: /edit in GMS v87\.1/i })).toBeDisabled();
    expect(screen.getByRole("button", { name: /delete in GMS v87\.1/i })).toBeDisabled();
    expect(screen.getByRole("button", { name: /open in GMS v87\.1/i })).toBeDisabled();
  });

  it("keeps Add, Copy and Mark Unsupported enabled where the definition is undefined", () => {
    renderDrawer("NoOpHandler", "b");
    expect(screen.getByRole("button", { name: /add to GMS v87\.1/i })).toBeEnabled();
    expect(screen.getByRole("button", { name: /copy into GMS v87\.1/i })).toBeEnabled();
    expect(
      screen.getByRole("button", { name: /mark unsupported in GMS v87\.1/i }),
    ).toBeEnabled();
  });

  // design §5.1 - all four routes individually addressable.
  it("lists every binding of a multi-binding definition with its own actions", () => {
    renderDrawer("NoOpHandler", "a");
    const list = screen.getByRole("list", { name: /bindings in GMS v83\.1/i });
    expect(within(list).getAllByRole("listitem")).toHaveLength(4);
    expect(within(list).getByText("0x17")).toBeInTheDocument();
    expect(within(list).getByText("0x24")).toBeInTheDocument();
  });

  it("dispatches an action carrying the scope and the addressed binding", async () => {
    const { onAction } = renderDrawer("NoOpHandler", "a");
    const list = screen.getByRole("list", { name: /bindings in GMS v83\.1/i });
    const row = within(list).getAllByRole("listitem")[1]!;
    await userEvent.click(within(row).getByRole("button", { name: /edit/i }));
    expect(onAction).toHaveBeenCalledWith({
      type: "edit",
      scopeKey: "a",
      name: "NoOpHandler",
      opCodeValue: 0x19,
    });
  });

  it("shows each object's state, opcode, validator and services in the Fields tab", () => {
    renderDrawer("Move", "a");
    const fields = screen.getByRole("tabpanel", { name: /fields/i });
    expect(within(fields).getByText("GMS v83.1")).toBeInTheDocument();
    expect(within(fields).getByText("0x29")).toBeInTheDocument();
    expect(within(fields).getAllByText("LoggedInValidator").length).toBeGreaterThan(0);
    expect(within(fields).getAllByText("channel").length).toBeGreaterThan(0);
  });

  it("renders the nested per-entry options matrix in the Options tab", async () => {
    renderDrawer("Move", "a");
    await userEvent.click(screen.getByRole("tab", { name: /options/i }));
    const panel = screen.getByRole("tabpanel", { name: /options/i });
    // Positional list: rows keyed by array index, index 1 missing for GMS v87.1.
    expect(within(panel).getByRole("rowheader", { name: "0" })).toBeInTheDocument();
    expect(within(panel).getByRole("rowheader", { name: "1" })).toBeInTheDocument();
    expect(within(panel).getByText("WALK")).toBeInTheDocument();
    expect(within(panel).getByText("STAND")).toBeInTheDocument();
  });

  it("marks options entries that differ from or are missing against the baseline", async () => {
    renderDrawer("Move", "a");
    await userEvent.click(screen.getByRole("tab", { name: /options/i }));
    const panel = screen.getByRole("tabpanel", { name: /options/i });
    expect(within(panel).getByLabelText(/missing in GMS v87\.1/i)).toBeInTheDocument();
  });

  // FR-5.1: validator is a handlers-only field.
  it("shows the validator for handlers but not for writers", () => {
    const rows = buildRows({ objects, kind: "handler", baselineKey: "a" });
    const row = rows.find((r) => r.name === "Move")!;
    const handlerRender = render(
      <DefinitionDrawer
        row={row}
        objects={objects}
        kind="handler"
        baselineKey="a"
        selection={{ name: "Move", scopeKey: "a" }}
        onClose={vi.fn()}
        onAction={vi.fn()}
      />,
    );
    const handlerFields = screen.getByRole("tabpanel", { name: /fields/i });
    expect(within(handlerFields).getByText("Validator")).toBeInTheDocument();
    handlerRender.unmount();

    const writerObj: SocketObject = {
      ...a,
      handlers: new Map(),
      writers: a.handlers,
    };
    const writerObjects = [writerObj];
    const writerRows = buildRows({
      objects: writerObjects,
      kind: "writer",
      baselineKey: "a",
    });
    const writerRow = writerRows.find((r) => r.name === "Move")!;
    render(
      <DefinitionDrawer
        row={writerRow}
        objects={writerObjects}
        kind="writer"
        baselineKey="a"
        selection={{ name: "Move", scopeKey: "a" }}
        onClose={vi.fn()}
        onAction={vi.fn()}
      />,
    );
    const writerFields = screen.getByRole("tabpanel", { name: /fields/i });
    expect(within(writerFields).queryByText("Validator")).not.toBeInTheDocument();
  });

  // Task 10 round-2 fix: `key` is always fully-qualified ("group.entry") and
  // set-independent; `label` collapses to the bare entry name only when every
  // compared object agrees on a single option group.
  it("renders OptionsMatrixTable rows keyed by the qualified key but labelled bare when there is one group", async () => {
    const single = obj("single", 92, {
      CharacterInteraction: [binding("0x30", { options: { operations: { INVITE: 0 } } })],
    });
    render(
      <DefinitionDrawer
        row={
          buildRows({ objects: [single], kind: "handler", baselineKey: "single" }).find(
            (r) => r.name === "CharacterInteraction",
          )!
        }
        objects={[single]}
        kind="handler"
        baselineKey="single"
        selection={{ name: "CharacterInteraction", scopeKey: "single" }}
        onClose={vi.fn()}
        onAction={vi.fn()}
      />,
    );
    await userEvent.click(screen.getByRole("tab", { name: /options/i }));
    const panel = screen.getByRole("tabpanel", { name: /options/i });
    // label is bare ("INVITE"), not qualified, because this compared set has
    // only one group.
    const cell = within(panel).getByRole("rowheader", { name: "INVITE" });
    expect(cell).toBeInTheDocument();
  });

  it("renders OptionsMatrixTable labels qualified once a second group is present", async () => {
    const multi = obj("multi", 83, {
      CharacterInteraction: [
        binding("0x30", {
          options: { operations: { INVITE: 0 }, enterError: { FULL: 1 } },
        }),
      ],
    });
    render(
      <DefinitionDrawer
        row={
          buildRows({ objects: [multi], kind: "handler", baselineKey: "multi" }).find(
            (r) => r.name === "CharacterInteraction",
          )!
        }
        objects={[multi]}
        kind="handler"
        baselineKey="multi"
        selection={{ name: "CharacterInteraction", scopeKey: "multi" }}
        onClose={vi.fn()}
        onAction={vi.fn()}
      />,
    );
    await userEvent.click(screen.getByRole("tab", { name: /options/i }));
    const panel = screen.getByRole("tabpanel", { name: /options/i });
    expect(
      within(panel).getByRole("rowheader", { name: "operations.INVITE" }),
    ).toBeInTheDocument();
    expect(
      within(panel).getByRole("rowheader", { name: "enterError.FULL" }),
    ).toBeInTheDocument();
  });

  // Every OptionsEntryCellState must be distinguishable without relying on
  // colour alone - each renders its own text/aria-label.
  it("renders differing and extra option cells distinguishably from same and missing", async () => {
    const base = obj("base", 83, {
      Move: [binding("0x29", { options: { types: ["WALK", "STAND"] } })],
    });
    const differing = obj("diff", 87, {
      Move: [binding("0x2A", { options: { types: ["RUN", "STAND", "JUMP"] } })],
    });
    const compareObjects = [base, differing];
    render(
      <DefinitionDrawer
        row={
          buildRows({ objects: compareObjects, kind: "handler", baselineKey: "base" }).find(
            (r) => r.name === "Move",
          )!
        }
        objects={compareObjects}
        kind="handler"
        baselineKey="base"
        selection={{ name: "Move", scopeKey: "base" }}
        onClose={vi.fn()}
        onAction={vi.fn()}
      />,
    );
    await userEvent.click(screen.getByRole("tab", { name: /options/i }));
    const panel = screen.getByRole("tabpanel", { name: /options/i });

    // index 0: base "WALK" vs diff "RUN" -> differs
    expect(within(panel).getByLabelText(/differs in GMS v87\.1/i)).toBeInTheDocument();
    expect(within(panel).getByText("RUN")).toBeInTheDocument();
    // index 1: base "STAND" vs diff "STAND" -> same (non-baseline shows "=")
    expect(within(panel).getByLabelText(/same as baseline in GMS v87\.1/i)).toBeInTheDocument();
    // index 2: base has no entry, diff has "JUMP" -> extra
    expect(within(panel).getByLabelText(/only in GMS v87\.1/i)).toBeInTheDocument();
    expect(within(panel).getByText("JUMP")).toBeInTheDocument();
  });
});
