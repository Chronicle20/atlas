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
  NoOpHandler: [
    binding("0x17"),
    binding("0x19"),
    binding("0x22"),
    binding("0x24"),
  ],
  Move: [binding("0x29", { options: { types: ["WALK", "STAND"] } })],
});
const b = obj("b", 87, {
  Move: [binding("0x2A", { options: { types: ["WALK"] } })],
});
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
    expect(
      screen.getByRole("button", { name: /edit in GMS v83\.1/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /undefine in GMS v83\.1/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /mark unsupported in GMS v83\.1/i }),
    ).toBeInTheDocument();
  });

  // The matrix is a wide grid; its detail panel docks along the bottom edge
  // so it reads across the same width rather than in a narrow side column.
  it("opens along the bottom edge", () => {
    renderDrawer("NoOpHandler", "a");
    expect(screen.getByRole("dialog").className).toContain("bottom-0");
  });

  it("relabels the actions when the scope moves to another object", () => {
    renderDrawer("Move", "b");
    expect(
      screen.getByRole("button", { name: /edit in GMS v87\.1/i }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /edit in GMS v83\.1/i }),
    ).not.toBeInTheDocument();
  });

  // FR-5.4
  it("disables Edit, Delete and Open where the definition is undefined for the scope", () => {
    renderDrawer("NoOpHandler", "b");
    expect(
      screen.getByRole("button", { name: /edit in GMS v87\.1/i }),
    ).toBeDisabled();
    expect(
      screen.getByRole("button", { name: /undefine in GMS v87\.1/i }),
    ).toBeDisabled();
    expect(
      screen.getByRole("button", { name: /open in GMS v87\.1/i }),
    ).toBeDisabled();
  });

  it("keeps Add, Copy and Mark Unsupported enabled where the definition is undefined", () => {
    renderDrawer("NoOpHandler", "b");
    expect(
      screen.getByRole("button", { name: /^define in GMS v87\.1/i }),
    ).toBeEnabled();
    expect(
      screen.getByRole("button", { name: /copy into GMS v87\.1/i }),
    ).toBeEnabled();
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

  // The Fields tab carries the object, its opcodes and (handlers only) its
  // validator. State is the card's tint plus its accessible label; services
  // and the options shape belong to their own tabs and are NOT repeated here.
  it("shows each object's opcode, state and validator in the Fields tab", () => {
    renderDrawer("Move", "a");
    const fields = screen.getByRole("tabpanel", { name: /fields/i });
    expect(within(fields).getByText("GMS v83.1")).toBeInTheDocument();
    expect(within(fields).getByText("0x29")).toBeInTheDocument();
    expect(
      within(fields).getAllByText("LoggedInValidator").length,
    ).toBeGreaterThan(0);
    expect(
      within(fields).getByRole("listitem", { name: /GMS v83\.1: Defined/i }),
    ).toBeInTheDocument();
    expect(within(fields).queryByText("channel")).not.toBeInTheDocument();
  });

  it("renders the nested per-entry options matrix in the Options tab", async () => {
    renderDrawer("Move", "a");
    await userEvent.click(screen.getByRole("tab", { name: /options/i }));
    const panel = screen.getByRole("tabpanel", { name: /options/i });
    // Positional list: rows keyed by array index, index 1 missing for GMS v87.1.
    expect(
      within(panel).getByRole("rowheader", { name: "0" }),
    ).toBeInTheDocument();
    expect(
      within(panel).getByRole("rowheader", { name: "1" }),
    ).toBeInTheDocument();
    expect(within(panel).getByText("WALK")).toBeInTheDocument();
    expect(within(panel).getByText("STAND")).toBeInTheDocument();
  });

  // Every card is three lines whatever its state, so the row scans as a row:
  // an object with no definition leaves the opcode line blank and spends the
  // footnote line - the one a defined card gives its validator - on the state.
  it("gives a card with no definition a blank opcode line and the state word", () => {
    renderDrawer("NoOpHandler", "a");
    const fields = screen.getByRole("tabpanel", { name: /fields/i });
    const card = within(fields).getByRole("listitem", {
      name: /GMS v87\.1: Undefined/i,
    });
    expect(within(card).getByText("Undefined")).toBeInTheDocument();
    expect(card).not.toHaveTextContent(/0x/);
  });

  // Real corpus shape: a list-shaped `types` table stores an OBJECT per index
  // (gms_87_1 CharacterMoveHandle), which used to render as the literal
  // "[object Object]" - the same string at every index in every column, so
  // the one thing this tab exists to show was unreadable on exactly the rows
  // that diverge.
  it("renders an object-valued options entry as its key/value pairs", async () => {
    const structured = [
      obj("a", 83, {
        Move: [
          binding("0x29", {
            options: { types: [{ Name: "NORMAL", Type: "NORMAL" }] },
          }),
        ],
      }),
      obj("b", 87, {
        Move: [
          binding("0x2A", {
            options: { types: [{ Name: "JUMP", Type: "JUMP" }] },
          }),
        ],
      }),
    ];
    const rows = buildRows({
      objects: structured,
      kind: "handler",
      baselineKey: "a",
    });
    render(
      <DefinitionDrawer
        row={rows.find((r) => r.name === "Move")!}
        objects={structured}
        kind="handler"
        baselineKey="a"
        selection={{ name: "Move", scopeKey: "a" }}
        onClose={vi.fn()}
        onAction={vi.fn()}
      />,
    );
    await userEvent.click(screen.getByRole("tab", { name: /options/i }));
    const panel = screen.getByRole("tabpanel", { name: /options/i });
    expect(
      within(panel).getByText("Name: NORMAL, Type: NORMAL"),
    ).toBeInTheDocument();
    expect(
      within(panel).getByText("Name: JUMP, Type: JUMP"),
    ).toBeInTheDocument();
    expect(
      within(panel).queryByText(/\[object Object\]/),
    ).not.toBeInTheDocument();
  });

  it("marks options entries that differ from or are missing against the baseline", async () => {
    renderDrawer("Move", "a");
    await userEvent.click(screen.getByRole("tab", { name: /options/i }));
    const panel = screen.getByRole("tabpanel", { name: /options/i });
    expect(
      within(panel).getByLabelText(/missing in GMS v87\.1/i),
    ).toBeInTheDocument();
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
    expect(
      within(handlerFields).getAllByText("LoggedInValidator").length,
    ).toBeGreaterThan(0);
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
    expect(
      within(writerFields).queryAllByText("LoggedInValidator"),
    ).toHaveLength(0);
  });

  // Task 10 round-2 fix: `key` is always fully-qualified ("group.entry") and
  // set-independent; `label` collapses to the bare entry name only when every
  // compared object agrees on a single option group.
  it("renders OptionsMatrixTable rows keyed by the qualified key but labelled bare when there is one group", async () => {
    const single = obj("single", 92, {
      CharacterInteraction: [
        binding("0x30", { options: { operations: { INVITE: 0 } } }),
      ],
    });
    render(
      <DefinitionDrawer
        row={buildRows({
          objects: [single],
          kind: "handler",
          baselineKey: "single",
        }).find((r) => r.name === "CharacterInteraction")!}
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
        row={buildRows({
          objects: [multi],
          kind: "handler",
          baselineKey: "multi",
        }).find((r) => r.name === "CharacterInteraction")!}
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
  // colour alone - each renders its own text/aria-label. Each assertion is
  // scoped to the ONE cell it means (found by its own unique aria-label,
  // then asserted on via toHaveTextContent) rather than a bare getByText
  // scan of the whole panel - the latter would only pass because of how many
  // OTHER cells happen to print the same string, which is an accident of
  // this fixture, not a property the test should depend on.
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
        row={buildRows({
          objects: compareObjects,
          kind: "handler",
          baselineKey: "base",
        }).find((r) => r.name === "Move")!}
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
    expect(
      within(panel).getByLabelText(/differs in GMS v87\.1/i),
    ).toHaveTextContent("RUN");
    // index 1: base "STAND" vs diff "STAND" -> same (non-baseline shows "=",
    // not the repeated value - see OptionsMatrix.tsx's cellText doc comment).
    expect(
      within(panel).getByLabelText(/same as baseline in GMS v87\.1/i),
    ).toHaveTextContent("=");
    // index 2: base has no entry, diff has "JUMP" -> extra
    expect(
      within(panel).getByLabelText(/only in GMS v87\.1/i),
    ).toHaveTextContent("JUMP");
  });

  // FINDING 1 fix: a definition with more than one binding has no single
  // unambiguous target for a definition-scoped Edit/Delete/Open - only the
  // per-binding rows (already exercised above) can act without guessing.
  it("disables top-level Edit, Delete and Open when the scope has more than one binding, and explains why", () => {
    renderDrawer("NoOpHandler", "a");
    const editButton = screen.getByRole("button", {
      name: /edit in GMS v83\.1/i,
    });
    const deleteButton = screen.getByRole("button", {
      name: /undefine in GMS v83\.1/i,
    });
    const openButton = screen.getByRole("button", {
      name: /open in GMS v83\.1/i,
    });
    expect(editButton).toBeDisabled();
    expect(deleteButton).toBeDisabled();
    expect(openButton).toBeDisabled();
    expect(editButton.getAttribute("title")).toMatch(/more than one binding/i);
    expect(deleteButton.getAttribute("title")).toMatch(
      /more than one binding/i,
    );
  });

  // FINDING 1 fix: exactly one binding is unambiguous, so the top-level
  // buttons stay enabled AND name the opcode they'd act on - the target is
  // never implicit.
  it("keeps top-level Edit, Delete and Open enabled and names the opcode when the scope has exactly one binding", () => {
    renderDrawer("Move", "a");
    expect(
      screen.getByRole("button", { name: /edit in GMS v83\.1 \(0x29\)/i }),
    ).toBeEnabled();
    expect(
      screen.getByRole("button", { name: /undefine in GMS v83\.1 \(0x29\)/i }),
    ).toBeEnabled();
    expect(
      screen.getByRole("button", { name: /open in GMS v83\.1 \(0x29\)/i }),
    ).toBeEnabled();
  });

  it("dispatches the top-level Edit action carrying the single binding's own opcode", async () => {
    const { onAction } = renderDrawer("Move", "a");
    await userEvent.click(
      screen.getByRole("button", { name: /edit in GMS v83\.1/i }),
    );
    expect(onAction).toHaveBeenCalledWith({
      type: "edit",
      scopeKey: "a",
      name: "Move",
      opCodeValue: 0x29,
    });
  });

  // FINDING 1 fix, second half: `defined` alone is not enough - a defined
  // scope whose only binding's opcode fails to parse has no resolvable
  // target either, so it must be treated the same as multi-binding.
  it("disables top-level Edit, Delete and Open when the scope's only binding has no resolvable opcode", () => {
    const malformed = obj("malformed", 61, {
      Move: [binding("not-a-hex-opcode")],
    });
    const malformedObjects = [malformed];
    render(
      <DefinitionDrawer
        row={buildRows({
          objects: malformedObjects,
          kind: "handler",
          baselineKey: "malformed",
        }).find((r) => r.name === "Move")!}
        objects={malformedObjects}
        kind="handler"
        baselineKey="malformed"
        selection={{ name: "Move", scopeKey: "malformed" }}
        onClose={vi.fn()}
        onAction={vi.fn()}
      />,
    );
    expect(
      screen.getByRole("button", { name: /edit in GMS v61\.1/i }),
    ).toBeDisabled();
    expect(
      screen.getByRole("button", { name: /undefine in GMS v61\.1/i }),
    ).toBeDisabled();
    expect(
      screen.getByRole("button", { name: /open in GMS v61\.1/i }),
    ).toBeDisabled();
  });

  // FINDING 2 fix: the unsupported <-> Clear Unsupported branch had no
  // coverage. A name that is ONLY in `unsupportedHandlers` (no bindings)
  // must show Clear Unsupported, not Mark Unsupported, and it must be
  // enabled and correctly scoped.
  it("shows Clear Unsupported instead of Mark Unsupported when the scope has the definition marked unsupported", () => {
    const legacy = obj("legacy", 61, {}, ["LegacyHandler"]);
    const legacyObjects = [legacy];
    render(
      <DefinitionDrawer
        row={buildRows({
          objects: legacyObjects,
          kind: "handler",
          baselineKey: "legacy",
        }).find((r) => r.name === "LegacyHandler")!}
        objects={legacyObjects}
        kind="handler"
        baselineKey="legacy"
        selection={{ name: "LegacyHandler", scopeKey: "legacy" }}
        onClose={vi.fn()}
        onAction={vi.fn()}
      />,
    );
    expect(
      screen.getByRole("button", { name: /clear unsupported in GMS v61\.1/i }),
    ).toBeEnabled();
    expect(
      screen.queryByRole("button", { name: /mark unsupported in GMS v61\.1/i }),
    ).not.toBeInTheDocument();
  });

  it("dispatches clear-unsupported for the scoped object", async () => {
    const legacy = obj("legacy", 61, {}, ["LegacyHandler"]);
    const legacyObjects = [legacy];
    const onAction = vi.fn();
    render(
      <DefinitionDrawer
        row={buildRows({
          objects: legacyObjects,
          kind: "handler",
          baselineKey: "legacy",
        }).find((r) => r.name === "LegacyHandler")!}
        objects={legacyObjects}
        kind="handler"
        baselineKey="legacy"
        selection={{ name: "LegacyHandler", scopeKey: "legacy" }}
        onClose={vi.fn()}
        onAction={onAction}
      />,
    );
    await userEvent.click(
      screen.getByRole("button", { name: /clear unsupported in GMS v61\.1/i }),
    );
    expect(onAction).toHaveBeenCalledWith({
      type: "clear-unsupported",
      scopeKey: "legacy",
      name: "LegacyHandler",
    });
  });

  // FINDING 2 fix: the ancestor / Reset to Ancestor branch had no coverage
  // either. It must be absent with no `ancestor` prop, and present only once
  // both an ancestor is supplied AND the scope itself is tenant-sourced.
  it("shows Reset to Ancestor only when an ancestor is supplied for a tenant-sourced scope", () => {
    const tenantScope: SocketObject = {
      ...obj("tenant-a", 83, { Move: [binding("0x29")] }),
      source: "tenant",
    };
    const ancestor = obj("ancestor-a", 83, { Move: [binding("0x29")] });
    const tenantObjects = [tenantScope];
    const row = buildRows({
      objects: tenantObjects,
      kind: "handler",
      baselineKey: "tenant-a",
    }).find((r) => r.name === "Move")!;

    const noAncestor = render(
      <DefinitionDrawer
        row={row}
        objects={tenantObjects}
        kind="handler"
        baselineKey="tenant-a"
        selection={{ name: "Move", scopeKey: "tenant-a" }}
        onClose={vi.fn()}
        onAction={vi.fn()}
      />,
    );
    expect(
      screen.queryByRole("button", { name: /reset to ancestor/i }),
    ).not.toBeInTheDocument();
    noAncestor.unmount();

    render(
      <DefinitionDrawer
        row={row}
        objects={tenantObjects}
        kind="handler"
        baselineKey="tenant-a"
        selection={{ name: "Move", scopeKey: "tenant-a" }}
        onClose={vi.fn()}
        onAction={vi.fn()}
        ancestor={ancestor}
      />,
    );
    expect(
      screen.getByRole("button", { name: /reset to ancestor in GMS v83\.1/i }),
    ).toBeInTheDocument();
  });

  it("hides Reset to Ancestor when an ancestor is supplied but the scope is template-sourced", () => {
    const row = buildRows({ objects, kind: "handler", baselineKey: "a" }).find(
      (r) => r.name === "Move",
    )!;
    render(
      <DefinitionDrawer
        row={row}
        objects={objects}
        kind="handler"
        baselineKey="a"
        selection={{ name: "Move", scopeKey: "a" }}
        onClose={vi.fn()}
        onAction={vi.fn()}
        ancestor={obj("ancestor-a", 83, { Move: [binding("0x29")] })}
      />,
    );
    expect(
      screen.queryByRole("button", { name: /reset to ancestor/i }),
    ).not.toBeInTheDocument();
  });

  it("dispatches reset-to-ancestor for the scoped tenant object", async () => {
    const tenantScope: SocketObject = {
      ...obj("tenant-a", 83, { Move: [binding("0x29")] }),
      source: "tenant",
    };
    const ancestor = obj("ancestor-a", 83, { Move: [binding("0x29")] });
    const tenantObjects = [tenantScope];
    const row = buildRows({
      objects: tenantObjects,
      kind: "handler",
      baselineKey: "tenant-a",
    }).find((r) => r.name === "Move")!;
    const onAction = vi.fn();
    render(
      <DefinitionDrawer
        row={row}
        objects={tenantObjects}
        kind="handler"
        baselineKey="tenant-a"
        selection={{ name: "Move", scopeKey: "tenant-a" }}
        onClose={vi.fn()}
        onAction={onAction}
        ancestor={ancestor}
      />,
    );
    await userEvent.click(
      screen.getByRole("button", { name: /reset to ancestor in GMS v83\.1/i }),
    );
    expect(onAction).toHaveBeenCalledWith({
      type: "reset-to-ancestor",
      scopeKey: "tenant-a",
      name: "Move",
    });
  });
});
