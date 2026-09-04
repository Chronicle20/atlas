import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import type { CellContext } from "@tanstack/react-table";
import type {
  DataTableColumnDef,
  DataTableFeatures,
} from "@/components/data-table-features";
import { describe, it, expect, vi } from "vitest";
import { getColumns } from "@/pages/tenants-columns";
import type { Tenant, TenantConfig } from "@/types/models/tenant";

function makeTenant(id: string): Tenant {
  return {
    id,
    attributes: {
      name: `Tenant ${id}`,
      region: "GMS",
      majorVersion: 83,
      minorVersion: 1,
    },
  } as unknown as Tenant;
}

type ActionsProps = {
  onDelete?: (id: string) => void;
  onRename?: (id: string) => void;
};

function renderActionsCell(props: ActionsProps) {
  const columns = getColumns(props);
  const actions = columns.find(
    (c: DataTableColumnDef<Tenant>) => c.id === "actions",
  );
  if (!actions || typeof actions.cell !== "function") {
    throw new Error("actions column missing");
  }

  const tenant = makeTenant("abc");
  const ctx = {
    row: {
      original: tenant,
      getValue: (key: string) =>
        (tenant as unknown as Record<string, unknown>)[key],
    },
    column: { id: "actions" },
  } as unknown as CellContext<DataTableFeatures, Tenant, string>;

  const CellComponent = actions.cell;
  const node = CellComponent(ctx) as React.ReactNode;
  return render(<>{node}</>);
}

describe("tenants-columns actions menu", () => {
  it("renders Rename menu item when onRename is provided and invokes with correct id", async () => {
    const onRename = vi.fn();
    const onDelete = vi.fn();
    renderActionsCell({ onDelete, onRename });

    await userEvent.click(screen.getByRole("button", { name: /open menu/i }));
    const renameItem = await screen.findByText("Rename");
    await userEvent.click(renameItem);

    expect(onRename).toHaveBeenCalledWith("abc");
    expect(onDelete).not.toHaveBeenCalled();
  });

  it("omits Rename when onRename is not provided", async () => {
    const onDelete = vi.fn();
    renderActionsCell({ onDelete });

    await userEvent.click(screen.getByRole("button", { name: /open menu/i }));
    expect(await screen.findByText("Delete")).toBeInTheDocument();
    expect(screen.queryByText("Rename")).not.toBeInTheDocument();
  });
});

function driftCell(configs: Map<string, TenantConfig>) {
  const columns = getColumns({ configs });
  const column = columns.find((c) => c.id === "templateDrift");
  if (!column?.cell || typeof column.cell !== "function") {
    throw new Error("templateDrift column is missing or has no cell renderer");
  }
  const row = { original: makeTenant("abc") };
  return column.cell({ row } as never);
}

describe("tenants-columns templateDrift", () => {
  it("renders a badge when the tenant has drifted", () => {
    const configs = new Map<string, TenantConfig>([
      [
        "abc",
        {
          id: "abc",
          attributes: {
            templateDrift: true,
            sectionDrift: {
              socket: true,
              properties: false,
              characters: false,
              npcs: false,
              cashShop: false,
              mapleLife: false,
            },
          },
        } as unknown as TenantConfig,
      ],
    ]);
    render(<MemoryRouter>{driftCell(configs)}</MemoryRouter>);
    expect(screen.getByText("Differs from template")).toBeInTheDocument();
  });

  it("renders nothing when the tenant has not drifted", () => {
    const configs = new Map<string, TenantConfig>([
      [
        "abc",
        {
          id: "abc",
          attributes: {
            templateDrift: false,
            sectionDrift: {
              socket: false,
              properties: false,
              characters: false,
              npcs: false,
              cashShop: false,
              mapleLife: false,
            },
          },
        } as unknown as TenantConfig,
      ],
    ]);
    const { container } = render(
      <MemoryRouter>{driftCell(configs)}</MemoryRouter>,
    );
    expect(container).not.toHaveTextContent("Differs from template");
  });

  it("renders nothing when templateDrift is absent", () => {
    const configs = new Map<string, TenantConfig>([
      ["abc", { id: "abc", attributes: {} } as unknown as TenantConfig],
    ]);
    const { container } = render(
      <MemoryRouter>{driftCell(configs)}</MemoryRouter>,
    );
    expect(container).not.toHaveTextContent("Differs from template");
  });

  it("renders nothing when the tenant has no configuration row", () => {
    const configs = new Map<string, TenantConfig>();
    const { container } = render(
      <MemoryRouter>{driftCell(configs)}</MemoryRouter>,
    );
    expect(container).not.toHaveTextContent("Differs from template");
  });
});
