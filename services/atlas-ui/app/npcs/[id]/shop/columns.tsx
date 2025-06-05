"use client"

import {ColumnDef} from "@tanstack/react-table"
import {Commodity} from "@/lib/npcs";
import {Button} from "@/components/ui/button";
import {MoreHorizontal} from "lucide-react";
import {DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger} from "@/components/ui/dropdown-menu";

interface ColumnProps {
    npcId: number;
    onEdit: (commodity: Commodity) => void;
    onDelete: (commodityId: string) => void;
}

export const getColumns = ({onEdit, onDelete}: ColumnProps): ColumnDef<Commodity>[] => {
    return [
        {
            accessorKey: "id",
            header: "ID",
        },
        {
            accessorFn: (row) => row.attributes.templateId,
            header: "Template ID",
        },
        {
            accessorFn: (row) => row.attributes.mesoPrice,
            header: "Meso Price",
        },
        {
            accessorFn: (row) => row.attributes.discountRate,
            header: "Discount Rate",
        },
        {
            accessorFn: (row) => row.attributes.tokenItemId,
            header: "Token Item ID",
        },
        {
            accessorFn: (row) => row.attributes.tokenPrice,
            header: "Token Price",
        },
        {
            accessorFn: (row) => row.attributes.period,
            header: "Period",
        },
        {
            accessorFn: (row) => row.attributes.levelLimit,
            header: "Level Limit",
        },
        {
            id: "actions",
            cell: ({row}) => {
                const commodity = row.original;

                return (
                    <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                            <Button variant="ghost" className="h-8 w-8 p-0">
                                <span className="sr-only">Open menu</span>
                                <MoreHorizontal className="h-4 w-4"/>
                            </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                            <DropdownMenuItem onClick={() => onEdit(commodity)}>
                                Edit Commodity
                            </DropdownMenuItem>
                            <DropdownMenuItem onClick={() => onDelete(commodity.id)}>
                                Delete Commodity
                            </DropdownMenuItem>
                        </DropdownMenuContent>
                    </DropdownMenu>
                )
            },
        }
    ];
};
