"use client"

import {useEffect, useState} from "react";
import {FieldValues, Path, PathValue, useFieldArray, useForm, UseFormReturn, useWatch} from "react-hook-form";
import {Form, FormControl, FormField, FormItem, FormLabel, FormMessage} from "@/components/ui/form";
import {Input} from "@/components/ui/input";
import {Button} from "@/components/ui/button";
import {useParams} from "next/navigation";
import {useTenant} from "@/context/tenant-context";
import {Plus, X} from "lucide-react"
import {Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle} from "@/components/ui/dialog";
import {tenantsService} from "@/services/api";
import {CharacterTemplate} from "@/types/models/template";
import {TenantConfig} from "@/types/models/tenant";
import {toast} from "sonner";

interface FormValues {
    templates: CharacterTemplate[];
}

export function TemplatesForm() {
    const {id} = useParams(); // Get tenants ID from URL
    const {fetchTenantConfiguration} = useTenant();
    const [tenant, setTenant] = useState<TenantConfig | null>(null);
    const [loading, setLoading] = useState(true);

    const form = useForm<FormValues>({
        defaultValues: {
            templates: []
        }
    });

    const {fields, remove} = useFieldArray({
        control: form.control,
        name: "templates"
    });

    // Fetch the full tenant configuration
    useEffect(() => {
        const fetchTenant = async () => {
            try {
                setLoading(true);
                if (id) {
                    const tenantConfig = await fetchTenantConfiguration(id as string);
                    setTenant(tenantConfig);

                    // Update form with tenant data
                    form.reset({
                        templates: tenantConfig.attributes.characters.templates.map(template => ({
                            jobIndex: template.jobIndex || 0,
                            subJobIndex: template.subJobIndex || 0,
                            gender: template.gender || 0,
                            mapId: template.mapId || 0,
                            faces: template.faces || [],
                            hairs: template.hairs || [],
                            hairColors: template.hairColors || [],
                            skinColors: template.skinColors || [],
                            tops: template.tops || [],
                            bottoms: template.bottoms || [],
                            shoes: template.shoes || [],
                            weapons: template.weapons || [],
                            items: template.items || [],
                            skills: template.skills || [],
                        }))
                    });
                }
            } catch (error) {
                console.error("Error fetching tenant configuration:", error);
                toast.error("Failed to load tenant configuration");
            } finally {
                setLoading(false);
            }
        };

        fetchTenant();
    }, [id, fetchTenantConfiguration, form]);

    const onSubmit = async (data: FormValues) => {
        if (!tenant) return;

        try {
            const updatedTenant = await tenantsService.updateTenantConfiguration(tenant, {
                characters: {
                    templates: data.templates,
                },
            });

            if (updatedTenant) {
                setTenant(updatedTenant);
                toast.success("Successfully saved tenant configuration.");

                form.reset({
                    templates: updatedTenant.attributes.characters.templates,
                });
            }
        } catch (error) {
            console.error("Error updating tenant configuration:", error);
            toast.error("Failed to update tenant configuration");
        }
    }

    if (loading) {
        return <div className="flex justify-center items-center p-8">Loading tenant configuration...</div>;
    }

    if (!tenant) {
        return <div className="flex justify-center items-center p-8">Tenant configuration not found</div>;
    }

    return (
        <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                {fields.map((field, index) => (
                    <div key={field.id}
                         className="border p-4 rounded-md gap-2 relative flex flex-col justify-stretch">
                        <div className="flex flex-row justify-stretch">
                            <FormField
                                control={form.control}
                                name={`templates.${index}.jobIndex`}
                                render={({field}) => (
                                    <FormItem>
                                        <FormLabel>Job Index</FormLabel>
                                        <FormControl>
                                            <Input placeholder="0" {...field} />
                                        </FormControl>
                                        <FormMessage/>
                                    </FormItem>
                                )}
                            />
                            <FormField
                                control={form.control}
                                name={`templates.${index}.subJobIndex`}
                                render={({field}) => (
                                    <FormItem>
                                        <FormLabel>Sub Job Index</FormLabel>
                                        <FormControl>
                                            <Input placeholder="0" {...field} />
                                        </FormControl>
                                        <FormMessage/>
                                    </FormItem>
                                )}
                            />
                        </div>
                        <FormField
                            control={form.control}
                            name={`templates.${index}.gender`}
                            render={({field}) => (
                                <FormItem>
                                    <FormLabel>Gender</FormLabel>
                                    <FormControl>
                                        <Input placeholder="0" {...field} />
                                    </FormControl>
                                    <FormMessage/>
                                </FormItem>
                            )}
                        />
                        <FormField
                            control={form.control}
                            name={`templates.${index}.mapId`}
                            render={({field}) => (
                                <FormItem>
                                    <FormLabel>Starting Map</FormLabel>
                                    <FormControl>
                                        <Input placeholder="0" {...field} />
                                    </FormControl>
                                    <FormMessage/>
                                </FormItem>
                            )}
                        />

                        <NumbersField form={form} name={`templates.${index}.faces`} title="Faces"/>
                        <NumbersField form={form} name={`templates.${index}.hairs`} title="Hairs"/>
                        <NumbersField form={form} name={`templates.${index}.hairColors`} title="Hair Colors"/>
                        <NumbersField form={form} name={`templates.${index}.skinColors`} title="Skin Colors"/>
                        <NumbersField form={form} name={`templates.${index}.tops`} title="Tops"/>
                        <NumbersField form={form} name={`templates.${index}.bottoms`} title="Bottoms"/>
                        <NumbersField form={form} name={`templates.${index}.shoes`} title="Shoes"/>
                        <NumbersField form={form} name={`templates.${index}.weapons`} title="Weapons"/>
                        <NumbersField form={form} name={`templates.${index}.items`} title="Items"/>
                        <NumbersField form={form} name={`templates.${index}.skills`} title="Skills"/>
                        <Button type="button" className="absolute top-0 right-0" variant="ghost" size="icon"
                                onClick={() => remove(index)}>
                            <X/>
                        </Button>
                    </div>
                ))}
                <div className="flex flex-row gap-2 justify-end">
                    <Button type="submit">Save</Button>
                </div>
            </form>
        </Form>
    );
}

interface NumbersFieldProps<T extends FieldValues> {
    form: UseFormReturn<T>;
    name: Path<T>;
    title: string;
}

function NumbersField<T extends FieldValues>({form, name, title}: NumbersFieldProps<T>) {
    const values = useWatch({control: form.control, name}) as number[] || [];
    const [newValue, setNewValue] = useState("");

    const [isDialogOpen, setDialogOpen] = useState(false);

    const handleAdd = () => {
        if (newValue.trim() !== "" && !isNaN(Number(newValue))) {
            form.setValue(name, [...values, Number(newValue)] as PathValue<T, typeof name>);
            setNewValue("");
        }
    };

    const handleRemove = (index: number) => {
        form.setValue(name, values.filter((_, i) => i !== index) as PathValue<T, typeof name>);
    };

    return (
        <div className="border p-2 rounded-md">
            <FormLabel>{title}</FormLabel>
            <div className="flex flex-row p-2 justify-start gap-2">
                {values.map((value, index) => (
                    <div key={index} className="flex items-center">
                        <Button type="button" variant="outline" onClick={() => handleRemove(index)}>
                            {value}
                            <X/>
                        </Button>
                    </div>
                ))}
                <Button type="button" variant="outline" onClick={() => setDialogOpen(true)} size="icon">
                    <Plus/>
                </Button>
            </div>

            <Dialog open={isDialogOpen} onOpenChange={setDialogOpen}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>Add a New Value</DialogTitle>
                    </DialogHeader>
                    <Input
                        value={newValue}
                        onChange={(e) => setNewValue(e.target.value)}
                        placeholder="Enter value..."
                    />
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setDialogOpen(false)}>Cancel</Button>
                        <Button onClick={handleAdd}>Add</Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </div>
    );
}
