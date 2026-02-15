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
import {charactersService} from "@/services/api/characters.service";
import {toast} from "sonner";
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from "@/components/ui/alert-dialog";

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
            cell: ({row}) => (
                <Link href={"/characters/" + row.original.id} className="font-medium text-primary hover:underline">
                    {row.original.attributes.name}
                </Link>
            ),
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
                    name = tenantConfig?.attributes.worlds[num]?.name || String(value)
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
                return (
                    <Link href={"/maps/" + mapId}>
                        <MapCell mapId={mapId} tenant={tenant}/>
                    </Link>
                )
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
                return <CharacterActions character={row.original} tenant={tenant} {...(onRefresh && { onRefresh })} />
            },
        }
    ];
};

function CharacterActions({ character, tenant, onRefresh }: { character: Character; tenant: Tenant | null; onRefresh?: () => void }) {
    const [changeMapOpen, setChangeMapOpen] = useState(false);
    const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
    const [deleting, setDeleting] = useState(false);

    const handleDelete = async () => {
        if (!tenant) return;
        try {
            setDeleting(true);
            await charactersService.deleteCharacter(tenant, character.id);
            toast.success("Successfully deleted character " + character.attributes.name);
            onRefresh?.();
        } catch (error) {
            toast.error("Failed to delete character: " + (error instanceof Error ? error.message : 'Unknown error'));
        } finally {
            setDeleting(false);
            setDeleteDialogOpen(false);
        }
    };

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
                    <DropdownMenuItem onClick={() => setChangeMapOpen(true)}>
                        Change Map
                    </DropdownMenuItem>
                    <DropdownMenuItem className="text-destructive" onClick={() => setDeleteDialogOpen(true)}>
                        Delete Character
                    </DropdownMenuItem>
                </DropdownMenuContent>
            </DropdownMenu>
            <ChangeMapDialog
                character={character}
                open={changeMapOpen}
                onOpenChange={setChangeMapOpen}
                {...(onRefresh && { onSuccess: onRefresh })}
            />
            <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>Delete Character</AlertDialogTitle>
                        <AlertDialogDescription>
                            This action cannot be undone. This will permanently delete the character &quot;{character.attributes.name}&quot;.
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                        <AlertDialogAction onClick={handleDelete} disabled={deleting}>
                            {deleting ? "Deleting..." : "Delete"}
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </>
    );
}
