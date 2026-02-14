"use client"

import { ColumnDef } from "@tanstack/react-table"
import {Tooltip, TooltipContent, TooltipProvider, TooltipTrigger} from "@/components/ui/tooltip";
import {Badge} from "@/components/ui/badge";
import {DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger} from "@/components/ui/dropdown-menu";
import {Button} from "@/components/ui/button";
import {MoreHorizontal, LogOut, Shield, ShieldOff, Loader2} from "lucide-react";
import {accountsService} from "@/services/api/accounts.service";
import {Account} from "@/types/models/account";
import type {Tenant} from "@/types/models/tenant";
import type {CheckBanAttributes} from "@/types/models/ban";
import {toast} from "sonner";

interface ColumnProps {
    tenant: Tenant | null;
    onRefresh?: () => void;
    banStatuses: Map<string, CheckBanAttributes>;
    banStatusLoading: boolean;
    onBanAccount?: (account: Account) => void;
    onRemoveBan?: (account: Account) => void;
}

export const hiddenColumns = ["id", "attributes.gm"];

export const getColumns = ({tenant, onRefresh, banStatuses, banStatusLoading, onBanAccount, onRemoveBan}: ColumnProps): ColumnDef<Account>[] => {
    const handleLogout = async (id: string, name: string) => {
        if (tenant === null) {
            return
        }

        try {
            await accountsService.terminateAccountSession(tenant, id);
            toast.success("Successfully logged out " + name);
            onRefresh?.();
        } catch (error) {
            toast.error("Failed to logout " + name + ": " + (error instanceof Error ? error.message : 'Unknown error'));
        }
    };
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
            id: "banStatus",
            header: "Ban Status",
            cell: ({ row }) => {
                if (banStatusLoading) {
                    return (
                        <Badge variant="outline" className="text-muted-foreground">
                            <Loader2 className="mr-1 h-3 w-3 animate-spin" />
                            Checking
                        </Badge>
                    );
                }

                const status = banStatuses.get(row.original.id);
                if (!status) {
                    return (
                        <Badge variant="outline" className="text-muted-foreground">
                            Unknown
                        </Badge>
                    );
                }

                if (status.banned) {
                    const label = status.permanent ? "Banned (Permanent)" : "Banned";
                    return (
                        <TooltipProvider>
                            <Tooltip>
                                <TooltipTrigger asChild>
                                    <Badge variant="destructive">
                                        {label}
                                    </Badge>
                                </TooltipTrigger>
                                <TooltipContent>
                                    <p>{status.reason || "No reason provided"}</p>
                                </TooltipContent>
                            </Tooltip>
                        </TooltipProvider>
                    );
                }

                return (
                    <Badge variant="secondary" className="bg-green-100 text-green-800 hover:bg-green-100">
                        Active
                    </Badge>
                );
            },
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
            accessorKey: "attributes.tos",
            header: "TOS",
        },
        {
            id: "actions",
            cell: ({ row }) => {
                const account = row.original;
                const status = banStatuses.get(account.id);
                const isBanned = status?.banned === true;

                return (
                    <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                            <Button variant="ghost" className="h-8 w-8 p-0">
                                <span className="sr-only">Open menu</span>
                                <MoreHorizontal className="h-4 w-4" />
                            </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                            <DropdownMenuItem
                                disabled={account.attributes.loggedIn === 0}
                                onClick={() => handleLogout(account.id, account.attributes.name)}
                            >
                                <LogOut className="mr-2 h-4 w-4" />
                                Logout
                            </DropdownMenuItem>
                            <DropdownMenuSeparator />
                            {isBanned ? (
                                <DropdownMenuItem onClick={() => onRemoveBan?.(account)}>
                                    <ShieldOff className="mr-2 h-4 w-4" />
                                    Remove Ban
                                </DropdownMenuItem>
                            ) : (
                                <DropdownMenuItem
                                    className="text-destructive focus:text-destructive"
                                    onClick={() => onBanAccount?.(account)}
                                >
                                    <Shield className="mr-2 h-4 w-4" />
                                    Ban Account
                                </DropdownMenuItem>
                            )}
                        </DropdownMenuContent>
                    </DropdownMenu>
                )
            },
        }
    ]
};
