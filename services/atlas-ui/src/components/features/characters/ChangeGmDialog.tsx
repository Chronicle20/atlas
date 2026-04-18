import { useState } from "react";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { charactersService } from "@/services/api/characters.service";
import { Character } from "@/types/models/character";
import { useTenant } from "@/context/tenant-context";
import { toast } from "sonner";

interface ChangeGmDialogProps {
  character: Character;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: () => void;
}

export function ChangeGmDialog({ character, open, onOpenChange, onSuccess }: ChangeGmDialogProps) {
  const currentGm = character.attributes.gm;
  const [gmLevel, setGmLevel] = useState<string>(currentGm > 0 ? "0" : "1");
  const [isLoading, setIsLoading] = useState(false);
  const [validationError, setValidationError] = useState<string>("");
  const { activeTenant } = useTenant();

  const validateGmLevel = (value: string): string => {
    if (!value.trim()) {
      return "GM level is required";
    }

    if (!/^\d+$/.test(value.trim())) {
      return "GM level must be a non-negative number";
    }

    const numValue = parseInt(value, 10);

    if (numValue < 0) {
      return "GM level must be non-negative";
    }

    if (numValue === currentGm) {
      return "Character already has this GM level";
    }

    return "";
  };

  const handleGmLevelChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    setGmLevel(value);
    setValidationError(validateGmLevel(value));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setValidationError("");

    if (!activeTenant) {
      toast.error("No active tenant selected");
      return;
    }

    const error = validateGmLevel(gmLevel);
    if (error) {
      setValidationError(error);
      return;
    }

    const gmValue = parseInt(gmLevel, 10);
    setIsLoading(true);

    try {
      await charactersService.update(activeTenant, character.id, { gm: gmValue });
      const action = gmValue > 0 ? "promoted to GM" : "demoted from GM";
      toast.success(`Successfully ${action}: ${character.attributes.name}`);

      onOpenChange(false);
      onSuccess?.();
    } catch (error: unknown) {
      let errorMessage = "Failed to update GM status";
      if (error instanceof Error) {
        errorMessage = error.message;
      }
      toast.error(errorMessage);

      if (import.meta.env.DEV) {
        console.error('GM change error:', error);
      }
    } finally {
      setIsLoading(false);
    }
  };

  const handleOpenChange = (newOpen: boolean) => {
    if (!isLoading) {
      onOpenChange(newOpen);
      if (!newOpen) {
        setGmLevel(currentGm > 0 ? "0" : "1");
        setValidationError("");
      }
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Change GM Status</DialogTitle>
            <DialogDescription>
              Change the GM status for character <strong>{character.attributes.name}</strong>.
              <br />
              Current GM level: <strong>{currentGm === 0 ? "None (0)" : currentGm}</strong>
              <br />
              <span className="text-xs text-muted-foreground mt-1 block">
                GM characters can use admin commands in-game (@ban, @item, @mob, etc.)
              </span>
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="gmLevel">New GM Level</Label>
              <Input
                id="gmLevel"
                type="text"
                value={gmLevel}
                onChange={handleGmLevelChange}
                placeholder="0 = normal, 1 = GM"
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
            <Button
              type="submit"
              disabled={isLoading || !!validationError}
              variant={parseInt(gmLevel, 10) > 0 ? "default" : "destructive"}
            >
              {isLoading ? "Updating..." : parseInt(gmLevel, 10) > 0 ? "Promote to GM" : "Remove GM"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
