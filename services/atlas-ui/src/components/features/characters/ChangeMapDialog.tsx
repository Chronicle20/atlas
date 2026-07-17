import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { type Character } from "@/types/models/character";
import { useTenant } from "@/context/tenant-context";
import { useCharacterLocation, characterLocationKeys } from "@/lib/hooks/api/useCharacterLocation";
import { locationsService } from "@/services/api/locations.service";
import { toast } from "sonner";

interface ChangeMapDialogProps {
  character: Character;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: () => void;
}

export function ChangeMapDialog({ character, open, onOpenChange, onSuccess }: ChangeMapDialogProps) {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();
  const { data: location } = useCharacterLocation(activeTenant, character.id);
  const currentMapId = location?.attributes.mapId;

  const [mapId, setMapId] = useState<string>(currentMapId != null ? String(currentMapId) : "");
  const [syncedMapId, setSyncedMapId] = useState<number | undefined>(currentMapId);
  const [isLoading, setIsLoading] = useState(false);
  const [validationError, setValidationError] = useState<string>("");

  // Adjust the field when the location query resolves to a new map id (React's
  // "adjust state during render" pattern — avoids a set-state-in-effect and won't
  // clobber in-progress edits on a same-value refetch).
  if (currentMapId != null && currentMapId !== syncedMapId) {
    setSyncedMapId(currentMapId);
    setMapId(String(currentMapId));
  }

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
    if (currentMapId != null && numValue === currentMapId) {
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
    
    // Clear any existing validation errors
    setValidationError("");
    
    if (!activeTenant) {
      toast.error("No active tenant selected");
      return;
    }

    // Validate the input before submission
    const error = validateMapId(mapId);
    if (error) {
      setValidationError(error);
      toast.error("Please fix the validation errors before submitting");
      return;
    }

    const mapIdNumber = parseInt(mapId, 10);

    setIsLoading(true);
    
    try {
      await locationsService.changeMap(character.id, { mapId: mapIdNumber });
      toast.success(`Successfully changed ${character.attributes.name}'s map to ${mapIdNumber}`);

      // Refresh the character's location so the dialog/table reflect the new map.
      queryClient.invalidateQueries({
        queryKey: characterLocationKeys.detail(activeTenant.id, character.id),
      });

      // Reset form state on success
      setMapId(String(mapIdNumber));
      setValidationError("");

      onOpenChange(false);
      onSuccess?.();
    } catch (error: unknown) {
      // Enhanced error handling with more specific messaging
      let errorMessage: string;

      if (error instanceof Error) {
        errorMessage = error.message;
        
        // Add contextual information for specific error types
        if (error.message.includes("Network error")) {
          errorMessage += ". Please check your internet connection and try again.";
        } else if (error.message.includes("Authentication failed")) {
          errorMessage += ". Please refresh the page and try again.";
        } else if (error.message.includes("Permission denied")) {
          errorMessage += ". You may not have the required permissions to perform this action.";
        } else if (error.message.includes("Server error")) {
          errorMessage += ". Please try again later or contact support if the issue persists.";
        } else if (error.message.includes("Invalid map ID")) {
          // This is a validation error from the server, reset to show it in validation
          setValidationError("The map ID is invalid or does not exist");
          errorMessage = "Invalid map ID provided";
        }
      } else {
        // Handle non-Error objects
        errorMessage = "An unexpected error occurred while updating the character map";
      }
      
      toast.error(errorMessage);
      
      // Log error for debugging (only in development)
      if (import.meta.env.DEV) {
        console.error('Map change error:', error);
      }
    } finally {
      setIsLoading(false);
    }
  };

  const handleOpenChange = (newOpen: boolean) => {
    if (!isLoading) {
      onOpenChange(newOpen);
      if (!newOpen) {
        // Reset form when dialog closes
        setMapId(currentMapId != null ? String(currentMapId) : "");
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
              Current map: <strong>{currentMapId != null ? currentMapId : "—"}</strong>
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