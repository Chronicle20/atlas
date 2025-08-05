"use client"

import { useTenant } from "@/context/tenant-context";
import { useCallback, useEffect, useState, useMemo } from "react";
import {npcsService} from "@/services/api";
import {NPC, Commodity} from "@/types/models/npc";
import { tenantHeaders } from "@/lib/headers";
import { Button } from "@/components/ui/button";
import { RefreshCw } from "lucide-react";
import { toast } from "sonner";
import { Toaster } from "@/components/ui/sonner";
import dynamic from "next/dynamic";
import {createErrorFromUnknown} from "@/types/api/errors";
import {ErrorDisplay} from "@/components/common/ErrorDisplay";
import {NpcPageSkeleton} from "@/components/common/skeletons/NpcPageSkeleton";
import { VirtualizedNpcGrid } from "@/components/features/npc/VirtualizedNpcGrid";
import { useOptimizedNpcBatchData } from "@/lib/hooks/useNpcData";
import { useNpcErrorHandler } from "@/lib/hooks/useNpcErrorHandler";
import { ErrorBoundary } from "@/components/common/ErrorBoundary";

// Dynamic imports for performance optimization
const NpcDialogs = dynamic(() => import("@/components/features/npc/NpcDialogs").then(mod => ({ default: mod.NpcDialogs })), {
  loading: () => null,
  ssr: false,
});

// Dynamic import for heavy UI components that aren't immediately needed
const AdvancedNpcActions = dynamic(() => import("@/components/features/npc/AdvancedNpcActions").then(mod => ({ default: mod.AdvancedNpcActions })), {
  loading: () => (
    <div className="flex items-center gap-2">
      <div className="h-8 w-20 bg-muted rounded animate-pulse" />
    </div>
  ),
  ssr: false,
});

export default function Page() {
    const { activeTenant } = useTenant();
    const [npcs, setNpcs] = useState<NPC[]>([]);
    const [npcsWithMetadata, setNpcsWithMetadata] = useState<NPC[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [isCreateShopDialogOpen, setIsCreateShopDialogOpen] = useState(false);
    const [isDeleteAllShopsDialogOpen, setIsDeleteAllShopsDialogOpen] = useState(false);
    const [isBulkUpdateShopDialogOpen, setIsBulkUpdateShopDialogOpen] = useState(false);
    const [selectedNpcId, setSelectedNpcId] = useState<number | null>(null);
    const [createShopJson, setCreateShopJson] = useState("");
    const [bulkUpdateShopJson, setBulkUpdateShopJson] = useState("");
    const [containerHeight, setContainerHeight] = useState(600);

    // Initialize error handler for batch operations
    const { handleErrors, handleError } = useNpcErrorHandler({
        showToasts: true,
        logErrors: true,
        maxToastsPerMinute: 5, // Allow more toasts for batch operations
    });

    const fetchDataAgain = useCallback(() => {
        if (!activeTenant) return;

        setLoading(true);

        npcsService.getAllNPCs(activeTenant)
            .then((npcData) => {
                setNpcs(npcData);
            })
            .catch((err: unknown) => {
                const errorInfo = createErrorFromUnknown(err, "Failed to fetch NPCs");
                setError(errorInfo.message);
            })
            .finally(() => setLoading(false));
    }, [activeTenant]);

    // Extract NPC IDs for batch data fetching
    const npcIds = useMemo(() => npcs.map(npc => npc.id), [npcs]);
    
    // Memoize batch data options to prevent unnecessary re-fetching
    const batchDataOptions = useMemo(() => ({
        enabled: npcIds.length > 0,
        staleTime: 30 * 60 * 1000, // 30 minutes
        onError: (error: Error) => {
            // Handle error without dependency on handleError function
            console.error('Batch metadata fetch error:', error);
        },
    }), [npcIds.length]);
    
    // Fetch NPC metadata (names and icons) in batch using optimized hook
    const { data: npcDataResults, isLoading: isMetadataLoading, error: metadataError } = useOptimizedNpcBatchData(npcIds, batchDataOptions);

    // Merge original NPC data with fetched metadata
    useEffect(() => {
        if (!npcDataResults || npcDataResults.length === 0) {
            setNpcsWithMetadata(npcs);
            return;
        }

        const updatedNpcs: NPC[] = npcs.map(npc => {
            const metadata = npcDataResults.find(result => 
                result && !result.error && result.id === npc.id
            );
            
            // Only apply metadata if it exists and has no error
            if (metadata && !metadata.error) {
                return {
                    ...npc,
                    ...('name' in metadata && metadata.name && typeof metadata.name === 'string' && { name: metadata.name }),
                    ...('iconUrl' in metadata && metadata.iconUrl && typeof metadata.iconUrl === 'string' && { iconUrl: metadata.iconUrl }),
                };
            }
            
            return npc;
        });
        
        setNpcsWithMetadata(updatedNpcs);
    }, [npcs, npcDataResults]);

    // Handle metadata errors separately
    useEffect(() => {
        if (metadataError) {
            // Log errors for debugging without causing re-renders
            console.error('Metadata fetch error:', metadataError);
        }
    }, [metadataError]); // Safe to depend on error object

    const handleCreateShop = async () => {
        if (!activeTenant) return;

        try {
            // Send the entire JSON as is to the server
            const jsonData = JSON.parse(createShopJson);

            if (!jsonData.data || !jsonData.data.attributes || !jsonData.data.attributes.npcId) {
                toast.error("Invalid JSON format. Missing npcId in data.attributes");
                return;
            }

            const npcId = parseInt(jsonData.data.attributes.npcId);
            if (isNaN(npcId)) {
                toast.error("Please provide a valid NPC ID in the JSON");
                return;
            }

            const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
            const response = await fetch(rootUrl + "/api/npcs/" + npcId + "/shop", {
                method: "POST",
                headers: tenantHeaders(activeTenant),
                body: createShopJson
            });

            if (!response.ok) {
                throw new Error("Failed to create shop.");
            }
            await response.json();
            toast.success("Shop created successfully");
            setIsCreateShopDialogOpen(false);
            setCreateShopJson("");
            fetchDataAgain();
        } catch (err: unknown) {
            toast.error("Failed to create shop: " + (err instanceof Error ? err.message : String(err)));
        }
    };

    const handleDeleteAllShops = async () => {
        if (!activeTenant) return;

        try {
            await npcsService.deleteAllShops(activeTenant);
            toast.success("All shops deleted successfully");
            setIsDeleteAllShopsDialogOpen(false);
            fetchDataAgain();
        } catch (err: unknown) {
            toast.error("Failed to delete all shops: " + (err instanceof Error ? err.message : String(err)));
        }
    };

    const handleBulkUpdateShop = async () => {
        if (!activeTenant || !selectedNpcId) return;

        try {
            const jsonData = JSON.parse(bulkUpdateShopJson);

            // Extract commodities from the included array if available
            let commoditiesToUpdate: Commodity[] = [];

            // Check for commodities in the root level included array (standard JSON:API format)
            if (jsonData.included && jsonData.included.length > 0) {
                commoditiesToUpdate = jsonData.included;
            }
            // Fallback to check for commodities in data.included (alternative format)
            else if (jsonData.data.included && jsonData.data.included.length > 0) {
                commoditiesToUpdate = jsonData.data.included;
            }

            // Get recharger value from JSON data if available
            const rechargerValue = jsonData.data.attributes?.recharger;

            await npcsService.updateShop(selectedNpcId, commoditiesToUpdate, activeTenant, rechargerValue);
            setIsBulkUpdateShopDialogOpen(false);
            setBulkUpdateShopJson("");
            fetchDataAgain();
            toast.success("Shop updated successfully");
        } catch (err: unknown) {
            toast.error("Failed to update shop: " + (err instanceof Error ? err.message : String(err)));
        }
    };

    useEffect(() => {
        fetchDataAgain();
    }, [activeTenant, fetchDataAgain]);

    // Prevent body scrolling when this page is mounted
    useEffect(() => {
        // Save the original styles
        const originalOverflow = document.documentElement.style.overflow;

        // Apply overflow: hidden to html element
        document.documentElement.style.overflow = 'hidden';

        // Cleanup function to restore original styles when component unmounts
        return () => {
            document.documentElement.style.overflow = originalOverflow;
        };
    }, []);

    // Calculate container height dynamically for virtual scrolling
    useEffect(() => {
        const calculateHeight = () => {
            // Calculate available height: viewport height - header - padding - controls
            const viewportHeight = window.innerHeight;
            const headerHeight = 64; // Approximate header height
            const paddingAndControls = 200; // Padding, title, controls space
            const availableHeight = viewportHeight - headerHeight - paddingAndControls;
            setContainerHeight(Math.max(400, availableHeight)); // Minimum 400px
        };

        calculateHeight();
        window.addEventListener('resize', calculateHeight);
        return () => window.removeEventListener('resize', calculateHeight);
    }, []);

    if (loading) return <NpcPageSkeleton />;
    if (error) return <ErrorDisplay error={error} retry={fetchDataAgain} />;

    return (
        <div className="flex flex-col flex-1 space-y-6 p-10 pb-4 h-[calc(100vh-4rem)] overflow-hidden">
            <div className="flex flex-col space-y-4">
                <h2 className="text-2xl font-bold tracking-tight">NPCs</h2>
                <div className="flex items-center justify-between">
                    <div className="flex gap-2 items-center">
                        <Button
                            variant="outline"
                            size="icon"
                            onClick={fetchDataAgain}
                            className="hover:bg-accent cursor-pointer"
                            title="Refresh"
                        >
                            <RefreshCw className="h-4 w-4" />
                        </Button>
                    </div>
                    <div className="flex items-center gap-2">
                        <AdvancedNpcActions
                            onCreateShop={() => setIsCreateShopDialogOpen(true)}
                            onDeleteAllShops={() => setIsDeleteAllShopsDialogOpen(true)}
                        />
                    </div>
                </div>
            </div>

            <div className="overflow-auto h-[calc(100vh-10rem)] pr-4">
                <ErrorBoundary
                    fallback={({ error, resetError }) => (
                        <div className="flex flex-col items-center justify-center py-10 space-y-4">
                            <div className="text-center">
                                <h3 className="text-lg font-medium text-destructive">Error Loading NPCs</h3>
                                <p className="text-sm text-muted-foreground mt-1">
                                    {error.message || 'An unexpected error occurred while loading the NPC grid.'}
                                </p>
                            </div>
                            <div className="flex gap-2">
                                <Button onClick={resetError} variant="outline">
                                    <RefreshCw className="h-4 w-4 mr-2" />
                                    Try Again
                                </Button>
                                <Button onClick={fetchDataAgain} variant="default">
                                    <RefreshCw className="h-4 w-4 mr-2" />
                                    Reload All Data
                                </Button>
                            </div>
                        </div>
                    )}
                    onError={(error, errorInfo) => {
                        handleError(error, 0, { 
                            context: 'npc_grid_rendering',
                            componentStack: errorInfo.componentStack,
                        });
                    }}
                >
                    <VirtualizedNpcGrid
                        npcs={npcsWithMetadata}
                        isLoading={loading || (isMetadataLoading && npcs.length === 0)}
                        containerHeight={containerHeight - 80} // Account for padding and headers
                        onBulkUpdateShop={(npcId) => {
                            setSelectedNpcId(npcId);
                            setIsBulkUpdateShopDialogOpen(true);
                        }}
                        enableVirtualization={true}
                        itemHeight={200} // Approximate height based on NPC card design
                        overscan={2} // Render 2 extra rows for smooth scrolling
                    />
                </ErrorBoundary>
            </div>

            {/* Dynamic dialogs - only loaded when needed */}
            <NpcDialogs
                isCreateShopDialogOpen={isCreateShopDialogOpen}
                setIsCreateShopDialogOpen={setIsCreateShopDialogOpen}
                isDeleteAllShopsDialogOpen={isDeleteAllShopsDialogOpen}
                setIsDeleteAllShopsDialogOpen={setIsDeleteAllShopsDialogOpen}
                isBulkUpdateShopDialogOpen={isBulkUpdateShopDialogOpen}
                setIsBulkUpdateShopDialogOpen={setIsBulkUpdateShopDialogOpen}
                createShopJson={createShopJson}
                setCreateShopJson={setCreateShopJson}
                bulkUpdateShopJson={bulkUpdateShopJson}
                setBulkUpdateShopJson={setBulkUpdateShopJson}
                handleCreateShop={handleCreateShop}
                handleDeleteAllShops={handleDeleteAllShops}
                handleBulkUpdateShop={handleBulkUpdateShop}
            />

            <Toaster richColors />
        </div>
    );
}
