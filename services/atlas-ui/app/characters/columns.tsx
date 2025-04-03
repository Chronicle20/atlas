"use client"

import {ColumnDef} from "@tanstack/react-table"
import {Tenant} from "@/lib/tenants";
import {getJobNameById} from "@/lib/jobs";
import {Badge} from "@/components/ui/badge";
import {Tooltip, TooltipContent, TooltipProvider, TooltipTrigger} from "@/components/ui/tooltip";
import {Character} from "@/lib/characters";
import {Account} from "@/lib/accounts";
import {useEffect, useState} from "react";
import {fetchMap} from "@/lib/maps";

interface ColumnProps {
    tenant: Tenant | null;
    accountMap: Map<string, Account>;
}

export const hiddenColumns = ["id", "attributes.gm"];

const mapNameCache = new Map<string, string>()

export const getColumns = ({tenant, accountMap}: ColumnProps): ColumnDef<Character>[] => {
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
            cell: ({row}) => {
                const accountId = String(row.getValue("attributes_accountId"))
                const account = accountMap.get(accountId)
                return account?.attributes.name ?? "Unknown"
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
            cell: ({ row }) => {
                const mapId = String(row.getValue("attributes_mapId"))
                const [name, setName] = useState<string>("Loading...")

                useEffect(() => {
                    if (!tenant || !mapId || mapNameCache.has(mapId)) return

                    fetchMap(tenant, mapId)
                        .then((map) => {
                            const mapName = map.attributes.name
                            mapNameCache.set(mapId, mapName)
                            setName(mapName)
                        })
                        .catch(() => setName("Unknown"))
                }, [tenant, mapId])

                return (
                    <TooltipProvider>
                        <Tooltip>
                            <TooltipTrigger asChild>
                                <Badge variant="secondary">
                                    {name}
                                </Badge>
                            </TooltipTrigger>
                            <TooltipContent>
                                <p>{mapId}</p>
                            </TooltipContent>
                        </Tooltip>
                    </TooltipProvider>
                )
            }
        },
        {
            accessorKey: "attributes.gm",
            header: "GM",
            enableHiding: false,
        },
    ];
};
