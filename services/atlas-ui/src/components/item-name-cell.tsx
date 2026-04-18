import type { Tenant } from "@/types/models/tenant";
import { useEffect, useState } from "react";
import { itemsService } from "@/services/api/items.service";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";

const itemNameCache = new Map<string, string>()

export function ItemNameCell({ itemId, tenant }: { itemId: string; tenant: Tenant | null }) {
    const [name, setName] = useState(() => itemNameCache.get(itemId) ?? null)
    const [isLoading, setIsLoading] = useState(() => !itemNameCache.has(itemId))

    useEffect(() => {
        if (!tenant || !itemId || itemNameCache.has(itemId)) return

        setIsLoading(true)
        itemsService.getItemName(itemId, tenant)
            .then((itemName) => {
                itemNameCache.set(itemId, itemName)
                setName(itemName)
                setIsLoading(false)
            })
            .catch(() => {
                itemNameCache.set(itemId, itemId)
                setName(itemId)
                setIsLoading(false)
            })
    }, [itemId, tenant])

    if (isLoading) {
        return <Skeleton className="h-6 w-20 rounded-full" />
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
                    <p>{itemId}</p>
                </TooltipContent>
            </Tooltip>
        </TooltipProvider>
    )
}
