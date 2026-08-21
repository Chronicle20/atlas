import { render, screen, fireEvent } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { DataTableColumnDef } from "@/components/data-table-features";
import { DataTableWrapper } from "../DataTableWrapper";

type Row = { id: string; name: string };
const columns: DataTableColumnDef<Row>[] = [
  { accessorKey: "name", header: "Name" },
];
const TS = 1_735_732_920_000;

describe("DataTableWrapper empty branch", () => {
  it("renders the empty-state refresh button when onRefresh is supplied", () => {
    const onRefresh = vi.fn();
    render(
      <DataTableWrapper columns={columns} data={[]} onRefresh={onRefresh} />,
    );
    const button = screen.getByTestId("empty-state-refresh");
    expect(button).toBeInTheDocument();
    fireEvent.click(button);
    expect(onRefresh).toHaveBeenCalledOnce();
  });

  it("renders no refresh button on the empty branch without onRefresh", () => {
    render(<DataTableWrapper columns={columns} data={[]} />);
    expect(screen.queryByTestId("empty-state-refresh")).toBeNull();
  });

  it("forwards isRefreshing to the empty-state button", () => {
    const onRefresh = vi.fn();
    render(
      <DataTableWrapper
        columns={columns}
        data={[]}
        onRefresh={onRefresh}
        isRefreshing
      />,
    );
    expect(screen.getByTestId("empty-state-refresh")).toBeDisabled();
  });

  it("renders both the custom action and refresh on the empty branch", () => {
    const onRefresh = vi.fn();
    const onAction = vi.fn();
    render(
      <DataTableWrapper
        columns={columns}
        data={[]}
        onRefresh={onRefresh}
        emptyState={{
          title: "No accounts found",
          action: { label: "Create Account", onClick: onAction },
        }}
      />,
    );
    expect(screen.getByText("No accounts found")).toBeInTheDocument();
    const buttons = screen.getAllByRole("button");
    expect(buttons).toHaveLength(2);
    expect(buttons[0]).toHaveTextContent("Create Account");
    expect(buttons[1]).toBe(screen.getByTestId("empty-state-refresh"));
  });

  it("forwards lastUpdatedAt to the caption", () => {
    const onRefresh = vi.fn();
    render(
      <DataTableWrapper
        columns={columns}
        data={[]}
        onRefresh={onRefresh}
        lastUpdatedAt={TS}
      />,
    );
    expect(screen.getByTestId("empty-state-last-updated")).toHaveAttribute(
      "title",
      new Date(TS).toISOString(),
    );
  });

  it("renders the default empty copy when no emptyState is supplied", () => {
    render(<DataTableWrapper columns={columns} data={[]} />);
    expect(screen.getByText("No data available")).toBeInTheDocument();
    expect(
      screen.getByText("There are no items to display at this time."),
    ).toBeInTheDocument();
  });

  it("renders the loader on the loading branch", () => {
    const onRefresh = vi.fn();
    render(
      <DataTableWrapper
        columns={columns}
        data={[]}
        loading
        onRefresh={onRefresh}
      />,
    );
    expect(screen.queryByTestId("empty-state")).toBeNull();
  });

  it("renders the error branch with a retry, not an empty state", () => {
    const onRefresh = vi.fn();
    render(
      <DataTableWrapper
        columns={columns}
        data={[]}
        error="boom"
        onRefresh={onRefresh}
      />,
    );
    expect(screen.queryByTestId("empty-state")).toBeNull();
    expect(
      screen.getByRole("button", { name: /try again/i }),
    ).toBeInTheDocument();
  });
});
