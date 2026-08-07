import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { MemoryRouter, useLocation } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";

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

const templates: SocketObject[] = [
  {
    key: "t83",
    label: "GMS v83.1",
    source: "template",
    region: "GMS",
    majorVersion: 83,
    minorVersion: 1,
    handlers: new Map([["LoginHandle", [binding("0x01")]]]),
    writers: new Map([["AuthSuccess", [binding("0x00")]]]),
    unsupportedHandlers: new Set(),
    unsupportedWriters: new Set(),
  },
  {
    key: "t95",
    label: "GMS v95.1",
    source: "template",
    region: "GMS",
    majorVersion: 95,
    minorVersion: 1,
    handlers: new Map([["LoginHandle", [binding("0x01")]]]),
    // Two writer opcodes, three apart, so the baseline has a real range for
    // the opcode-gap scan to find holes in (0x9B and 0x9C).
    writers: new Map([
      ["PetActivated", [binding("0x9A")]],
      ["PetMovement", [binding("0x9D")]],
    ]),
    unsupportedHandlers: new Set(),
    unsupportedWriters: new Set(),
  },
];

vi.mock("@/lib/hooks/api/useSocketObjects", () => ({
  useSocketMatrixTemplates: () => ({
    data: templates,
    isLoading: false,
    error: null,
  }),
  useSocketMatrixTenants: () => ({ data: [], isLoading: false, error: null }),
  useSocketMutation: () => ({ mutateAsync: vi.fn(), isPending: false }),
  socketKeys: { all: ["socket"] },
}));

import { PacketMatrixPage } from "@/pages/PacketMatrixPage";

/**
 * `MemoryRouter` keeps its history internally and never touches
 * `window.location`, so the only way to actually observe a `setSearchParams`
 * write-back is to read the router's own location from inside the router
 * tree - a sibling that calls `useLocation()` and renders the current
 * search string.
 */
function LocationSpy() {
  const location = useLocation();
  return <div data-testid="location-search">{location.search}</div>;
}

function renderPage(initialPath = "/packet-matrix") {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[initialPath]}>{children}</MemoryRouter>
    </QueryClientProvider>
  );
  return render(
    <>
      <PacketMatrixPage />
      <LocationSpy />
    </>,
    { wrapper },
  );
}

describe("PacketMatrixPage", () => {
  it("defaults to handlers mode with the highest-version template as baseline", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByRole("grid")).toBeInTheDocument());
    const header = screen.getByRole("columnheader", { name: /GMS v95\.1/ });
    expect(within(header).getByText(/baseline/i)).toBeInTheDocument();
    expect(
      screen.getByRole("row", { name: /LoginHandle/ }),
    ).toBeInTheDocument();
  });

  it("switches to writers mode and shows the writer row set", async () => {
    renderPage();
    await userEvent.click(screen.getByRole("radio", { name: /writers/i }));
    expect(
      screen.getByRole("row", { name: /PetActivated/ }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("row", { name: /LoginHandle/ }),
    ).not.toBeInTheDocument();
  });

  it("reads mode, baseline and definition from the URL", async () => {
    renderPage("/packet-matrix?mode=writers&baseline=t83&def=AuthSuccess");
    await waitFor(() => expect(screen.getByRole("grid")).toBeInTheDocument());
    const header = screen.getByRole("columnheader", { name: /GMS v83\.1/ });
    expect(within(header).getByText(/baseline/i)).toBeInTheDocument();
    expect(screen.getByRole("row", { name: /AuthSuccess/ })).toHaveAttribute(
      "aria-selected",
      "true",
    );
  });

  it("explains the cell states in a legend below the grid", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByRole("grid")).toBeInTheDocument());
    expect(screen.getByText("Defined")).toBeInTheDocument();
    expect(screen.getByText("Unsupported (audited)")).toBeInTheDocument();
    expect(screen.getByText("Undefined")).toBeInTheDocument();
    expect(screen.getByText(/1 definition ·/)).toBeInTheDocument();
  });

  // Both templates bind LoginHandle at 0x01 only, so the handlers view has
  // no range to scan. The writers view does: baseline v95 spans 0x9A-0x9D,
  // and the baseline binds neither 0x9B nor 0x9C.
  it("inserts blank rows for opcodes the baseline's range leaves unbound", async () => {
    renderPage("/packet-matrix?mode=writers");
    await waitFor(() => expect(screen.getByRole("grid")).toBeInTheDocument());
    expect(screen.getAllByTestId("opcode-gap-row")).toHaveLength(2);
    expect(
      screen.getByRole("row", { name: /0x9B — no definition/ }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("row", { name: /0x9C — no definition/ }),
    ).toBeInTheDocument();
  });

  it("runs no gap scan where the baseline binds fewer than two opcodes", async () => {
    renderPage("/packet-matrix?mode=writers&baseline=t83");
    await waitFor(() => expect(screen.getByRole("grid")).toBeInTheDocument());
    expect(screen.queryAllByTestId("opcode-gap-row")).toHaveLength(0);
  });

  it("drops the gap rows as soon as a filter narrows the view", async () => {
    renderPage("/packet-matrix?mode=writers");
    await waitFor(() => expect(screen.getByRole("grid")).toBeInTheDocument());
    expect(screen.getAllByTestId("opcode-gap-row")).toHaveLength(2);
    await userEvent.click(
      screen.getByRole("button", { name: /options not supplied/i }),
    );
    expect(screen.queryAllByTestId("opcode-gap-row")).toHaveLength(0);
  });

  it("writes the mode back to the URL so the view is shareable", async () => {
    renderPage();
    expect(screen.getByTestId("location-search")).toHaveTextContent("");
    await userEvent.click(screen.getByRole("radio", { name: /writers/i }));
    // The router's own location - not window.location, which MemoryRouter
    // never touches - actually carries the write-back this test is named for.
    expect(screen.getByTestId("location-search")).toHaveTextContent(
      "?mode=writers",
    );
  });
});
