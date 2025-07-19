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
  const [validationError, setValidationError] = useState<string>("");
  const { activeTenant } = useTenant();

  const validateMapId = (value: string): string => {
    // Clear any existing validation error
    setValidationError("");
    
    // Check if empty
    if (!value.trim()) {
      return "Map ID is required";
    }
    
    // Check if contains only digits (no decimals, no scientific notation, no negative signs)
    if (!/^\d+$/.test(value.trim())) {
      return "Map ID must contain only numbers";
    }
    
    const numValue = parseInt(value, 10);
    
    // Check for valid integer range (Map IDs are typically positive integers)
    if (numValue < 0) {
      return "Map ID must be a positive number";
    }
    
    // Check for reasonable upper bound (prevent extremely large numbers)
    if (numValue > 2147483647) {
      return "Map ID is too large";
    }
    
    // Check if same as current map
    if (numValue === character.attributes.mapId) {
      return "Character is already on this map";
    }
    
    return "";
  };

  const handleMapIdChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    setMapId(value);
    
    const error = validateMapId(value);
    setValidationError(error);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!activeTenant) {
      toast.error("No active tenant selected");
      return;
    }

    // Validate the input before submission
    const error = validateMapId(mapId);
    if (error) {
      setValidationError(error);
      return;
    }

    const mapIdNumber = parseInt(mapId, 10);

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
        setValidationError("");
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
                type="text"
                value={mapId}
                onChange={handleMapIdChange}
                placeholder="Enter map ID"
                disabled={isLoading}
                required
                className={validationError ? "border-red-500 focus-visible:ring-red-500" : ""}
              />
              {validationError && (
                <p className="text-sm text-red-500 mt-1">{validationError}</p>
              )}
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
            <Button type="submit" disabled={isLoading || !!validationError}>
              {isLoading ? "Updating..." : "Change Map"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}