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
import {CharacterTemplate} from "@/lib/templates";
import {updateTenant} from "@/lib/tenants";
import {toast} from "sonner";

interface FormValues {
    templates: CharacterTemplate[];
}

export function TemplatesForm() {
    const {id} = useParams(); // Get tenants ID from URL
    const {tenants} = useTenant()
    let tenant = tenants.find((t) => t.id === id);

    const form = useForm<FormValues>({
        defaultValues: {
            templates: tenant?.attributes.characters.templates.map(template => ({
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
        }
    });

    const {fields, remove} = useFieldArray({
        control: form.control,
        name: "templates"
    });

    useEffect(() => {
        form.reset({
            templates: tenant?.attributes.characters.templates.map(template => ({
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
    }, [tenant, form.reset, form]);

    const onSubmit = async (data: FormValues) => {
        tenant = await updateTenant(tenant, {
            characters: {
                templates: data.templates,
            },
        });
        toast.success("Successfully saved tenant.");
        form.reset({
            templates: tenant?.attributes.characters.templates,
        });
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