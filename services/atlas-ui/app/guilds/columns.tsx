"use client"

import {ColumnDef} from "@tanstack/react-table"
import {DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger} from "@/components/ui/dropdown-menu";
import {Button} from "@/components/ui/button";
import {MoreHorizontal} from "lucide-react";
import type {TenantConfig} from "@/types/models/tenant";
import {Guild} from "@/types/models/guild";
import {Character} from "@/types/models/character";
import {Tooltip, TooltipContent, TooltipProvider, TooltipTrigger} from "@/components/ui/tooltip";
import {Badge} from "@/components/ui/badge";
import Link from "next/link";

interface ColumnProps {
    tenant: TenantConfig | null;
    characterMap: Map<string, Character>;
}

export const hiddenColumns = ["id"];

export function getColumns({tenant, characterMap}: ColumnProps): ColumnDef<Guild>[] {
    return [
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
            accessorKey: "attributes.leaderId",
            header: "Leader",
            cell: ({row}) => {
                const leaderId = String(row.getValue("attributes_leaderId"))
                const leader = characterMap.get(leaderId)
                return leader?.attributes.name ?? "Unknown"
            }
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
            accessorKey: "attributes.points",
            header: "Points",
        },
        {
            header: "Members",
            cell: ({ row }) => {
                const members = row.original.attributes.members
                return members?.length ?? 0
            }
        },
        {
            accessorKey: "attributes.capacity",
            header: "Capacity",
        },
        {
            id: "actions",
            cell: ({row}) => {
                return (
                    <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                            <Button variant="ghost" className="h-8 w-8 p-0">
                                <span className="sr-only">Open menu</span>
                                <MoreHorizontal className="h-4 w-4"/>
                            </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                            <DropdownMenuItem asChild>
                                <Link href={"/guilds/" + row.getValue("id")}>
                                    View Guild
                                </Link>
                            </DropdownMenuItem>
                        </DropdownMenuContent>
                    </DropdownMenu>
                )
            },
        }
    ]
}
