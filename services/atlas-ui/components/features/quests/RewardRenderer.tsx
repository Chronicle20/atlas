"use client"

import { Badge } from "@/components/ui/badge"
import {
    Star,
    Coins,
    Package,
    Wand2,
    ArrowRight,
    User,
    Sparkles,
} from "lucide-react"
import type { QuestActions, ItemReward, SkillReward } from "@/types/models/quest"

interface RewardRendererProps {
    actions: QuestActions
    type: "start" | "end"
}

export function RewardRenderer({ actions, type }: RewardRendererProps) {
    const items: React.ReactNode[] = []

    // NPC (for start actions)
    if (actions.npcId) {
        items.push(
            <RewardItem
                key="npc"
                icon={<User className="h-4 w-4" />}
                label="NPC"
                value={`ID: ${actions.npcId}`}
            />
        )
    }

    // Experience reward
    if (actions.exp && actions.exp !== 0) {
        items.push(
            <RewardItem
                key="exp"
                icon={<Star className="h-4 w-4" />}
                label="Experience"
                value={`${actions.exp > 0 ? '+' : ''}${actions.exp.toLocaleString()} EXP`}
                variant={actions.exp > 0 ? "success" : "destructive"}
            />
        )
    }

    // Meso reward
    if (actions.money && actions.money !== 0) {
        items.push(
            <RewardItem
                key="money"
                icon={<Coins className="h-4 w-4" />}
                label="Meso"
                value={`${actions.money > 0 ? '+' : ''}${actions.money.toLocaleString()}`}
                variant={actions.money > 0 ? "success" : "destructive"}
            />
        )
    }

    // Fame reward
    if (actions.fame && actions.fame !== 0) {
        items.push(
            <RewardItem
                key="fame"
                icon={<Star className="h-4 w-4" />}
                label="Fame"
                value={`${actions.fame > 0 ? '+' : ''}${actions.fame}`}
                variant={actions.fame > 0 ? "success" : "destructive"}
            />
        )
    }

    // Item rewards
    if (actions.items && actions.items.length > 0) {
        actions.items.forEach((item, index) => {
            const propLabel = item.prop !== undefined
                ? item.prop === -1
                    ? " (Guaranteed)"
                    : item.prop === 0
                        ? " (Selection)"
                        : ` (${item.prop}% chance)`
                : ""
            const periodLabel = item.period && item.period > 0
                ? ` for ${formatPeriod(item.period)}`
                : ""
            const genderLabel = item.gender !== undefined && item.gender !== -1
                ? item.gender === 0 ? " (Male)" : " (Female)"
                : ""

            items.push(
                <RewardItem
                    key={`item-${index}`}
                    icon={<Package className="h-4 w-4" />}
                    label={`Item #${item.id}`}
                    value={`${item.count > 0 ? '+' : ''}${item.count}${propLabel}${periodLabel}${genderLabel}`}
                    variant={item.count > 0 ? "default" : "destructive"}
                />
            )
        })
    }

    // Skill rewards
    if (actions.skills && actions.skills.length > 0) {
        actions.skills.forEach((skill, index) => {
            const levelLabel = skill.level === -1
                ? "Remove"
                : skill.level
                    ? `Lv.${skill.level}`
                    : "Grant"
            const masterLabel = skill.masterLevel ? ` (Master: ${skill.masterLevel})` : ""
            const jobLabel = skill.jobs && skill.jobs.length > 0
                ? ` [Jobs: ${skill.jobs.join(', ')}]`
                : ""

            items.push(
                <RewardItem
                    key={`skill-${index}`}
                    icon={<Wand2 className="h-4 w-4" />}
                    label={`Skill #${skill.id}`}
                    value={`${levelLabel}${masterLabel}${jobLabel}`}
                    variant={skill.level === -1 ? "destructive" : "default"}
                />
            )
        })
    }

    // Buff item
    if (actions.buffItemId) {
        items.push(
            <RewardItem
                key="buff"
                icon={<Sparkles className="h-4 w-4" />}
                label="Buff Item"
                value={`ID: ${actions.buffItemId}`}
            />
        )
    }

    // Next quest (chain)
    if (actions.nextQuest) {
        items.push(
            <RewardItem
                key="nextQuest"
                icon={<ArrowRight className="h-4 w-4" />}
                label="Next Quest"
                value={`Quest #${actions.nextQuest}`}
                variant="default"
                isLink
            />
        )
    }

    // Minimum level for rewards
    if (actions.levelMin) {
        items.push(
            <RewardItem
                key="levelMin"
                icon={<Star className="h-4 w-4" />}
                label="Level Required"
                value={`${actions.levelMin}+`}
            />
        )
    }

    if (items.length === 0) {
        return (
            <div className="text-muted-foreground text-sm">
                No {type === "start" ? "start" : "completion"} rewards
            </div>
        )
    }

    return <div className="space-y-2">{items}</div>
}

interface RewardItemProps {
    icon: React.ReactNode
    label: string
    value: string
    variant?: "default" | "success" | "destructive"
    isLink?: boolean
}

function RewardItem({ icon, label, value, variant = "default", isLink = false }: RewardItemProps) {
    const badgeVariant = variant === "success"
        ? "default"
        : variant === "destructive"
            ? "destructive"
            : "secondary"

    return (
        <div className="flex items-center gap-2 text-sm">
            <span className="text-muted-foreground">{icon}</span>
            <span className="font-medium">{label}:</span>
            <Badge
                variant={badgeVariant}
                className={isLink ? "cursor-pointer hover:underline" : ""}
            >
                {value}
            </Badge>
        </div>
    )
}

function formatPeriod(minutes: number): string {
    if (minutes < 60) return `${minutes}m`
    if (minutes < 1440) return `${Math.floor(minutes / 60)}h`
    return `${Math.floor(minutes / 1440)}d`
}
