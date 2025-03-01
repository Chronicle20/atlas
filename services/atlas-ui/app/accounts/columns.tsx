"use client"

import { ColumnDef } from "@tanstack/react-table"
import {Tooltip, TooltipContent, TooltipProvider, TooltipTrigger} from "@/components/ui/tooltip";
import {Badge} from "@/components/ui/badge";

export type Account = {
    id: string
    attributes: {
        name: string
        gender: number
        banned: boolean
        tos: boolean
    }
}

export const hiddenColumns = ["id", "attributes.gm"];

export const columns: ColumnDef<Account>[] = [
    {
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
]
