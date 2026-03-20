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
import type {Template} from "@/types/models/template";

interface ColumnProps {
    onDelete?: (id: string) => void;
    onClone?: (id: string) => void;
    onCreateTenant?: (id: string) => void;
}

export const getColumns = ({ onDelete, onClone, onCreateTenant }: ColumnProps): ColumnDef<Template>[] => [
    {
        accessorKey: "id",
        header: "Id",
        cell: ({ row }) => {
            const id = row.getValue("id") as string;
            return (
                <Link href={"/templates/" + id + "/properties"} className="font-mono text-primary hover:underline">
                    {id}
                </Link>
            );
        },
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
                        {onClone && (
                            <DropdownMenuItem 
                                onClick={() => onClone(id)}
                            >
                                Clone Template
                            </DropdownMenuItem>
                        )}
                        {onCreateTenant && (
                            <DropdownMenuItem 
                                onClick={() => onCreateTenant(id)}
                            >
                                Create Tenant from Template
                            </DropdownMenuItem>
                        )}
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
