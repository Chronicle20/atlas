"use client"

import { ColumnDef } from "@tanstack/react-table";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { MoreHorizontal, Eye, Trash2 } from "lucide-react";
import { BanTypeBadge } from "@/components/features/bans/BanTypeBadge";
import { BanStatusBadge } from "@/components/features/bans/BanStatusBadge";
import { Ban, BanReasonCodeLabels } from "@/types/models/ban";

interface ColumnProps {
    onView?: (ban: Ban) => void;
    onDelete?: (ban: Ban) => void;
}

export const hiddenColumns = ["attributes.reasonCode"];

export const getColumns = ({ onView, onDelete }: ColumnProps): ColumnDef<Ban>[] => {
    return [
        {
            accessorKey: "id",
            header: "ID",
            enableHiding: false,
        },
        {
            accessorKey: "attributes.banType",
            header: "Type",
            cell: ({ row }) => {
                return <BanTypeBadge type={row.original.attributes.banType} />;
            },
        },
        {
            accessorKey: "attributes.value",
            header: "Value",
            cell: ({ row }) => {
                return (
                    <span className="font-mono text-sm">
                        {row.original.attributes.value}
                    </span>
                );
            },
        },
        {
            id: "status",
            header: "Status",
            cell: ({ row }) => {
                return (
                    <BanStatusBadge
                        permanent={row.original.attributes.permanent}
                        expiresAt={row.original.attributes.expiresAt}
                    />
                );
            },
        },
        {
            accessorKey: "attributes.reasonCode",
            header: "Reason Code",
            cell: ({ row }) => {
                const code = row.original.attributes.reasonCode;
                return BanReasonCodeLabels[code] || "Unknown";
            },
        },
        {
            accessorKey: "attributes.reason",
            header: "Reason",
            cell: ({ row }) => {
                const reason = row.original.attributes.reason;
                if (!reason) return <span className="text-muted-foreground">-</span>;
                return (
                    <span className="max-w-[200px] truncate block" title={reason}>
                        {reason}
                    </span>
                );
            },
        },
        {
            accessorKey: "attributes.expiresAt",
            header: "Expires At",
            cell: ({ row }) => {
                const { permanent, expiresAt } = row.original.attributes;
                if (permanent) return <span className="text-muted-foreground">Never</span>;
                if (!expiresAt) return <span className="text-muted-foreground">-</span>;
                return new Date(expiresAt).toLocaleString();
            },
        },
        {
            accessorKey: "attributes.issuedBy",
            header: "Issued By",
            cell: ({ row }) => {
                const issuedBy = row.original.attributes.issuedBy;
                if (!issuedBy) return <span className="text-muted-foreground">-</span>;
                return issuedBy;
            },
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
                            <DropdownMenuItem onClick={() => onView?.(row.original)}>
                                <Eye className="mr-2 h-4 w-4" />
                                View Details
                            </DropdownMenuItem>
                            <DropdownMenuSeparator />
                            <DropdownMenuItem
                                className="text-destructive focus:text-destructive"
                                onClick={() => onDelete?.(row.original)}
                            >
                                <Trash2 className="mr-2 h-4 w-4" />
                                Delete
                            </DropdownMenuItem>
                        </DropdownMenuContent>
                    </DropdownMenu>
                );
            },
        },
    ];
};
