import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { PoolItemsTable } from "../PoolItemsTable";
import type { RewardPoolItemData } from "@/types/models/reward-pool-item";

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: {
      id: "t1",
      attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
    },
  }),
}));
vi.mock("@/components/item-name-cell", () => ({
  ItemNameCell: ({ itemId }: { itemId: string }) => <span>item-{itemId}</span>,
}));

function cashSurpriseItem(
  overrides: Partial<RewardPoolItemData["attributes"]> = {},
): RewardPoolItemData {
  return {
    id: "1",
    type: "gachapon-items",
    attributes: {
      gachaponId: "5910000",
      itemId: 5510000,
      quantity: 1,
      tier: "common",
      weight: 100,
      commodityId: 5300000,
      ...overrides,
    },
  };
}

const noop = () => {};

describe("PoolItemsTable", () => {
  it("cash-surprise: renders the Serial column with each row's commodityId", () => {
    render(
      <PoolItemsTable
        kind="cash-surprise"
        poolId="5910000"
        tierWeights={{ common: 0, uncommon: 0, rare: 0 }}
        items={[cashSurpriseItem()]}
        globalItems={[]}
        onEdit={noop}
        onDelete={noop}
      />,
    );
    expect(screen.getByText("Serial")).toBeInTheDocument();
    expect(screen.getByText("5300000")).toBeInTheDocument();
    expect(screen.getByText("item-5510000")).toBeInTheDocument();
  });

  it("incubator: renders the flat layout without a Serial column", () => {
    render(
      <PoolItemsTable
        kind="incubator"
        poolId="4170001"
        tierWeights={{ common: 0, uncommon: 0, rare: 0 }}
        items={[cashSurpriseItem({ commodityId: 0 })]}
        globalItems={[]}
        onEdit={noop}
        onDelete={noop}
      />,
    );
    expect(screen.queryByText("Serial")).not.toBeInTheDocument();
  });

  it("gachapon: renders the tiered layout, unaffected by the Serial column addition", () => {
    render(
      <PoolItemsTable
        kind="gachapon"
        poolId="henesys"
        tierWeights={{ common: 70, uncommon: 25, rare: 5 }}
        items={[
          {
            id: "1",
            type: "gachapon-items",
            attributes: {
              gachaponId: "henesys",
              itemId: 2000000,
              quantity: 1,
              tier: "common",
              weight: 0,
              commodityId: 0,
            },
          },
        ]}
        globalItems={[]}
        onEdit={noop}
        onDelete={noop}
      />,
    );
    expect(screen.queryByText("Serial")).not.toBeInTheDocument();
    expect(screen.getByText("item-2000000")).toBeInTheDocument();
  });
});
