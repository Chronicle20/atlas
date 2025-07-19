"use client"

import {useEffect, useState} from "react";
import {useForm, useFieldArray, UseFormReturn, FieldValues, Path, useWatch, PathValue, SubmitHandler} from "react-hook-form";
import {Form, FormField, FormItem, FormLabel, FormControl, FormMessage} from "@/components/ui/form";
import {Input} from "@/components/ui/input";
import {Button} from "@/components/ui/button";
import {useParams} from "next/navigation";
import {X, Plus} from "lucide-react"
import {Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle} from "@/components/ui/dialog";
import {fetchTemplates, updateTemplate} from "@/lib/templates";
import type {Template} from "@/types/models/template";
import {toast} from "sonner";

interface FormValues {
    templates: {
        jobIndex: number;
        subJobIndex: number;
        gender: number;
        mapId: number;
        faces: number[];
        hairs: number[];
        hairColors: number[];
        skinColors: number[];
        tops: number[];
        bottoms: number[];
        shoes: number[];
        weapons: number[];
        items: number[];
        skills: number[];
    }[];
}

export function TemplatesForm() {
    const { id } = useParams(); // Get templates ID from URL

    const [template, setTemplate] = useState<Template>();
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);

    const form = useForm<FormValues>({
        defaultValues: {
            templates: []
        }
    });

    useEffect(() => {
        if (!id) return; // Ensure id is available

        setLoading(true); // Show loading while fetching

        fetchTemplates()
            .then((data) => {
                const template = data.find((t) => String(t.id) === String(id));
                setTemplate(template);

                if (template) {
                    const formValues: FormValues = {
                        templates: template.attributes.characters.templates.map(t => ({
                            jobIndex: t.jobIndex,
                            subJobIndex: t.subJobIndex,
                            gender: t.gender,
                            mapId: t.mapId,
                            faces: t.faces,
                            hairs: t.hairs,
                            hairColors: t.hairColors,
                            skinColors: t.skinColors,
                            tops: t.tops,
                            bottoms: t.bottoms,
                            shoes: t.shoes,
                            weapons: t.weapons,
                            items: t.items,
                            skills: t.skills,
                        }))
                    };
                    form.reset(formValues);
                }
            })
            .catch((err) => setError(err.message))
            .finally(() => setLoading(false));
    }, [id, form]);

    const {fields, remove} = useFieldArray({
        control: form.control,
        name: "templates"
    });

    const onSubmit: SubmitHandler<FormValues> = async (data: FormValues) => {
        updateTemplate(template, {
            characters: {
                templates: data.templates,
            },
        }).then(() => {
            toast.success("Successfully saved template.");
        });
    }

    if (loading) return <div>Loading...</div>; // Show loading message while fetching data
    if (error) return <div>Error: {error}</div>; // Show error message if fetching failed

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