import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
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
  FormDescription,
} from "@/components/ui/form";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { useTenantConfiguration } from "@/lib/hooks/api/useTenants";
import { useNameValidity } from "@/lib/hooks/api/useNameValidity";
import { useCreateCharacterFromPreset } from "@/lib/hooks/api/useCharacterFromPresetMutation";
import { toast } from "sonner";
import type { Tenant } from "@/types/models/tenant";
import { applyPresetSchema, type ApplyPresetFormValues } from "@/lib/schemas/apply-preset.schema";
import { createErrorFromUnknown } from "@/types/api/errors";

type FormValues = ApplyPresetFormValues;

interface ApplyPresetDialogProps {
  tenant: Tenant;
  accountId: number;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ApplyPresetDialog({ tenant, accountId, open, onOpenChange }: ApplyPresetDialogProps) {
  const tenantConfigQuery = useTenantConfiguration(tenant.id);
  const presets = (tenantConfigQuery.data?.attributes?.characters?.presets ?? []).filter(
    (p): p is typeof p & { id: string } => !!p.id,
  );
  const mutation = useCreateCharacterFromPreset(tenant);

  const form = useForm<FormValues>({
    resolver: zodResolver(applyPresetSchema),
    defaultValues: { presetId: "", worldId: 0, name: "" },
  });

  const name = form.watch("name");
  const worldId = form.watch("worldId");

  const validityQuery = useNameValidity(tenant, name, worldId, {
    enabled: !!name && name.length >= 3,
  });
  const validity = validityQuery.data;

  // Reset form on open/close
  useEffect(() => {
    if (open) {
      form.reset({ presetId: "", worldId: 0, name: "" });
    }
  }, [open, form]);

  const submitDisabled =
    mutation.isPending ||
    !validity ||
    !validity.valid ||
    !form.formState.isValid;

  const onSubmit = form.handleSubmit((values) => {
    mutation.mutate(
      { presetId: values.presetId, accountId, worldId: values.worldId, name: values.name },
      {
        onSuccess: () => {
          toast.success("Character creation started.");
          onOpenChange(false);
        },
        onError: (err: unknown) => {
          const error = err as { status?: number; response?: { status?: number } };
          const status = error?.status ?? error?.response?.status;
          if (status === 409) {
            form.setError("name", { message: "Name already taken." });
          } else {
            toast.error(createErrorFromUnknown(err).message);
          }
        },
      },
    );
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add character from preset</DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={onSubmit} className="space-y-4">
            <FormField
              control={form.control}
              name="presetId"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Preset</FormLabel>
                  <Select onValueChange={field.onChange} value={field.value}>
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder="Select a preset" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {presets.map((p) => (
                        <SelectItem key={p.id} value={p.id}>
                          {p.attributes.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="worldId"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>World ID</FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      value={field.value}
                      onChange={(e) => field.onChange(Number(e.target.value))}
                      min={0}
                      max={255}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Character name</FormLabel>
                  <FormControl>
                    <Input {...field} placeholder="3-12 characters" />
                  </FormControl>
                  {validity && !validity.valid && (
                    <FormDescription className="text-destructive">
                      {validity.detail ?? `Name invalid (${validity.reason})`}
                    </FormDescription>
                  )}
                  {validity?.valid && (
                    <FormDescription>Name is available.</FormDescription>
                  )}
                  <FormMessage />
                </FormItem>
              )}
            />

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={submitDisabled}>
                {mutation.isPending ? "Creating..." : "Apply"}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
