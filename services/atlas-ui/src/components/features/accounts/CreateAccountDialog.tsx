// services/atlas-ui/src/components/features/accounts/CreateAccountDialog.tsx
import { useEffect, useRef } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useNavigate } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  Form,
  FormField,
  FormItem,
  FormLabel,
  FormControl,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { useCreateAndPollAccount } from "@/lib/hooks/api/useCreateAndPollAccount";
import { accountKeys } from "@/lib/hooks/api/useAccounts";
import {
  createAccountSchema,
  type CreateAccountFormValues,
} from "@/lib/schemas/create-account.schema";
import type { Tenant } from "@/types/models/tenant";

interface CreateAccountDialogProps {
  tenant: Tenant;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function CreateAccountDialog({
  tenant,
  open,
  onOpenChange,
}: CreateAccountDialogProps) {
  const machine = useCreateAndPollAccount(tenant);
  const navigate = useNavigate();
  const qc = useQueryClient();

  const form = useForm<CreateAccountFormValues>({
    resolver: zodResolver(createAccountSchema),
    defaultValues: { name: "", password: "" },
    mode: "onSubmit",
  });

  // Reset form + machine when the dialog is reopened.
  useEffect(() => {
    if (open) {
      form.reset({ name: "", password: "" });
      machine.reset();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  // Surface duplicate-name as an inline name error.
  const errorAcknowledged = useRef(false);
  useEffect(() => {
    if (machine.status === "error" && machine.errorKind === "duplicate-name") {
      form.setError("name", {
        message: machine.errorMessage ?? "Name already taken.",
      });
      errorAcknowledged.current = true;
      machine.reset();
    } else if (machine.status === "error" && machine.errorKind === "generic") {
      toast.error(machine.errorMessage ?? "Failed to create account");
      errorAcknowledged.current = true;
      machine.reset();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [machine.status, machine.errorKind]);

  // Success: close, toast, navigate.
  useEffect(() => {
    if (machine.status === "success" && machine.accountId != null) {
      qc.invalidateQueries({ queryKey: accountKeys.all });
      const submittedName = form.getValues("name");
      toast.success(`Account ${submittedName} created`);
      onOpenChange(false);
      navigate(`/accounts/${machine.accountId}`);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [machine.status, machine.accountId]);

  const submitting =
    machine.status === "submitting" || machine.status === "polling";
  const showTimeout = machine.status === "timeout";

  const onSubmit = form.handleSubmit(async (values) => {
    await machine.submit(values);
  });

  const handleCancel = () => {
    machine.reset();
    onOpenChange(false);
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) handleCancel();
        else onOpenChange(true);
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create Account</DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={onSubmit} className="space-y-4">
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel htmlFor="create-account-name">Account name</FormLabel>
                  <FormControl>
                    <Input
                      id="create-account-name"
                      autoComplete="off"
                      disabled={submitting}
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="password"
              render={({ field }) => (
                <FormItem>
                  <FormLabel htmlFor="create-account-password">Password</FormLabel>
                  <FormControl>
                    <Input
                      id="create-account-password"
                      type="password"
                      autoComplete="new-password"
                      disabled={submitting}
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            {machine.status === "submitting" && (
              <p className="text-sm text-muted-foreground">Creating account…</p>
            )}
            {machine.status === "polling" && (
              <p className="text-sm text-muted-foreground">
                Waiting for account to appear…
              </p>
            )}
            {showTimeout && (
              <Alert variant="destructive">
                <AlertDescription>
                  Timed out waiting for account to appear.
                </AlertDescription>
              </Alert>
            )}

            <DialogFooter>
              <Button type="button" variant="outline" onClick={handleCancel}>
                Cancel
              </Button>
              {showTimeout ? (
                <Button
                  type="button"
                  onClick={() => {
                    void machine.retry();
                  }}
                >
                  Retry
                </Button>
              ) : (
                <Button type="submit" disabled={submitting}>
                  {submitting ? "Working…" : "Create"}
                </Button>
              )}
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
