"use client"

import { ColumnDef } from "@tanstack/react-table"
import type { Tenant } from "@/types/models/tenant";
import { Badge } from "@/components/ui/badge";
import Link from "next/link";
import { MapCell } from "@/components/map-cell";
import {
  type MerchantShop,
  getShopTypeName,
  getShopTypeBadgeVariant,
  getShopStateName,
  getShopStateBadgeVariant,
} from "@/types/models/merchant";

interface ColumnProps {
  tenant: Tenant | null;
}

export const hiddenColumns = ["id"];

export const getColumns = ({ tenant }: ColumnProps): ColumnDef<MerchantShop>[] => {
  return [
    {
      accessorKey: "id",
      header: "Id",
      enableHiding: false,
    },
    {
      accessorKey: "attributes.title",
      header: "Shop Name",
      cell: ({ row }) => (
        <Link href={"/merchants/" + row.original.id} className="font-medium text-primary hover:underline">
          {row.original.attributes.title || "Untitled"}
        </Link>
      ),
    },
    {
      accessorKey: "attributes.shopType",
      header: "Type",
      cell: ({ row }) => {
        const shopType = row.original.attributes.shopType;
        return (
          <Badge variant="secondary" className={getShopTypeBadgeVariant(shopType)}>
            {getShopTypeName(shopType)}
          </Badge>
        );
      },
    },
    {
      accessorKey: "attributes.state",
      header: "State",
      cell: ({ row }) => {
        const state = row.original.attributes.state;
        return (
          <Badge variant="secondary" className={getShopStateBadgeVariant(state)}>
            {getShopStateName(state)}
          </Badge>
        );
      },
    },
    {
      accessorKey: "attributes.mapId",
      header: "Map",
      cell: ({ row }) => (
        <MapCell mapId={String(row.original.attributes.mapId)} tenant={tenant} />
      ),
    },
    {
      accessorKey: "attributes.characterId",
      header: "Owner",
      cell: ({ row }) => (
        <Link href={"/characters/" + row.original.attributes.characterId} className="font-medium text-primary hover:underline">
          {row.original.attributes.characterId}
        </Link>
      ),
    },
    {
      accessorKey: "attributes.listingCount",
      header: "Items",
      cell: ({ row }) => (
        <Badge variant="secondary">
          {row.original.attributes.listingCount}
        </Badge>
      ),
    },
  ];
};
