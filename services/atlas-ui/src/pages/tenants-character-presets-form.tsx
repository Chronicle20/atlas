import { useEffect, useState } from "react";
import { useForm, useFieldArray, type SubmitHandler, type UseFormReturn } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Form, FormField, FormItem, FormLabel, FormControl, FormMessage } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { useParams } from "react-router-dom";
import { useTenantConfiguration, useUpdateTenantConfiguration } from "@/lib/hooks/api/useTenants";
import { toast } from "sonner";
import { presetsFormSchema, type PresetsFormValues } from "@/pages/character-presets-schema";
import { Plus, Trash, X } from "lucide-react";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";

const DEFAULT_PRESET_ATTRIBUTES = {
    name: "New preset",
    description: "",
    tags: [] as string[],
    jobId: 0,
    gender: 0 as 0 | 1,
    face: 20000,
    hair: 30030,
    hairColor: 0,
    skinColor: 0,
    mapId: 0,
    level: 1,
    meso: 0,
    gm: 0,
    stats: { str: 4, dex: 4, int: 4, luk: 4, hp: 50, mp: 5 },
    defaultName: "",
    equipment: [] as { templateId: number; useAverageStats: boolean }[],
    inventory: [] as { templateId: number; quantity: number }[],
    skills: [] as { skillId: number; level: number }[],
};

export function TenantsPresetsForm() {
    const { id } = useParams();
    const tenantQuery = useTenantConfiguration(String(id ?? ""));
    const updateTenantConfig = useUpdateTenantConfiguration();
    const tenant = tenantQuery.data ?? null;

    const form = useForm<PresetsFormValues>({
        resolver: zodResolver(presetsFormSchema),
        defaultValues: { presets: [] },
    });

    useEffect(() => {
        if (tenant) {
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const presets = (tenant.attributes.characters as any).presets ?? [];
            form.reset({
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                presets: presets.map((p: any) => ({
                    id: p.id,
                    attributes: { ...DEFAULT_PRESET_ATTRIBUTES, ...p.attributes },
                })),
            });
        }
    }, [tenant, form]);

    const { fields, append, remove } = useFieldArray({ control: form.control, name: "presets" });

    const onSubmit: SubmitHandler<PresetsFormValues> = (data) => {
        if (!tenant) return;
        updateTenantConfig.mutate(
            {
                tenant,
                updates: {
                    characters: {
                        ...tenant.attributes.characters,
                        presets: data.presets,
                    },
                },
            },
            {
                onSuccess: () => toast.success("Successfully saved presets."),
                onError: (err: unknown) => {
                    const apiErr = err as { errors?: { meta?: { path?: string }; detail?: string }[]; message?: string };
                    if (apiErr?.errors && Array.isArray(apiErr.errors)) {
                        apiErr.errors.forEach((e) => {
                            const path = e?.meta?.path;
                            const detail = e?.detail ?? "validation error";
                            if (path) {
                                const m = path.match(/^presets\[([^\]]+)\]\.(.+)$/);
                                if (m) {
                                    const presetId = m[1];
                                    const field = m[2];
                                    const idx = (data.presets ?? []).findIndex((p) => p.id === presetId);
                                    if (idx >= 0) {
                                        form.setError(`presets.${idx}.attributes.${field}` as Parameters<typeof form.setError>[0], { message: detail });
                                    }
                                }
                            }
                        });
                        toast.error("Validation failed; see field errors below.");
                    } else {
                        toast.error(apiErr?.message ?? "Save failed");
                    }
                },
            },
        );
    };

    if (tenantQuery.isLoading) return <div className="flex justify-center items-center p-8">Loading tenant configuration...</div>;
    if (!tenant) return <div className="flex justify-center items-center p-8">Tenant configuration not found</div>;

    return (
        <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                <div className="flex justify-end">
                    <Button
                        type="button"
                        onClick={() => append({ id: undefined, attributes: DEFAULT_PRESET_ATTRIBUTES })}
                    >
                        <Plus className="mr-2 h-4 w-4" /> Add preset
                    </Button>
                </div>

                <div className="space-y-2">
                    {fields.map((field, index) => (
                        <PresetItem key={field.id} index={index} form={form} onRemove={() => remove(index)} />
                    ))}
                </div>

                <div className="flex justify-end">
                    <Button type="submit">Save</Button>
                </div>
            </form>
        </Form>
    );
}

interface PresetItemProps {
    index: number;
    form: UseFormReturn<PresetsFormValues>;
    onRemove: () => void;
}

function PresetItem({ index, form, onRemove }: PresetItemProps) {
    const [open, setOpen] = useState(false);
    const name = form.watch(`presets.${index}.attributes.name`);

    return (
        <div className="border rounded-md">
            <button
                type="button"
                className="w-full flex items-center justify-between px-4 py-3 text-left font-medium hover:bg-accent"
                onClick={() => setOpen((v) => !v)}
            >
                <span>{name || `Preset ${index + 1}`}</span>
                <span className="text-xs text-muted-foreground">{open ? "▲" : "▼"}</span>
            </button>

            {open && (
                <div className="px-4 pb-4 space-y-4 border-t pt-4">
                    {/* Identity section */}
                    <section>
                        <h4 className="font-semibold mb-2">Identity</h4>
                        <div className="grid grid-cols-2 gap-2">
                            <FormField
                                control={form.control}
                                name={`presets.${index}.attributes.name`}
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>Name</FormLabel>
                                        <FormControl><Input placeholder="Preset name" {...field} /></FormControl>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />
                            <FormField
                                control={form.control}
                                name={`presets.${index}.attributes.defaultName`}
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>Default Character Name</FormLabel>
                                        <FormControl><Input placeholder="" {...field} /></FormControl>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />
                        </div>
                        <FormField
                            control={form.control}
                            name={`presets.${index}.attributes.description`}
                            render={({ field }) => (
                                <FormItem className="mt-2">
                                    <FormLabel>Description</FormLabel>
                                    <FormControl><Input placeholder="" {...field} /></FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />
                        <TagsField form={form} name={`presets.${index}.attributes.tags`} title="Tags" />
                    </section>

                    {/* Character section */}
                    <section>
                        <h4 className="font-semibold mb-2">Character</h4>
                        <div className="grid grid-cols-3 gap-2">
                            <FormField
                                control={form.control}
                                name={`presets.${index}.attributes.jobId`}
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>Job ID</FormLabel>
                                        <FormControl>
                                            <Input type="number" {...field} onChange={(e) => field.onChange(Number(e.target.value))} />
                                        </FormControl>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />
                            <FormField
                                control={form.control}
                                name={`presets.${index}.attributes.gender`}
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>Gender (0=M, 1=F)</FormLabel>
                                        <FormControl>
                                            <Input type="number" min={0} max={1} {...field} onChange={(e) => field.onChange(Number(e.target.value) as 0 | 1)} />
                                        </FormControl>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />
                            <FormField
                                control={form.control}
                                name={`presets.${index}.attributes.mapId`}
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>Starting Map</FormLabel>
                                        <FormControl>
                                            <Input type="number" {...field} onChange={(e) => field.onChange(Number(e.target.value))} />
                                        </FormControl>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />
                            <FormField
                                control={form.control}
                                name={`presets.${index}.attributes.face`}
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>Face ID</FormLabel>
                                        <FormControl>
                                            <Input type="number" {...field} onChange={(e) => field.onChange(Number(e.target.value))} />
                                        </FormControl>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />
                            <FormField
                                control={form.control}
                                name={`presets.${index}.attributes.hair`}
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>Hair ID</FormLabel>
                                        <FormControl>
                                            <Input type="number" {...field} onChange={(e) => field.onChange(Number(e.target.value))} />
                                        </FormControl>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />
                            <FormField
                                control={form.control}
                                name={`presets.${index}.attributes.hairColor`}
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>Hair Color</FormLabel>
                                        <FormControl>
                                            <Input type="number" {...field} onChange={(e) => field.onChange(Number(e.target.value))} />
                                        </FormControl>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />
                            <FormField
                                control={form.control}
                                name={`presets.${index}.attributes.skinColor`}
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>Skin Color</FormLabel>
                                        <FormControl>
                                            <Input type="number" {...field} onChange={(e) => field.onChange(Number(e.target.value))} />
                                        </FormControl>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />
                        </div>
                    </section>

                    {/* Stats section */}
                    <section>
                        <h4 className="font-semibold mb-2">Stats</h4>
                        <div className="grid grid-cols-3 gap-2">
                            <FormField
                                control={form.control}
                                name={`presets.${index}.attributes.level`}
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>Level</FormLabel>
                                        <FormControl>
                                            <Input type="number" {...field} onChange={(e) => field.onChange(Number(e.target.value))} />
                                        </FormControl>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />
                            <FormField
                                control={form.control}
                                name={`presets.${index}.attributes.meso`}
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>Meso</FormLabel>
                                        <FormControl>
                                            <Input type="number" {...field} onChange={(e) => field.onChange(Number(e.target.value))} />
                                        </FormControl>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />
                            <FormField
                                control={form.control}
                                name={`presets.${index}.attributes.gm`}
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>GM Level</FormLabel>
                                        <FormControl>
                                            <Input type="number" {...field} onChange={(e) => field.onChange(Number(e.target.value))} />
                                        </FormControl>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />
                            {(["str", "dex", "int", "luk", "hp", "mp"] as const).map((stat) => (
                                <FormField
                                    key={stat}
                                    control={form.control}
                                    name={`presets.${index}.attributes.stats.${stat}`}
                                    render={({ field }) => (
                                        <FormItem>
                                            <FormLabel>{stat.toUpperCase()}</FormLabel>
                                            <FormControl>
                                                <Input type="number" {...field} onChange={(e) => field.onChange(Number(e.target.value))} />
                                            </FormControl>
                                            <FormMessage />
                                        </FormItem>
                                    )}
                                />
                            ))}
                        </div>
                    </section>

                    {/* Equipment section */}
                    <EquipmentSection index={index} form={form} />

                    {/* Inventory section */}
                    <InventorySection index={index} form={form} />

                    {/* Skills section */}
                    <SkillsSection index={index} form={form} />

                    <Button type="button" variant="destructive" size="sm" onClick={onRemove}>
                        <Trash className="mr-2 h-4 w-4" /> Delete preset
                    </Button>
                </div>
            )}
        </div>
    );
}

interface SectionProps {
    index: number;
    form: UseFormReturn<PresetsFormValues>;
}

function EquipmentSection({ index, form }: SectionProps) {
    const { fields, append, remove } = useFieldArray({
        control: form.control,
        name: `presets.${index}.attributes.equipment`,
    });

    return (
        <section>
            <div className="flex items-center justify-between mb-2">
                <h4 className="font-semibold">Equipment</h4>
                <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => append({ templateId: 0, useAverageStats: true })}
                >
                    <Plus className="mr-1 h-3 w-3" /> Add
                </Button>
            </div>
            {fields.map((f, i) => (
                <div key={f.id} className="flex gap-2 items-end mb-2">
                    <FormField
                        control={form.control}
                        name={`presets.${index}.attributes.equipment.${i}.templateId`}
                        render={({ field }) => (
                            <FormItem className="flex-1">
                                <FormLabel>Template ID</FormLabel>
                                <FormControl>
                                    <Input type="number" {...field} onChange={(e) => field.onChange(Number(e.target.value))} />
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        )}
                    />
                    <FormField
                        control={form.control}
                        name={`presets.${index}.attributes.equipment.${i}.useAverageStats`}
                        render={({ field }) => (
                            <FormItem className="flex items-center gap-2 mb-2">
                                <FormControl>
                                    <input
                                        type="checkbox"
                                        checked={field.value}
                                        onChange={(e) => field.onChange(e.target.checked)}
                                    />
                                </FormControl>
                                <FormLabel className="!mt-0">Avg Stats</FormLabel>
                                <FormMessage />
                            </FormItem>
                        )}
                    />
                    <Button type="button" variant="ghost" size="icon" onClick={() => remove(i)}>
                        <X className="h-4 w-4" />
                    </Button>
                </div>
            ))}
        </section>
    );
}

function InventorySection({ index, form }: SectionProps) {
    const { fields, append, remove } = useFieldArray({
        control: form.control,
        name: `presets.${index}.attributes.inventory`,
    });

    return (
        <section>
            <div className="flex items-center justify-between mb-2">
                <h4 className="font-semibold">Inventory</h4>
                <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => append({ templateId: 0, quantity: 1 })}
                >
                    <Plus className="mr-1 h-3 w-3" /> Add
                </Button>
            </div>
            {fields.map((f, i) => (
                <div key={f.id} className="flex gap-2 items-end mb-2">
                    <FormField
                        control={form.control}
                        name={`presets.${index}.attributes.inventory.${i}.templateId`}
                        render={({ field }) => (
                            <FormItem className="flex-1">
                                <FormLabel>Template ID</FormLabel>
                                <FormControl>
                                    <Input type="number" {...field} onChange={(e) => field.onChange(Number(e.target.value))} />
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        )}
                    />
                    <FormField
                        control={form.control}
                        name={`presets.${index}.attributes.inventory.${i}.quantity`}
                        render={({ field }) => (
                            <FormItem className="flex-1">
                                <FormLabel>Quantity</FormLabel>
                                <FormControl>
                                    <Input type="number" min={1} {...field} onChange={(e) => field.onChange(Number(e.target.value))} />
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        )}
                    />
                    <Button type="button" variant="ghost" size="icon" onClick={() => remove(i)}>
                        <X className="h-4 w-4" />
                    </Button>
                </div>
            ))}
        </section>
    );
}

function SkillsSection({ index, form }: SectionProps) {
    const { fields, append, remove } = useFieldArray({
        control: form.control,
        name: `presets.${index}.attributes.skills`,
    });

    return (
        <section>
            <div className="flex items-center justify-between mb-2">
                <h4 className="font-semibold">Skills</h4>
                <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => append({ skillId: 0, level: 1 })}
                >
                    <Plus className="mr-1 h-3 w-3" /> Add
                </Button>
            </div>
            {fields.map((f, i) => (
                <div key={f.id} className="flex gap-2 items-end mb-2">
                    <FormField
                        control={form.control}
                        name={`presets.${index}.attributes.skills.${i}.skillId`}
                        render={({ field }) => (
                            <FormItem className="flex-1">
                                <FormLabel>Skill ID</FormLabel>
                                <FormControl>
                                    <Input type="number" {...field} onChange={(e) => field.onChange(Number(e.target.value))} />
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        )}
                    />
                    <FormField
                        control={form.control}
                        name={`presets.${index}.attributes.skills.${i}.level`}
                        render={({ field }) => (
                            <FormItem className="flex-1">
                                <FormLabel>Level</FormLabel>
                                <FormControl>
                                    <Input type="number" min={1} {...field} onChange={(e) => field.onChange(Number(e.target.value))} />
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        )}
                    />
                    <Button type="button" variant="ghost" size="icon" onClick={() => remove(i)}>
                        <X className="h-4 w-4" />
                    </Button>
                </div>
            ))}
        </section>
    );
}

interface TagsFieldProps {
    form: UseFormReturn<PresetsFormValues>;
    name: `presets.${number}.attributes.tags`;
    title: string;
}

function TagsField({ form, name, title }: TagsFieldProps) {
    const tags = form.watch(name) as string[] || [];
    const [isDialogOpen, setDialogOpen] = useState(false);
    const [newTag, setNewTag] = useState("");

    const handleAdd = () => {
        const trimmed = newTag.trim();
        if (trimmed) {
            form.setValue(name, [...tags, trimmed]);
            setNewTag("");
        }
    };

    const handleRemove = (i: number) => {
        form.setValue(name, tags.filter((_, idx) => idx !== i));
    };

    return (
        <div className="border p-2 rounded-md mt-2">
            <FormLabel>{title}</FormLabel>
            <div className="flex flex-row p-2 justify-start gap-2 flex-wrap">
                {tags.map((tag, i) => (
                    <Button key={i} type="button" variant="outline" size="sm" onClick={() => handleRemove(i)}>
                        {tag} <X className="ml-1 h-3 w-3" />
                    </Button>
                ))}
                <Button type="button" variant="outline" size="icon" onClick={() => setDialogOpen(true)}>
                    <Plus />
                </Button>
            </div>
            <Dialog open={isDialogOpen} onOpenChange={setDialogOpen}>
                <DialogContent>
                    <DialogHeader><DialogTitle>Add Tag</DialogTitle></DialogHeader>
                    <Input value={newTag} onChange={(e) => setNewTag(e.target.value)} placeholder="Tag..." />
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setDialogOpen(false)}>Cancel</Button>
                        <Button onClick={handleAdd}>Add</Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </div>
    );
}
