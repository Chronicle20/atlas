"use client"

import { useTenant } from "@/context/tenant-context";
import { useEffect, useState } from "react";
import {NPC, fetchNPCs, bulkCreateShops, Shop, deleteAllShops, updateShop, Commodity} from "@/lib/npcs";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { MoreHorizontal, RefreshCw, Upload, Trash2, ShoppingBag } from "lucide-react";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import Link from "next/link";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Textarea } from "@/components/ui/textarea";
import { toast } from "sonner";
import { Toaster } from "@/components/ui/sonner";

export default function Page() {
    const { activeTenant } = useTenant();
    const [npcs, setNpcs] = useState<NPC[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [isBulkCreateDialogOpen, setIsBulkCreateDialogOpen] = useState(false);
    const [isDeleteAllShopsDialogOpen, setIsDeleteAllShopsDialogOpen] = useState(false);
    const [isBulkUpdateShopDialogOpen, setIsBulkUpdateShopDialogOpen] = useState(false);
    const [selectedNpcId, setSelectedNpcId] = useState<number | null>(null);
    const [bulkCreateJson, setBulkCreateJson] = useState("");
    const [bulkUpdateShopJson, setBulkUpdateShopJson] = useState("");

    const fetchDataAgain = () => {
        if (!activeTenant) return;

        setLoading(true);

        fetchNPCs(activeTenant)
            .then((npcData) => {
                setNpcs(npcData);
            })
            .catch((err) => setError(err.message))
            .finally(() => setLoading(false));
    };

    const handleBulkCreateShops = async () => {
        if (!activeTenant) return;

        try {
            const jsonData = JSON.parse(bulkCreateJson);
            await bulkCreateShops(activeTenant, jsonData.data.map((shop: Shop) => ({
                npcId: shop.attributes.npcId,
                commodities: shop.included
            })));

            toast.success("Shops created successfully");
            setIsBulkCreateDialogOpen(false);
            setBulkCreateJson("");
            fetchDataAgain();
        } catch (err) {
            toast.error("Failed to create shops: " + (err instanceof Error ? err.message : String(err)));
        }
    };

    const handleDeleteAllShops = async () => {
        if (!activeTenant) return;

        try {
            await deleteAllShops(activeTenant);
            toast.success("All shops deleted successfully");
            setIsDeleteAllShopsDialogOpen(false);
            fetchDataAgain();
        } catch (err) {
            toast.error("Failed to delete all shops: " + (err instanceof Error ? err.message : String(err)));
        }
    };

    const handleBulkUpdateShop = async () => {
        if (!activeTenant || !selectedNpcId) return;

        try {
            const jsonData = JSON.parse(bulkUpdateShopJson);

            // Extract commodities from the included array if available
            let commoditiesToUpdate: Commodity[] = [];
            if (jsonData.data.included && jsonData.data.included.length > 0) {
                commoditiesToUpdate = jsonData.data.included;
            }

            await updateShop(activeTenant, selectedNpcId, commoditiesToUpdate);
            setIsBulkUpdateShopDialogOpen(false);
            setBulkUpdateShopJson("");
            fetchDataAgain();
            toast.success("Shop updated successfully");
        } catch (err) {
            toast.error("Failed to update shop: " + (err instanceof Error ? err.message : String(err)));
        }
    };

    useEffect(() => {
        fetchDataAgain();
    }, [activeTenant]);

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

    if (loading) return <div>Loading...</div>;
    if (error) return <div>Error: {error}</div>;

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
                        <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                                <Button variant="outline" size="sm">
                                    <MoreHorizontal className="h-4 w-4 mr-2" />
                                    Actions
                                </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="end">
                                <DropdownMenuItem onClick={() => setIsBulkCreateDialogOpen(true)}>
                                    <Upload className="h-4 w-4 mr-2" />
                                    Bulk Create Shops
                                </DropdownMenuItem>
                                <DropdownMenuItem onClick={() => setIsDeleteAllShopsDialogOpen(true)}>
                                    <Trash2 className="h-4 w-4 mr-2" />
                                    Delete All Shops
                                </DropdownMenuItem>
                            </DropdownMenuContent>
                        </DropdownMenu>
                    </div>
                </div>
            </div>

            <div className="overflow-auto h-[calc(100vh-10rem)] pr-4">
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
                    {npcs.map((npc) => (
                        <Card key={npc.id} className="overflow-hidden">
                            <CardHeader className="pb-2 flex justify-between items-start">
                                <CardTitle className="text-lg">NPC #{npc.id}</CardTitle>
                                <DropdownMenu>
                                    <DropdownMenuTrigger asChild>
                                        <Button variant="ghost" className="h-8 w-8 p-0">
                                            <span className="sr-only">Open menu</span>
                                            <MoreHorizontal className="h-4 w-4" />
                                        </Button>
                                    </DropdownMenuTrigger>
                                    <DropdownMenuContent align="end">
                                        {npc.hasShop && (
                                            <DropdownMenuItem onClick={() => {
                                                setSelectedNpcId(npc.id);
                                                setIsBulkUpdateShopDialogOpen(true);
                                            }}>
                                                <Upload className="h-4 w-4 mr-2" />
                                                Bulk Update Shop
                                            </DropdownMenuItem>
                                        )}
                                    </DropdownMenuContent>
                                </DropdownMenu>
                            </CardHeader>
                            <CardContent className="pb-2">
                                <div className="text-sm">
                                    {npc.hasShop ? (
                                        <Button 
                                            variant="default" 
                                            size="sm"
                                            className="cursor-pointer"
                                            asChild
                                            title="Shop Active"
                                        >
                                            <Link href={`/npcs/${npc.id}/shop`}>
                                                <ShoppingBag className="h-4 w-4" />
                                            </Link>
                                        </Button>
                                    ) : (
                                        <Button 
                                            variant="outline" 
                                            size="sm"
                                            className="cursor-not-allowed opacity-50"
                                            disabled
                                            title="Shop Inactive"
                                        >
                                            <ShoppingBag className="h-4 w-4" />
                                        </Button>
                                    )}
                                </div>
                            </CardContent>
                        </Card>
                    ))}
                    {npcs.length === 0 && (
                        <div className="col-span-full text-center py-10">
                            No NPCs found.
                        </div>
                    )}
                </div>
            </div>

            <Dialog open={isBulkCreateDialogOpen} onOpenChange={setIsBulkCreateDialogOpen}>
                <DialogContent className="sm:max-w-[600px]">
                    <DialogHeader>
                        <DialogTitle>Bulk Create Shops</DialogTitle>
                    </DialogHeader>
                    <div className="grid gap-4 py-4">
                        <Textarea
                            placeholder="Paste JSON data here..."
                            value={bulkCreateJson}
                            onChange={(e) => setBulkCreateJson(e.target.value)}
                            className="min-h-[300px] font-mono"
                        />
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setIsBulkCreateDialogOpen(false)}>
                            Cancel
                        </Button>
                        <Button onClick={handleBulkCreateShops}>
                            Create Shops
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            <Dialog open={isDeleteAllShopsDialogOpen} onOpenChange={setIsDeleteAllShopsDialogOpen}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>Delete All Shops</DialogTitle>
                    </DialogHeader>
                    <div className="py-4">
                        <p className="text-destructive font-semibold">Warning: This action cannot be undone.</p>
                        <p>Are you sure you want to delete all shops for the current tenant?</p>
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setIsDeleteAllShopsDialogOpen(false)}>
                            Cancel
                        </Button>
                        <Button variant="destructive" onClick={handleDeleteAllShops}>
                            Delete All Shops
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            <Dialog open={isBulkUpdateShopDialogOpen} onOpenChange={setIsBulkUpdateShopDialogOpen}>
                <DialogContent className="sm:max-w-[600px]">
                    <DialogHeader>
                        <DialogTitle>Bulk Update Shop</DialogTitle>
                    </DialogHeader>
                    <div className="grid gap-4 py-4">
                        <Textarea
                            placeholder="Paste JSON data here..."
                            value={bulkUpdateShopJson}
                            onChange={(e) => setBulkUpdateShopJson(e.target.value)}
                            className="min-h-[300px] font-mono"
                        />
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setIsBulkUpdateShopDialogOpen(false)}>
                            Cancel
                        </Button>
                        <Button onClick={handleBulkUpdateShop}>
                            Update Shop
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            <Toaster richColors />
        </div>
    );
}
