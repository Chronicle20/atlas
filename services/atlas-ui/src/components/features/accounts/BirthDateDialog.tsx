import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Pencil } from "lucide-react";
import { toast } from "sonner";
import type { Account } from "@/types/models/account";
import type { Tenant } from "@/types/models/tenant";
import { useUpdateAccountBirthDate } from "@/lib/hooks/api/useAccounts";
import {
  birthDateToInput,
  formatBirthDate,
  inputToBirthDate,
} from "@/lib/utils/birth-date";

interface BirthDateDialogProps {
  account: Account;
  tenant: Tenant;
}

/**
 * The account's birth date, with a button opening a one-field editor —
 * the same shape as WalletPanel's "Add balance" dialog.
 *
 * The birth date is the account's second-password credential on pre-v95
 * clients: atlas-channel checks the value the client sends on the cash-shop
 * name-change and world-transfer requests against this stored number, and a
 * stored 0 fails that check outright rather than matching a client-sent 0.
 * So setting it here is what makes those flows usable on a v83 tenant.
 */
export function BirthDateDialog({ account, tenant }: BirthDateDialogProps) {
  const [dialogOpen, setDialogOpen] = useState(false);
  const [value, setValue] = useState("");
  const updateBirthDate = useUpdateAccountBirthDate();

  const openDialog = () => {
    setValue(birthDateToInput(account.attributes.birthDate));
    setDialogOpen(true);
  };

  const handleSave = () => {
    const parsed = inputToBirthDate(value);
    if (parsed === null) {
      toast.error("Please enter a valid date");
      return;
    }

    updateBirthDate.mutate(
      { tenant, account, birthDate: parsed },
      {
        onSuccess: () => {
          toast.success("Birthday updated");
          setDialogOpen(false);
        },
        onError: (error) => {
          toast.error(
            "Failed to update birthday: " +
              (error instanceof Error ? error.message : "Unknown error"),
          );
        },
      },
    );
  };

  return (
    <>
      <div className="flex items-center justify-between gap-2">
        <div>
          <p className="text-muted-foreground">Birthday</p>
          <p className="font-medium">
            {formatBirthDate(account.attributes.birthDate)}
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={openDialog}
          aria-label="Edit birthday"
        >
          <Pencil className="h-4 w-4 mr-1" />
          Edit
        </Button>
      </div>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Set Birthday</DialogTitle>
            <DialogDescription>
              Stored as an integer in yyyymmdd form. Pre-v95 clients use it as
              the account&apos;s second password for cash-shop name changes and
              world transfers.
            </DialogDescription>
          </DialogHeader>
          <div className="py-4">
            <Label htmlFor="birthDate">Birthday</Label>
            <Input
              id="birthDate"
              type="date"
              value={value}
              onChange={(e) => setValue(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") handleSave();
              }}
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>
              Cancel
            </Button>
            <Button onClick={handleSave} disabled={updateBirthDate.isPending}>
              {updateBirthDate.isPending ? "Saving..." : "Save"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
