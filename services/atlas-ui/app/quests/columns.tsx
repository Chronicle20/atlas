"use client"

import { ColumnDef } from "@tanstack/react-table"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { MoreHorizontal, Eye, Clock, Zap, CheckCircle } from "lucide-react"
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
    Tooltip,
    TooltipContent,
    TooltipProvider,
    TooltipTrigger,
} from "@/components/ui/tooltip"
import type { QuestDefinition } from "@/types/models/quest"
import Link from "next/link"

export const hiddenColumns = ["attributes.area", "attributes.order"]

export const getColumns = (): ColumnDef<QuestDefinition>[] => {
    return [
        {
            accessorKey: "id",
            header: "ID",
            enableHiding: false,
            cell: ({ getValue }) => {
                const id = getValue() as string
                return (
                    <Link
                        href={`/quests/${id}`}
                        className="font-mono text-sm hover:underline"
                    >
                        {id}
                    </Link>
                )
            },
        },
        {
            accessorKey: "attributes.name",
            header: "Name",
            cell: ({ row }) => {
                const name = row.original.attributes.name || "(Unnamed)"
                return (
                    <Link
                        href={`/quests/${row.original.id}`}
                        className="font-medium hover:underline"
                    >
                        {name}
                    </Link>
                )
            },
        },
        {
            accessorKey: "attributes.parent",
            header: "Category",
            cell: ({ getValue }) => {
                const category = getValue() as string | undefined
                if (!category) return <span className="text-muted-foreground">-</span>
                return <Badge variant="outline">{category}</Badge>
            },
        },
        {
            id: "levelRange",
            header: "Level",
            cell: ({ row }) => {
                const startReqs = row.original.attributes.startRequirements
                const minLevel = startReqs?.levelMin
                const maxLevel = startReqs?.levelMax

                if (!minLevel && !maxLevel) {
                    return <span className="text-muted-foreground">Any</span>
                }

                if (minLevel && maxLevel) {
                    return <span>{minLevel} - {maxLevel}</span>
                }

                if (minLevel) {
                    return <span>{minLevel}+</span>
                }

                return <span>1 - {maxLevel}</span>
            },
        },
        {
            id: "flags",
            header: "Flags",
            cell: ({ row }) => {
                const attrs = row.original.attributes
                const flags = []

                if (attrs.autoStart) {
                    flags.push(
                        <TooltipProvider key="autoStart">
                            <Tooltip>
                                <TooltipTrigger>
                                    <Badge variant="default" className="gap-1">
                                        <Zap className="h-3 w-3" />
                                        Auto
                                    </Badge>
                                </TooltipTrigger>
                                <TooltipContent>Auto-start quest</TooltipContent>
                            </Tooltip>
                        </TooltipProvider>
                    )
                }

                if (attrs.autoComplete) {
                    flags.push(
                        <TooltipProvider key="autoComplete">
                            <Tooltip>
                                <TooltipTrigger>
                                    <Badge variant="secondary" className="gap-1">
                                        <CheckCircle className="h-3 w-3" />
                                        Complete
                                    </Badge>
                                </TooltipTrigger>
                                <TooltipContent>Auto-complete quest</TooltipContent>
                            </Tooltip>
                        </TooltipProvider>
                    )
                }

                if (attrs.timeLimit && attrs.timeLimit > 0) {
                    flags.push(
                        <TooltipProvider key="timed">
                            <Tooltip>
                                <TooltipTrigger>
                                    <Badge variant="outline" className="gap-1">
                                        <Clock className="h-3 w-3" />
                                        Timed
                                    </Badge>
                                </TooltipTrigger>
                                <TooltipContent>Time limit: {attrs.timeLimit}s</TooltipContent>
                            </Tooltip>
                        </TooltipProvider>
                    )
                }

                if (flags.length === 0) {
                    return <span className="text-muted-foreground">-</span>
                }

                return <div className="flex gap-1 flex-wrap">{flags}</div>
            },
        },
        {
            id: "requirements",
            header: "Requirements",
            cell: ({ row }) => {
                const startReqs = row.original.attributes.startRequirements
                const endReqs = row.original.attributes.endRequirements
                const counts = []

                const mobCount = (endReqs?.mobs?.length || 0)
                const itemCount = (startReqs?.items?.length || 0) + (endReqs?.items?.length || 0)
                const questCount = (startReqs?.quests?.length || 0)

                if (mobCount > 0) counts.push(`${mobCount} mob${mobCount > 1 ? 's' : ''}`)
                if (itemCount > 0) counts.push(`${itemCount} item${itemCount > 1 ? 's' : ''}`)
                if (questCount > 0) counts.push(`${questCount} quest${questCount > 1 ? 's' : ''}`)

                if (counts.length === 0) {
                    return <span className="text-muted-foreground">None</span>
                }

                return <span className="text-sm">{counts.join(", ")}</span>
            },
        },
        {
            id: "rewards",
            header: "Rewards",
            cell: ({ row }) => {
                const endActions = row.original.attributes.endActions
                const rewards = []

                if (endActions?.exp && endActions.exp > 0) {
                    rewards.push(`${endActions.exp.toLocaleString()} EXP`)
                }
                if (endActions?.money && endActions.money > 0) {
                    rewards.push(`${endActions.money.toLocaleString()} Meso`)
                }
                if (endActions?.items && endActions.items.length > 0) {
                    rewards.push(`${endActions.items.length} item${endActions.items.length > 1 ? 's' : ''}`)
                }

                if (rewards.length === 0) {
                    return <span className="text-muted-foreground">None</span>
                }

                return <span className="text-sm">{rewards.join(", ")}</span>
            },
        },
        {
            accessorKey: "attributes.area",
            header: "Area",
        },
        {
            accessorKey: "attributes.order",
            header: "Order",
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
                            <DropdownMenuItem asChild>
                                <Link href={`/quests/${row.original.id}`}>
                                    <Eye className="h-4 w-4 mr-2" />
                                    View Details
                                </Link>
                            </DropdownMenuItem>
                        </DropdownMenuContent>
                    </DropdownMenu>
                )
            },
        },
    ]
}
