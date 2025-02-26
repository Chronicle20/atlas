"use client"

import { ColumnDef } from "@tanstack/react-table"
import {Tenant} from "@/lib/tenants";
import {getJobNameById} from "@/lib/jobs";

// This type is used to define the shape of our data.
// You can use a Zod schema here if you want.
export type Character = {
    id: string
    attributes: {
        name: string
    }
}

interface ColumnProps {
    tenant: Tenant | null;
}

export const getColumns = ({ tenant }: ColumnProps): ColumnDef<Character>[] => {
    return [
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
            cell: ({getValue}) => {
                const value = getValue();
                const num = Number(value);
                let name = String(value);
                if (!isNaN(num)) {
                    name = tenant?.attributes.worlds[num].name || String(value)
                }
                return (
                    <div>{name}</div>
                );
            }
        },
        {
            accessorKey: "attributes.level",
            header: "Level",
        },
        {
            accessorKey: "attributes.jobId",
            header: "Job",
            cell: ({getValue}) => {
                const value = getValue();
                const id = Number(value);
                let name = String(value);
                if (!isNaN(id)) {
                    name = getJobNameById(id) || String(value)
                }
                return (
                    <div>{name}</div>
                );
            }
        },
        {
            accessorKey: "attributes.mapId",
            header: "Map",
        },
        {
            accessorKey: "attributes.gm",
            header: "GM",
        },
    ];
};
