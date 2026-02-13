"use client"

/**
 * Entity name components for displaying resolved names for NPCs, items, mobs, skills, and maps.
 * Uses atlas-data API for name lookups with graceful fallback to IDs.
 */

import { useNpcData } from "@/lib/hooks/useNpcData"
import { useItemData } from "@/lib/hooks/useItemData"
import { useMobData } from "@/lib/hooks/useMobData"
import { useSkillData } from "@/lib/hooks/useSkillData"
import { Skeleton } from "@/components/ui/skeleton"

interface EntityNameProps {
    id: number
    showId?: boolean
    className?: string
}

/**
 * Display NPC name with fallback to ID
 */
export function NpcName({ id, showId = false, className }: EntityNameProps) {
    const { name, isLoading, hasError } = useNpcData(id)

    if (isLoading) {
        return <Skeleton className="h-4 w-16 inline-block" />
    }

    if (hasError || !name) {
        return <span className={className}>NPC #{id}</span>
    }

    return (
        <span className={className}>
            {name}
            {showId && <span className="text-muted-foreground ml-1">(#{id})</span>}
        </span>
    )
}

/**
 * Display Item name with fallback to ID
 */
export function ItemName({ id, showId = false, className }: EntityNameProps) {
    const { name, isLoading, hasError } = useItemData(id)

    if (isLoading) {
        return <Skeleton className="h-4 w-16 inline-block" />
    }

    if (hasError || !name) {
        return <span className={className}>Item #{id}</span>
    }

    return (
        <span className={className}>
            {name}
            {showId && <span className="text-muted-foreground ml-1">(#{id})</span>}
        </span>
    )
}

/**
 * Display Mob/Monster name with fallback to ID
 */
export function MobName({ id, showId = false, className }: EntityNameProps) {
    const { name, isLoading, hasError } = useMobData(id)

    if (isLoading) {
        return <Skeleton className="h-4 w-16 inline-block" />
    }

    if (hasError || !name) {
        return <span className={className}>Monster #{id}</span>
    }

    return (
        <span className={className}>
            {name}
            {showId && <span className="text-muted-foreground ml-1">(#{id})</span>}
        </span>
    )
}

/**
 * Display Skill name with fallback to ID
 */
export function SkillName({ id, showId = false, className }: EntityNameProps) {
    const { name, isLoading, hasError } = useSkillData(id)

    if (isLoading) {
        return <Skeleton className="h-4 w-16 inline-block" />
    }

    if (hasError || !name) {
        return <span className={className}>Skill #{id}</span>
    }

    return (
        <span className={className}>
            {name}
            {showId && <span className="text-muted-foreground ml-1">(#{id})</span>}
        </span>
    )
}

/**
 * Job name mapping (static since jobs are fixed)
 */
const JOB_NAMES: Record<number, string> = {
    0: "Beginner",
    100: "Warrior",
    110: "Fighter",
    111: "Crusader",
    112: "Hero",
    120: "Page",
    121: "White Knight",
    122: "Paladin",
    130: "Spearman",
    131: "Dragon Knight",
    132: "Dark Knight",
    200: "Magician",
    210: "Wizard (F/P)",
    211: "Mage (F/P)",
    212: "Arch Mage (F/P)",
    220: "Wizard (I/L)",
    221: "Mage (I/L)",
    222: "Arch Mage (I/L)",
    230: "Cleric",
    231: "Priest",
    232: "Bishop",
    300: "Archer",
    310: "Hunter",
    311: "Ranger",
    312: "Bowmaster",
    320: "Crossbowman",
    321: "Sniper",
    322: "Marksman",
    400: "Rogue",
    410: "Assassin",
    411: "Hermit",
    412: "Night Lord",
    420: "Bandit",
    421: "Chief Bandit",
    422: "Shadower",
    500: "Pirate",
    510: "Brawler",
    511: "Marauder",
    512: "Buccaneer",
    520: "Gunslinger",
    521: "Outlaw",
    522: "Corsair",
    900: "GM",
    910: "Super GM",
    1000: "Noblesse",
    1100: "Dawn Warrior 1",
    1110: "Dawn Warrior 2",
    1111: "Dawn Warrior 3",
    1112: "Dawn Warrior 4",
    1200: "Blaze Wizard 1",
    1210: "Blaze Wizard 2",
    1211: "Blaze Wizard 3",
    1212: "Blaze Wizard 4",
    1300: "Wind Archer 1",
    1310: "Wind Archer 2",
    1311: "Wind Archer 3",
    1312: "Wind Archer 4",
    1400: "Night Walker 1",
    1410: "Night Walker 2",
    1411: "Night Walker 3",
    1412: "Night Walker 4",
    1500: "Thunder Breaker 1",
    1510: "Thunder Breaker 2",
    1511: "Thunder Breaker 3",
    1512: "Thunder Breaker 4",
    2000: "Legend",
    2001: "Evan 1",
    2100: "Aran 1",
    2110: "Aran 2",
    2111: "Aran 3",
    2112: "Aran 4",
    2200: "Evan 2",
    2210: "Evan 3",
    2211: "Evan 4",
    2212: "Evan 5",
    2213: "Evan 6",
    2214: "Evan 7",
    2215: "Evan 8",
    2216: "Evan 9",
    2217: "Evan 10",
    2218: "Evan Master",
}

/**
 * Display Job name with fallback to ID
 */
export function JobName({ id, showId = false, className }: Omit<EntityNameProps, 'region' | 'version'>) {
    const name = JOB_NAMES[id]

    if (!name) {
        return <span className={className}>Job #{id}</span>
    }

    return (
        <span className={className}>
            {name}
            {showId && <span className="text-muted-foreground ml-1">(#{id})</span>}
        </span>
    )
}
