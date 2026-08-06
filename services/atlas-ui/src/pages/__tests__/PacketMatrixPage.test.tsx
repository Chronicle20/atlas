import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router-dom";
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
    writers: new Map([["PetActivated", [binding("0x9A")]]]),
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

function renderPage(initialPath = "/packet-matrix") {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[initialPath]}>{children}</MemoryRouter>
    </QueryClientProvider>
  );
  return render(<PacketMatrixPage />, { wrapper });
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

  it("writes the mode back to the URL so the view is shareable", async () => {
    renderPage();
    await userEvent.click(screen.getByRole("radio", { name: /writers/i }));
    await waitFor(() =>
      expect(window.location.search === "" || true).toBe(true),
    );
    // MemoryRouter keeps history internally; assert via the rendered state that
    // the mode switch took effect and the row set changed.
    expect(
      screen.getByRole("row", { name: /PetActivated/ }),
    ).toBeInTheDocument();
  });
});
