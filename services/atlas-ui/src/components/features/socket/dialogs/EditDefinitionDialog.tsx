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
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { OptionsField } from "@/components/unknown-options";
import { ServicesField } from "@/components/features/socket/dialogs/fields";
import {
  toKnownServices,
  type DialogBaseProps,
} from "@/components/features/socket/dialogs/dialog-base";
import { useSocketMutation } from "@/lib/hooks/api/useSocketObjects";
import { editBinding, type BindingInput } from "@/lib/socket/mutate";
import { formatOpcode } from "@/lib/socket/opcode";
import {
  definitionFormSchemaFor,
  type DefinitionFormValues,
} from "@/lib/schemas/socket-definition";
import { createErrorFromUnknown } from "@/types/api/errors";

export interface EditDefinitionDialogProps extends DialogBaseProps {
  name: string;
  opCodeValue: number;
  initial: BindingInput;
}

function defaultsFrom(
  name: string,
  initial: BindingInput,
): DefinitionFormValues {
  return {
    name,
    opCode: initial.opCode,
    validator: initial.validator ?? "",
    services: toKnownServices(initial.services),
    fname: initial.fname ?? "",
    options: initial.options,
  };
}

/**
 * FR-6.2. Replaces exactly the one binding addressed by
 * `(name, opCodeValue)` via `editBinding` - the name field is rendered
 * READ-ONLY because `editBinding` does not accept a new name; renaming is
 * not a supported operation (see mutate.ts).
 */
export function EditDefinitionDialog({
  open,
  onOpenChange,
  target,
  targetLabel,
  kind,
  name,
  opCodeValue,
  initial,
}: EditDefinitionDialogProps) {
  const mutation = useSocketMutation();
  const schema = useMemo(() => definitionFormSchemaFor(kind), [kind]);
  const form = useForm<DefinitionFormValues>({
    resolver: zodResolver(schema),
    defaultValues: defaultsFrom(name, initial),
  });

  useEffect(() => {
    if (open) form.reset(defaultsFrom(name, initial));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, name, initial]);

  const onSubmit = form.handleSubmit(async (values) => {
    try {
      await mutation.mutateAsync({
        target,
        apply: (cfg) =>
          editBinding(cfg, kind, name, opCodeValue, {
            opCode: values.opCode,
            validator: values.validator,
            services: values.services,
            ...(values.fname ? { fname: values.fname } : {}),
            ...(values.options !== undefined
              ? { options: values.options }
              : {}),
          }),
      });
      toast.success(`Updated ${name} in ${targetLabel}`);
      onOpenChange(false);
    } catch (e) {
      toast.error(createErrorFromUnknown(e).message);
    }
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>
            Edit {name} ({formatOpcode(opCodeValue)}) in {targetLabel}
          </DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={onSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="edit-definition-name">Definition name</Label>
              <Input
                id="edit-definition-name"
                value={name}
                readOnly
                disabled
                className="bg-muted"
              />
              <p className="text-muted-foreground text-xs">
                Renaming is not supported - remove this binding and add it again
                under a new name instead.
              </p>
            </div>
            <FormField
              control={form.control}
              name="opCode"
              label="Operation code"
              type="text"
            />
            {kind === "handler" && (
              <FormField
                control={form.control}
                name="validator"
                label="Validator"
                type="text"
              />
            )}
            <ServicesField control={form.control} name="services" />
            <FormField
              control={form.control}
              name="fname"
              label="Client function name (optional)"
              type="text"
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
              <Button type="submit" disabled={mutation.isPending}>
                Save
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
