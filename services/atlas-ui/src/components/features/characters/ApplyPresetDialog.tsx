import { useEffect, useMemo } from "react";
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
import { CharacterRenderer } from "@/components/features/characters/CharacterRenderer";
import { useTenantConfiguration } from "@/lib/hooks/api/useTenants";
import { useNameValidity } from "@/lib/hooks/api/useNameValidity";
import { useCreateCharacterFromPreset } from "@/lib/hooks/api/useCharacterFromPresetMutation";
import { useServices } from "@/lib/hooks/api/useServices";
import { isChannelService } from "@/services/api";
import { synthesizeEquippedAssetsFromTemplateIds } from "@/lib/utils/maplestory";
import { cn } from "@/lib/utils";
import { toast } from "sonner";
import type { Tenant } from "@/types/models/tenant";
import type { Character } from "@/types/models/character";
import type { TenantConfigAttributes } from "@/services/api";
import {
  applyPresetSchema,
  type ApplyPresetFormValues,
} from "@/lib/schemas/apply-preset.schema";
import { createErrorFromUnknown } from "@/types/api/errors";

type FormValues = ApplyPresetFormValues;

interface ApplyPresetDialogProps {
  tenant: Tenant;
  accountId: number;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

type CharacterPreset =
  TenantConfigAttributes["characters"]["presets"][number];

type PresetWithId = CharacterPreset & { id: string };

function presetToCharacter(preset: PresetWithId): Character {
  return {
    id: `preset-${preset.id}`,
    attributes: {
      accountId: 0,
      worldId: 0,
      name: preset.attributes.name,
      level: preset.attributes.level,
      experience: 0,
      gachaponExperience: 0,
      strength: 0,
      dexterity: 0,
      intelligence: 0,
      luck: 0,
      hp: 0,
      maxHp: 0,
      mp: 0,
      maxMp: 0,
      meso: 0,
      hpMpUsed: 0,
      jobId: preset.attributes.jobId,
      skinColor: preset.attributes.skinColor,
      gender: preset.attributes.gender,
      fame: 0,
      hair: preset.attributes.hair + preset.attributes.hairColor,
      face: preset.attributes.face,
      ap: 0,
      sp: "",
      mapId: preset.attributes.mapId,
      spawnPoint: 0,
      gm: 0,
      x: 0,
      y: 0,
      stance: 0,
    },
  };
}

export function ApplyPresetDialog({
  tenant,
  accountId,
  open,
  onOpenChange,
}: ApplyPresetDialogProps) {
  const tenantConfigQuery = useTenantConfiguration(tenant.id);
  const servicesQuery = useServices();

  const serviceableWorldIds = useMemo<Set<number>>(() => {
    const ids = new Set<number>();
    const services = servicesQuery.data ?? [];
    for (const svc of services) {
      if (!isChannelService(svc)) continue;
      for (const channelTenant of svc.attributes.tenants) {
        if (channelTenant.id !== tenant.id) continue;
        for (const w of channelTenant.worlds) {
          ids.add(w.id);
        }
      }
    }
    return ids;
  }, [servicesQuery.data, tenant.id]);

  const worlds = useMemo(() => {
    const all = tenantConfigQuery.data?.attributes?.worlds ?? [];
    return all
      .map((w, i) => ({ world: w, worldId: i }))
      .filter(({ worldId }) => serviceableWorldIds.has(worldId));
  }, [tenantConfigQuery.data, serviceableWorldIds]);

  const presets = (
    tenantConfigQuery.data?.attributes?.characters?.presets ?? []
  ).filter((p): p is PresetWithId => !!p.id);
  const mutation = useCreateCharacterFromPreset(tenant);

  const form = useForm<FormValues>({
    resolver: zodResolver(applyPresetSchema),
    mode: "onChange",
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
      <DialogContent className="max-w-2xl">
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
                  <FormControl>
                    <div
                      role="radiogroup"
                      aria-label="Preset"
                      className="grid grid-cols-2 sm:grid-cols-3 gap-3"
                    >
                      {presets.map((p) => {
                        const selected = field.value === p.id;
                        const character = presetToCharacter(p);
                        const inventory = synthesizeEquippedAssetsFromTemplateIds(
                          p.attributes.equipment.map((e) => e.templateId),
                        );
                        return (
                          <button
                            key={p.id}
                            type="button"
                            role="radio"
                            aria-checked={selected}
                            onClick={() => field.onChange(p.id)}
                            className={cn(
                              "flex flex-col items-center gap-1 rounded-md border p-2 hover:bg-accent/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
                              selected && "ring-2 ring-primary border-primary",
                            )}
                          >
                            <div className="aspect-square w-full flex items-center justify-center bg-muted/30 rounded">
                              <CharacterRenderer
                                character={character}
                                inventory={inventory}
                                size="small"
                                lazy
                                {...(tenant.attributes.region && {
                                  region: tenant.attributes.region,
                                })}
                                {...(tenant.attributes.majorVersion && {
                                  majorVersion: tenant.attributes.majorVersion,
                                })}
                              />
                            </div>
                            <span className="text-xs font-medium text-center leading-tight">
                              {p.attributes.name}
                            </span>
                          </button>
                        );
                      })}
                    </div>
                  </FormControl>
                  {presets.length === 0 && (
                    <FormDescription className="text-muted-foreground">
                      No presets configured. Configure them under Tenant Details &rarr; Character Presets.
                    </FormDescription>
                  )}
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="worldId"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>World</FormLabel>
                  <Select
                    onValueChange={(v) => field.onChange(Number(v))}
                    value={field.value !== undefined ? String(field.value) : ""}
                    disabled={worlds.length === 0}
                  >
                    <FormControl>
                      <SelectTrigger aria-label="World">
                        <SelectValue placeholder="Select a world" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {worlds.map(({ world, worldId: i }) => (
                        <SelectItem key={i} value={String(i)}>
                          {world.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  {worlds.length === 0 && (
                    <FormDescription className="text-muted-foreground">
                      No worlds are serviced for this tenant. Configure a channel
                      service for this tenant under Services to enable a world.
                    </FormDescription>
                  )}
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

