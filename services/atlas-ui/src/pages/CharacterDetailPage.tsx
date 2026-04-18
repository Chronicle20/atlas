import { useParams, Link } from "react-router-dom";
import { useState } from "react";
import { Toaster } from "@/components/ui/sonner";
import { useTenant } from "@/context/tenant-context";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useCharacter, useInvalidateCharacters } from "@/lib/hooks/api/useCharacters";
import { useInventory, useDeleteAsset, useInvalidateInventory } from "@/lib/hooks/api/useInventory";
import { useTenantConfiguration } from "@/lib/hooks/api/useTenants";
import { inventoryService, type Compartment, type Asset } from "@/services/api/inventory.service";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { MapPin, Shield } from "lucide-react";
import { MapCell } from "@/components/map-cell";
import { ChangeMapDialog } from "@/components/features/characters/ChangeMapDialog";
import { ChangeGmDialog } from "@/components/features/characters/ChangeGmDialog";
import { CharacterRenderer } from "@/components/features/characters/CharacterRenderer";
import { InventoryGrid } from "@/components/features/characters/InventoryGrid";
import { QuestStatusTabs } from "@/components/features/quests";
import { ErrorDisplay } from "@/components/common";
import { CharacterDetailSkeleton } from "@/components/common/skeletons/CharacterDetailSkeleton";
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

export function CharacterDetailPage() {
  const { id } = useParams();
  const { activeTenant } = useTenant();

  const characterQuery = useCharacter(activeTenant!, id ?? "");
  const inventoryQuery = useInventory(activeTenant!, id ?? "");
  const tenantConfigQuery = useTenantConfiguration(activeTenant?.id ?? "");
  const deleteAsset = useDeleteAsset();
  const { invalidateAll: invalidateCharacters } = useInvalidateCharacters();
  const { invalidateAll: invalidateInventory } = useInvalidateInventory();

  const character = characterQuery.data ?? null;
  const inventory = inventoryQuery.data ?? null;
  const tenantConfig = tenantConfigQuery.data ?? null;
  const loading = characterQuery.isLoading || inventoryQuery.isLoading || tenantConfigQuery.isLoading;
  const error = characterQuery.error?.message ?? tenantConfigQuery.error?.message ?? null;

  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [assetToDelete, setAssetToDelete] = useState<{ compartmentId: string; assetId: string } | null>(null);
  const [changeMapDialogOpen, setChangeMapDialogOpen] = useState(false);
  const [changeGmDialogOpen, setChangeGmDialogOpen] = useState(false);

  const openDeleteDialog = (compartmentId: string, assetId: string) => {
    setAssetToDelete({ compartmentId, assetId });
    setDeleteDialogOpen(true);
  };

  const handleDeleteAsset = async () => {
    if (!activeTenant || !id || !assetToDelete) return;
    try {
      await deleteAsset.mutateAsync({
        tenant: activeTenant,
        characterId: String(id),
        compartmentId: assetToDelete.compartmentId,
        assetId: assetToDelete.assetId,
      });
      invalidateInventory();
    } catch (err) {
      console.error("Failed to delete asset:", err);
    } finally {
      setDeleteDialogOpen(false);
      setAssetToDelete(null);
    }
  };

  const handleCharacterRefetch = () => {
    invalidateCharacters();
  };

  if (loading) return <CharacterDetailSkeleton />;
  if (error || !character || !tenantConfig) {
    return <ErrorDisplay error={error || "Character or tenant configuration not found"} className="p-4" />;
  }

  const compartments = inventory?.included?.filter(
    (item): item is Compartment => item.type === "compartments",
  ) || [];
  const equippedItems = inventory?.included?.filter(
    (item): item is Asset =>
      item.type === "assets" && "slot" in item.attributes && item.attributes.slot < 0,
  ) || [];
  const sortedCompartments = [...compartments].sort((a, b) => a.attributes.type - b.attributes.type);

  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16 h-screen overflow-auto">
      <div className="items-center justify-between space-y-2">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">{character.attributes.name}</h2>
        </div>
      </div>
      <div className="flex flex-row gap-6">
        <Card className="w-auto flex-shrink-0">
          <CardContent className="flex justify-center pt-4 pb-4">
            <CharacterRenderer
              character={character}
              inventory={equippedItems}
              size="large"
              scale={2}
              {...(activeTenant?.attributes.region && { region: activeTenant.attributes.region })}
              {...(activeTenant?.attributes.majorVersion && { majorVersion: activeTenant.attributes.majorVersion })}
              className="character-renderer"
            />
          </CardContent>
        </Card>

        <Card className="flex-1">
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle>Attributes</CardTitle>
              <div className="flex items-center gap-2">
                {character.attributes.gm > 0 && (
                  <Badge variant="destructive">GM {character.attributes.gm}</Badge>
                )}
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setChangeGmDialogOpen(true)}
                  className="flex items-center gap-2"
                >
                  <Shield className="h-4 w-4" />
                  {character.attributes.gm > 0 ? "Change GM" : "Promote to GM"}
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setChangeMapDialogOpen(true)}
                  className="flex items-center gap-2"
                >
                  <MapPin className="h-4 w-4" />
                  Change Map
                </Button>
              </div>
            </div>
          </CardHeader>
          <CardContent className="grid grid-cols-4 gap-2 text-sm text-muted-foreground">
            <div>
              <strong>World:</strong> {tenantConfig.attributes.worlds[character.attributes.worldId]?.name || "Unknown"}
            </div>
            <div><strong>Gender:</strong> {character.attributes.gender}</div>
            <div><strong>Level:</strong> {character.attributes.level}</div>
            <div><strong>Experience:</strong> {character.attributes.experience}</div>
            <div className="flex items-center gap-1">
              <strong>Map:</strong>
              <Link to={"/maps/" + character.attributes.mapId}>
                <MapCell mapId={String(character.attributes.mapId)} tenant={activeTenant} />
              </Link>
            </div>
            <div><strong>Strength:</strong> {character.attributes.strength}</div>
            <div><strong>Dexterity:</strong> {character.attributes.dexterity}</div>
            <div><strong>Intelligence:</strong> {character.attributes.intelligence}</div>
            <div><strong>Luck:</strong> {character.attributes.luck}</div>
          </CardContent>
        </Card>
      </div>

      {activeTenant && (
        <div className="space-y-4">
          <h3 className="text-xl font-bold tracking-tight">Quest Status</h3>
          <QuestStatusTabs characterId={String(id)} tenant={activeTenant} />
        </div>
      )}

      {inventory && (
        <div className="space-y-4">
          <h3 className="text-xl font-bold tracking-tight">Inventory</h3>
          <div className="grid grid-cols-1 gap-4">
            {sortedCompartments.map((compartment) => {
              try {
                const assets = inventoryService.getAssetsForCompartment(compartment, inventory.included || []);
                return (
                  <Collapsible key={compartment.id} className="border rounded-md">
                    <CollapsibleTrigger className="flex justify-between items-center w-full p-4 hover:bg-muted/50">
                      <div className="flex items-center gap-2">
                        <h4 className="text-lg font-semibold">{inventoryService.getCompartmentTypeName(compartment.attributes.type)}</h4>
                      </div>
                      <span className="text-sm text-muted-foreground">
                        {assets.length} / {compartment.attributes.capacity}
                      </span>
                    </CollapsibleTrigger>
                    <CollapsibleContent className="p-4 pt-0">
                      <div className="pt-4">
                        <InventoryGrid
                          compartment={compartment}
                          assets={assets}
                          onDeleteAsset={(assetId) => {
                            const asset = assets.find(a => a.id === assetId);
                            if (asset) {
                              openDeleteDialog(compartment.id, asset.id);
                            }
                          }}
                          deletingAssetId={deleteAsset.isPending ? assetToDelete?.assetId ?? null : null}
                          isLoading={loading}
                        />
                      </div>
                    </CollapsibleContent>
                  </Collapsible>
                );
              } catch (e) {
                console.error("Error rendering compartment:", compartment.id, e);
                return (
                  <div key={compartment.id} className="border rounded-md p-4 bg-red-50">
                    <p className="text-red-600">Error loading compartment: {inventoryService.getCompartmentTypeName(compartment.attributes?.type || 0)}</p>
                  </div>
                );
              }
            })}
          </div>
        </div>
      )}

      <Toaster richColors />

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
              disabled={deleteAsset.isPending}
            >
              {deleteAsset.isPending ? "Deleting..." : "Delete"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <ChangeGmDialog
        character={character}
        open={changeGmDialogOpen}
        onOpenChange={setChangeGmDialogOpen}
        onSuccess={handleCharacterRefetch}
      />

      <ChangeMapDialog
        character={character}
        open={changeMapDialogOpen}
        onOpenChange={setChangeMapDialogOpen}
        onSuccess={handleCharacterRefetch}
      />
    </div>
  );
}
