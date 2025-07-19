"use client";

import { useState } from "react";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Character, updateCharacter } from "@/lib/characters";
import { useTenant } from "@/context/tenant-context";
import { toast } from "sonner";

interface ChangeMapDialogProps {
  character: Character;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: () => void;
}

export function ChangeMapDialog({ character, open, onOpenChange, onSuccess }: ChangeMapDialogProps) {
  const [mapId, setMapId] = useState<string>(character.attributes.mapId.toString());
  const [isLoading, setIsLoading] = useState(false);
  const { activeTenant } = useTenant();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!activeTenant) {
      toast.error("No active tenant selected");
      return;
    }

    const mapIdNumber = parseInt(mapId);
    if (isNaN(mapIdNumber) || mapIdNumber < 0) {
      toast.error("Map ID must be a valid positive number");
      return;
    }

    if (mapIdNumber === character.attributes.mapId) {
      toast.error("Character is already on this map");
      return;
    }

    setIsLoading(true);
    
    try {
      await updateCharacter(activeTenant, character.id, { mapId: mapIdNumber });
      toast.success(`Successfully changed ${character.attributes.name}'s map to ${mapIdNumber}`);
      onOpenChange(false);
      onSuccess?.();
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : "Failed to update character map";
      toast.error(errorMessage);
    } finally {
      setIsLoading(false);
    }
  };

  const handleOpenChange = (newOpen: boolean) => {
    if (!isLoading) {
      onOpenChange(newOpen);
      if (!newOpen) {
        // Reset form when dialog closes
        setMapId(character.attributes.mapId.toString());
      }
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Change Map Location</DialogTitle>
            <DialogDescription>
              Change the map location for character <strong>{character.attributes.name}</strong>.
              <br />
              Current map: <strong>{character.attributes.mapId}</strong>
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="mapId">New Map ID</Label>
              <Input
                id="mapId"
                type="number"
                value={mapId}
                onChange={(e) => setMapId(e.target.value)}
                placeholder="Enter map ID"
                min="0"
                step="1"
                disabled={isLoading}
                required
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => handleOpenChange(false)}
              disabled={isLoading}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isLoading}>
              {isLoading ? "Updating..." : "Change Map"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}