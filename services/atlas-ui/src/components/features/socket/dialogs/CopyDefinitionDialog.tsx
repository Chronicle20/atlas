import { useEffect, useMemo, useState } from "react";
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
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { OptionsField } from "@/components/unknown-options";
import { ServicesField } from "@/components/features/socket/dialogs/fields";
import {
  toKnownServices,
  type DialogBaseProps,
} from "@/components/features/socket/dialogs/dialog-base";
import { useSocketMutation } from "@/lib/hooks/api/useSocketObjects";
import { copyBindings } from "@/lib/socket/mutate";
import {
  entriesOf,
  type Binding,
  type DefinitionKind,
  type SocketObject,
} from "@/lib/socket/model";
import { formatOpcode } from "@/lib/socket/opcode";
import {
  definitionFormSchemaFor,
  type DefinitionFormValues,
} from "@/lib/schemas/socket-definition";
import { createErrorFromUnknown } from "@/types/api/errors";

export interface CopyDefinitionDialogProps extends DialogBaseProps {
  /** The Definition being copied - fixed by the drawer action that opened this dialog. */
  name: string;
  /** Candidate sources, e.g. every other column on the grid. */
  sourceObjects: SocketObject[];
}

/** The opcode of the first binding `name` has in `obj`, or "" if none. */
function firstOpCodeFor(
  obj: SocketObject | undefined,
  kind: DefinitionKind,
  name: string,
): string {
  return obj ? (entriesOf(obj, kind).get(name)?.[0]?.opCode ?? "") : "";
}

function bindingToDefaults(
  name: string,
  binding: Binding,
): DefinitionFormValues {
  return {
    name,
    opCode: binding.opCode,
    validator: binding.validator ?? "",
    services: toKnownServices(binding.services),
    fname: binding.fname ?? "",
    options: binding.options,
  };
}

/**
 * FR-6.5. Choose a source object, then (when the name has more than one
 * binding there, e.g. `NoOpHandler`'s four opcodes) the specific source
 * binding to load - then edit and confirm. Submits via `copyBindings`, which
 * deep-clones every value, so the written binding shares no structure with
 * the source object's in-memory copy.
 */
export function CopyDefinitionDialog({
  open,
  onOpenChange,
  target,
  targetLabel,
  kind,
  name,
  sourceObjects,
}: CopyDefinitionDialogProps) {
  const mutation = useSocketMutation();

  const candidates = useMemo(
    () =>
      sourceObjects.filter(
        (o) => (entriesOf(o, kind).get(name)?.length ?? 0) > 0,
      ),
    [sourceObjects, kind, name],
  );

  const [sourceKey, setSourceKey] = useState<string>(candidates[0]?.key ?? "");
  const [sourceOpCode, setSourceOpCode] = useState<string>(
    firstOpCodeFor(candidates[0], kind, name),
  );

  // Reset the picker state whenever the dialog transitions from closed to
  // open (adjust state during render per
  // https://react.dev/learn/you-might-not-need-an-effect, instead of an
  // effect that would fire a synchronous setState on mount).
  const [wasOpen, setWasOpen] = useState(open);
  if (open !== wasOpen) {
    setWasOpen(open);
    if (open) {
      const first = candidates[0];
      setSourceKey(first?.key ?? "");
      setSourceOpCode(firstOpCodeFor(first, kind, name));
    }
  }

  const sourceObject = candidates.find((o) => o.key === sourceKey);
  const sourceBindings = sourceObject
    ? (entriesOf(sourceObject, kind).get(name) ?? [])
    : [];
  const selectedBinding =
    sourceBindings.find((b) => b.opCode === sourceOpCode) ?? sourceBindings[0];

  /** Selecting a different source object also picks its first binding, in one user action. */
  const handleSourceChange = (key: string) => {
    setSourceKey(key);
    setSourceOpCode(
      firstOpCodeFor(
        candidates.find((o) => o.key === key),
        kind,
        name,
      ),
    );
  };

  const schema = useMemo(() => definitionFormSchemaFor(kind), [kind]);
  const form = useForm<DefinitionFormValues>({
    resolver: zodResolver(schema),
    defaultValues: selectedBinding
      ? bindingToDefaults(name, selectedBinding)
      : {
          name,
          opCode: "",
          validator: "",
          services: [],
          fname: "",
          options: undefined,
        },
  });

  // Reload the form whenever the dialog reopens or the chosen source
  // binding changes - form.reset() is an imperative call into
  // react-hook-form's external store (not a React state setter), so it
  // stays in an effect rather than the render-time adjustment above.
  useEffect(() => {
    if (selectedBinding) form.reset(bindingToDefaults(name, selectedBinding));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, sourceKey, sourceOpCode]);

  const onSubmit = form.handleSubmit(async (values) => {
    try {
      await mutation.mutateAsync({
        target,
        apply: (cfg) =>
          copyBindings(cfg, kind, name, [
            {
              opCode: values.opCode,
              validator: values.validator,
              services: values.services,
              ...(values.fname ? { fname: values.fname } : {}),
              ...(values.options !== undefined
                ? { options: values.options }
                : {}),
            },
          ]),
      });
      toast.success(`Copied ${name} into ${targetLabel}`);
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
            Copy {name} into {targetLabel}
          </DialogTitle>
        </DialogHeader>

        {candidates.length === 0 ? (
          <p className="text-muted-foreground text-sm">
            No other object defines {name} to copy from.
          </p>
        ) : (
          <>
            <div className="space-y-2">
              <Label htmlFor="copy-source-object">Source object</Label>
              <Select value={sourceKey} onValueChange={handleSourceChange}>
                <SelectTrigger id="copy-source-object">
                  <SelectValue placeholder="Select a source object" />
                </SelectTrigger>
                <SelectContent>
                  {candidates.map((o) => (
                    <SelectItem key={o.key} value={o.key}>
                      {o.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {sourceBindings.length > 1 && (
              <div className="space-y-2">
                <Label htmlFor="copy-source-binding">Source binding</Label>
                <Select value={sourceOpCode} onValueChange={setSourceOpCode}>
                  <SelectTrigger id="copy-source-binding">
                    <SelectValue placeholder="Select a binding" />
                  </SelectTrigger>
                  <SelectContent>
                    {sourceBindings.map((b) => (
                      <SelectItem key={b.opCode} value={b.opCode}>
                        {b.opCodeValue !== null
                          ? formatOpcode(b.opCodeValue)
                          : b.opCode}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}

            <Form {...form}>
              <form onSubmit={onSubmit} className="space-y-4">
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
                    Copy
                  </Button>
                </DialogFooter>
              </form>
            </Form>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}
