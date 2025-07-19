"use client"

import {ColumnDef} from "@tanstack/react-table"
import type {Tenant, TenantConfig} from "@/types/models/tenant";
import {getJobNameById} from "@/lib/jobs";
import {Badge} from "@/components/ui/badge";
import {Tooltip, TooltipContent, TooltipProvider, TooltipTrigger} from "@/components/ui/tooltip";
import {Character} from "@/types/models/character";
import {Account} from "@/types/models/account";
import {MapCell} from "@/components/map-cell";
import {DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger} from "@/components/ui/dropdown-menu";
import {Button} from "@/components/ui/button";
import {MoreHorizontal} from "lucide-react";
import Link from "next/link";
import {ChangeMapDialog} from "@/components/features/characters/ChangeMapDialog";
import {useState} from "react";

interface ColumnProps {
    tenant: Tenant | null;
    tenantConfig: TenantConfig | null;
    accountMap: Map<string, Account>;
    onRefresh?: () => void;
}

export const hiddenColumns = ["id", "attributes.gm"];

export const getColumns = ({tenant, tenantConfig, accountMap, onRefresh}: ColumnProps): ColumnDef<Character>[] => {
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
                    name = tenantConfig?.attributes.worlds[num].name || String(value)
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
                                <TooltipContent copyable>
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
            cell: ({row}) => {
                const mapId = String(row.getValue("attributes_mapId"))
                return <MapCell mapId={mapId} tenant={tenant}/>
            }
        },
        {
            accessorKey: "attributes.gm",
            header: "GM",
            enableHiding: false,
        },
        {
            id: "actions",
            cell: ({row}) => {
                return <CharacterActions character={row.original} onRefresh={onRefresh} />
            },
        }
    ];
};

function CharacterActions({ character, onRefresh }: { character: Character; onRefresh?: () => void }) {
    const [changeMapOpen, setChangeMapOpen] = useState(false);

    return (
        <>
            <DropdownMenu>
                <DropdownMenuTrigger asChild>
                    <Button variant="ghost" className="h-8 w-8 p-0">
                        <span className="sr-only">Open menu</span>
                        <MoreHorizontal className="h-4 w-4"/>
                    </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                    <DropdownMenuItem asChild>
                        <Link href={"/characters/" + character.id}>
                            View Character
                        </Link>
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => setChangeMapOpen(true)}>
                        Change Map
                    </DropdownMenuItem>
                </DropdownMenuContent>
            </DropdownMenu>
            <ChangeMapDialog 
                character={character}
                open={changeMapOpen}
                onOpenChange={setChangeMapOpen}
                onSuccess={onRefresh}
            />
        </>
    );
}
