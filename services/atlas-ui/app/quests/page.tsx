"use client"

import { useTenant } from "@/context/tenant-context"
import { DataTableWrapper } from "@/components/common/DataTableWrapper"
import { getColumns, hiddenColumns } from "@/app/quests/columns"
import { useCallback, useEffect, useState, useMemo } from "react"
import { questsService } from "@/services/api"
import type { QuestDefinition } from "@/types/models/quest"
import { Toaster } from "sonner"
import { createErrorFromUnknown } from "@/types/api/errors"
import { Skeleton } from "@/components/ui/skeleton"
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select"
import { Label } from "@/components/ui/label"

function QuestPageSkeleton() {
    return (
        <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
            <div className="items-center justify-between space-y-2">
                <Skeleton className="h-8 w-32" />
            </div>
            <div className="flex gap-4">
                <Skeleton className="h-10 w-48" />
            </div>
            <div className="space-y-2">
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
            </div>
        </div>
    )
}

export default function Page() {
    const { activeTenant } = useTenant()
    const [quests, setQuests] = useState<QuestDefinition[]>([])
    const [categories, setCategories] = useState<string[]>([])
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)
    const [selectedCategory, setSelectedCategory] = useState<string>("all")

    const fetchData = useCallback(async () => {
        if (!activeTenant) return

        setLoading(true)
        setError(null)

        try {
            const [questData, categoryData] = await Promise.all([
                questsService.getAllQuests(activeTenant),
                questsService.getCategories(activeTenant),
            ])
            setQuests(questData)
            setCategories(categoryData)
        } catch (err: unknown) {
            const errorInfo = createErrorFromUnknown(err, "Failed to fetch quests")
            setError(errorInfo.message)
        } finally {
            setLoading(false)
        }
    }, [activeTenant])

    useEffect(() => {
        fetchData()
    }, [fetchData])

    const filteredQuests = useMemo(() => {
        if (selectedCategory === "all") {
            return quests
        }
        return quests.filter(q => q.attributes.parent === selectedCategory)
    }, [quests, selectedCategory])

    const columns = useMemo(() => getColumns(), [])

    if (loading) {
        return <QuestPageSkeleton />
    }

    return (
        <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
            <div className="items-center justify-between space-y-2">
                <div>
                    <h2 className="text-2xl font-bold tracking-tight">Quests</h2>
                    <p className="text-muted-foreground">
                        Browse and manage quest definitions ({quests.length} total)
                    </p>
                </div>
            </div>

            <div className="flex gap-4 items-end">
                <div className="space-y-1">
                    <Label htmlFor="category-filter">Category</Label>
                    <Select
                        value={selectedCategory}
                        onValueChange={setSelectedCategory}
                    >
                        <SelectTrigger id="category-filter" className="w-48">
                            <SelectValue placeholder="All Categories" />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="all">All Categories</SelectItem>
                            {categories.map((category) => (
                                <SelectItem key={category} value={category}>
                                    {category}
                                </SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                </div>
            </div>

            <div className="mt-4">
                <DataTableWrapper
                    columns={columns}
                    data={filteredQuests}
                    error={error}
                    onRefresh={fetchData}
                    initialVisibilityState={hiddenColumns}
                    emptyState={{
                        title: "No quests found",
                        description: selectedCategory !== "all"
                            ? `No quests found in the "${selectedCategory}" category.`
                            : "There are no quest definitions to display.",
                    }}
                />
            </div>
            <Toaster richColors />
        </div>
    )
}
