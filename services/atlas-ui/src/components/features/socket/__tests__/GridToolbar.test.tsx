import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

// Radix Select (used by the column picker's Region/Version filters) relies
// on DOM APIs jsdom does not implement - same polyfill already used by
// IdentitySection.test.tsx wherever a Radix Select is exercised.
Element.prototype.hasPointerCapture ||= () => false;
Element.prototype.scrollIntoView ||= () => {};

import { GridToolbar } from "@/components/features/socket/GridToolbar";
import { emptyFilters } from "@/lib/socket/matrix";
import type { GridFilters } from "@/lib/socket/matrix";
import type { SocketObject } from "@/lib/socket/model";

function obj(key: string, major: number): SocketObject {
  return {
    key,
    label: `GMS v${major}.1`,
    source: "template",
    region: "GMS",
    majorVersion: major,
    minorVersion: 1,
    handlers: new Map(),
    writers: new Map(),
    unsupportedHandlers: new Set(),
    unsupportedWriters: new Set(),
  };
}

const objects = [obj("a", 83), obj("b", 95)];

function renderToolbar(
  overrides: Partial<Parameters<typeof GridToolbar>[0]> = {},
) {
  const onFiltersChange = vi.fn();
  const props = {
    kind: "writer" as const,
    objects,
    selectedKeys: ["a", "b"],
    baselineKey: "b",
    showFName: false,
    onShowFNameChange: vi.fn(),
    filters: emptyFilters(),
    onFiltersChange,
    sort: { key: "opcode" as const, direction: "asc" as const },
    onSortChange: vi.fn(),
    ...overrides,
  };
  render(<GridToolbar {...props} />);
  // `overrides` never supplies `onFiltersChange` in this file, but its
  // declared type on GridToolbarProps still widens the merged object's
  // property to a union that drops the concrete Mock type (and `.mock`
  // with it) - return the local Mock reference directly so assertions
  // that read `.mock.calls` stay typed.
  return { ...props, onFiltersChange };
}

describe("GridToolbar", () => {
  it("renders the mode switch only when a handler is supplied", () => {
    renderToolbar({ onKindChange: vi.fn() });
    expect(
      screen.getByRole("radio", { name: /handlers/i }),
    ).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: /writers/i })).toBeInTheDocument();
  });

  it("renders the column picker only when a handler is supplied", () => {
    renderToolbar({ onSelectedKeysChange: vi.fn() });
    expect(
      screen.getByRole("button", { name: /columns/i }),
    ).toBeInTheDocument();
  });

  it("renders the baseline selector only when a handler is supplied", () => {
    renderToolbar({ onBaselineChange: vi.fn() });
    expect(
      screen.getByRole("button", { name: /baseline/i }),
    ).toBeInTheDocument();
  });

  // FR-7.3: on the four per-object pages these controls are ABSENT, not disabled.
  it("omits the mode switch, column picker and baseline selector on a locked page", () => {
    renderToolbar();
    expect(
      screen.queryByRole("radio", { name: /handlers/i }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /columns/i }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /baseline/i }),
    ).not.toBeInTheDocument();
  });

  it("reports a search query", () => {
    // GridToolbar is fully controlled (owns no internal query state), so a
    // real <input value=...> resets to the unchanged `filters.query` prop
    // after every keystroke event when nothing re-supplies a new prop
    // in between - exactly React's controlled-input contract. A single
    // change event is the accurate simulation of "the field reports a new
    // value"; simulating keystroke-by-keystroke typing here would only be
    // exercising that same controlled-reset behavior, not the toolbar.
    const props = renderToolbar();
    fireEvent.change(screen.getByRole("searchbox"), {
      target: { value: "Login" },
    });
    expect(props.onFiltersChange).toHaveBeenCalled();
    const last = props.onFiltersChange.mock.calls.at(-1)![0] as GridFilters;
    expect(last.query).toBe("Login");
  });

  it("toggles the fname column", async () => {
    const props = renderToolbar();
    await userEvent.click(screen.getByRole("switch", { name: /fname/i }));
    expect(props.onShowFNameChange).toHaveBeenCalledWith(true);
  });

  it("changes the mode via the switch", async () => {
    const onKindChange = vi.fn();
    renderToolbar({ onKindChange });
    await userEvent.click(screen.getByRole("radio", { name: /handlers/i }));
    expect(onKindChange).toHaveBeenCalledWith("handler");
  });

  it("changes selected columns when a handler is supplied", async () => {
    const onSelectedKeysChange = vi.fn();
    renderToolbar({ onSelectedKeysChange });
    await userEvent.click(screen.getByRole("button", { name: /columns/i }));
    await userEvent.click(
      screen.getByRole("checkbox", { name: /gms v95\.1/i }),
    );
    expect(onSelectedKeysChange).toHaveBeenCalledWith(["a"]);
  });

  it("narrows the column picker's object list by version (FR-2.12)", async () => {
    renderToolbar({ onSelectedKeysChange: vi.fn() });
    await userEvent.click(screen.getByRole("button", { name: /columns/i }));
    expect(screen.getAllByRole("checkbox", { name: /^gms v/i })).toHaveLength(
      2,
    );

    await userEvent.click(
      screen.getByRole("combobox", { name: /filter columns by version/i }),
    );
    await userEvent.click(screen.getByRole("option", { name: "83.1" }));

    const remaining = screen.getAllByRole("checkbox", { name: /^gms v/i });
    expect(remaining).toHaveLength(1);
    expect(
      screen.getByRole("checkbox", { name: "GMS v83.1" }),
    ).toBeInTheDocument();
  });

  it("names the current baseline on the selector's trigger", () => {
    renderToolbar({ onBaselineChange: vi.fn() });
    expect(
      screen.getByRole("button", { name: /baseline:\s*GMS v95\.1/i }),
    ).toBeInTheDocument();
  });

  it("changes the baseline when a selector is supplied", async () => {
    const onBaselineChange = vi.fn();
    renderToolbar({ onBaselineChange });
    await userEvent.click(screen.getByRole("button", { name: /baseline/i }));
    await userEvent.click(screen.getByRole("option", { name: "GMS v83.1" }));
    expect(onBaselineChange).toHaveBeenCalledWith("a");
  });

  it("reports a state filter", async () => {
    const props = renderToolbar();
    await userEvent.click(screen.getByRole("button", { name: /state/i }));
    await userEvent.click(screen.getByRole("option", { name: /unsupported/i }));
    const last = props.onFiltersChange.mock.calls.at(-1)![0] as GridFilters;
    expect(last.states).toContain("unsupported");
  });

  // A boolean filter, so its chip is the toggle itself - no popover to open.
  it("reports the options-omission filter", async () => {
    const props = renderToolbar();
    const chip = screen.getByRole("button", { name: /options not supplied/i });
    expect(chip).toHaveAttribute("aria-pressed", "false");
    await userEvent.click(chip);
    const last = props.onFiltersChange.mock.calls.at(-1)![0] as GridFilters;
    expect(last.optionsMissingOnly).toBe(true);
  });

  it("marks an active filter chip and offers a clear affordance", () => {
    renderToolbar({
      filters: { ...emptyFilters(), states: ["defined", "undefined"] },
    });
    expect(
      screen.getByRole("button", { name: "State: Defined, Undefined" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /clear state filter/i }),
    ).toBeInTheDocument();
  });

  it("clears one filter without touching the others", async () => {
    const filters: GridFilters = {
      ...emptyFilters(),
      states: ["defined"],
      services: ["login"],
    };
    const props = renderToolbar({ filters });
    await userEvent.click(
      screen.getByRole("button", { name: /clear state filter/i }),
    );
    const last = props.onFiltersChange.mock.calls.at(-1)![0] as GridFilters;
    expect(last.states).toEqual([]);
    expect(last.services).toEqual(["login"]);
  });

  it("offers exactly the two real service options", async () => {
    renderToolbar();
    await userEvent.click(screen.getByRole("button", { name: /^service/i }));
    const options = screen.getAllByRole("option");
    expect(options.map((o) => o.textContent)).toEqual(["Login", "Channel"]);
  });

  it("reports a service filter", async () => {
    const props = renderToolbar();
    await userEvent.click(screen.getByRole("button", { name: /^service/i }));
    await userEvent.click(screen.getByRole("option", { name: /login/i }));
    const last = props.onFiltersChange.mock.calls.at(-1)![0] as GridFilters;
    expect(last.services).toContain("login");
  });

  it("reports a has-options filter", async () => {
    const props = renderToolbar();
    await userEvent.click(screen.getByRole("button", { name: /has options/i }));
    await userEvent.click(screen.getByRole("option", { name: "Has options" }));
    const last = props.onFiltersChange.mock.calls.at(-1)![0] as GridFilters;
    expect(last.hasOptions).toBe(true);
  });

  it("changes the sort key", async () => {
    const props = renderToolbar();
    await userEvent.click(screen.getByRole("radio", { name: /^name$/i }));
    expect(props.onSortChange).toHaveBeenCalledWith({
      key: "name",
      direction: "asc",
    });
  });

  it("toggles the sort direction", async () => {
    const props = renderToolbar();
    await userEvent.click(
      screen.getByRole("button", { name: /toggle sort direction/i }),
    );
    expect(props.onSortChange).toHaveBeenCalledWith({
      key: "opcode",
      direction: "desc",
    });
  });

  it("renders the ancestry filter only when its options are supplied", async () => {
    renderToolbar();
    expect(
      screen.queryByRole("button", { name: /vs template/i }),
    ).not.toBeInTheDocument();

    const onAncestryChange = vi.fn();
    renderToolbar({
      ancestryFilterOptions: { value: [], onChange: onAncestryChange },
    });
    expect(
      screen.getByRole("button", { name: /vs template/i }),
    ).toBeInTheDocument();
  });

  it("reports an ancestry filter change", async () => {
    const onAncestryChange = vi.fn();
    renderToolbar({
      ancestryFilterOptions: { value: [], onChange: onAncestryChange },
    });
    await userEvent.click(screen.getByRole("button", { name: /vs template/i }));
    await userEvent.click(screen.getByRole("option", { name: /^modified$/i }));
    expect(onAncestryChange).toHaveBeenCalledWith(["modified"]);
  });

  it("does not mutate the filters object passed in", async () => {
    const filters = emptyFilters();
    const before = JSON.stringify(filters);
    renderToolbar({ filters });
    await userEvent.click(screen.getByRole("button", { name: /state/i }));
    await userEvent.click(screen.getByRole("option", { name: /unsupported/i }));
    expect(JSON.stringify(filters)).toBe(before);
  });

  it("round-trips through emptyFilters when a filter is toggled on and back off", async () => {
    let filters = emptyFilters();
    const onFiltersChange = vi.fn((next: GridFilters) => {
      filters = next;
    });
    const { rerender } = render(
      <GridToolbar
        kind="writer"
        objects={objects}
        selectedKeys={["a", "b"]}
        baselineKey="b"
        showFName={false}
        onShowFNameChange={vi.fn()}
        filters={filters}
        onFiltersChange={onFiltersChange}
        sort={{ key: "opcode", direction: "asc" }}
        onSortChange={vi.fn()}
      />,
    );

    // `/^state/i` rather than `/state/i`: once the filter is active the chip
    // grows a "Clear State filter" sibling, which an unanchored match would
    // find too.
    await userEvent.click(screen.getByRole("button", { name: /^state/i }));
    await userEvent.click(screen.getByRole("option", { name: /unsupported/i }));
    // Close the popover explicitly rather than relying on its open/closed
    // state across the rerender - the trigger button toggles, so a stray
    // "still open" popover would make the next click close it instead of
    // reopening it.
    await userEvent.keyboard("{Escape}");
    rerender(
      <GridToolbar
        kind="writer"
        objects={objects}
        selectedKeys={["a", "b"]}
        baselineKey="b"
        showFName={false}
        onShowFNameChange={vi.fn()}
        filters={filters}
        onFiltersChange={onFiltersChange}
        sort={{ key: "opcode", direction: "asc" }}
        onSortChange={vi.fn()}
      />,
    );

    await userEvent.click(screen.getByRole("button", { name: /^state/i }));
    await userEvent.click(screen.getByRole("option", { name: /unsupported/i }));

    expect(filters).toEqual(emptyFilters());
  });
});
