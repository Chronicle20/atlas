"use client"

import { ColumnDef } from "@tanstack/react-table"

// This type is used to define the shape of our data.
// You can use a Zod schema here if you want.
export type Character = {
    id: string
    attributes: {
        name: string
    }
}

export const columns: ColumnDef<Character>[] = [
    {
        accessorKey: "id",
        header: "Id",
    },
    {
        accessorKey: "attributes.name",
        header: "Name",
    },
    {
        accessorKey: "attributes.accountId",
        header: "Account",
    },
    {
        accessorKey: "attributes.worldId",
        header: "World",
    },
    {
        accessorKey: "attributes.level",
        header: "Level",
    },
    {
        accessorKey: "attributes.jobId",
        header: "Job",
    },
    {
        accessorKey: "attributes.mapId",
        header: "Map",
    },
    {
        accessorKey: "attributes.gm",
        header: "GM",
    },
]
