import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type {
  SocketHandlerEntry,
  SocketWriterEntry,
} from "@/types/models/socket";

const {
  useTemplateMock,
  useTenantConfigurationMock,
  useSocketMatrixTemplatesMock,
  mutateAsync,
} = vi.hoisted(() => ({
  useTemplateMock: vi.fn(),
  useTenantConfigurationMock: vi.fn(),
  useSocketMatrixTemplatesMock: vi.fn(),
  mutateAsync: vi.fn(),
}));

vi.mock("@/lib/hooks/api/useTemplates", () => ({
  useTemplate: useTemplateMock,
}));
vi.mock("@/lib/hooks/api/useTenants", () => ({
  useTenantConfiguration: useTenantConfigurationMock,
}));
vi.mock("@/lib/hooks/api/useSocketObjects", () => ({
  useSocketMatrixTemplates: useSocketMatrixTemplatesMock,
  useSocketMutation: () => ({ mutateAsync, isPending: false }),
  socketKeys: { all: ["socket"] },
}));

import { DefinitionGridPage } from "@/components/features/socket/DefinitionGridPage";

interface Fixture {
  id: string;
  region: string;
  majorVersion: number;
  minorVersion: number;
  handlers?: SocketHandlerEntry[];
  writers?: SocketWriterEntry[];
}

function objectDoc({
  id,
  region,
  majorVersion,
  minorVersion,
  handlers,
  writers,
}: Fixture) {
  return {
    id,
    attributes: {
      region,
      majorVersion,
      minorVersion,
      usesPin: false,
      characters: { templates: [], presets: [] },
      npcs: [],
      worlds: [],
      socket: {
        handlers: handlers ?? [
          {
            opCode: "0x01",
            validator: "LoggedInValidator",
            handler: "LoginHandle",
            services: ["channel"],
          },
        ],
        writers: writers ?? [
          { opCode: "0x00", writer: "AuthSuccess", services: ["channel"] },
        ],
        unsupported: { handlers: [], writers: [] },
      },
    },
  };
}

function renderTenantHandlers(id = "tenant-1") {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[`/tenants/${id}/handlers`]}>
        {children}
      </MemoryRouter>
    </QueryClientProvider>
  );
  return render(
    <Routes>
      <Route
        path="/tenants/:id/handlers"
        element={<DefinitionGridPage kind="handler" scope="tenant" />}
      />
    </Routes>,
    { wrapper },
  );
}

function renderTemplateHandlers(id = "template-1") {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[`/templates/${id}/handlers`]}>
        {children}
      </MemoryRouter>
    </QueryClientProvider>
  );
  return render(
    <Routes>
      <Route
        path="/templates/:id/handlers"
        element={<DefinitionGridPage kind="handler" scope="template" />}
      />
    </Routes>,
    { wrapper },
  );
}

beforeEach(() => {
  mutateAsync.mockReset().mockResolvedValue(undefined);
  // `enabled: !!id` disables whichever of the two detail queries doesn't
  // match the page's scope, but both hooks are still called unconditionally
  // (Rules of Hooks) - give both a safe default so the disabled one doesn't
  // read `.data` off `undefined`.
  useTemplateMock
    .mockReset()
    .mockReturnValue({ data: undefined, isLoading: false, error: null });
  useTenantConfigurationMock
    .mockReset()
    .mockReturnValue({ data: undefined, isLoading: false, error: null });
  useSocketMatrixTemplatesMock
    .mockReset()
    .mockReturnValue({ data: [], isLoading: false, error: null });
});

describe("DefinitionGridPage - per-object controls (FR-7.3)", () => {
  it("renders the grid without the mode switch, column picker or baseline selector", async () => {
    useTemplateMock.mockReturnValue({
      data: objectDoc({
        id: "template-1",
        region: "GMS",
        majorVersion: 83,
        minorVersion: 1,
      }),
      isLoading: false,
      error: null,
    });

    renderTemplateHandlers();

    await waitFor(() => expect(screen.getByRole("grid")).toBeInTheDocument());
    expect(
      screen.queryByRole("radiogroup", { name: /mode/i }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /^columns/i }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /^baseline/i }),
    ).not.toBeInTheDocument();
  });
});

describe("DefinitionGridPage - Tenant ancestry (FR-7.2/FR-8.2)", () => {
  it("renders the inferred ancestor as a second column when one matches", async () => {
    useTenantConfigurationMock.mockReturnValue({
      data: objectDoc({
        id: "tenant-1",
        region: "GMS",
        majorVersion: 83,
        minorVersion: 1,
      }),
      isLoading: false,
      error: null,
    });
    useSocketMatrixTemplatesMock.mockReturnValue({
      data: [
        {
          key: "template-1",
          label: "GMS v83.1",
          source: "template",
          region: "GMS",
          majorVersion: 83,
          minorVersion: 1,
          handlers: new Map([
            [
              "LoginHandle",
              [
                {
                  opCode: "0x01",
                  opCodeValue: 1,
                  validator: "LoggedInValidator",
                  services: ["channel"],
                  index: 0,
                },
              ],
            ],
          ]),
          writers: new Map(),
          unsupportedHandlers: new Set(),
          unsupportedWriters: new Set(),
        },
      ],
      isLoading: false,
      error: null,
    });

    renderTenantHandlers();

    await waitFor(() => expect(screen.getByRole("grid")).toBeInTheDocument());
    // Definition + the tenant's own column + the read-only ancestor column.
    expect(screen.getAllByRole("columnheader")).toHaveLength(3);
    expect(
      screen.getAllByRole("columnheader", { name: /GMS v83\.1/ }),
    ).toHaveLength(2);
    // The ancestry filter is only offered once an ancestor is inferred.
    expect(
      screen.getByRole("button", { name: /vs template/i }),
    ).toBeInTheDocument();
  });

  it("disables the ancestor column's mutating drawer actions but leaves the tenant's own column editable", async () => {
    useTenantConfigurationMock.mockReturnValue({
      data: objectDoc({
        id: "tenant-1",
        region: "GMS",
        majorVersion: 83,
        minorVersion: 1,
      }),
      isLoading: false,
      error: null,
    });
    useSocketMatrixTemplatesMock.mockReturnValue({
      data: [
        {
          key: "template-1",
          label: "GMS v83.1",
          source: "template",
          region: "GMS",
          majorVersion: 83,
          minorVersion: 1,
          handlers: new Map([
            [
              "LoginHandle",
              [
                {
                  opCode: "0x01",
                  opCodeValue: 1,
                  validator: "LoggedInValidator",
                  services: ["channel"],
                  index: 0,
                },
              ],
            ],
          ]),
          writers: new Map(),
          unsupportedHandlers: new Set(),
          unsupportedWriters: new Set(),
        },
      ],
      isLoading: false,
      error: null,
    });

    renderTenantHandlers();
    await waitFor(() => expect(screen.getByRole("grid")).toBeInTheDocument());

    // objects = [tenantObject, ancestor] - the cell buttons appear in that
    // column order, and both columns share the same label since the
    // ancestor is inferred by an EXACT (region, majorVersion, minorVersion)
    // match, so index (not name) is what distinguishes them here.
    const cells = screen.getAllByRole("button", {
      name: /LoginHandle in GMS v83\.1/,
    });
    expect(cells).toHaveLength(2);

    // Scope the drawer to the read-only ancestor column (index 1).
    await userEvent.click(cells[1]!);
    const addOnAncestor = screen.getByRole("button", {
      name: /^Add binding to GMS v83\.1…$/,
    });
    expect(addOnAncestor).toBeDisabled();
    expect(addOnAncestor).toHaveAttribute(
      "title",
      expect.stringMatching(/ancestor template/i),
    );
    expect(
      screen.getByRole("button", { name: /^Copy into GMS v83\.1…$/ }),
    ).toBeDisabled();
    expect(
      screen.getByRole("button", { name: /^Mark unsupported in GMS v83\.1…$/ }),
    ).toBeDisabled();
  });

  it("leaves the tenant's own column fully editable in the drawer", async () => {
    useTenantConfigurationMock.mockReturnValue({
      data: objectDoc({
        id: "tenant-1",
        region: "GMS",
        majorVersion: 83,
        minorVersion: 1,
      }),
      isLoading: false,
      error: null,
    });
    useSocketMatrixTemplatesMock.mockReturnValue({
      data: [
        {
          key: "template-1",
          label: "GMS v83.1",
          source: "template",
          region: "GMS",
          majorVersion: 83,
          minorVersion: 1,
          handlers: new Map([
            [
              "LoginHandle",
              [
                {
                  opCode: "0x01",
                  opCodeValue: 1,
                  validator: "LoggedInValidator",
                  services: ["channel"],
                  index: 0,
                },
              ],
            ],
          ]),
          writers: new Map(),
          unsupportedHandlers: new Set(),
          unsupportedWriters: new Set(),
        },
      ],
      isLoading: false,
      error: null,
    });

    renderTenantHandlers();
    await waitFor(() => expect(screen.getByRole("grid")).toBeInTheDocument());

    const cells = screen.getAllByRole("button", {
      name: /LoginHandle in GMS v83\.1/,
    });
    // Scope the drawer to the tenant's own column (index 0) instead.
    await userEvent.click(cells[0]!);
    expect(
      screen.getByRole("button", { name: /^Add binding to GMS v83\.1…$/ }),
    ).toBeEnabled();
    expect(
      screen.getByRole("button", { name: /^Copy into GMS v83\.1…$/ }),
    ).toBeEnabled();
    expect(
      screen.getByRole("button", { name: /^Mark unsupported in GMS v83\.1…$/ }),
    ).toBeEnabled();
  });

  it("renders a single column with ancestry affordances absent when no Template matches", async () => {
    useTenantConfigurationMock.mockReturnValue({
      data: objectDoc({
        id: "tenant-1",
        region: "GMS",
        majorVersion: 999,
        minorVersion: 1,
      }),
      isLoading: false,
      error: null,
    });
    useSocketMatrixTemplatesMock.mockReturnValue({
      data: [],
      isLoading: false,
      error: null,
    });

    renderTenantHandlers();

    await waitFor(() => expect(screen.getByRole("grid")).toBeInTheDocument());
    // Definition column + exactly the tenant's own single object column.
    expect(screen.getAllByRole("columnheader")).toHaveLength(2);
    expect(
      screen.queryByRole("button", { name: /vs template/i }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /copy missing from ancestor/i }),
    ).not.toBeInTheDocument();
  });
});

describe("DefinitionGridPage - missing validator banner (FR-11.4)", () => {
  it("shows the banner with the empty-validator count when the document has one", async () => {
    useTemplateMock.mockReturnValue({
      data: objectDoc({
        id: "template-1",
        region: "GMS",
        majorVersion: 83,
        minorVersion: 1,
        handlers: [
          {
            opCode: "0x01",
            validator: "",
            handler: "LoginHandle",
            services: ["channel"],
          },
          {
            opCode: "0x02",
            validator: "   ",
            handler: "PongHandle",
            services: ["channel"],
          },
          {
            opCode: "0x03",
            validator: "LoggedInValidator",
            handler: "MoveHandle",
            services: ["channel"],
          },
        ],
      }),
      isLoading: false,
      error: null,
    });

    renderTemplateHandlers();

    await waitFor(() => expect(screen.getByRole("grid")).toBeInTheDocument());
    // Anchored so it doesn't also match the "Fill missing validators…" button.
    expect(screen.getByText(/^missing validators$/i)).toBeInTheDocument();
    expect(screen.getByText(/2 handler entries/i)).toBeInTheDocument();
  });

  it("does not show the banner when every handler has a validator", async () => {
    useTemplateMock.mockReturnValue({
      data: objectDoc({
        id: "template-1",
        region: "GMS",
        majorVersion: 83,
        minorVersion: 1,
        handlers: [
          {
            opCode: "0x01",
            validator: "LoggedInValidator",
            handler: "LoginHandle",
            services: ["channel"],
          },
        ],
      }),
      isLoading: false,
      error: null,
    });

    renderTemplateHandlers();

    await waitFor(() => expect(screen.getByRole("grid")).toBeInTheDocument());
    expect(screen.queryByText(/missing validators/i)).not.toBeInTheDocument();
  });
});

describe("DefinitionGridPage - deep link (FR-12.2)", () => {
  it("pre-selects the row named by ?def= without forcing the drawer open", async () => {
    useTemplateMock.mockReturnValue({
      data: objectDoc({
        id: "template-1",
        region: "GMS",
        majorVersion: 83,
        minorVersion: 1,
        handlers: [
          {
            opCode: "0x01",
            validator: "LoggedInValidator",
            handler: "LoginHandle",
            services: ["channel"],
          },
          {
            opCode: "0x02",
            validator: "LoggedInValidator",
            handler: "PongHandle",
            services: ["channel"],
          },
        ],
      }),
      isLoading: false,
      error: null,
    });

    const qc = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={qc}>
        <MemoryRouter
          initialEntries={["/templates/template-1/handlers?def=PongHandle"]}
        >
          {children}
        </MemoryRouter>
      </QueryClientProvider>
    );
    render(
      <Routes>
        <Route
          path="/templates/:id/handlers"
          element={<DefinitionGridPage kind="handler" scope="template" />}
        />
      </Routes>,
      { wrapper },
    );

    await waitFor(() => expect(screen.getByRole("grid")).toBeInTheDocument());
    // The grid stays queryable - no modal Sheet swallowed the background by
    // auto-opening the drawer from the deep link.
    expect(screen.getByRole("row", { name: /PongHandle/ })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    // "grid filtered to it" (FR-12.2): the non-matching row drops out.
    expect(
      screen.queryByRole("row", { name: /LoginHandle/ }),
    ).not.toBeInTheDocument();
  });
});
