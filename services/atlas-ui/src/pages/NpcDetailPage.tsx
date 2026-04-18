
import { useTenant } from "@/context/tenant-context";
import { useQuery } from "@tanstack/react-query";
import { npcsService } from "@/services/api";
import { useParams } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { ShoppingBag, MessageCircle, RefreshCw } from "lucide-react";
import { Link } from "react-router-dom";
import { NpcImage } from "@/components/features/npc/NpcImage";
import { useNpcData } from "@/lib/hooks/useNpcData";
import { Skeleton } from "@/components/ui/skeleton";
import { ErrorDisplay } from "@/components/common/ErrorDisplay";

export function NpcDetailPage() {
    const { activeTenant } = useTenant();
    const params = useParams();
    const npcId = Number(params.id);

    // Fetch NPC metadata (name, icon) from external API
    const {
        name: npcName,
        iconUrl: npcIconUrl,
        isLoading: isMetadataLoading
    } = useNpcData(npcId, {
        enabled: npcId > 0,
    });

    const npcQuery = useQuery({
        queryKey: ["npcs", "detail", activeTenant?.id ?? "no-tenant", npcId],
        queryFn: async () => {
            const npcData = await npcsService.getNPCById(npcId, activeTenant!);
            // If the NPC isn't in shops/conversations, synthesise a basic entry.
            return npcData ?? { id: npcId, hasShop: false, hasConversation: false };
        },
        enabled: !!activeTenant && npcId > 0,
        staleTime: 60 * 1000,
    });

    const npc = npcQuery.data ?? null;
    const loading = npcQuery.isLoading;
    const error = npcQuery.error?.message ?? null;
    const fetchNpcData = () => { npcQuery.refetch(); };

    if (loading) {
        return (
            <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
                <div className="flex items-center space-x-4">
                    <Skeleton className="h-8 w-48" />
                </div>
                <Card className="max-w-md">
                    <CardHeader className="pb-4">
                        <div className="flex items-center space-x-4">
                            <Skeleton className="h-24 w-24 rounded-lg" />
                            <div className="space-y-2">
                                <Skeleton className="h-6 w-32" />
                                <Skeleton className="h-4 w-20" />
                            </div>
                        </div>
                    </CardHeader>
                    <CardContent>
                        <div className="flex space-x-3">
                            <Skeleton className="h-10 w-32" />
                            <Skeleton className="h-10 w-32" />
                        </div>
                    </CardContent>
                </Card>
            </div>
        );
    }

    if (error) {
        return (
            <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
                <ErrorDisplay error={error} retry={fetchNpcData} />
            </div>
        );
    }

    const displayName = npcName || `NPC #${npcId}`;

    return (
        <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
            <div className="flex items-center justify-between">
                <h2 className="text-2xl font-bold tracking-tight">{displayName}</h2>
                <Button
                    variant="outline"
                    size="icon"
                    onClick={fetchNpcData}
                    className="hover:bg-accent cursor-pointer"
                    title="Refresh"
                >
                    <RefreshCw className="h-4 w-4" />
                </Button>
            </div>

            <Card className="max-w-md">
                <CardHeader className="pb-4">
                    <div className="flex items-center space-x-4">
                        <NpcImage
                            npcId={npcId}
                            name={npcName}
                            iconUrl={npcIconUrl}
                            size={96}
                            className="rounded-lg bg-muted"
                            lazy={false}
                            showRetryButton={true}
                            maxRetries={3}
                        />
                        <div className="flex flex-col space-y-1">
                            {isMetadataLoading ? (
                                <>
                                    <Skeleton className="h-6 w-32" />
                                    <Skeleton className="h-4 w-20" />
                                </>
                            ) : (
                                <>
                                    <CardTitle className="text-xl">
                                        {displayName}
                                    </CardTitle>
                                    <p className="text-sm text-muted-foreground">
                                        ID: {npcId}
                                    </p>
                                </>
                            )}
                        </div>
                    </div>
                </CardHeader>

                <CardContent>
                    <div className="flex flex-col space-y-4">
                        <div className="flex space-x-3">
                            {npc?.hasShop ? (
                                <Button
                                    variant="default"
                                    asChild
                                    className="cursor-pointer"
                                >
                                    <Link to={`/npcs/${npcId}/shop`}>
                                        <ShoppingBag className="h-4 w-4 mr-2" />
                                        View Shop
                                    </Link>
                                </Button>
                            ) : (
                                <Button
                                    variant="outline"
                                    disabled
                                    className="cursor-not-allowed opacity-50"
                                    title="No Shop Available"
                                >
                                    <ShoppingBag className="h-4 w-4 mr-2" />
                                    No Shop
                                </Button>
                            )}

                            {npc?.hasConversation ? (
                                <Button
                                    variant="default"
                                    asChild
                                    className="cursor-pointer"
                                >
                                    <Link to={`/npcs/${npcId}/conversations`}>
                                        <MessageCircle className="h-4 w-4 mr-2" />
                                        View Conversation
                                    </Link>
                                </Button>
                            ) : (
                                <Button
                                    variant="outline"
                                    disabled
                                    className="cursor-not-allowed opacity-50"
                                    title="No Conversation Available"
                                >
                                    <MessageCircle className="h-4 w-4 mr-2" />
                                    No Conversation
                                </Button>
                            )}
                        </div>

                        {!npc?.hasShop && !npc?.hasConversation && (
                            <p className="text-sm text-muted-foreground">
                                This NPC has no shop or conversation configured.
                            </p>
                        )}
                    </div>
                </CardContent>
            </Card>
        </div>
    );
}
