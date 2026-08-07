import { useEffect, useMemo } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Form } from "@/components/ui/form";
import { FormField } from "@/components/common";
import { OptionsField } from "@/components/unknown-options";
import { ServicesField } from "@/components/features/socket/dialogs/fields";
import type { DialogBaseProps } from "@/components/features/socket/dialogs/dialog-base";
import { useSocketMutation } from "@/lib/hooks/api/useSocketObjects";
import { addBinding } from "@/lib/socket/mutate";
import {
  definitionFormSchemaFor,
  type DefinitionFormValues,
} from "@/lib/schemas/socket-definition";
import { createErrorFromUnknown } from "@/types/api/errors";

export type { DialogBaseProps };

const EMPTY_DEFAULTS: DefinitionFormValues = {
  name: "",
  opCode: "",
  validator: "",
  services: [],
  fname: "",
  options: undefined,
};

/**
 * FR-6.1. Adds a new binding for a new-or-existing name and, via
 * `addBinding`, clears any Unsupported marker for that name in the same
 * write - a name cannot come out of this dialog both Defined and
 * Unsupported. Composes `addBinding` as the splice handed to
 * `useSocketMutation`; never calls a service or builds a PATCH body itself.
 */
export function AddDefinitionDialog({
  open,
  onOpenChange,
  target,
  targetLabel,
  kind,
}: DialogBaseProps) {
  const mutation = useSocketMutation();
  const schema = useMemo(() => definitionFormSchemaFor(kind), [kind]);
  const form = useForm<DefinitionFormValues>({
    resolver: zodResolver(schema),
    defaultValues: EMPTY_DEFAULTS,
  });

  useEffect(() => {
    if (open) form.reset(EMPTY_DEFAULTS);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  const onSubmit = form.handleSubmit(async (values) => {
    try {
      await mutation.mutateAsync({
        target,
        apply: (cfg) =>
          addBinding(cfg, kind, values.name, {
            opCode: values.opCode,
            validator: values.validator,
            services: values.services,
            ...(values.fname ? { fname: values.fname } : {}),
            ...(values.options !== undefined
              ? { options: values.options }
              : {}),
          }),
      });
      toast.success(`Added ${values.name} to ${targetLabel}`);
      onOpenChange(false);
    } catch (e) {
      toast.error(createErrorFromUnknown(e).message);
    }
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Add definition to {targetLabel}</DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={onSubmit} className="space-y-4">
            <FormField
              control={form.control}
              name="name"
              label="Definition name"
              type="text"
              placeholder="PongHandle"
            />
            <FormField
              control={form.control}
              name="opCode"
              label="Operation code"
              type="text"
              placeholder="0x2A"
            />
            {kind === "handler" && (
              <FormField
                control={form.control}
                name="validator"
                label="Validator"
                type="text"
                placeholder="LoggedInValidator"
              />
            )}
            <ServicesField control={form.control} name="services" />
            <FormField
              control={form.control}
              name="fname"
              label="Client function name (optional)"
              type="text"
              placeholder="CLogin::SendCheckPasswordPacket"
            />
            <OptionsField form={form} path="options" />
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => onOpenChange(false)}
              >
                Cancel
              </Button>
              {/*
                Not bare "Add": OptionsField (embedded below) renders its own
                "Add" button for adding an option key, and the two would be
                indistinguishable by accessible name if this one weren't
                more specific.
              */}
              <Button type="submit" disabled={mutation.isPending}>
                Add definition
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
