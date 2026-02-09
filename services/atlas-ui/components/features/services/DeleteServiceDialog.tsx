"use client";

import { useState } from "react";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";

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
import { buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { useDeleteService } from "@/lib/hooks/api/useServices";
import type { Service } from "@/types/models/service";
import { getServiceTypeDisplayName } from "@/types/models/service";

interface DeleteServiceDialogProps {
  service: Service | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: () => void;
}

/**
 * Confirmation dialog for deleting a service.
 *
 * Shows service ID and type, requires explicit confirmation before deletion.
 */
export function DeleteServiceDialog({
  service,
  open,
  onOpenChange,
  onSuccess,
}: DeleteServiceDialogProps) {
  const [isDeleting, setIsDeleting] = useState(false);
  const deleteService = useDeleteService();

  const handleDelete = async () => {
    if (!service) return;

    setIsDeleting(true);
    try {
      await deleteService.mutateAsync({ id: service.id });
      toast.success("Service deleted successfully");
      onOpenChange(false);
      onSuccess?.();
    } catch (error) {
      console.error("Failed to delete service:", error);
      toast.error("Failed to delete service. Please try again.");
    } finally {
      setIsDeleting(false);
    }
  };

  if (!service) return null;

  const typeName = getServiceTypeDisplayName(service.attributes.type);

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete {typeName}?</AlertDialogTitle>
          <AlertDialogDescription>
            This action cannot be undone. This will permanently delete the
            service configuration.
          </AlertDialogDescription>
        </AlertDialogHeader>

        <div className="rounded-md bg-muted p-3 text-sm">
          <div className="flex flex-col gap-1">
            <div className="flex justify-between">
              <span className="text-muted-foreground">Service ID:</span>
              <span className="font-mono text-xs">{service.id}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Type:</span>
              <span>{typeName}</span>
            </div>
          </div>
        </div>

        <AlertDialogFooter>
          <AlertDialogCancel disabled={isDeleting}>Cancel</AlertDialogCancel>
          <AlertDialogAction
            onClick={(e) => {
              e.preventDefault();
              handleDelete();
            }}
            disabled={isDeleting}
            className={cn(buttonVariants({ variant: "destructive" }))}
          >
            {isDeleting ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Deleting...
              </>
            ) : (
              "Delete Service"
            )}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
