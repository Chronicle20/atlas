"use client"

import {ColumnDef} from "@tanstack/react-table"
import {Tenant} from "@/lib/tenants";
import {getJobNameById} from "@/lib/jobs";
import {Badge} from "@/components/ui/badge";
import {Tooltip, TooltipContent, TooltipProvider, TooltipTrigger} from "@/components/ui/tooltip";

export type Character = {
    id: string
    attributes: {
        name: string
    }
}

interface ColumnProps {
    tenant: Tenant | null;
}

export const hiddenColumns = ["id", "attributes.gm"];

export const getColumns = ({tenant}: ColumnProps): ColumnDef<Character>[] => {
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
            accessorKey: "attributes.level",
            header: "Level",
        },
        {
            accessorKey: "attributes.jobId",
            header: "Role",
            cell: ({row, getValue}) => {
                const value = getValue();
                const id = Number(value);
                let name = String(value);
                if (!isNaN(id)) {
                    name = getJobNameById(id) || String(value)
                }

                const gm = row.getValue("attributes_gm");
                let isGm = false
                const gmVal = Number(gm);
                if (gmVal > 0) {
                    isGm = true;
                }

                return (
                    <div className="flex flex-rows justify-start gap-2">
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
                        {
                            isGm && (
                                <TooltipProvider>
                                    <Tooltip>
                                        <TooltipTrigger asChild>
                                            <Badge variant="destructive">
                                                GM
                                            </Badge>
                                        </TooltipTrigger>
                                        <TooltipContent>
                                            <p>{String(gm)}</p>
                                        </TooltipContent>
                                    </Tooltip>
                                </TooltipProvider>
                            )
                        }
                    </div>
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
            enableHiding: false,
        },
    ];
};
