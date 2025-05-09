"use client"

import { useTenant } from "@/context/tenant-context";
import { useEffect, useState } from "react";
import { NPC, fetchNPCs } from "@/lib/npcs";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { MoreHorizontal } from "lucide-react";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import Link from "next/link";
import { RefreshCw } from "lucide-react";

export default function Page() {
    const { activeTenant } = useTenant();
    const [npcs, setNpcs] = useState<NPC[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

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

    useEffect(() => {
        fetchDataAgain();
    }, [activeTenant]);

    if (loading) return <div>Loading...</div>;
    if (error) return <div>Error: {error}</div>;

    return (
        <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
            <div className="flex items-center justify-between">
                <div>
                    <h2 className="text-2xl font-bold tracking-tight">NPCs</h2>
                </div>
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
                                        <DropdownMenuItem>
                                            <Link href={`/npcs/${npc.id}/shop`}>
                                                <span>View Shop</span>
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
        </div>
    );
}