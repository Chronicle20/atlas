"use client"

import { ColumnDef } from "@tanstack/react-table"
import {Tooltip, TooltipContent, TooltipProvider, TooltipTrigger} from "@/components/ui/tooltip";
import {Badge} from "@/components/ui/badge";
import {DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger} from "@/components/ui/dropdown-menu";
import {Button} from "@/components/ui/button";
import {MoreHorizontal} from "lucide-react";
import {Account, terminateAccountSession} from "@/lib/accounts";
import type {Tenant} from "@/types/models/tenant";
import {toast} from "sonner";

const onLogout = (tenant: Tenant | null, id: string, name: string) => {
    if (tenant === null) {
        return
    }

    terminateAccountSession(tenant, id);
    toast.success("Successfully logged out " + name);
};

interface ColumnProps {
    tenant: Tenant | null;
}

export const hiddenColumns = ["id", "attributes.gm"];

export const getColumns = ({tenant}: ColumnProps): ColumnDef<Account>[] => {
    return [ {
            accessorKey: "id",
            header: "Id",
            enableHiding: false,
        },
        {
            accessorKey: "attributes.name",
            header: "Name",
        },
        {
            accessorKey: "attributes.loggedIn",
            header: "State",
            cell: ({getValue}) => {
                const value = getValue();
                const num = Number(value);
                let name = String(value);
                if (!isNaN(num)) {
                    if (num === 0) {
                        name = "Logged Out";
                    } else if (num === 1) {
                        name = "Logged In";
                    } else {
                        name = "In Transition";
                    }
                }
                return (
                    <TooltipProvider>
                        <Tooltip>
                            <TooltipTrigger asChild>
                                <Badge variant="secondary">
                                    {name}
                                </Badge>
                            </TooltipTrigger>
                            <TooltipContent>
                                <p>{String(value)}</p>
                            </TooltipContent>
                        </Tooltip>
                    </TooltipProvider>
                );
            }
        },
        {
            accessorKey: "attributes.gender",
            header: "Gender",
            cell: ({getValue}) => {
                const value = getValue();
                const num = Number(value);
                let name = String(value);
                if (!isNaN(num)) {
                    name = num === 0 ? "Male" : "Female";
                }
                return (
                    <TooltipProvider>
                        <Tooltip>
                            <TooltipTrigger asChild>
                                <Badge variant="secondary">
                                    {name}
                                </Badge>
                            </TooltipTrigger>
                            <TooltipContent>
                                <p>{String(value)}</p>
                            </TooltipContent>
                        </Tooltip>
                    </TooltipProvider>
                );
            }
        },
        {
            accessorKey: "attributes.banned",
            header: "Banned",
        },
        {
            accessorKey: "attributes.tos",
            header: "TOS",
        },
        {
            id: "actions",
            cell: ({ row }) => {
                return (
                    <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                            <Button variant="ghost" className="h-8 w-8 p-0">
                                <span className="sr-only">Open menu</span>
                                <MoreHorizontal className="h-4 w-4" />
                            </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                            <DropdownMenuItem disabled={row.original.attributes.loggedIn === 0} onClick={() => {onLogout(tenant, row.original.id, row.original.attributes.name)}}>
                                Logout
                            </DropdownMenuItem>
                        </DropdownMenuContent>
                    </DropdownMenu>
                )
            },
        }
    ]
};
