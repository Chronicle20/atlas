"use client"

import { useTenant } from "@/context/tenant-context"
import { useParams, useRouter } from "next/navigation"
import { useCallback, useEffect, useState } from "react"
import { questsService } from "@/services/api"
import type { QuestDefinition } from "@/types/models/quest"
import { createErrorFromUnknown } from "@/types/api/errors"
import { Skeleton } from "@/components/ui/skeleton"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
} from "@/components/ui/card"
import {
    Collapsible,
    CollapsibleContent,
    CollapsibleTrigger,
} from "@/components/ui/collapsible"
import {
    ArrowLeft,
    ChevronDown,
    Clock,
    Zap,
    CheckCircle,
    RefreshCw,
    ArrowRight,
} from "lucide-react"
import { RequirementRenderer } from "@/components/features/quests/RequirementRenderer"
import { RewardRenderer } from "@/components/features/quests/RewardRenderer"
import { Toaster } from "sonner"
import Link from "next/link"

function QuestDetailSkeleton() {
    return (
        <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
            <Skeleton className="h-8 w-32" />
            <Skeleton className="h-12 w-96" />
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <Skeleton className="h-48" />
                <Skeleton className="h-48" />
                <Skeleton className="h-48" />
                <Skeleton className="h-48" />
            </div>
        </div>
    )
}

export default function QuestDetailPage() {
    const { activeTenant } = useTenant()
    const params = useParams()
    const router = useRouter()
    const questId = params.id as string

    const [quest, setQuest] = useState<QuestDefinition | null>(null)
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)
    const [startReqsOpen, setStartReqsOpen] = useState(true)
    const [endReqsOpen, setEndReqsOpen] = useState(true)
    const [startActionsOpen, setStartActionsOpen] = useState(false)
    const [endActionsOpen, setEndActionsOpen] = useState(true)

    const fetchQuest = useCallback(async () => {
        if (!activeTenant || !questId) return

        setLoading(true)
        setError(null)

        try {
            const questData = await questsService.getQuestById(activeTenant, questId)
            setQuest(questData)
        } catch (err: unknown) {
            const errorInfo = createErrorFromUnknown(err, "Failed to fetch quest")
            setError(errorInfo.message)
        } finally {
            setLoading(false)
        }
    }, [activeTenant, questId])

    useEffect(() => {
        fetchQuest()
    }, [fetchQuest])

    if (loading) {
        return <QuestDetailSkeleton />
    }

    if (error) {
        return (
            <div className="flex flex-col flex-1 items-center justify-center space-y-4 p-10">
                <p className="text-destructive">{error}</p>
                <div className="flex gap-2">
                    <Button variant="outline" onClick={() => router.back()}>
                        <ArrowLeft className="h-4 w-4 mr-2" />
                        Go Back
                    </Button>
                    <Button onClick={fetchQuest}>
                        <RefreshCw className="h-4 w-4 mr-2" />
                        Retry
                    </Button>
                </div>
            </div>
        )
    }

    if (!quest) {
        return (
            <div className="flex flex-col flex-1 items-center justify-center space-y-4 p-10">
                <p className="text-muted-foreground">Quest not found</p>
                <Button variant="outline" onClick={() => router.back()}>
                    <ArrowLeft className="h-4 w-4 mr-2" />
                    Go Back
                </Button>
            </div>
        )
    }

    const attrs = quest.attributes

    return (
        <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
            {/* Header */}
            <div className="flex items-center gap-4">
                <Button variant="ghost" size="icon" onClick={() => router.back()}>
                    <ArrowLeft className="h-4 w-4" />
                </Button>
                <div>
                    <div className="flex items-center gap-2">
                        <h2 className="text-2xl font-bold tracking-tight">
                            Quest #{questId}
                        </h2>
                        {attrs.parent && (
                            <Badge variant="outline">{attrs.parent}</Badge>
                        )}
                    </div>
                    <p className="text-xl text-muted-foreground">
                        {attrs.name || "(Unnamed Quest)"}
                    </p>
                </div>
            </div>

            {/* Metadata Card */}
            <Card>
                <CardHeader>
                    <CardTitle>Quest Information</CardTitle>
                    <CardDescription>Basic quest metadata and flags</CardDescription>
                </CardHeader>
                <CardContent>
                    <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                        <div className="space-y-1">
                            <p className="text-sm text-muted-foreground">Auto Start</p>
                            <div className="flex items-center gap-1">
                                {attrs.autoStart ? (
                                    <Badge className="gap-1">
                                        <Zap className="h-3 w-3" />
                                        Yes
                                    </Badge>
                                ) : (
                                    <span className="text-sm">No</span>
                                )}
                            </div>
                        </div>
                        <div className="space-y-1">
                            <p className="text-sm text-muted-foreground">Auto Complete</p>
                            <div className="flex items-center gap-1">
                                {attrs.autoComplete ? (
                                    <Badge variant="secondary" className="gap-1">
                                        <CheckCircle className="h-3 w-3" />
                                        Yes
                                    </Badge>
                                ) : (
                                    <span className="text-sm">No</span>
                                )}
                            </div>
                        </div>
                        <div className="space-y-1">
                            <p className="text-sm text-muted-foreground">Time Limit</p>
                            {attrs.timeLimit && attrs.timeLimit > 0 ? (
                                <Badge variant="outline" className="gap-1">
                                    <Clock className="h-3 w-3" />
                                    {formatTime(attrs.timeLimit)}
                                </Badge>
                            ) : (
                                <span className="text-sm">None</span>
                            )}
                        </div>
                        <div className="space-y-1">
                            <p className="text-sm text-muted-foreground">Area / Order</p>
                            <span className="text-sm">
                                {attrs.area || 0} / {attrs.order || 0}
                            </span>
                        </div>
                    </div>

                    {(attrs.summary || attrs.demandSummary || attrs.rewardSummary) && (
                        <div className="mt-4 pt-4 border-t space-y-2">
                            {attrs.summary && (
                                <div>
                                    <p className="text-sm font-medium">Summary</p>
                                    <p className="text-sm text-muted-foreground">{attrs.summary}</p>
                                </div>
                            )}
                            {attrs.demandSummary && (
                                <div>
                                    <p className="text-sm font-medium">Demand</p>
                                    <p className="text-sm text-muted-foreground">{attrs.demandSummary}</p>
                                </div>
                            )}
                            {attrs.rewardSummary && (
                                <div>
                                    <p className="text-sm font-medium">Reward</p>
                                    <p className="text-sm text-muted-foreground">{attrs.rewardSummary}</p>
                                </div>
                            )}
                        </div>
                    )}
                </CardContent>
            </Card>

            {/* Requirements and Rewards Grid */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                {/* Start Requirements */}
                <Collapsible open={startReqsOpen} onOpenChange={setStartReqsOpen}>
                    <Card>
                        <CardHeader className="pb-3">
                            <CollapsibleTrigger className="flex items-center justify-between w-full">
                                <CardTitle className="text-lg">Start Requirements</CardTitle>
                                <ChevronDown
                                    className={`h-4 w-4 transition-transform ${
                                        startReqsOpen ? "rotate-180" : ""
                                    }`}
                                />
                            </CollapsibleTrigger>
                        </CardHeader>
                        <CollapsibleContent>
                            <CardContent>
                                <RequirementRenderer
                                    requirements={attrs.startRequirements}
                                    type="start"
                                />
                            </CardContent>
                        </CollapsibleContent>
                    </Card>
                </Collapsible>

                {/* Completion Requirements */}
                <Collapsible open={endReqsOpen} onOpenChange={setEndReqsOpen}>
                    <Card>
                        <CardHeader className="pb-3">
                            <CollapsibleTrigger className="flex items-center justify-between w-full">
                                <CardTitle className="text-lg">Completion Requirements</CardTitle>
                                <ChevronDown
                                    className={`h-4 w-4 transition-transform ${
                                        endReqsOpen ? "rotate-180" : ""
                                    }`}
                                />
                            </CollapsibleTrigger>
                        </CardHeader>
                        <CollapsibleContent>
                            <CardContent>
                                <RequirementRenderer
                                    requirements={attrs.endRequirements}
                                    type="end"
                                />
                            </CardContent>
                        </CollapsibleContent>
                    </Card>
                </Collapsible>

                {/* Start Actions */}
                <Collapsible open={startActionsOpen} onOpenChange={setStartActionsOpen}>
                    <Card>
                        <CardHeader className="pb-3">
                            <CollapsibleTrigger className="flex items-center justify-between w-full">
                                <CardTitle className="text-lg">Start Actions</CardTitle>
                                <ChevronDown
                                    className={`h-4 w-4 transition-transform ${
                                        startActionsOpen ? "rotate-180" : ""
                                    }`}
                                />
                            </CollapsibleTrigger>
                        </CardHeader>
                        <CollapsibleContent>
                            <CardContent>
                                <RewardRenderer
                                    actions={attrs.startActions}
                                    type="start"
                                />
                            </CardContent>
                        </CollapsibleContent>
                    </Card>
                </Collapsible>

                {/* Completion Rewards */}
                <Collapsible open={endActionsOpen} onOpenChange={setEndActionsOpen}>
                    <Card>
                        <CardHeader className="pb-3">
                            <CollapsibleTrigger className="flex items-center justify-between w-full">
                                <CardTitle className="text-lg">Completion Rewards</CardTitle>
                                <ChevronDown
                                    className={`h-4 w-4 transition-transform ${
                                        endActionsOpen ? "rotate-180" : ""
                                    }`}
                                />
                            </CollapsibleTrigger>
                        </CardHeader>
                        <CollapsibleContent>
                            <CardContent>
                                <RewardRenderer
                                    actions={attrs.endActions}
                                    type="end"
                                />
                            </CardContent>
                        </CollapsibleContent>
                    </Card>
                </Collapsible>
            </div>

            {/* Quest Chain */}
            {attrs.endActions.nextQuest && (
                <Card>
                    <CardHeader>
                        <CardTitle className="text-lg flex items-center gap-2">
                            <ArrowRight className="h-4 w-4" />
                            Quest Chain
                        </CardTitle>
                    </CardHeader>
                    <CardContent>
                        <p className="text-sm text-muted-foreground mb-2">
                            This quest leads to another quest upon completion:
                        </p>
                        <Link href={`/quests/${attrs.endActions.nextQuest}`}>
                            <Button variant="outline">
                                View Quest #{attrs.endActions.nextQuest}
                                <ArrowRight className="h-4 w-4 ml-2" />
                            </Button>
                        </Link>
                    </CardContent>
                </Card>
            )}

            <Toaster richColors />
        </div>
    )
}

function formatTime(seconds: number): string {
    if (seconds < 60) return `${seconds}s`
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`
    const hours = Math.floor(seconds / 3600)
    const mins = Math.floor((seconds % 3600) / 60)
    return `${hours}h ${mins}m`
}
