"use client"

import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { BanType, BanTypeLabels } from "@/types/models/ban";
import { Globe, Cpu, User } from "lucide-react";

interface BanTypeBadgeProps {
    type: BanType;
}

const typeConfig: Record<BanType, { color: string; icon: React.ReactNode; label: string }> = {
    [BanType.IP]: {
        color: "bg-blue-100 text-blue-800 hover:bg-blue-100",
        icon: <Globe className="h-3 w-3 mr-1" />,
        label: "IP"
    },
    [BanType.HWID]: {
        color: "bg-purple-100 text-purple-800 hover:bg-purple-100",
        icon: <Cpu className="h-3 w-3 mr-1" />,
        label: "HWID"
    },
    [BanType.Account]: {
        color: "bg-orange-100 text-orange-800 hover:bg-orange-100",
        icon: <User className="h-3 w-3 mr-1" />,
        label: "Account"
    }
};

export function BanTypeBadge({ type }: BanTypeBadgeProps) {
    const config = typeConfig[type] || typeConfig[BanType.IP];

    return (
        <TooltipProvider>
            <Tooltip>
                <TooltipTrigger asChild>
                    <Badge variant="secondary" className={`${config.color} flex items-center`}>
                        {config.icon}
                        {config.label}
                    </Badge>
                </TooltipTrigger>
                <TooltipContent>
                    <p>{BanTypeLabels[type] || 'Unknown Type'}</p>
                </TooltipContent>
            </Tooltip>
        </TooltipProvider>
    );
}
