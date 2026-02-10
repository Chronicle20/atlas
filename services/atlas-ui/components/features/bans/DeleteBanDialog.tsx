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
import { Loader2, AlertTriangle } from "lucide-react";

interface DeleteBanDialogProps {
    ban: Ban | null;
    open: boolean;
    onOpenChange: (open: boolean) => void;
    tenant: Tenant | null;
    onSuccess?: () => void;
}

export function DeleteBanDialog({ ban, open, onOpenChange, tenant, onSuccess }: DeleteBanDialogProps) {
    const [isDeleting, setIsDeleting] = useState(false);

    const handleDelete = async () => {
        if (!tenant || !ban) {
            toast.error("No tenant or ban selected");
            return;
        }

        setIsDeleting(true);

        try {
            await bansService.deleteBan(tenant, ban.id);
            toast.success("Ban deleted successfully");
            onOpenChange(false);
            onSuccess?.();
        } catch (error) {
            toast.error("Failed to delete ban: " + (error instanceof Error ? error.message : "Unknown error"));
        } finally {
            setIsDeleting(false);
        }
    };

    if (!ban) return null;

    return (
        <AlertDialog open={open} onOpenChange={onOpenChange}>
            <AlertDialogContent>
                <AlertDialogHeader>
                    <AlertDialogTitle className="flex items-center gap-2">
                        <AlertTriangle className="h-5 w-5 text-destructive" />
                        Delete Ban
                    </AlertDialogTitle>
                    <AlertDialogDescription asChild>
                        <div className="space-y-4">
                            <p>
                                Are you sure you want to delete this ban? This action cannot be undone.
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
                        disabled={isDeleting}
                    >
                        Cancel
                    </Button>
                    <Button
                        variant="destructive"
                        onClick={handleDelete}
                        disabled={isDeleting}
                    >
                        {isDeleting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                        Delete Ban
                    </Button>
                </AlertDialogFooter>
            </AlertDialogContent>
        </AlertDialog>
    );
}
