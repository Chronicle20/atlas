"use client"

import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { Shield, Clock, Ban } from "lucide-react";
import { isZeroTime } from "@/types/models/ban";

interface BanStatusBadgeProps {
    permanent: boolean;
    expiresAt: string;
}

function isExpired(expiresAt: string): boolean {
    return !isZeroTime(expiresAt) && new Date(expiresAt) < new Date();
}

export function BanStatusBadge({ permanent, expiresAt }: BanStatusBadgeProps) {
    if (permanent) {
        return (
            <TooltipProvider>
                <Tooltip>
                    <TooltipTrigger asChild>
                        <Badge variant="destructive" className="flex items-center">
                            <Ban className="h-3 w-3 mr-1" />
                            Permanent
                        </Badge>
                    </TooltipTrigger>
                    <TooltipContent>
                        <p>This ban will never expire</p>
                    </TooltipContent>
                </Tooltip>
            </TooltipProvider>
        );
    }

    if (isExpired(expiresAt)) {
        return (
            <TooltipProvider>
                <Tooltip>
                    <TooltipTrigger asChild>
                        <Badge variant="secondary" className="bg-gray-100 text-gray-600 hover:bg-gray-100 flex items-center">
                            <Clock className="h-3 w-3 mr-1" />
                            Expired
                        </Badge>
                    </TooltipTrigger>
                    <TooltipContent>
                        <p>Expired on {new Date(expiresAt).toLocaleString()}</p>
                    </TooltipContent>
                </Tooltip>
            </TooltipProvider>
        );
    }

    return (
        <TooltipProvider>
            <Tooltip>
                <TooltipTrigger asChild>
                    <Badge variant="secondary" className="bg-yellow-100 text-yellow-800 hover:bg-yellow-100 flex items-center">
                        <Shield className="h-3 w-3 mr-1" />
                        Active
                    </Badge>
                </TooltipTrigger>
                <TooltipContent>
                    <p>Expires on {new Date(expiresAt).toLocaleString()}</p>
                </TooltipContent>
            </Tooltip>
        </TooltipProvider>
    );
}
