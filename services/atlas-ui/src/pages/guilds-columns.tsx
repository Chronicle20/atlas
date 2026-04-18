
import {ColumnDef} from "@tanstack/react-table"
import type {TenantConfig} from "@/types/models/tenant";
import {Guild} from "@/types/models/guild";
import {Character} from "@/types/models/character";
import {Tooltip, TooltipContent, TooltipProvider, TooltipTrigger} from "@/components/ui/tooltip";
import {Badge} from "@/components/ui/badge";
import { Link } from "react-router-dom";

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
            cell: ({row}) => (
                <Link to={"/guilds/" + row.original.id} className="font-medium text-primary hover:underline">
                    {row.original.attributes.name}
                </Link>
            ),
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
                    name = tenant?.attributes.worlds[num]?.name || String(value)
                }
                return (
                    <TooltipProvider>
                        <Tooltip>
                            <TooltipTrigger asChild>
                                <Badge variant="secondary">
                                    {name}
                                </Badge>
                            </TooltipTrigger>
                            <TooltipContent copyable>
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
    ]
}
