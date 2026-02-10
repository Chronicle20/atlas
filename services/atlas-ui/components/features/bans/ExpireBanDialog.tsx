"use client"

import { useState } from "react";
import {
    AlertDialog,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";
import { bansService } from "@/services/api/bans.service";
import { BanTypeBadge } from "./BanTypeBadge";
import { BanStatusBadge } from "./BanStatusBadge";
import type { Ban } from "@/types/models/ban";
import type { Tenant } from "@/types/models/tenant";
import { Loader2, Clock } from "lucide-react";

interface ExpireBanDialogProps {
    ban: Ban | null;
    open: boolean;
    onOpenChange: (open: boolean) => void;
    tenant: Tenant | null;
    onSuccess?: () => void;
}

export function ExpireBanDialog({ ban, open, onOpenChange, tenant, onSuccess }: ExpireBanDialogProps) {
    const [isExpiring, setIsExpiring] = useState(false);

    const handleExpire = async () => {
        if (!tenant || !ban) {
            toast.error("No tenant or ban selected");
            return;
        }

        setIsExpiring(true);

        try {
            await bansService.expireBan(tenant, ban.id);
            toast.success("Ban expired successfully");
            onOpenChange(false);
            onSuccess?.();
        } catch (error) {
            toast.error("Failed to expire ban: " + (error instanceof Error ? error.message : "Unknown error"));
        } finally {
            setIsExpiring(false);
        }
    };

    if (!ban) return null;

    return (
        <AlertDialog open={open} onOpenChange={onOpenChange}>
            <AlertDialogContent>
                <AlertDialogHeader>
                    <AlertDialogTitle className="flex items-center gap-2">
                        <Clock className="h-5 w-5 text-amber-500" />
                        Expire Ban Early
                    </AlertDialogTitle>
                    <AlertDialogDescription asChild>
                        <div className="space-y-4">
                            <p>
                                Are you sure you want to expire this ban early? The player will be able to resume playing immediately.
                            </p>
                            <div className="rounded-lg border p-4 space-y-3 bg-muted/50">
                                <div className="flex items-center gap-2">
                                    <span className="text-sm font-medium text-foreground">Type:</span>
                                    <BanTypeBadge type={ban.attributes.banType} />
                                </div>
                                <div className="flex items-center gap-2">
                                    <span className="text-sm font-medium text-foreground">Value:</span>
                                    <span className="text-sm text-foreground font-mono">{ban.attributes.value}</span>
                                </div>
                                <div className="flex items-center gap-2">
                                    <span className="text-sm font-medium text-foreground">Status:</span>
                                    <BanStatusBadge
                                        permanent={ban.attributes.permanent}
                                        expiresAt={ban.attributes.expiresAt}
                                    />
                                </div>
                                {ban.attributes.reason && (
                                    <div>
                                        <span className="text-sm font-medium text-foreground">Reason:</span>
                                        <p className="text-sm text-muted-foreground mt-1">{ban.attributes.reason}</p>
                                    </div>
                                )}
                            </div>
                        </div>
                    </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                    <Button
                        variant="outline"
                        onClick={() => onOpenChange(false)}
                        disabled={isExpiring}
                    >
                        Cancel
                    </Button>
                    <Button
                        variant="default"
                        onClick={handleExpire}
                        disabled={isExpiring}
                    >
                        {isExpiring && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                        Expire Ban
                    </Button>
                </AlertDialogFooter>
            </AlertDialogContent>
        </AlertDialog>
    );
}
