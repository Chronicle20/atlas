"use client"

import {useEffect, useState} from "react";
import {useForm, useFieldArray} from "react-hook-form";
import {Form, FormField, FormItem, FormLabel, FormControl, FormMessage} from "@/components/ui/form";
import {Input} from "@/components/ui/input";
import {Button} from "@/components/ui/button";
import {useParams} from "next/navigation";
import {X, Plus} from "lucide-react"
import {Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle} from "@/components/ui/dialog";
import {fetchTemplates, Template, updateTemplate} from "@/lib/templates";

export function TemplatesForm() {
    const { id } = useParams(); // Get templates ID from URL

    const [template, setTemplate] = useState<Template>();
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);

    useEffect(() => {
        const loadTemplates = async () => {
            try {
                const data : Template[] = await fetchTemplates();
                const template = data.find((t) => t.id === id);
                setTemplate(template);
            } catch (err) {
                setError(err.message);
            } finally {
                setLoading(false);
            }
        };

        loadTemplates();
    }, []);

    const form = useForm({
        defaultValues: {
            templates: template?.attributes.characters.templates.map(template => ({
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
            templates: template?.attributes.characters.templates.map(template => ({
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
    }, [template, form.reset, form]);

    const onSubmit = async (data) => {
        await updateTemplate(template, {
            characters: {
                templates: data.templates,
            },
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
                        <NumbersField templateIndex={index} templateProperty={"faces"} form={form}/>
                        <NumbersField templateIndex={index} templateProperty={"hairs"} form={form}/>
                        <NumbersField templateIndex={index} templateProperty={"hairColors"} form={form}/>
                        <NumbersField templateIndex={index} templateProperty={"skinColors"} form={form}/>
                        <NumbersField templateIndex={index} templateProperty={"tops"} form={form}/>
                        <NumbersField templateIndex={index} templateProperty={"bottoms"} form={form}/>
                        <NumbersField templateIndex={index} templateProperty={"shoes"} form={form}/>
                        <NumbersField templateIndex={index} templateProperty={"weapons"} form={form}/>
                        <NumbersField templateIndex={index} templateProperty={"items"} form={form}/>
                        <NumbersField templateIndex={index} templateProperty={"skills"} form={form}/>
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

// Component to handle numbers for a specific world
function NumbersField({templateIndex, templateProperty, form}) {
    const {fields: fields, append: add, remove: remove} = useFieldArray({
        control: form.control,
        name: `templates.${templateIndex}.${templateProperty}`
    });

    const values = form.watch(`templates.${templateIndex}.${templateProperty}`);

    const [isDialogOpen, setDialogOpen] = useState(false);
    const [newValue, setNewValue] = useState("");

    const handleAdd = () => {
        if (newValue.trim() !== "") {
            add(newValue);
            setNewValue("");
            setDialogOpen(false);
        }
    };

    return (
        <div className="border p-2 rounded-md">
            <FormLabel>{templateProperty ? templateProperty.charAt(0).toUpperCase() + templateProperty.slice(1) : ""}</FormLabel>
            <div className="flex flex-row p-2 justify-start gap-2">
                {fields.map((field, index) => (
                    <div key={field.id} className="flex items-center">
                        <Button type="button" variant="outline" onClick={() => remove(index)}>
                            {values[index]}
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