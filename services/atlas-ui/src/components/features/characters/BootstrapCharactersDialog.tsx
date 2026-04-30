import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { PresetApplicationFlow } from "./PresetApplicationFlow";
import type { Tenant } from "@/types/models/tenant";

interface BootstrapCharactersDialogProps {
  tenant: Tenant;
  accountId: number;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function BootstrapCharactersDialog({
  tenant,
  accountId,
  open,
  onOpenChange,
}: BootstrapCharactersDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle>Bootstrap characters</DialogTitle>
        </DialogHeader>
        <PresetApplicationFlow
          tenant={tenant}
          accountId={accountId}
          onClose={() => onOpenChange(false)}
        />
      </DialogContent>
    </Dialog>
  );
}
