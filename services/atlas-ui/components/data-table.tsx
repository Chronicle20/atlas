"use client"

import {ColumnDef, flexRender, getCoreRowModel, useReactTable, VisibilityState,} from "@tanstack/react-table"

import {Table, TableBody, TableCell, TableHead, TableHeader, TableRow,} from "@/components/ui/table"
import React from "react";
import {Button} from "./ui/button";
import {RefreshCw, MoreVertical} from "lucide-react";
import {DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger} from "@/components/ui/dropdown-menu";

interface DataTableHeaderAction {
    icon?: React.ReactNode
    label: string
    onClick: () => void
}

interface DataTableProps<TData, TValue> {
    initialVisibilityState?: string[]
    columns: ColumnDef<TData, TValue>[]
    data: TData[]
    onRefresh?: () => void
    headerActions?: DataTableHeaderAction[]
}

export function DataTable<TData, TValue>({
                                             initialVisibilityState,
                                             columns,
                                             data,
                                             onRefresh,
                                             headerActions,
                                         }: DataTableProps<TData, TValue>) {
    const state = Object.fromEntries((initialVisibilityState || []).map((col) => [col.replaceAll(".", "_"), false]));
    const [columnVisibility, setColumnVisibility] = React.useState<VisibilityState>(state)

    const table = useReactTable({
        data,
        columns,
        getCoreRowModel: getCoreRowModel(),
        onColumnVisibilityChange: setColumnVisibility,
        state: {
            columnVisibility,
        }
    })

    return (
        <div className="w-full space-y-4">
            <div className="flex items-center justify-between">
                <div className="flex gap-2 items-center">
                    {onRefresh && (
                        <Button
                            variant="outline"
                            size="icon"
                            onClick={onRefresh}
                            className="hover:bg-accent cursor-pointer"
                            title="Refresh"
                        >
                            <RefreshCw className="h-4 w-4"/>
                        </Button>
                    )}
                </div>
                <div className="flex items-center gap-2">
                    {headerActions && headerActions.length > 0 && (
                        <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                                <Button variant="outline" size="sm">
                                    <MoreVertical className="h-4 w-4 mr-2" />
                                    Actions
                                </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="end">
                                {headerActions.map((action, index) => (
                                    <DropdownMenuItem key={index} onClick={action.onClick}>
                                        {action.icon && (
                                            <span className="mr-2">{action.icon}</span>
                                        )}
                                        {action.label}
                                    </DropdownMenuItem>
                                ))}
                            </DropdownMenuContent>
                        </DropdownMenu>
                    )}
                </div>
            </div>

            <div className="w-full h-[calc(100vh-10rem)]">
                <div className="w-full h-auto">
                    <div className="rounded-md border flex flex-col">
                        <div className="w-full">
                            <Table className="[&>div]:overflow-visible table-fixed w-full">
                                <TableHeader>
                                    {table.getHeaderGroups().map((headerGroup) => (
                                        <TableRow key={headerGroup.id}>
                                            {headerGroup.headers.map((header) => {
                                                return (
                                                    <TableHead key={header.id} style={{ width: header.getSize() }}>
                                                        {header.isPlaceholder
                                                            ? null
                                                            : flexRender(
                                                                header.column.columnDef.header,
                                                                header.getContext()
                                                            )}
                                                    </TableHead>
                                                )
                                            })}
                                        </TableRow>
                                    ))}
                                </TableHeader>
                            </Table>
                        </div>
                        <div className="overflow-auto max-h-[calc(100vh-20rem)]">
                            <Table className="[&>div]:overflow-visible table-fixed w-full">
                                <TableBody>
                                    {table.getRowModel().rows?.length ? (
                                        table.getRowModel().rows.map((row) => (
                                            <TableRow
                                                key={row.id}
                                                data-state={row.getIsSelected() && "selected"}
                                            >
                                                {row.getVisibleCells().map((cell) => (
                                                    <TableCell key={cell.id} style={{ width: cell.column.getSize() }}>
                                                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                                                    </TableCell>
                                                ))}
                                            </TableRow>
                                        ))
                                    ) : (
                                        <TableRow>
                                            <TableCell colSpan={columns.length} className="h-24 text-center">
                                                No results.
                                            </TableCell>
                                        </TableRow>
                                    )}
                                </TableBody>
                            </Table>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    )
}
