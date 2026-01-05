"use client"

import { Badge } from "@/components/ui/badge"
import {
    User,
    Sword,
    Package,
    Skull,
    MapPin,
    Star,
    Coins,
    ScrollText,
    Clock,
    Calendar,
    Dog,
} from "lucide-react"
import type { QuestRequirements, QuestRequirement, ItemRequirement, MobRequirement } from "@/types/models/quest"

interface RequirementRendererProps {
    requirements: QuestRequirements
    type: "start" | "end"
}

export function RequirementRenderer({ requirements, type }: RequirementRendererProps) {
    const items: React.ReactNode[] = []

    // NPC requirement
    if (requirements.npcId) {
        items.push(
            <RequirementItem
                key="npc"
                icon={<User className="h-4 w-4" />}
                label="NPC"
                value={`ID: ${requirements.npcId}`}
            />
        )
    }

    // Level requirements
    if (requirements.levelMin || requirements.levelMax) {
        const levelText = requirements.levelMin && requirements.levelMax
            ? `${requirements.levelMin} - ${requirements.levelMax}`
            : requirements.levelMin
                ? `${requirements.levelMin}+`
                : `1 - ${requirements.levelMax}`
        items.push(
            <RequirementItem
                key="level"
                icon={<Star className="h-4 w-4" />}
                label="Level"
                value={levelText}
            />
        )
    }

    // Fame requirement
    if (requirements.fameMin) {
        items.push(
            <RequirementItem
                key="fame"
                icon={<Star className="h-4 w-4" />}
                label="Fame"
                value={`${requirements.fameMin}+`}
            />
        )
    }

    // Meso requirements
    if (requirements.mesoMin || requirements.mesoMax) {
        const mesoText = requirements.mesoMin && requirements.mesoMax
            ? `${requirements.mesoMin.toLocaleString()} - ${requirements.mesoMax.toLocaleString()}`
            : requirements.mesoMin
                ? `${requirements.mesoMin.toLocaleString()}+`
                : `Max ${requirements.mesoMax?.toLocaleString()}`
        items.push(
            <RequirementItem
                key="meso"
                icon={<Coins className="h-4 w-4" />}
                label="Meso"
                value={mesoText}
            />
        )
    }

    // Job requirements
    if (requirements.jobs && requirements.jobs.length > 0) {
        items.push(
            <RequirementItem
                key="jobs"
                icon={<Sword className="h-4 w-4" />}
                label="Job"
                value={requirements.jobs.map(j => `ID: ${j}`).join(", ")}
            />
        )
    }

    // Quest requirements
    if (requirements.quests && requirements.quests.length > 0) {
        requirements.quests.forEach((quest, index) => {
            const stateLabel = quest.state === 0 ? "Not Started" : quest.state === 1 ? "Started" : "Completed"
            items.push(
                <RequirementItem
                    key={`quest-${index}`}
                    icon={<ScrollText className="h-4 w-4" />}
                    label={`Quest #${quest.id}`}
                    value={stateLabel}
                />
            )
        })
    }

    // Item requirements
    if (requirements.items && requirements.items.length > 0) {
        requirements.items.forEach((item, index) => {
            items.push(
                <RequirementItem
                    key={`item-${index}`}
                    icon={<Package className="h-4 w-4" />}
                    label={`Item #${item.id}`}
                    value={item.count > 0 ? `x${item.count}` : `Remove ${Math.abs(item.count)}`}
                    variant={item.count < 0 ? "destructive" : "default"}
                />
            )
        })
    }

    // Mob kill requirements
    if (requirements.mobs && requirements.mobs.length > 0) {
        requirements.mobs.forEach((mob, index) => {
            items.push(
                <RequirementItem
                    key={`mob-${index}`}
                    icon={<Skull className="h-4 w-4" />}
                    label={`Monster #${mob.id}`}
                    value={`Kill x${mob.count}`}
                />
            )
        })
    }

    // Map requirements
    if (requirements.fieldEnter && requirements.fieldEnter.length > 0) {
        requirements.fieldEnter.forEach((mapId, index) => {
            items.push(
                <RequirementItem
                    key={`map-${index}`}
                    icon={<MapPin className="h-4 w-4" />}
                    label="Visit Map"
                    value={`ID: ${mapId}`}
                />
            )
        })
    }

    // Pet requirements
    if (requirements.pet && requirements.pet.length > 0) {
        items.push(
            <RequirementItem
                key="pet"
                icon={<Dog className="h-4 w-4" />}
                label="Pet"
                value={requirements.pet.map(p => `ID: ${p}`).join(", ")}
            />
        )
    }

    // Pet tameness requirement
    if (requirements.petTamenessMin) {
        items.push(
            <RequirementItem
                key="petTameness"
                icon={<Dog className="h-4 w-4" />}
                label="Pet Tameness"
                value={`${requirements.petTamenessMin}+`}
            />
        )
    }

    // Time-based requirements
    if (requirements.dayOfWeek) {
        items.push(
            <RequirementItem
                key="dayOfWeek"
                icon={<Calendar className="h-4 w-4" />}
                label="Day of Week"
                value={requirements.dayOfWeek}
            />
        )
    }

    if (requirements.start || requirements.end) {
        const timeText = requirements.start && requirements.end
            ? `${requirements.start} - ${requirements.end}`
            : requirements.start || requirements.end
        items.push(
            <RequirementItem
                key="time"
                icon={<Clock className="h-4 w-4" />}
                label="Time"
                value={timeText || ""}
            />
        )
    }

    // Interval requirement
    if (requirements.interval) {
        items.push(
            <RequirementItem
                key="interval"
                icon={<Clock className="h-4 w-4" />}
                label="Interval"
                value={formatInterval(requirements.interval)}
            />
        )
    }

    // Completion count requirement
    if (requirements.completionCount) {
        items.push(
            <RequirementItem
                key="completionCount"
                icon={<ScrollText className="h-4 w-4" />}
                label="Completions Required"
                value={`${requirements.completionCount}`}
            />
        )
    }

    if (items.length === 0) {
        return (
            <div className="text-muted-foreground text-sm">
                No {type === "start" ? "start" : "completion"} requirements
            </div>
        )
    }

    return <div className="space-y-2">{items}</div>
}

interface RequirementItemProps {
    icon: React.ReactNode
    label: string
    value: string
    variant?: "default" | "destructive"
}

function RequirementItem({ icon, label, value, variant = "default" }: RequirementItemProps) {
    return (
        <div className="flex items-center gap-2 text-sm">
            <span className="text-muted-foreground">{icon}</span>
            <span className="font-medium">{label}:</span>
            <Badge variant={variant === "destructive" ? "destructive" : "secondary"}>
                {value}
            </Badge>
        </div>
    )
}

function formatInterval(seconds: number): string {
    if (seconds < 60) return `${seconds}s`
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m`
    if (seconds < 86400) return `${Math.floor(seconds / 3600)}h`
    return `${Math.floor(seconds / 86400)}d`
}
