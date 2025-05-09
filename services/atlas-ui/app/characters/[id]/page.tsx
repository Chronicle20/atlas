"use client"

import {useEffect, useState} from "react"
import {useParams} from "next/navigation"
import {Toaster} from "@/components/ui/sonner"
import {useTenant} from "@/context/tenant-context";
import {Card, CardContent, CardHeader, CardTitle} from "@/components/ui/card";
import {Character, fetchCharacter} from "@/lib/characters";
import {Collapsible, CollapsibleContent, CollapsibleTrigger} from "@/components/ui/collapsible";
import {InventoryResponse, fetchInventory, getCompartmentTypeName, getAssetsForCompartment, deleteAsset, Compartment} from "@/lib/inventory";
import { Button } from "@/components/ui/button";
import { X } from "lucide-react";
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from "@/components/ui/alert-dialog";

export default function CharacterDetailPage() {
    const {id} = useParams()
    const {activeTenant} = useTenant()

    const [character, setCharacter] = useState<Character | null>(null)
    const [inventory, setInventory] = useState<InventoryResponse | null>(null)
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)
    const [deletingAsset, setDeletingAsset] = useState<string | null>(null)
    const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
    const [assetToDelete, setAssetToDelete] = useState<{ compartmentId: string, assetId: string } | null>(null)

    // Function to open delete confirmation dialog
    const openDeleteDialog = (compartmentId: string, assetId: string) => {
        setAssetToDelete({ compartmentId, assetId });
        setDeleteDialogOpen(true);
    };

    // Function to handle asset deletion
    const handleDeleteAsset = async () => {
        if (!activeTenant || !id || !assetToDelete) return;

        try {
            setDeletingAsset(assetToDelete.assetId);
            await deleteAsset(activeTenant, String(id), assetToDelete.compartmentId, assetToDelete.assetId);

            // Refresh inventory data after deletion
            const updatedInventory = await fetchInventory(activeTenant, String(id));
            setInventory(updatedInventory);
        } catch (err) {
            console.error("Failed to delete asset:", err);
        } finally {
            setDeletingAsset(null);
            setDeleteDialogOpen(false);
            setAssetToDelete(null);
        }
    };

    useEffect(() => {
        if (!activeTenant || !id) return

        setLoading(true)

        // Fetch character data
        const characterPromise = fetchCharacter(activeTenant, String(id))
            .then(setCharacter)
            .catch((err) => setError(err.message));

        // Fetch inventory data
        const inventoryPromise = fetchInventory(activeTenant, String(id))
            .then(setInventory)
            .catch((err) => console.error("Failed to fetch inventory:", err));

        // Wait for both requests to complete
        Promise.all([characterPromise, inventoryPromise])
            .finally(() => setLoading(false));

    }, [activeTenant, id])

    if (loading) return <div className="p-4">Loading...</div>
    if (error || !character) return <div className="p-4 text-red-500">Error: {error || "Character not found"}</div>

    // Get compartments from inventory data
    const compartments = inventory?.included.filter(
        (item): item is Compartment => item.type === 'compartments'
    ) || [];

    // Sort compartments by type
    const sortedCompartments = [...compartments].sort((a, b) => a.attributes.type - b.attributes.type);

    return (
        <div className="flex flex-col flex-1 space-y-6 p-10 pb-16 h-screen overflow-auto">
            <div className="items-center justify-between space-y-2">
                <div>
                    <h2 className="text-2xl font-bold tracking-tight">{character.attributes.name}</h2>
                </div>
            </div>
            <div className="flex flex-row gap-6">
                <Card className="w-[100%]">
                    <CardHeader>
                        <CardTitle>Attributes</CardTitle>
                    </CardHeader>
                    <CardContent className="grid grid-cols-4 gap-2 text-sm text-muted-foreground">
                        <div>
                            <strong>World:</strong> {activeTenant?.attributes.worlds[character.attributes.worldId].name}
                        </div>
                        <div><strong>Gender:</strong> {character.attributes.gender}</div>
                        <div><strong>Level:</strong> {character.attributes.level}</div>
                        <div><strong>Experience:</strong> {character.attributes.experience}</div>
                        <div><strong>Strength:</strong> {character.attributes.strength}</div>
                        <div><strong>Dexterity:</strong> {character.attributes.dexterity}</div>
                        <div><strong>Intelligence:</strong> {character.attributes.intelligence}</div>
                        <div><strong>Luck:</strong> {character.attributes.luck}</div>
                    </CardContent>
                </Card>
            </div>

            {/* Inventory Section */}
            {inventory && (
                <div className="space-y-4">
                    <h3 className="text-xl font-bold tracking-tight">Inventory</h3>
                    <div className="grid grid-cols-1 gap-4">
                        {sortedCompartments.map((compartment) => {
                            const assets = getAssetsForCompartment(compartment, inventory.included);
                            return (
                                <Collapsible key={compartment.id} className="border rounded-md">
                                    <CollapsibleTrigger className="flex justify-between items-center w-full p-4 hover:bg-muted/50">
                                        <div className="flex items-center gap-2">
                                            <h4 className="text-lg font-semibold">{getCompartmentTypeName(compartment.attributes.type)}</h4>
                                        </div>
                                        <span className="text-sm text-muted-foreground">
                                            {assets.length} / {compartment.attributes.capacity}
                                        </span>
                                    </CollapsibleTrigger>
                                    <CollapsibleContent className="p-4 pt-0">
                                        <div className="flex flex-wrap gap-3 pt-4">
                                            {assets.length > 0 ? (
                                                assets.map((asset) => (
                                                    <Card key={asset.id} className="overflow-hidden relative py-0 w-[100px]">
                                                        <Button
                                                            variant="ghost" 
                                                            size="icon" 
                                                            className="absolute top-0 right-0"
                                                            onClick={(e) => {
                                                                e.stopPropagation();
                                                                openDeleteDialog(compartment.id, asset.id);
                                                            }}
                                                            disabled={deletingAsset === asset.id}
                                                        >
                                                            <X/>
                                                        </Button>
                                                        <CardHeader className="p-1 pl-3 pb-1 text-left">
                                                            <div>{asset.attributes.slot}</div>
                                                        </CardHeader>
                                                        <CardContent className="p-2 pt-0 text-base text-center">
                                                            <CardTitle>{asset.attributes.templateId}</CardTitle>
                                                        </CardContent>
                                                    </Card>
                                                ))
                                            ) : (
                                                <div className="col-span-full text-center text-muted-foreground py-4">
                                                    No items in this compartment
                                                </div>
                                            )}
                                        </div>
                                    </CollapsibleContent>
                                </Collapsible>
                            );
                        })}
                    </div>
                </div>
            )}

            <Toaster richColors/>

            {/* Delete Confirmation Dialog */}
            <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>Are you sure?</AlertDialogTitle>
                        <AlertDialogDescription>
                            This action cannot be undone. This will permanently delete the item from your inventory.
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                        <AlertDialogAction 
                            onClick={handleDeleteAsset}
                            disabled={deletingAsset !== null}
                        >
                            {deletingAsset ? "Deleting..." : "Delete"}
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </div>
    )
}
