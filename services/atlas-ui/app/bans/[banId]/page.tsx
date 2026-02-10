"use client"

import { useTenant } from "@/context/tenant-context";
import { useCallback, useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { bansService } from "@/services/api/bans.service";
import { Ban, BanTypeLabels, BanReasonCodeLabels, formatBanExpiration, isBanActive } from "@/types/models/ban";
import { BanTypeBadge } from "@/components/features/bans/BanTypeBadge";
import { BanStatusBadge } from "@/components/features/bans/BanStatusBadge";
import { DeleteBanDialog } from "@/components/features/bans/DeleteBanDialog";
import { Toaster, toast } from "sonner";
import { createErrorFromUnknown } from "@/types/api/errors";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Separator } from "@/components/ui/separator";
import { ArrowLeft, Trash2, Shield, Calendar, User, FileText, Hash } from "lucide-react";

function BanDetailSkeleton() {
    return (
        <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
            <div className="flex items-center gap-4">
                <Skeleton className="h-9 w-9" />
                <Skeleton className="h-8 w-48" />
            </div>
            <Card>
                <CardHeader>
                    <Skeleton className="h-6 w-32" />
                    <Skeleton className="h-4 w-48" />
                </CardHeader>
                <CardContent className="space-y-6">
                    {Array.from({ length: 6 }).map((_, i) => (
                        <div key={i} className="space-y-2">
                            <Skeleton className="h-4 w-24" />
                            <Skeleton className="h-6 w-48" />
                        </div>
                    ))}
                </CardContent>
            </Card>
        </div>
    );
}

export default function BanDetailPage() {
    const { activeTenant } = useTenant();
    const params = useParams();
    const router = useRouter();
    const banId = params.banId as string;

    const [ban, setBan] = useState<Ban | null>(null);
    const [loading, setLoading] = useState(true);
    const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);

    const fetchBan = useCallback(async () => {
        if (!activeTenant || !banId) return;

        setLoading(true);

        try {
            const data = await bansService.getBanById(activeTenant, banId);
            setBan(data);
        } catch (err: unknown) {
            const errorInfo = createErrorFromUnknown(err, "Failed to fetch ban");
            toast.error(errorInfo.message);
            router.push("/bans");
        } finally {
            setLoading(false);
        }
    }, [activeTenant, banId, router]);

    useEffect(() => {
        fetchBan();
    }, [fetchBan]);

    const handleDeleteSuccess = () => {
        router.push("/bans");
    };

    if (loading) {
        return <BanDetailSkeleton />;
    }

    if (!ban) {
        return (
            <div className="flex flex-col flex-1 items-center justify-center p-10">
                <p className="text-muted-foreground">Ban not found</p>
                <Button variant="outline" className="mt-4" onClick={() => router.push("/bans")}>
                    <ArrowLeft className="mr-2 h-4 w-4" />
                    Back to Bans
                </Button>
            </div>
        );
    }

    return (
        <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                    <Button variant="ghost" size="icon" onClick={() => router.push("/bans")}>
                        <ArrowLeft className="h-4 w-4" />
                    </Button>
                    <div className="flex items-center gap-2">
                        <Shield className="h-6 w-6" />
                        <h2 className="text-2xl font-bold tracking-tight">Ban Details</h2>
                    </div>
                </div>
                <Button
                    variant="destructive"
                    onClick={() => setDeleteDialogOpen(true)}
                >
                    <Trash2 className="mr-2 h-4 w-4" />
                    Delete Ban
                </Button>
            </div>

            <Card>
                <CardHeader>
                    <div className="flex items-center justify-between">
                        <div>
                            <CardTitle className="flex items-center gap-2">
                                <Hash className="h-5 w-5" />
                                Ban #{ban.id}
                            </CardTitle>
                            <CardDescription>
                                {isBanActive(ban) ? "This ban is currently active" : "This ban has expired"}
                            </CardDescription>
                        </div>
                        <div className="flex items-center gap-2">
                            <BanTypeBadge type={ban.attributes.banType} />
                            <BanStatusBadge
                                permanent={ban.attributes.permanent}
                                expiresAt={ban.attributes.expiresAt}
                            />
                        </div>
                    </div>
                </CardHeader>
                <CardContent className="space-y-6">
                    <div className="grid gap-6 md:grid-cols-2">
                        <div className="space-y-1">
                            <label className="text-sm font-medium text-muted-foreground flex items-center gap-2">
                                <Shield className="h-4 w-4" />
                                Ban Type
                            </label>
                            <p className="text-lg font-medium">
                                {BanTypeLabels[ban.attributes.banType]}
                            </p>
                        </div>

                        <div className="space-y-1">
                            <label className="text-sm font-medium text-muted-foreground flex items-center gap-2">
                                <Hash className="h-4 w-4" />
                                Value
                            </label>
                            <p className="text-lg font-mono">
                                {ban.attributes.value}
                            </p>
                        </div>

                        <div className="space-y-1">
                            <label className="text-sm font-medium text-muted-foreground flex items-center gap-2">
                                <FileText className="h-4 w-4" />
                                Reason Code
                            </label>
                            <p className="text-lg">
                                {BanReasonCodeLabels[ban.attributes.reasonCode]}
                            </p>
                        </div>

                        <div className="space-y-1">
                            <label className="text-sm font-medium text-muted-foreground flex items-center gap-2">
                                <Calendar className="h-4 w-4" />
                                Expires At
                            </label>
                            <p className="text-lg">
                                {formatBanExpiration(ban)}
                            </p>
                        </div>

                        <div className="space-y-1">
                            <label className="text-sm font-medium text-muted-foreground flex items-center gap-2">
                                <User className="h-4 w-4" />
                                Issued By
                            </label>
                            <p className="text-lg">
                                {ban.attributes.issuedBy || <span className="text-muted-foreground">Not specified</span>}
                            </p>
                        </div>
                    </div>

                    {ban.attributes.reason && (
                        <>
                            <Separator />
                            <div className="space-y-1">
                                <label className="text-sm font-medium text-muted-foreground flex items-center gap-2">
                                    <FileText className="h-4 w-4" />
                                    Reason Details
                                </label>
                                <p className="text-base whitespace-pre-wrap rounded-lg bg-muted p-4">
                                    {ban.attributes.reason}
                                </p>
                            </div>
                        </>
                    )}
                </CardContent>
            </Card>

            <DeleteBanDialog
                ban={ban}
                open={deleteDialogOpen}
                onOpenChange={setDeleteDialogOpen}
                tenant={activeTenant}
                onSuccess={handleDeleteSuccess}
            />

            <Toaster richColors />
        </div>
    );
}
