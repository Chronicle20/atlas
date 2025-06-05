"use client"

import { useTenant } from "@/context/tenant-context";
import { useEffect, useState } from "react";
import {NPC, fetchNPCs, bulkCreateShops, Shop} from "@/lib/npcs";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { MoreHorizontal, RefreshCw, Upload } from "lucide-react";
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
    const [bulkCreateJson, setBulkCreateJson] = useState("");

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
                commodities: shop.attributes.commodities
            })));

            toast.success("Shops created successfully");
            setIsBulkCreateDialogOpen(false);
            setBulkCreateJson("");
            fetchDataAgain();
        } catch (err) {
            toast.error("Failed to create shops: " + (err instanceof Error ? err.message : String(err)));
        }
    };

    useEffect(() => {
        fetchDataAgain();
    }, [activeTenant]);

    if (loading) return <div>Loading...</div>;
    if (error) return <div>Error: {error}</div>;

    return (
        <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
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
                            </DropdownMenuContent>
                        </DropdownMenu>
                    </div>
                </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
                {npcs.map((npc) => (
                    <Card key={npc.id} className="overflow-hidden">
                        <CardHeader className="pb-2">
                            <CardTitle className="text-lg">NPC #{npc.id}</CardTitle>
                        </CardHeader>
                        <CardContent className="pb-2">
                            <div className="text-sm">
                                <p>Has Shop: {npc.hasShop ? "Yes" : "No"}</p>
                            </div>
                        </CardContent>
                        <CardFooter className="flex justify-end">
                            <DropdownMenu>
                                <DropdownMenuTrigger asChild>
                                    <Button variant="ghost" className="h-8 w-8 p-0">
                                        <span className="sr-only">Open menu</span>
                                        <MoreHorizontal className="h-4 w-4" />
                                    </Button>
                                </DropdownMenuTrigger>
                                <DropdownMenuContent align="end">
                                    {npc.hasShop && (
                                        <DropdownMenuItem asChild>
                                            <Link href={`/npcs/${npc.id}/shop`}>
                                                View Shop
                                            </Link>
                                        </DropdownMenuItem>
                                    )}
                                </DropdownMenuContent>
                            </DropdownMenu>
                        </CardFooter>
                    </Card>
                ))}
                {npcs.length === 0 && (
                    <div className="col-span-full text-center py-10">
                        No NPCs found.
                    </div>
                )}
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

            <Toaster richColors />
        </div>
    );
}
