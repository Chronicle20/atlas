import type {Tenant} from "@/types/models/tenant";
import {useEffect, useState} from "react";
import {mapsService} from "@/services/api";
import {Tooltip, TooltipContent, TooltipProvider, TooltipTrigger} from "@/components/ui/tooltip";
import {Badge} from "@/components/ui/badge";
import {Skeleton} from "@/components/ui/skeleton";

const mapNameCache = new Map<string, string>()

export function MapCell({ mapId, tenant }: { mapId: string; tenant: Tenant | null }) {
    const [name, setName] = useState(() => mapNameCache.get(mapId) ?? null)
    const [isLoading, setIsLoading] = useState(() => !mapNameCache.has(mapId))

    useEffect(() => {
        if (!tenant || !mapId || mapNameCache.has(mapId)) return

        setIsLoading(true)
        mapsService.getMapById(mapId)
            .then((map) => {
                const mapName = map.attributes.name
                mapNameCache.set(mapId, mapName)
                setName(mapName)
                setIsLoading(false)
            })
            .catch(() => {
                setName("Unknown")
                setIsLoading(false)
            })
    }, [mapId, tenant])

    if (isLoading) {
        return <Skeleton className="h-6 w-16 rounded-full" />
    }

    return (
        <TooltipProvider>
            <Tooltip>
                <TooltipTrigger asChild>
                    <Badge variant="secondary">
                        {name}
                    </Badge>
                </TooltipTrigger>
                <TooltipContent copyable>
                    <p>{mapId}</p>
                </TooltipContent>
            </Tooltip>
        </TooltipProvider>
    )
}
