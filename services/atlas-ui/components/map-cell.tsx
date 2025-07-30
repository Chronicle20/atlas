import type {Tenant} from "@/types/models/tenant";
import {useEffect, useState} from "react";
import {mapsService} from "@/services/api";
import {Tooltip, TooltipContent, TooltipProvider, TooltipTrigger} from "@/components/ui/tooltip";
import {Badge} from "@/components/ui/badge";

const mapNameCache = new Map<string, string>()

export function MapCell({ mapId, tenant }: { mapId: string; tenant: Tenant | null }) {
    const [name, setName] = useState(() => mapNameCache.get(mapId) ?? "Loading...")

    useEffect(() => {
        if (!tenant || !mapId || mapNameCache.has(mapId)) return

        mapsService.fetchMap(tenant, mapId)
            .then((map) => {
                const mapName = map.attributes.name
                mapNameCache.set(mapId, mapName)
                setName(mapName)
            })
            .catch(() => setName("Unknown"))
    }, [mapId])

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
