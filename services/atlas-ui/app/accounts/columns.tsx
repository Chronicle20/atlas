"use client"

import { ColumnDef } from "@tanstack/react-table"

// This type is used to define the shape of our data.
// You can use a Zod schema here if you want.
export type Account = {
    id: string
    attributes: {
        name: string
        gender: number
        banned: boolean
        tos: boolean
    }
}

export const columns: ColumnDef<Account>[] = [
    {
        accessorKey: "id",
        header: "Id",
    },
    {
        accessorKey: "attributes.name",
        header: "Name",
    },
    {
        accessorKey: "attributes.gender",
        header: "Gender",
    },
    {
        accessorKey: "attributes.banned",
        header: "Banned",
    },
    {
        accessorKey: "attributes.tos",
        header: "TOS",
    },
]
