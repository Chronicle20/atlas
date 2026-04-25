import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Badge } from "@/components/ui/badge"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Clock, CheckCircle, Play } from "lucide-react"
import { Link } from "react-router-dom"
import { useCharacterQuestStatus } from "@/lib/hooks/api/useCharacterQuestStatus"
import type { CharacterQuestStatus } from "@/types/models/quest"
import type { Tenant } from "@/types/models/tenant"
import { QuestStatusSkeleton } from "./QuestStatusSkeleton"
import { QuestName } from "./EntityName"

interface QuestStatusTabsProps {
    characterId: string
    tenant: Tenant
}

export function QuestStatusTabs({ characterId, tenant }: QuestStatusTabsProps) {
    const { data, isLoading, error } = useCharacterQuestStatus(tenant, characterId)

    if (!characterId) {
        return <p className="text-sm text-muted-foreground">No character selected.</p>
    }

    if (isLoading) {
        return <QuestStatusSkeleton />
    }

    if (error) {
        return <p className="text-sm text-destructive">Failed to load quests.</p>
    }

    const startedQuests = data?.started ?? []
    const completedQuests = data?.completed ?? []

    return (
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
