import { useCallback, useEffect, useState } from "react"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { ScrollArea } from "@/components/ui/scroll-area"
import { RefreshCw, Clock, CheckCircle, Play } from "lucide-react"
import { questStatusService } from "@/services/api/quest-status.service"
import type { CharacterQuestStatus } from "@/types/models/quest"
import type { Tenant } from "@/types/models/tenant"
import { createErrorFromUnknown } from "@/types/api/errors"
import { Link } from "react-router-dom";
import { QuestName } from "./EntityName"

interface QuestStatusTabsProps {
    characterId: string
    tenant: Tenant
}

export function QuestStatusTabs({ characterId, tenant }: QuestStatusTabsProps) {
    const [startedQuests, setStartedQuests] = useState<CharacterQuestStatus[]>([])
    const [completedQuests, setCompletedQuests] = useState<CharacterQuestStatus[]>([])
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)

    const fetchQuestStatuses = useCallback(async () => {
        if (!tenant || !characterId) return

        setLoading(true)
        setError(null)

        try {
            const [started, completed] = await Promise.all([
                questStatusService.getStartedQuests( characterId),
                questStatusService.getCompletedQuests( characterId),
            ])
            setStartedQuests(started)
            setCompletedQuests(completed)
        } catch (err: unknown) {
            const errorInfo = createErrorFromUnknown(err, "Failed to fetch quest statuses")
            setError(errorInfo.message)
        } finally {
            setLoading(false)
        }
    }, [tenant, characterId])

    useEffect(() => {
        fetchQuestStatuses()
    }, [fetchQuestStatuses])

    if (loading) {
        return <QuestStatusSkeleton />
    }

    if (error) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle className="flex items-center justify-between">
                        Quests
                        <Button variant="outline" size="sm" onClick={fetchQuestStatuses}>
                            <RefreshCw className="h-4 w-4 mr-2" />
                            Retry
                        </Button>
                    </CardTitle>
                </CardHeader>
                <CardContent>
                    <p className="text-destructive text-sm">{error}</p>
                </CardContent>
            </Card>
        )
    }

    return (
        <Card>
            <CardHeader>
                <div className="flex items-center justify-between">
                    <div>
                        <CardTitle>Quests</CardTitle>
                        <CardDescription>
                            {startedQuests.length} in progress, {completedQuests.length} completed
                        </CardDescription>
                    </div>
                    <Button variant="outline" size="icon" onClick={fetchQuestStatuses}>
                        <RefreshCw className="h-4 w-4" />
                    </Button>
                </div>
            </CardHeader>
            <CardContent>
                <Tabs defaultValue="started">
                    <TabsList className="grid w-full grid-cols-2">
                        <TabsTrigger value="started" className="flex items-center gap-2">
                            <Play className="h-4 w-4" />
                            Started ({startedQuests.length})
                        </TabsTrigger>
                        <TabsTrigger value="completed" className="flex items-center gap-2">
                            <CheckCircle className="h-4 w-4" />
                            Completed ({completedQuests.length})
                        </TabsTrigger>
                    </TabsList>

                    <TabsContent value="started" className="mt-4">
                        <ScrollArea className="h-[300px]">
                            {startedQuests.length === 0 ? (
                                <div className="text-center text-muted-foreground py-8">
                                    No quests in progress
                                </div>
                            ) : (
                                <div
                                    data-testid="quest-grid"
                                    className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3"
                                >
                                    {startedQuests.map((quest) => (
                                        <QuestStatusWidget key={quest.id} quest={quest} />
                                    ))}
                                </div>
                            )}
                        </ScrollArea>
                    </TabsContent>

                    <TabsContent value="completed" className="mt-4">
                        <ScrollArea className="h-[300px]">
                            {completedQuests.length === 0 ? (
                                <div className="text-center text-muted-foreground py-8">
                                    No completed quests
                                </div>
                            ) : (
                                <div
                                    data-testid="quest-grid"
                                    className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3"
                                >
                                    {completedQuests.map((quest) => (
                                        <QuestStatusWidget
                                            key={quest.id}
                                            quest={quest}
                                            showCompletionTime
                                        />
                                    ))}
                                </div>
                            )}
                        </ScrollArea>
                    </TabsContent>
                </Tabs>
            </CardContent>
        </Card>
    )
}

interface QuestStatusWidgetProps {
    quest: CharacterQuestStatus
    showCompletionTime?: boolean
}

function QuestStatusWidget({ quest, showCompletionTime }: QuestStatusWidgetProps) {
    const attrs = quest.attributes

    return (
        <Link
            to={`/quests/${attrs.questId}`}
            className="block border rounded-lg p-3 overflow-hidden hover:bg-muted/50 transition-colors"
        >
            <div className="flex items-center justify-between gap-2 min-w-0">
                <QuestName
                    id={attrs.questId}
                    className="font-medium truncate"
                />
                {attrs.completedCount > 1 && (
                    <Badge variant="outline" className="text-xs shrink-0">
                        x{attrs.completedCount}
                    </Badge>
                )}
            </div>
            {showCompletionTime && attrs.completedAt && (
                <div
                    data-testid="completion-time"
                    className="mt-1 text-sm text-muted-foreground flex items-center gap-1"
                >
                    <Clock className="h-3 w-3" />
                    {formatDate(attrs.completedAt)}
                </div>
            )}
        </Link>
    )
}

function QuestStatusSkeleton() {
    return (
        <Card>
            <CardHeader>
                <Skeleton className="h-6 w-24" />
                <Skeleton className="h-4 w-48 mt-1" />
            </CardHeader>
            <CardContent>
                <Skeleton className="h-10 w-full mb-4" />
                <div className="space-y-3">
                    <Skeleton className="h-16 w-full" />
                    <Skeleton className="h-16 w-full" />
                    <Skeleton className="h-16 w-full" />
                </div>
            </CardContent>
        </Card>
    )
}

function formatDate(dateString: string): string {
    try {
        const date = new Date(dateString)
        return date.toLocaleDateString(undefined, {
            year: "numeric",
            month: "short",
            day: "numeric",
            hour: "2-digit",
            minute: "2-digit",
        })
    } catch {
        return dateString
    }
}
