"use client"

import { ColumnDef } from "@tanstack/react-table"
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger
} from "@/components/ui/dropdown-menu";
import {Button} from "@/components/ui/button";
import {MoreHorizontal} from "lucide-react";
import Link from "next/link";

export type Tenant = {
    id: string
    attributes: {
        name: string
        region: string
        majorVersion: number
        minorVersion: number
    }
}

interface ColumnProps {
    onDelete?: (id: string) => void;
}

export const getColumns = ({ onDelete }: ColumnProps): ColumnDef<Tenant>[] => [
    {
        accessorKey: "id",
        header: "Id",
    },
    {
        accessorKey: "attributes.name",
        header: "Name",
    },
    {
        accessorKey: "attributes.region",
        header: "Region",
    },
    {
        accessorKey: "attributes.majorVersion",
        header: "Major",
    },
    {
        accessorKey: "attributes.minorVersion",
        header: "Minor",
    },
    {
        id: "actions",
        cell: ({ row }) => {
            const id = row.getValue("id") as string;

            return (
                <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                        <Button variant="ghost" className="h-8 w-8 p-0">
                            <span className="sr-only">Open menu</span>
                            <MoreHorizontal className="h-4 w-4" />
                        </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                        <DropdownMenuItem asChild>
                            <Link href={"/tenants/" + id + "/properties"}>
                                View Tenant
                            </Link>
                        </DropdownMenuItem>
                        {onDelete && (
                            <DropdownMenuItem 
                                className="text-destructive focus:text-destructive"
                                onClick={() => onDelete(id)}
                            >
                                Delete
                            </DropdownMenuItem>
                        )}
                    </DropdownMenuContent>
                </DropdownMenu>
            )
        },
    },
]
